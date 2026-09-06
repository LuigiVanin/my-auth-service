package entity

import (
	"encoding/json"
	"time"
)

type Profile struct {
	ID   string `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()" json:"id"`
	Name string `gorm:"not null" json:"name"`
	Key  string `gorm:"unique;not null" json:"key"`

	// NULL is global; set makes the row exclusive to that organization.
	OrganizationId *string `gorm:"type:uuid;default:null;index" json:"organization_id,omitempty"`

	Permissions json.RawMessage `gorm:"type:jsonb;default:'{}';not null" json:"permissions"`
	Metadata    json.RawMessage `gorm:"type:jsonb;default:'{}';not null" json:"metadata"`

	Organization *Organization `gorm:"foreignKey:OrganizationId" json:"-"`

	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null" json:"createdAt"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null" json:"updatedAt"`
}

func (Profile) TableName() string {
	return "profiles"
}
