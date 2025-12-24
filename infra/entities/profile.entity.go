package entity

import (
	"encoding/json"
	"time"
)

type Profile struct {
	ID              string          `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	ParentProfileId *string         `gorm:"type:uuid"`
	Name            string          `gorm:"not null"`
	Key             string          `gorm:"unique;not null"`
	Permissions     json.RawMessage `gorm:"type:jsonb;default:'{}';not null"`
	Metadata        json.RawMessage `gorm:"type:jsonb;default:'{}';not null"`

	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null"`
}

func (Profile) TableName() string {
	return "profiles"
}
