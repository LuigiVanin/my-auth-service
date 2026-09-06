package services

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	as "auth_service/app/modules/authorize/services"
	odto "auth_service/app/modules/core/organization/models"
	orep "auth_service/app/modules/core/organization/repository"
	otpdto "auth_service/app/modules/core/otp/models"
	os "auth_service/app/modules/core/otp/services"
	prep "auth_service/app/modules/core/participant/repository"
	ps "auth_service/app/modules/core/participant/services"
	ss "auth_service/app/modules/core/session/services"
	udto "auth_service/app/modules/core/user/models"
	ur "auth_service/app/modules/core/user/repository"
	us "auth_service/app/modules/core/user/services"
	dto "auth_service/app/modules/register/models"
	hs "auth_service/app/modules/utils/hash/services"
	repo "auth_service/shared/repository"

	e "auth_service/app/errors"
	entity "auth_service/infra/entities"
	"auth_service/shared/constants"
	sharedDto "auth_service/shared/models"
	"auth_service/shared/utils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var _ IRegisterService = &RegisterService{}

// NOTE: the three writes of a provisioning go straight to the repositories, not
// through the organization and participant services. Those services wrap the same
// calls one to one, and the unit of work needs every one of them on the same
// repo.Option - the layer that owns it. The services stay for the routes that will
// perform these mutations on their own.
type RegisterService struct {
	userRepository         ur.IUserRepository
	organizationRepository orep.IOrganizationRepository
	participantRepository  prep.IParticipantRepository
	userService            us.IUserService
	participantService     ps.IParticipantService
	logger                 *zap.Logger
	hashService            hs.IHashService
	otpService             os.IOtpService
	sessionService         ss.ISessionService
	authorizeService       as.IAuthorizeService
	txManager              repo.ITransactionManager
}

func NewRegisterService(
	userRepository ur.IUserRepository,
	organizationRepository orep.IOrganizationRepository,
	participantRepository prep.IParticipantRepository,
	userService us.IUserService,
	participantService ps.IParticipantService,
	logger *zap.Logger,
	hashService hs.IHashService,
	otpService os.IOtpService,
	sessionService ss.ISessionService,
	authorizeService as.IAuthorizeService,
	txManager repo.ITransactionManager,
) *RegisterService {

	return &RegisterService{
		userRepository:         userRepository,
		organizationRepository: organizationRepository,
		participantRepository:  participantRepository,
		userService:            userService,
		participantService:     participantService,
		logger:                 logger,
		hashService:            hashService,
		otpService:             otpService,
		sessionService:         sessionService,
		authorizeService:       authorizeService,
		txManager:              txManager,
	}
}

// ProvisionUser writes a user together with the organization it owns. The order is
// the only one the cycle between the two tables allows: organization with no owner,
// user pointing at it, owner stamped, participation created.
//
// NOTE: the session stays out of this transaction and is created by the caller.
// SessionService.CreateNew takes no repo.Option and already fires the invalidation
// of the other sessions in a loose goroutine, so it cannot join a unit of work.
func (this *RegisterService) ProvisionUser(
	app *entity.App,
	user entity.User,
) (*ProvisionedUser, error) {
	if app.UsersPool.DefaultProfileId == "" {
		return nil, e.ThrowInternalServerError("The users pool of this app has no default profile")
	}

	tx, err := this.txManager.Tx()

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to open transaction")
	}

	defer tx.Rollback()

	option := repo.Option{Tx: tx}

	organization, err := this.organizationRepository.Create(entity.Organization{
		UsersPoolId: app.UsersPool.ID,
		ProfileId:   app.UsersPool.DefaultProfileId,
		Name:        fmt.Sprintf("%s's organization", user.Name),
	}, option)

	if err != nil {
		this.logger.Error("Failed to create organization", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to create organization")
	}

	user.UsersPoolId = app.UsersPool.ID
	user.CurrentOrganizationId = organization.ID

	created, err := this.userRepository.Create(user, option)

	if err != nil {
		this.logger.Error("Failed to create user", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to create user")
	}

	affected, err := this.organizationRepository.Update(
		entity.Organization{ID: organization.ID},
		odto.OrganizationUpdateDao{OwnerUserId: &created.ID},
		option,
	)

	if err != nil || affected == 0 {
		this.logger.Error("Failed to set the organization owner", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to set the organization owner")
	}

	// The owner participates on the very profile that is the ceiling of its
	// organization, so it holds the most that organization can hold and not a
	// token more. What a signup actually gets is decided in one place:
	// users_pool.default_profile_id.
	if _, err := this.participantRepository.Create(entity.Participant{
		OrganizationId: organization.ID,
		UserId:         created.ID,
		ProfileId:      app.UsersPool.DefaultProfileId,
	}, option); err != nil {
		this.logger.Error("Failed to create participant", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to create participant")
	}

	if err := tx.Commit(); err != nil {
		return nil, e.ThrowInternalServerError("Failed to commit transaction")
	}

	this.logger.Info(
		"User created Successfully!",
		zap.Uint("userID", created.ID),
		zap.String("email", created.Email),
		zap.String("organizationID", organization.ID),
	)

	createdUser, err := this.userRepository.FindOne(
		entity.User{ID: created.ID},
		repo.Option{With: []string{"CurrentOrganization", "CurrentOrganization.Profile"}},
	)

	if err != nil {
		this.logger.Error("Failed to find user", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to find user")
	}

	if createdUser == nil {
		return nil, e.ThrowInternalServerError("User disappeared right after being created")
	}

	participation, err := this.participantService.FindForCurrentOrganization(createdUser)

	if err != nil {
		return nil, err
	}

	return &ProvisionedUser{
		User:        createdUser,
		Participant: participation.Participant,
		Permissions: participation.Permissions,
	}, nil
}

func (this *RegisterService) RegisterWithPassword(app *entity.App, userData dto.RegisterPayloadWithPassoword, request sharedDto.RequestInfo) (*dto.RegisterResponse, error) {

	// TODO: move this logic to a permission guard or something like that
	if !slices.Contains(app.LoginTypes, "WITH_PASSWORD") {
		return nil, e.ThrowNotAllowed("This app does not allow login with password")
	}

	// LEGACY: the logic will change dramatically, email verification will be guard now
	// if app.VerifyEmail {
	// 	return nil, e.ThrowNotAllowed("This app requires email verification, to do that you will need to register with OTP!")
	// }

	// Check if user already exists
	exists, err := this.userService.IsAlreadyCreated(userData.Email, app)
	if err != nil {
		this.logger.Error("Failed to check if user exists", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to check user existence")
	}
	if exists {
		return nil, e.ThrowUserAlreadyExists("User already exists")
	}

	// NOTE: Here I am using a random uuid as Salt. The Salt is stored inside the hashed password in argon2
	// 			 the Compare method take this in consideration.
	hashedPassword, err := this.hashService.HashText(
		userData.Password,
		uuid.New().String(),
	)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to hash password")
	}

	var phone string

	if userData.Phone != nil {
		phone = *userData.Phone
	}

	provisioned, err := this.ProvisionUser(app, entity.User{
		Email:        strings.ToLower(userData.Email),
		PasswordHash: hashedPassword,
		Name:         userData.Name,
		Phone:        phone,
		Metadata:     userData.Metadata,
	})

	if err != nil {
		return nil, err
	}

	// Create session for the newly registered user
	session, err := this.sessionService.CreateNew(app, provisioned.User, request, "WITH_PASSWORD")
	if err != nil {
		this.logger.Error("Failed to create session", zap.Error(err))
		return nil, err
	}

	this.logger.Info("Session created successfully", zap.String("session_id", session.ID))

	// Populate User field in session for CreateAuthorizationCredentials
	session.User = *provisioned.User

	// Generate authorization credentials
	credentials, err := this.authorizeService.CreateAuthorizationCredentials(app, session)
	if err != nil {
		this.logger.Error("Failed to create authorization credentials", zap.Error(err))
		return nil, err
	}

	return &dto.RegisterResponse{
		SessionId:        session.ID,
		AccessToken:      credentials.AccessToken,
		RefreshToken:     credentials.RefreshToken,
		ExpiresAt:        session.ExpiresAt,
		RefreshExpiresAt: session.RefreshExpiresAt,
		User: udto.UserResponse{
			User: *provisioned.User,
			Profile: &udto.ProfileResponse{
				Profile:     *provisioned.Participant.Profile,
				Permissions: provisioned.Permissions,
			},
		},
	}, nil
}

func (this *RegisterService) RegisterWithOtp(app *entity.App, userData dto.RegisterPayloadWithOtp, request sharedDto.RequestInfo) (*dto.RegisterResponse, error) {
	if !slices.Contains(app.LoginTypes, "WITH_OTP") {
		return nil, e.ThrowNotAllowed("This app does not allow login with OTP")
	}

	if userData.Password == "" && slices.Contains(app.LoginTypes, "WITH_PASSWORD") {
		return nil, e.ThrowUnprocessableEntity("Password is required to register with OTP",
			utils.JSON{
				"errors": utils.JSON{
					"field": "Password",
					"tag":   "password",
					"param": "",
					"value": "-",
				},
			},
		)
	}

	// Check if user already exists
	exists, err := this.userService.IsAlreadyCreated(userData.Email, app)
	if err != nil {
		this.logger.Error("Failed to check if user exists", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to check user existence")
	}
	if exists {
		return nil, e.ThrowUserAlreadyExists("User already exists")
	}

	var phone string

	if userData.Phone != nil {
		phone = *userData.Phone
	}

	otp, err := this.otpService.ValidateConsumable(userData.Otp, app.ID, constants.ActionRegister)

	if err != nil {
		return nil, err
	}

	// Validate that the data sent matches the OTP metadata
	if _, err := this.CompareOtpMetadataWithPayload(otp, userData); err != nil {
		return nil, err
	}

	go this.otpService.Invalidate(otp.ID)

	var hashedPassword string

	if userData.Password != "" && slices.Contains(app.LoginTypes, "WITH_PASSWORD") {
		hashedPassword, err = this.hashService.HashText(userData.Password, uuid.New().String())
		if err != nil {
			return nil, e.ThrowInternalServerError("Failed to hash password")
		}
	}

	provisioned, err := this.ProvisionUser(app, entity.User{
		Email:        strings.ToLower(userData.Email),
		Name:         userData.Name,
		Phone:        phone,
		VerifyEmail:  true,
		PasswordHash: hashedPassword,
	})

	if err != nil {
		return nil, err
	}

	// Create session for the newly registered user
	session, err := this.sessionService.CreateNew(app, provisioned.User, request, "WITH_OTP")
	if err != nil {
		this.logger.Error("Failed to create session", zap.Error(err))
		return nil, err
	}

	this.logger.Info("Session created successfully", zap.String("session_id", session.ID))

	// Populate User field in session for CreateAuthorizationCredentials
	session.User = *provisioned.User

	// Generate authorization credentials
	credentials, err := this.authorizeService.CreateAuthorizationCredentials(app, session)
	if err != nil {
		this.logger.Error("Failed to create authorization credentials", zap.Error(err))
		return nil, err
	}

	return &dto.RegisterResponse{
		SessionId:        session.ID,
		AccessToken:      credentials.AccessToken,
		RefreshToken:     credentials.RefreshToken,
		ExpiresAt:        session.ExpiresAt,
		RefreshExpiresAt: session.RefreshExpiresAt,
		User: udto.UserResponse{
			User: *provisioned.User,
			Profile: &udto.ProfileResponse{
				Profile:     *provisioned.Participant.Profile,
				Permissions: provisioned.Permissions,
			},
		},
	}, nil
}

func (this *RegisterService) CompareOtpMetadataWithPayload(otp *entity.Otp, userData dto.RegisterPayloadWithOtp) (*otpdto.OtpRegisterActionMetadata, error) {
	var metadata otpdto.OtpRegisterActionMetadata

	if err := json.Unmarshal(otp.Metadata, &metadata); err != nil {
		return nil, e.ThrowInternalServerError("Failed to parse OTP metadata")
	}

	if !strings.EqualFold(metadata.Payload.Email, userData.Email) {
		return nil, e.ThrowBadRequest("Email provided does not match the one used to generate the OTP")
	}

	if metadata.Payload.Name != userData.Name {
		return nil, e.ThrowBadRequest("Name provided does not match the one used to generate the OTP")
	}

	return &metadata, nil
}

func (service *RegisterService) Register() error {

	return nil
}
