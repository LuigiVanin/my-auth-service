package entity

import (
	"encoding/json"
	"time"
)

// NOTE: OwnerUserId is nullable because users and organizations reference each
// other. A user needs a CurrentOrganizationId to be inserted, so the organization
// is written first with a null owner and stamped once the user exists.
type Organization struct {
	ID          string `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()" json:"id"`
	UsersPoolId string `gorm:"type:uuid;not null;index" json:"users_pool_id"`
	OwnerUserId *uint  `gorm:"type:bigint;default:null;index" json:"owner_user_id,omitempty"`
	ProfileId   string `gorm:"type:uuid;not null" json:"profile_id"`

	Name        string `gorm:"not null" json:"name"`
	Description string `gorm:"not null;default:''" json:"description"`

	OwnerUser    *User         `gorm:"foreignKey:OwnerUserId" json:"owner_user,omitempty"`
	UsersPool    *UsersPool    `gorm:"foreignKey:UsersPoolId" json:"-"`
	Profile      *Profile      `gorm:"foreignKey:ProfileId" json:"-"`
	Participants []Participant `gorm:"foreignKey:OrganizationId" json:"participants,omitempty"`

	Metadata  json.RawMessage `gorm:"type:jsonb;default:'{}';not null" json:"metadata"`
	CreatedAt time.Time       `gorm:"default:CURRENT_TIMESTAMP;not null" json:"created_at"`
	UpdatedAt time.Time       `gorm:"default:CURRENT_TIMESTAMP;not null" json:"updated_at"`
}

func (Organization) TableName() string {
	return "organizations"
}
