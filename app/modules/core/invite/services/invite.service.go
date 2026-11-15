package services

import (
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	e "auth_service/app/errors"
	dto "auth_service/app/modules/core/invite/models"
	ir "auth_service/app/modules/core/invite/repository"
	otpdto "auth_service/app/modules/core/otp/models"
	os "auth_service/app/modules/core/otp/services"
	prep "auth_service/app/modules/core/participant/repository"
	ps "auth_service/app/modules/core/participant/services"
	prs "auth_service/app/modules/core/profile/services"
	udto "auth_service/app/modules/core/user/models"
	ur "auth_service/app/modules/core/user/repository"
	us "auth_service/app/modules/core/user/services"
	"auth_service/infra/config"
	entity "auth_service/infra/entities"
	"auth_service/shared/constants"
	"auth_service/shared/email"
	smodels "auth_service/shared/models"
	"auth_service/shared/permissions"
	repo "auth_service/shared/repository"
	"auth_service/shared/storage"
	"auth_service/shared/utils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	DefaultInviteTtl = 7 * 24 * time.Hour

	tokenSeparator = "."
	inviteSender   = "No Reply <contact@vanin.dev>"
)

type InviteService struct {
	inviteRepository ir.IInviteRepository

	// The accept writes a participation, the invite and maybe the user's
	// verify_email in one transaction, so those two writes go to the repositories.
	// Reads stay on the services.
	participantRepository prep.IParticipantRepository
	userRepository        ur.IUserRepository

	participantService ps.IParticipantService
	profileService     prs.IProfileService
	userService        us.IUserService
	otpService         os.IOtpService

	txManager    repo.ITransactionManager
	emailManager *email.EmailManager
	cfg          *config.Config
	logger       *zap.Logger
	urls         storage.IUrlResolver
}

var _ IInviteService = &InviteService{}

func NewInviteService(
	inviteRepository ir.IInviteRepository,
	participantRepository prep.IParticipantRepository,
	userRepository ur.IUserRepository,
	participantService ps.IParticipantService,
	profileService prs.IProfileService,
	userService us.IUserService,
	otpService os.IOtpService,
	txManager repo.ITransactionManager,
	emailManager *email.EmailManager,
	cfg *config.Config,
	logger *zap.Logger,
	urls storage.IUrlResolver,
) *InviteService {
	return &InviteService{
		inviteRepository:      inviteRepository,
		participantRepository: participantRepository,
		userRepository:        userRepository,
		participantService:    participantService,
		profileService:        profileService,
		userService:           userService,
		otpService:            otpService,
		txManager:             txManager,
		emailManager:          emailManager,
		cfg:                   cfg,
		logger:                logger,
		urls:                  urls,
	}
}

func (this *InviteService) Create(
	organizationId string,
	currentOrganization *entity.Organization,
	caller *entity.Participant,
	inviter *entity.User,
	app *entity.App,
	payload *dto.CreateInvite,
) (*dto.InviteResponse, error) {
	if currentOrganization == nil || caller == nil || inviter == nil || app == nil || payload == nil {
		return nil, e.ThrowInternalServerError("Current organization, caller, inviter, app and payload are required")
	}

	if currentOrganization.Profile == nil || caller.Profile == nil {
		return nil, e.ThrowInternalServerError("The current organization or the caller has no profile loaded")
	}

	if organizationId != currentOrganization.ID {
		return nil, e.ThrowPermissionDeniedError("Invites can only be issued for the organization you are currently in")
	}

	if this.cfg.Invite.AcceptUrl == "" {
		return nil, e.ThrowNotImplementedError("No invite accept page is configured on this server")
	}

	// The invite is an OTP of this app, and the invitee is looked up in the pool of
	// the organization: the two have to be the same pool for either to make sense.
	if app.UsersPool.ID != currentOrganization.UsersPoolId {
		return nil, e.ThrowPermissionDeniedError("Invites are issued through an application of the organization's users pool")
	}

	profile, err := this.handableProfile(payload.ProfileId, currentOrganization, caller)

	if err != nil {
		return nil, err
	}

	address := strings.ToLower(strings.TrimSpace(payload.Email))

	if err := this.refuseParticipant(address, currentOrganization); err != nil {
		return nil, err
	}

	this.revokePendingFor(organizationId, address)

	issued, err := this.otpService.Issue(constants.ActionInvite, app, address, this.ttlFor(payload, currentOrganization))

	if err != nil {
		return nil, err
	}

	invite, err := this.inviteRepository.Create(entity.Invite{
		OrganizationId:  organizationId,
		ProfileId:       profile.ID,
		InvitedByUserId: inviter.ID,
		Email:           address,
		OtpId:           issued.Otp.ID,
		ExpiresAt:       issued.Otp.ExpiresAt,
	})

	if err != nil {
		this.logger.Error("Failed to create an invite", zap.String("organization_id", organizationId), zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to create the invite")
	}

	if err := this.send(invite, issued.Secret, currentOrganization, inviter); err != nil {
		return nil, err
	}

	invite.Profile = profile

	return &dto.InviteResponse{Invite: *invite, Status: dto.InviteStatusPending}, nil
}

// handableProfile is the third refusal of UpdateParticipant: nobody hands another
// a profile wider than it holds - see steering/modules/profiles.md.
func (this *InviteService) handableProfile(
	profileId string,
	organization *entity.Organization,
	caller *entity.Participant,
) (*entity.Profile, error) {
	profile, err := this.profileService.FindByIdVisibleTo(profileId, organization.ID)

	if err != nil {
		return nil, err
	}

	if profile == nil {
		return nil, e.ThrowBadRequest(fmt.Sprintf("Profile `%s` not found", profileId), utils.JSON{"field": "profile_id"})
	}

	ceiling, err := permissions.Resolve(organization.Profile.Permissions, caller.Profile.Permissions)

	if err != nil {
		this.logger.Error("Failed to resolve the ceiling of the caller", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to resolve permissions")
	}

	within, err := permissions.IsWithin(profile.Permissions, ceiling)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to compare permissions")
	}

	if !within {
		return nil, e.ThrowPermissionDeniedError(
			fmt.Sprintf("The `%s` profile grants more than you hold in this organization", profile.Key),
		)
	}

	return profile, nil
}

// Only "already participates" is told apart. Whether the address has an account
// in the pool without participating is not revealed to the inviter.
func (this *InviteService) refuseParticipant(address string, organization *entity.Organization) error {
	user, err := this.userService.FindUserInPool(address, organization.UsersPoolId)

	if err != nil {
		this.logger.Error("Failed to look up the invited user", zap.Error(err))
		return e.ThrowInternalServerError("Failed to look up the invited user")
	}

	if user == nil {
		return nil
	}

	participant, err := this.participantService.FindForUserInOrganization(organization.ID, user.ID)

	if err != nil {
		return err
	}

	if participant != nil {
		return e.ThrowConflict(fmt.Sprintf("`%s` already participates in this organization", address))
	}

	return nil
}

// A failure here is logged and not raised: the new invite is still issued, and a
// pending row left behind is revoked by the next one or by hand.
func (this *InviteService) revokePendingFor(organizationId string, address string) {
	pending, err := this.inviteRepository.FindPendingFor(organizationId, address)

	if err != nil {
		this.logger.Error("Failed to look up the pending invites being replaced", zap.Error(err))
		return
	}

	now := time.Now()

	for _, invite := range pending {
		if _, err := this.inviteRepository.Revoke(invite.ID, now); err != nil {
			this.logger.Error("Failed to revoke a replaced invite", zap.String("invite_id", invite.ID), zap.Error(err))
			continue
		}

		this.otpService.Invalidate(invite.OtpId)
	}
}

func (this *InviteService) ttlFor(payload *dto.CreateInvite, organization *entity.Organization) time.Duration {
	if payload.TtlSeconds != nil {
		return time.Duration(*payload.TtlSeconds) * time.Second
	}

	if organization.InviteTtlSeconds != nil {
		return time.Duration(*organization.InviteTtlSeconds) * time.Second
	}

	if this.cfg.Invite.TtlSeconds > 0 {
		return time.Duration(this.cfg.Invite.TtlSeconds) * time.Second
	}

	return DefaultInviteTtl
}

func (this *InviteService) send(invite *entity.Invite, secret string, organization *entity.Organization, inviter *entity.User) error {
	link := this.acceptLink(Token(invite.OtpId, secret), organization.ID)

	sentId, err := this.emailManager.Sender().Send(email.EmailPayload{
		From:    inviteSender,
		To:      []string{invite.Email},
		Subject: fmt.Sprintf("You were invited to %s", organization.Name),
		Body: fmt.Sprintf(
			`<p>%s invited you to join <strong>%s</strong>.</p><p><a href="%s">Accept the invite</a></p><p>This link expires on %s.</p>`,
			html.EscapeString(inviter.Name),
			html.EscapeString(organization.Name),
			html.EscapeString(link),
			invite.ExpiresAt.UTC().Format(time.RFC1123),
		),
	})

	if err != nil {
		this.logger.Error("Failed to send an invite", zap.String("invite_id", invite.ID), zap.Error(err))
		return e.ThrowInternalServerError("Failed to send the invite email")
	}

	this.logger.Info("Invite sent", zap.String("invite_id", invite.ID), zap.String("sent_id", sentId))

	return nil
}

func (this *InviteService) acceptLink(token string, organizationId string) string {
	return strings.NewReplacer(
		"{token}", url.QueryEscape(token),
		"{organization_id}", url.PathEscape(organizationId),
	).Replace(this.cfg.Invite.AcceptUrl)
}

// Token is what the link carries: the id locates the OTP, because its code is
// hashed with a salt and cannot be looked up, and the secret proves it.
func Token(otpId string, secret string) string {
	return otpId + tokenSeparator + secret
}

func parseToken(token string) (otpId string, secret string, ok bool) {
	otpId, secret, found := strings.Cut(strings.TrimSpace(token), tokenSeparator)

	if !found || secret == "" {
		return "", "", false
	}

	if _, err := uuid.Parse(otpId); err != nil {
		return "", "", false
	}

	return otpId, secret, true
}

func (this *InviteService) List(
	organizationId string,
	currentOrganization *entity.Organization,
	query *dto.GetInvitesQuery,
) (*dto.GetInvitesResponse, error) {
	if currentOrganization == nil || query == nil {
		return nil, e.ThrowInternalServerError("Current organization and query are required")
	}

	if organizationId != currentOrganization.ID {
		return nil, e.ThrowPermissionDeniedError("Invites can only be listed for the organization you are currently in")
	}

	status := dto.InviteStatus(query.Status)

	switch status {
	case "", dto.InviteStatusPending, dto.InviteStatusAccepted, dto.InviteStatusRevoked, dto.InviteStatusExpired:
	default:
		return nil, e.ThrowBadRequest(
			"`status` has to be one of pending, accepted, revoked or expired",
			utils.JSON{"source": "query", "field": "status"},
		)
	}

	option := repo.Option{With: []string{"Profile"}}.Paginate(query.Skip, query.Limit)

	invites, err := this.inviteRepository.FindForOrganization(organizationId, status, option)

	if err != nil {
		this.logger.Error("Failed to list invites", zap.String("organization_id", organizationId), zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to list invites")
	}

	total, err := this.inviteRepository.FindForOrganizationCount(organizationId, status)

	if err != nil {
		this.logger.Error("Failed to count invites", zap.String("organization_id", organizationId), zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to count invites")
	}

	now := time.Now()
	data := make([]dto.InviteResponse, 0, len(invites))

	for _, invite := range invites {
		data = append(data, dto.InviteResponse{Invite: invite, Status: StatusOf(&invite, now)})
	}

	return &dto.GetInvitesResponse{
		Total:  total,
		Amount: len(data),
		Skip:   option.Skip,
		Limit:  option.Size(),
		Data:   data,
	}, nil
}

// StatusOf is the predicate of the repository read in Go, so a row and the
// listing filter it came from always agree.
func StatusOf(invite *entity.Invite, now time.Time) dto.InviteStatus {
	switch {
	case invite.AcceptedAt != nil:
		return dto.InviteStatusAccepted
	case invite.RevokedAt != nil:
		return dto.InviteStatusRevoked
	case !invite.ExpiresAt.After(now):
		return dto.InviteStatusExpired
	default:
		return dto.InviteStatusPending
	}
}

func (this *InviteService) Revoke(organizationId string, inviteId string, currentOrganization *entity.Organization) (*dto.InviteResponse, error) {
	if currentOrganization == nil {
		return nil, e.ThrowInternalServerError("Current organization is required")
	}

	if organizationId != currentOrganization.ID {
		return nil, e.ThrowPermissionDeniedError("Invites can only be revoked in the organization you are currently in")
	}

	notFound := e.ThrowNotFound(fmt.Sprintf("Invite `%s` not found in this organization", inviteId))

	if _, err := uuid.Parse(inviteId); err != nil {
		return nil, notFound
	}

	invite, err := this.inviteRepository.FindOne(
		entity.Invite{ID: inviteId, OrganizationId: organizationId},
		repo.Option{With: []string{"Profile"}},
	)

	if err != nil {
		this.logger.Error("Failed to look up an invite", zap.String("invite_id", inviteId), zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to find the invite")
	}

	if invite == nil {
		return nil, notFound
	}

	now := time.Now()

	affected, err := this.inviteRepository.Revoke(invite.ID, now)

	if err != nil {
		this.logger.Error("Failed to revoke an invite", zap.String("invite_id", inviteId), zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to revoke the invite")
	}

	if affected == 0 {
		return nil, e.ThrowConflict(
			fmt.Sprintf("Only a pending invite can be revoked - this one is %s", StatusOf(invite, now)),
		)
	}

	this.otpService.Invalidate(invite.OtpId)

	invite.RevokedAt = &now

	return &dto.InviteResponse{Invite: *invite, Status: dto.InviteStatusRevoked}, nil
}

func (this *InviteService) Resolve(token string, app *entity.App) (*entity.Invite, error) {
	if app == nil {
		return nil, e.ThrowInternalServerError("App is required")
	}

	notFound := e.ThrowNotFound("Invite not found")

	otpId, secret, ok := parseToken(token)

	if !ok {
		return nil, notFound
	}

	invite, err := this.inviteRepository.FindOne(
		entity.Invite{OtpId: otpId},
		repo.Option{With: []string{"Organization", "Profile", "InvitedBy", "Otp"}},
	)

	if err != nil {
		this.logger.Error("Failed to look up an invite", zap.String("otp_id", otpId), zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to find the invite")
	}

	if invite == nil || invite.Otp == nil || invite.Organization == nil || invite.Otp.AppId != app.ID {
		return nil, notFound
	}

	switch StatusOf(invite, time.Now()) {
	case dto.InviteStatusAccepted:
		return nil, e.ThrowBadRequest("This invite was already accepted")
	case dto.InviteStatusRevoked:
		return nil, e.ThrowBadRequest("This invite was revoked or replaced by a newer one")
	case dto.InviteStatusExpired:
		return nil, e.ThrowBadRequest("This invite has expired")
	}

	// Every refusal of the OTP but a 500 is a wrong secret by now, and a wrong
	// secret must read exactly like an invite that does not exist.
	if _, err := this.otpService.ValidateConsumable(
		otpdto.PayloadOtpData{Id: otpId, Code: secret},
		app.ID,
		constants.ActionInvite,
	); err != nil {
		if isServerError(err) {
			return nil, err
		}

		return nil, notFound
	}

	return invite, nil
}

func isServerError(err error) bool {
	appError, ok := err.(*e.AppError)

	return !ok || appError.Code.Second >= 500
}

func (this *InviteService) Describe(token string, app *entity.App) (*dto.InviteDescription, error) {
	invite, err := this.Resolve(token, app)

	if err != nil {
		return nil, err
	}

	account, err := this.userService.FindUserInPool(invite.Email, invite.Organization.UsersPoolId)

	if err != nil {
		this.logger.Error("Failed to look up the invited user", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to look up the invited user")
	}

	description := &dto.InviteDescription{
		Organization: dto.InvitedOrganization{
			Id:        invite.Organization.ID,
			Name:      invite.Organization.Name,
			AvatarUrl: this.urls.PublicUrl(invite.Organization.AvatarKey),
		},
		Email:         invite.Email,
		ExpiresAt:     invite.ExpiresAt,
		AccountExists: account != nil,
	}

	if invite.InvitedBy != nil {
		description.InvitedBy.Name = invite.InvitedBy.Name
	}

	if invite.Profile != nil {
		description.Profile = dto.InvitedProfile{Id: invite.Profile.ID, Name: invite.Profile.Name, Key: invite.Profile.Key}
	}

	return description, nil
}

func (this *InviteService) Accept(token string, app *entity.App) (*dto.AcceptInviteResponse, error) {
	invite, err := this.Resolve(token, app)

	if err != nil {
		return nil, err
	}

	user, err := this.userService.FindUserInPool(invite.Email, invite.Organization.UsersPoolId)

	if err != nil {
		this.logger.Error("Failed to look up the invited user", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to look up the invited user")
	}

	if user == nil {
		return nil, e.ThrowNotFound("No account of this users pool uses the invited email - sign up with `POST /auth/register?method=invite`")
	}

	return this.AcceptFor(invite, user)
}

func (this *InviteService) AcceptFor(invite *entity.Invite, user *entity.User) (*dto.AcceptInviteResponse, error) {
	if invite == nil || user == nil || invite.Organization == nil {
		return nil, e.ThrowInternalServerError("A resolved invite and a user are required")
	}

	if user.UsersPoolId != invite.Organization.UsersPoolId {
		return nil, e.ThrowInternalServerError("The user and the organization of the invite are in different pools")
	}

	// The invariant that a participation points at a profile visible to its
	// organization is checked again: the profile may have changed since the invite.
	profile, err := this.profileService.FindByIdVisibleTo(invite.ProfileId, invite.OrganizationId)

	if err != nil {
		return nil, err
	}

	if profile == nil {
		return nil, e.ThrowBadRequest("The profile of this invite is no longer available to the organization")
	}

	participantId, err := this.consume(invite, user, profile)

	if err != nil {
		return nil, err
	}

	otpId := invite.OtpId

	utils.Detach(this.logger, "invalidate invite otp", func() error {
		this.otpService.Invalidate(otpId)
		return nil
	})

	participant, err := this.participantRepository.FindOne(
		entity.Participant{ID: participantId},
		repo.Option{With: []string{"User", "Profile"}},
	)

	if err != nil || participant == nil {
		this.logger.Error("Failed to read the accepted participation back", zap.String("participant_id", participantId), zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to find participant")
	}

	return &dto.AcceptInviteResponse{
		Organization: *smodels.NewOrganizationView(invite.Organization, this.urls),
		Participant:  *smodels.NewParticipantView(participant, this.urls),
	}, nil
}

// consume is the unit of work: the participation, the invite and the proof of the
// email land together or not at all. An existing participation is kept as is -
// changing its profile is UpdateParticipant, with its own refusals.
func (this *InviteService) consume(invite *entity.Invite, user *entity.User, profile *entity.Profile) (string, error) {
	tx, err := this.txManager.Tx()

	if err != nil {
		return "", e.ThrowInternalServerError("Failed to open transaction")
	}

	defer tx.Rollback()

	option := repo.Option{Tx: tx}

	participant, err := this.participantRepository.FindByOrganizationAndUser(invite.OrganizationId, user.ID, option)

	if err != nil {
		this.logger.Error("Failed to look up a participation", zap.Error(err))
		return "", e.ThrowInternalServerError("Failed to find participant")
	}

	if participant == nil {
		participant, err = this.participantRepository.Create(entity.Participant{
			OrganizationId: invite.OrganizationId,
			UserId:         user.ID,
			ProfileId:      profile.ID,
		}, option)

		if err != nil {
			this.logger.Error("Failed to create participant", zap.Error(err))
			return "", e.ThrowInternalServerError("Failed to create participant")
		}
	}

	affected, err := this.inviteRepository.Accept(invite.ID, time.Now(), option)

	if err != nil {
		this.logger.Error("Failed to accept an invite", zap.String("invite_id", invite.ID), zap.Error(err))
		return "", e.ThrowInternalServerError("Failed to accept the invite")
	}

	if affected == 0 {
		return "", e.ThrowBadRequest("This invite was already used")
	}

	if !user.VerifyEmail {
		verified := true

		if _, err := this.userRepository.Update(entity.User{ID: user.ID}, udto.UserUpdateDao{VerifyEmail: &verified}, option); err != nil {
			this.logger.Error("Failed to mark the email verified", zap.Uint("user_id", user.ID), zap.Error(err))
			return "", e.ThrowInternalServerError("Failed to verify the email")
		}
	}

	if err := tx.Commit(); err != nil {
		return "", e.ThrowInternalServerError("Failed to commit transaction")
	}

	return participant.ID, nil
}
