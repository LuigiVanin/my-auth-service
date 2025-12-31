package entity

import (
	"encoding/json"
	"time"
)

type Profile struct {
	ID              string          `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()" json:"id"`
	ParentProfileId *string         `gorm:"type:uuid" json:"-"`
	Name            string          `gorm:"not null" json:"name"`
	Key             string          `gorm:"unique;not null" json:"key"`
	Permissions     json.RawMessage `gorm:"type:jsonb;default:'{}';not null" json:"permissions"`
	Metadata        json.RawMessage `gorm:"type:jsonb;default:'{}';not null" json:"metadata"`

	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null" json:"createdAt"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null" json:"updatedAt"`
}

func (Profile) TableName() string {
	return "profiles"
}
