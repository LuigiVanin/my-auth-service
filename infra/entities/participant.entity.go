package entity

import (
	"encoding/json"
	"time"
)

type Participant struct {
	ID string `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()" json:"id"`

	OrganizationId string `gorm:"type:uuid;not null;uniqueIndex:participants_org_user_unique,priority:1" json:"organization_id"`
	UserId         uint   `gorm:"type:bigint;not null;uniqueIndex:participants_org_user_unique,priority:2" json:"user_id"`
	ProfileId      string `gorm:"type:uuid;not null" json:"profile_id"`

	Organization *Organization `gorm:"foreignKey:OrganizationId" json:"-"`
	User         *User         `gorm:"foreignKey:UserId" json:"user,omitempty"`
	Profile      *Profile      `gorm:"foreignKey:ProfileId" json:"profile,omitempty"`

	Metadata  json.RawMessage `gorm:"type:jsonb;default:'{}';not null" json:"metadata"`
	CreatedAt time.Time       `gorm:"default:CURRENT_TIMESTAMP;not null" json:"created_at"`
	UpdatedAt time.Time       `gorm:"default:CURRENT_TIMESTAMP;not null" json:"updated_at"`
}

func (Participant) TableName() string {
	return "participants"
}
