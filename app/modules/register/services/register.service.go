package services

import (
	"auth_service/app/models/dto"
	hs "auth_service/app/modules/hash/services"
	ur "auth_service/app/modules/user/repository"
	upr "auth_service/app/modules/user_pool/repository"
	e "auth_service/common/errors"
	entity "auth_service/infra/entities"

	"errors"
	"slices"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var _ IRegisterService = &RegisterService{}

type RegisterService struct {
	userPoolRepository upr.IUserPoolRepository
	userRepository     ur.IUserRepository
	logger             *zap.Logger
	hashService        hs.IHashService
}

func NewRegisterService(userPoolRepository upr.IUserPoolRepository, userRepository ur.IUserRepository, logger *zap.Logger, hashService hs.IHashService) *RegisterService {

	return &RegisterService{
		userPoolRepository: userPoolRepository,
		userRepository:     userRepository,
		logger:             logger,
		hashService:        hashService,
	}
}

func (this *RegisterService) RegisterWithPassword(app *entity.App, userData dto.RegisterPayloadWithPassoword) (*entity.User, error) {

	// TODO: move this logic to a permission guard or something like that
	if !slices.Contains(app.LoginTypes, "WITH_LOGIN") {
		return nil, e.ThrowNotAllowed("This app does not allow login with password")
	}

	_, err := this.userRepository.FindWhere(entity.User{
		Email:       userData.Email,
		UsersPoolId: app.UsersPool.ID,
	})

	// the Find process should be unsucessfull, if the email exist then it should throw an error
	if err == nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, e.ThrowBadRequest("User already exists")
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

	var profileId string = "f11796ff-cdc1-405d-93aa-e3b5919c92c9"

	// TODO: This should be moved to a profile repository
	switch app.Role {
	case "ADMIN":
		// MANAGER user profile | ADMIN apps always create MANAGER users
		profileId = "6ac35160-764b-42e9-bd1e-2e837384f73f"
	case "USER":
		// CONSUMER user profile | USER apps always create CONSUMER users
		profileId = "f11796ff-cdc1-405d-93aa-e3b5919c92c9"
	}

	var phone string

	if userData.Phone != nil {
		phone = *userData.Phone
	}

	user, err := this.userRepository.Create(entity.User{
		Email:        userData.Email,
		PasswordHash: hashedPassword,
		Name:         userData.Name,
		Phone:        phone,
		Metadata:     userData.Metadata,
		UsersPoolId:  app.UsersPool.ID,
		ProfileId:    profileId,
	})

	this.logger.Info("User created Successfully!", zap.Uint("userID", user.ID), zap.String("email", user.Email))

	if err != nil {
		this.logger.Error("Failed to create user", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to create user")
	}

	createdUser, err := this.userRepository.FindWhere(entity.User{
		ID: user.ID,
	}, "Profile")

	if err != nil {
		this.logger.Error("Failed to find user", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to find user")
	}

	// TODO: replace the createdUser from a User.entity to a DTO RegisterResponse
	return createdUser, nil
}

func (this *RegisterService) RegisterWithOtp() error {

	return e.ThrowNotImplementedError("RegisterWithOtp is not implemented")
}

func (service *RegisterService) Register() error {

	return nil
}
