package entity

import (
	"encoding/json"
	"time"
)

type User struct {
	ID   uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Uuid string `gorm:"type:uuid;default:uuid_generate_v4();unique;not null" json:"-"`

	UsersPoolId string `gorm:"type:uuid;not null;uniqueIndex:users_email_users_pool_unique,priority:2" json:"-"`

	CurrentOrganizationId string `gorm:"type:uuid;not null" json:"current_organization_id"`

	Name  string `gorm:"not null" json:"name"`
	Email string `gorm:"not null;uniqueIndex:users_email_users_pool_unique,priority:1" json:"email"`
	Phone string `gorm:"default:null" json:"phone"`

	VerifyEmail       bool       `gorm:"not null;default:false" json:"verify_email"`
	VerifiedEmailDate *time.Time `gorm:"column:verified_email_date" json:"verified_email_date,omitempty"`

	TwoFactorEnabled bool            `gorm:"not null;default:false" json:"two_factor_enabled"`
	PasswordHash     string          `gorm:"not null" json:"-"`
	Metadata         json.RawMessage `gorm:"type:jsonb;default:'{}';not null" json:"metadata"`

	// Written only by UserRepository.WriteTracking, and never serialized through
	// the entity: it carries the ip history of the user, and entity.User is embedded
	// in the login, register, refresh and listing responses. See
	// docs/features/2026-09-09-tracking/spec.md.
	Tracking  json.RawMessage `gorm:"type:jsonb;default:'{}';not null" json:"-"`
	CreatedAt time.Time       `gorm:"default:CURRENT_TIMESTAMP;not null" json:"created_at"`
	UpdatedAt time.Time       `gorm:"default:CURRENT_TIMESTAMP;not null" json:"updated_at"`

	UsersPool           *UsersPool    `gorm:"foreignKey:UsersPoolId" json:"-"`
	CurrentOrganization *Organization `gorm:"foreignKey:CurrentOrganizationId" json:"current_organization,omitempty"`
}

func (User) TableName() string {
	return "users"
}
