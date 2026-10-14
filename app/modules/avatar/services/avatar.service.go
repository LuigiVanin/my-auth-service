package services

import (
	"fmt"
	"time"

	e "auth_service/app/errors"
	as "auth_service/app/modules/core/app/services"
	os "auth_service/app/modules/core/organization/services"
	us "auth_service/app/modules/core/user/services"
	"auth_service/infra/config"
	entity "auth_service/infra/entities"
	"auth_service/shared/storage"
	"auth_service/shared/utils"

	"go.uber.org/zap"
)

const (
	defaultMaxBytes = 2 * 1024 * 1024
	defaultGrantTtl = 5 * time.Minute
)

// owner is a subject the caller was allowed to reach: where its avatar lives
// and how its key is written. Resolving one is where the scope rule runs.
type owner struct {
	path      string
	avatarKey *string
	write     func(key *string) error
}

type AvatarService struct {
	storage             storage.IStorage
	policy              storage.Policy
	userService         us.IUserService
	organizationService os.IOrganizationService
	appService          as.IAppService
	logger              *zap.Logger
}

var _ IAvatarService = &AvatarService{}

func NewAvatarService(
	files storage.IStorage,
	userService us.IUserService,
	organizationService os.IOrganizationService,
	appService as.IAppService,
	cfg *config.Config,
	logger *zap.Logger,
) *AvatarService {
	return &AvatarService{
		storage:             files,
		policy:              AvatarPolicy(cfg.Avatar),
		userService:         userService,
		organizationService: organizationService,
		appService:          appService,
		logger:              logger,
	}
}

func AvatarPolicy(settings config.AvatarConfig) storage.Policy {
	policy := storage.Policy{
		Prefix:        "avatar/",
		PendingPrefix: "pending/avatar/",
		MaxBytes:      defaultMaxBytes,
		ContentTypes: map[string]string{
			"image/png":  "png",
			"image/jpeg": "jpg",
			"image/webp": "webp",
		},
		GrantTtl:     defaultGrantTtl,
		CacheControl: "public, max-age=31536000, immutable",
	}

	if settings.MaxBytes > 0 {
		policy.MaxBytes = settings.MaxBytes
	}

	if settings.GrantTtlSeconds > 0 {
		policy.GrantTtl = time.Duration(settings.GrantTtlSeconds) * time.Second
	}

	return policy
}

func (this *AvatarService) RequestUserUpload(currentUser *entity.User, contentType string) (*storage.UploadGrant, error) {
	if err := this.requireStorage(); err != nil {
		return nil, err
	}

	return this.grant(this.userOwner(currentUser), contentType)
}

func (this *AvatarService) ConfirmUser(currentUser *entity.User, currentOrganization *entity.Organization, key string) (*entity.User, error) {
	if err := this.requireStorage(); err != nil {
		return nil, err
	}

	if err := this.confirm(this.userOwner(currentUser), key); err != nil {
		return nil, err
	}

	return this.userService.FindById(currentUser, currentOrganization, currentUser.ID)
}

func (this *AvatarService) RemoveUser(currentUser *entity.User, currentOrganization *entity.Organization) (*entity.User, error) {
	if err := this.requireStorage(); err != nil {
		return nil, err
	}

	if err := this.remove(this.userOwner(currentUser)); err != nil {
		return nil, err
	}

	return this.userService.FindById(currentUser, currentOrganization, currentUser.ID)
}

func (this *AvatarService) RequestOrganizationUpload(organizationId string, currentOrganization *entity.Organization, contentType string) (*storage.UploadGrant, error) {
	if err := this.requireStorage(); err != nil {
		return nil, err
	}

	target, err := this.organizationOwner(organizationId, currentOrganization)

	if err != nil {
		return nil, err
	}

	return this.grant(target, contentType)
}

func (this *AvatarService) ConfirmOrganization(organizationId string, currentOrganization *entity.Organization, key string) (*entity.Organization, error) {
	if err := this.requireStorage(); err != nil {
		return nil, err
	}

	target, err := this.organizationOwner(organizationId, currentOrganization)

	if err != nil {
		return nil, err
	}

	if err := this.confirm(target, key); err != nil {
		return nil, err
	}

	return this.findOrganization(organizationId)
}

func (this *AvatarService) RemoveOrganization(organizationId string, currentOrganization *entity.Organization) (*entity.Organization, error) {
	if err := this.requireStorage(); err != nil {
		return nil, err
	}

	target, err := this.organizationOwner(organizationId, currentOrganization)

	if err != nil {
		return nil, err
	}

	if err := this.remove(target); err != nil {
		return nil, err
	}

	return this.findOrganization(organizationId)
}

func (this *AvatarService) RequestAppUpload(appId string, currentOrganization *entity.Organization, contentType string) (*storage.UploadGrant, error) {
	if err := this.requireStorage(); err != nil {
		return nil, err
	}

	target, err := this.appOwner(appId, currentOrganization)

	if err != nil {
		return nil, err
	}

	return this.grant(target, contentType)
}

func (this *AvatarService) ConfirmApp(appId string, currentOrganization *entity.Organization, key string) (*entity.App, error) {
	if err := this.requireStorage(); err != nil {
		return nil, err
	}

	target, err := this.appOwner(appId, currentOrganization)

	if err != nil {
		return nil, err
	}

	if err := this.confirm(target, key); err != nil {
		return nil, err
	}

	return this.appService.FindById(appId, currentOrganization)
}

func (this *AvatarService) RemoveApp(appId string, currentOrganization *entity.Organization) (*entity.App, error) {
	if err := this.requireStorage(); err != nil {
		return nil, err
	}

	target, err := this.appOwner(appId, currentOrganization)

	if err != nil {
		return nil, err
	}

	if err := this.remove(target); err != nil {
		return nil, err
	}

	return this.appService.FindById(appId, currentOrganization)
}

func (this *AvatarService) userOwner(currentUser *entity.User) *owner {
	return &owner{
		path:      fmt.Sprintf("users/%d", currentUser.ID),
		avatarKey: currentUser.AvatarKey,
		write: func(key *string) error {
			return this.userService.SetAvatarKey(currentUser.ID, key)
		},
	}
}

// The same refusal as OrganizationService.UpdateForCaller: the fix is to switch.
func (this *AvatarService) organizationOwner(organizationId string, currentOrganization *entity.Organization) (*owner, error) {
	if currentOrganization == nil {
		return nil, e.ThrowInternalServerError("Current organization is required")
	}

	if organizationId != currentOrganization.ID {
		return nil, e.ThrowPermissionDeniedError(
			"The avatar of an organization can only be changed while you are currently in it",
		)
	}

	return &owner{
		path:      "organizations/" + currentOrganization.ID,
		avatarKey: currentOrganization.AvatarKey,
		write: func(key *string) error {
			return this.organizationService.SetAvatarKey(currentOrganization.ID, key)
		},
	}, nil
}

// AppService.FindById carries the scope rule: an app of another organization is
// a 404.
func (this *AvatarService) appOwner(appId string, currentOrganization *entity.Organization) (*owner, error) {
	app, err := this.appService.FindById(appId, currentOrganization)

	if err != nil {
		return nil, err
	}

	return &owner{
		path:      "apps/" + app.ID,
		avatarKey: app.AvatarKey,
		write: func(key *string) error {
			return this.appService.SetAvatarKey(app.ID, key)
		},
	}, nil
}

func (this *AvatarService) findOrganization(organizationId string) (*entity.Organization, error) {
	organization, err := this.organizationService.FindById(organizationId)

	if err != nil {
		return nil, err
	}

	if organization == nil {
		return nil, e.ThrowNotFound("Organization not found")
	}

	return organization, nil
}

func (this *AvatarService) requireStorage() error {
	if !storage.IsConfigured(this.storage) {
		return e.ThrowNotImplementedError("No file storage is configured on this server")
	}

	return nil
}

func (this *AvatarService) grant(target *owner, contentType string) (*storage.UploadGrant, error) {
	key, err := this.policy.NewPendingKey(target.path, contentType)

	if err != nil {
		return nil, e.ThrowBadRequest(
			fmt.Sprintf("`content_type` %q is not accepted for an avatar", contentType),
			utils.JSON{"field": "content_type"},
		)
	}

	grant, err := this.storage.PresignUpload(key, contentType, this.policy)

	if err != nil {
		this.logger.Error("Failed to sign an avatar upload", zap.String("key", key), zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to authorize the upload")
	}

	return grant, nil
}

// The client names the key, so nothing about it is trusted: it has to be one
// this owner could have been granted, it has to exist, and it has to be what the
// policy signed for. See RC-3 in docs/features/2026-10-11-avatars/requirements.md.
func (this *AvatarService) confirm(target *owner, key string) error {
	if !this.policy.OwnsPendingKey(target.path, key) {
		return e.ThrowBadRequest("`key` is not an upload granted for this avatar", utils.JSON{"field": "key"})
	}

	object, err := this.storage.Head(key)

	if err != nil {
		this.logger.Error("Failed to inspect an uploaded avatar", zap.String("key", key), zap.Error(err))
		return e.ThrowInternalServerError("Failed to inspect the upload")
	}

	if object == nil {
		return e.ThrowNotFound("No upload under this key: it expired, was never sent, or was already confirmed")
	}

	if !this.policy.Accepts(key, *object) {
		return e.ThrowUnprocessableEntity("The uploaded file is not an image this avatar accepts")
	}

	promoted := this.policy.Promoted(key)

	if err := this.storage.Copy(key, promoted); err != nil {
		this.logger.Error("Failed to promote an avatar", zap.String("key", key), zap.Error(err))
		return e.ThrowInternalServerError("Failed to store the avatar")
	}

	if err := target.write(&promoted); err != nil {
		this.discard(promoted)
		return err
	}

	this.discard(key)

	if target.avatarKey != nil && *target.avatarKey != promoted {
		this.discard(*target.avatarKey)
	}

	return nil
}

func (this *AvatarService) remove(target *owner) error {
	if target.avatarKey == nil {
		return nil
	}

	if err := target.write(nil); err != nil {
		return err
	}

	this.discard(*target.avatarKey)

	return nil
}

// discard never fails the request: the database already says what is current,
// and an object left behind costs cents. The lifecycle rule sweeps pending/.
func (this *AvatarService) discard(key string) {
	if err := this.storage.Delete(key); err != nil {
		this.logger.Warn("Failed to delete a stored object, left as an orphan", zap.String("key", key), zap.Error(err))
	}
}
