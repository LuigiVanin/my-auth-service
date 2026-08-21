package dto

import (
	entity "auth_service/infra/entities"
	"encoding/json"
)

type GetUsersAppQuery struct {
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

type GetUsersAppResponse struct {
	Total  int64         `json:"total"`
	Amount int           `json:"amount"`
	Skip   int           `json:"skip"`
	Data   []entity.User `json:"data"`
}
