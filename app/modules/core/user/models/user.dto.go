package models

import (
	entity "auth_service/infra/entities"
	"auth_service/shared/permissions"
	"encoding/json"
)

// UserListQuery is the query of GET /core/users. AppId and PoolId both name the
// pool to list, so exactly one of them has to be filled.
type UserListQuery struct {
	AppId  string `query:"app_id"`
	PoolId string `query:"pool_id"`

	Skip  int    `query:"skip"`
	Limit int    `query:"limit"`
	Name  string `query:"name"`
	Email string `query:"email"`
}

// UpdateUser is PUT /core/users/{id}, the administration route: whoever owns the
// pool of the target user. Every field is a pointer, so an absent one is not the
// same as an empty one, and Metadata is merged rather than replaced - see
// steering/models-layer.md.
//
// The password is absent: it is never carried by a payload, only derived by the
// hash service from the forgot password flow.
type UpdateUser struct {
	Name             *string          `json:"name" validate:"omitnil,min=1"`
	Email            *string          `json:"email" validate:"omitnil,email"`
	Phone            *string          `json:"phone"`
	VerifyEmail      *bool            `json:"verify_email"`
	TwoFactorEnabled *bool            `json:"two_factor_enabled"`
	Metadata         *json.RawMessage `json:"metadata"`
}

// UpdateUserSelf is PUT /core/users/me, which needs no scope check because the
// target is the caller. It is narrower than UpdateUser on purpose: email and the
// two verification flags decide how the user authenticates, so moving them is an
// administration act and not a profile edit.
type UpdateUserSelf struct {
	Name     *string          `json:"name" validate:"omitnil,min=1"`
	Phone    *string          `json:"phone"`
	Metadata *json.RawMessage `json:"metadata"`
}

// GetUserResponse is the answer of the two individual user reads, and the only
// place `users.tracking` is exposed: the column is `json:"-"` on the entity, so
// the login, register, refresh and listing responses that embed entity.User
// cannot leak the ip history of a user. The outer Tracking shadows the embedded
// one, the same way ProfileResponse shadows Permissions.
type GetUserResponse struct {
	entity.User
	Tracking json.RawMessage `json:"tracking"`
}

func UserWithTracking(user *entity.User) *GetUserResponse {
	return &GetUserResponse{User: *user, Tracking: user.Tracking}
}

type GetUsersResponse struct {
	Total  int64         `json:"total"`
	Amount int           `json:"amount"`
	Skip   int           `json:"skip"`
	Limit  int           `json:"limit"`
	Data   []entity.User `json:"data"`
}

// UserResponse is the authenticated user as login, register and refresh answer it.
//
// Profile is what the user holds in the organization it is currently scoped to: the
// profile of its participation there. It sits here and not on entity.User because
// the participations of a user are a list, and picking the one that matches the
// current organization is a transport concern.
type UserResponse struct {
	entity.User
	Profile *ProfileResponse `json:"profile,omitempty"`
}

// ProfileResponse is a profile whose permissions have already been resolved
// against every ceiling above it, so a caller never has to stack documents itself.
//
// The embedded raw Permissions is shadowed on purpose: a participant profile read
// on its own overstates whenever the ceiling above it is narrower.
type ProfileResponse struct {
	entity.Profile
	Permissions *permissions.Resolved `json:"permissions"`
}
