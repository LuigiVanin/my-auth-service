package services

import (
	dto "auth_service/app/modules/core/user/models"
	entity "auth_service/infra/entities"
	"auth_service/shared/tracking"
)

// TrackLogin is one login. It is the whole event and not a counter, which is the
// one place the tracking rule is broken on purpose - see
// docs/features/2026-09-09-tracking/spec.md.
type TrackLogin struct {
	UserId uint
	Event  tracking.LoginEvent
}

// LoginFrom reads the event off the session that was just minted: the session
// already carries the app, the ip, the user agent, the method and the moment, so
// a login has nothing to gather that is not already there.
func LoginFrom(session *entity.Session) TrackLogin {
	return TrackLogin{
		UserId: session.UserId,
		Event: tracking.LoginEvent{
			AppId:     session.AppId,
			At:        session.CreatedAt,
			Ip:        session.IpAddress,
			UserAgent: session.UserAgent,
			LoginType: session.LoginType,
		},
	}
}

type IUserService interface {
	IsAlreadyCreated(email string, app *entity.App) (bool, error)
	FindUserInPool(email string, usersPoolId string) (*entity.User, error)

	// The low level write, for callers that already know which column they are
	// moving - SetPassword and the organization switch. A payload never reaches it.
	Update(where entity.User, data dto.UserUpdateDao) (int64, error)

	// UpdateForOrganization is the administration write: the target has to sit in a
	// pool the current organization owns, and there is no self escape.
	UpdateForOrganization(
		currentOrganization *entity.Organization,
		targetUserId uint,
		payload *dto.UpdateUser,
	) (*entity.User, error)

	// UpdateSelf is the same write on the caller itself, over a narrower payload.
	UpdateSelf(currentUser *entity.User, payload *dto.UpdateUserSelf) (*entity.User, error)

	List(
		currentUser *entity.User,
		currentOrganization *entity.Organization,
		query *dto.UserListQuery,
	) (*dto.GetUsersResponse, error)

	FindById(
		currentUser *entity.User,
		currentOrganization *entity.Organization,
		targetUserId uint,
	) (*entity.User, error)

	// Track folds one event into the tracking column of the user. It is called from
	// a detached goroutine, so it reports its error rather than shaping a response,
	// and the only tag it accepts is tracking.TagLogin.
	Track(tag tracking.Tag, payload TrackLogin) error
}
