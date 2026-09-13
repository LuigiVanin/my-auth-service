package entity

import (
	"encoding/json"
	"time"
)

type UsersPool struct {
	ID          string `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()" json:"id"`
	Name        string `gorm:"not null" json:"name"`
	Description string `gorm:"not null;default:''" json:"description"`
	PublicKey   string `gorm:"not null" json:"public_key,omitempty"`

	// The ceiling handed to every organization created in this pool.
	DefaultProfileId string `gorm:"type:uuid;not null" json:"default_profile_id"`

	// Users registered in this pool. It only ever grows: deleting a user does not
	// decrement it, so it counts signups and not rows.
	UsersCount int `gorm:"not null;default:0" json:"users_count"`

	OwnerUserId *uint `gorm:"type:bigint;default:null" json:"owner_user_id,omitempty"`

	// Nullable: the seeded pool is owned by an organization living inside itself,
	// the platform ADMIN case and the only place that is allowed.
	OrganizationId *string `gorm:"type:uuid;default:null;index" json:"organization_id,omitempty"`

	OwnerUser      *User         `gorm:"foreignKey:OwnerUserId" json:"owner_user,omitempty"`
	Organization   *Organization `gorm:"foreignKey:OrganizationId" json:"organization,omitempty"`
	DefaultProfile *Profile      `gorm:"foreignKey:DefaultProfileId" json:"default_profile,omitempty"`

	Metadata json.RawMessage `gorm:"type:jsonb;default:'{}';not null" json:"metadata"`

	// Signup counters, written only by UserPoolRepository.WriteTracking. Blanked
	// row by row in the listing, so `omitempty` keeps it out of that page.
	Tracking json.RawMessage `gorm:"type:jsonb;default:'{}';not null" json:"tracking,omitempty"`

	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null" json:"updated_at"`
}

func (UsersPool) TableName() string {
	return "users_pool"
}
