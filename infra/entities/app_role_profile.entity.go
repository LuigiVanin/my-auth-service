package entity

import (
	"encoding/json"
	"time"
)

type AppRoleProfile struct {
	ID        string `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	ProfileId string `gorm:"type:uuid;not null;index:idx_app_role_profiles_profile_id"`
	Role      string `gorm:"type:APP_ROLE;not null;index:idx_app_role_profiles_role"`

	Priority int `gorm:"not null;default:999"`

	Permission json.RawMessage `gorm:"type:jsonb;default:'{}';not null"`
	Relation   json.RawMessage `gorm:"type:jsonb;default:'{}';not null"`
	Metadata   json.RawMessage `gorm:"type:jsonb;default:'{}';not null"`

	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null"`

	Profile Profile `gorm:"foreignKey:ProfileId"`
}

func (AppRoleProfile) TableName() string { return "app_role_profiles" }
