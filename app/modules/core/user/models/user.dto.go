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

type UpdateUser struct {
	// All the elements can be nil to indicate that the field should not be touched
	Name             *string          `gorm:"not null" json:"name"`
	Email            *string          `gorm:"not null;uniqueIndex:users_email_users_pool_unique,priority:1" json:"email"`
	Phone            *string          `gorm:"default:null" json:"phone"`
	VerifyEmail      *bool            `gorm:"not null;default:false" json:"verifyEmail"`
	TwoFactorEnabled *bool            `gorm:"not null;default:false" json:"twoFactorEnabled"`
	PasswordHash     *string          `gorm:"not null" json:"-"`
	Metadata         *json.RawMessage `gorm:"type:jsonb;default:'{}';not null" json:"metadata"`
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
