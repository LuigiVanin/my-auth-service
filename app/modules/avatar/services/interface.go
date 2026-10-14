package services

import (
	entity "auth_service/infra/entities"
	"auth_service/shared/storage"
)

// Three steps per subject: Request signs an upload into the pending area, the
// client posts the file straight to the bucket, Confirm promotes it. Remove
// clears the avatar and is idempotent.
type IAvatarService interface {
	RequestUserUpload(currentUser *entity.User, contentType string) (*storage.UploadGrant, error)
	ConfirmUser(currentUser *entity.User, currentOrganization *entity.Organization, key string) (*entity.User, error)
	RemoveUser(currentUser *entity.User, currentOrganization *entity.Organization) (*entity.User, error)

	RequestOrganizationUpload(organizationId string, currentOrganization *entity.Organization, contentType string) (*storage.UploadGrant, error)
	ConfirmOrganization(organizationId string, currentOrganization *entity.Organization, key string) (*entity.Organization, error)
	RemoveOrganization(organizationId string, currentOrganization *entity.Organization) (*entity.Organization, error)

	RequestAppUpload(appId string, currentOrganization *entity.Organization, contentType string) (*storage.UploadGrant, error)
	ConfirmApp(appId string, currentOrganization *entity.Organization, key string) (*entity.App, error)
	RemoveApp(appId string, currentOrganization *entity.Organization) (*entity.App, error)
}
