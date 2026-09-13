package services

import (
	"time"

	dto "auth_service/app/modules/core/user_pool/models"
	entity "auth_service/infra/entities"
	"auth_service/shared/tracking"
)

type CreateUserPoolData struct {
	Name             string
	Description      string
	OwnerUserId      *uint
	OrganizationId   *string
	DefaultProfileId string
}

// TrackSignup is one signup, reduced to what the counters group by: nothing about
// the user that signed up is kept, only the pool it landed in, the app it came
// through and the moment.
type TrackSignup struct {
	PoolId string
	AppId  string
	At     time.Time
}

type IUserPoolService interface {
	// Refuses when DefaultProfileId grants more than caller holds in granter.
	Create(data CreateUserPoolData, granter *entity.Organization, caller *entity.Participant) (*entity.UsersPool, error)

	// Update writes only the fields the payload filled, and merges Metadata into
	// what is stored. granter and caller are the ceiling a new DefaultProfileId is
	// clamped to, the same pair Create takes.
	Update(
		id string,
		granter *entity.Organization,
		caller *entity.Participant,
		payload *dto.UpdateUserPool,
	) (*entity.UsersPool, error)

	// Unscoped, and only safe where the id did not come from the caller.
	FindById(id string) (*entity.UsersPool, error)

	// The scoped lookup, for whenever the id comes from a request.
	FindByIdInOrganization(id string, organizationId string) (*entity.UsersPool, error)

	List(
		currentOrganization *entity.Organization,
		query *dto.UserPoolListQuery,
	) (*dto.GetUserPoolsResponse, error)

	// Track folds one event into the tracking column of the pool. It is called from
	// a detached goroutine, so it reports its error rather than shaping a response,
	// and the only tag it accepts is tracking.TagSignup.
	Track(tag tracking.Tag, payload TrackSignup) error
}
