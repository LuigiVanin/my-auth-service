package entity

import "time"

type UsersPool struct {
	ID        string `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Name      string `gorm:"not null"`
	PublicKey string `gorm:"not null"`

	// The ceiling handed to every organization created in this pool.
	DefaultProfileId string `gorm:"type:uuid;not null" json:"default_profile_id"`

	OwnerUserId *uint `gorm:"type:bigint;default:null" json:"owner_user_id,omitempty"`

	// Nullable: the seeded pool is owned by an organization living inside itself,
	// the platform ADMIN case and the only place that is allowed.
	OrganizationId *string `gorm:"type:uuid;default:null;index" json:"organization_id,omitempty"`

	OwnerUser      *User         `gorm:"foreignKey:OwnerUserId" json:"owner_user,omitempty"`
	Organization   *Organization `gorm:"foreignKey:OrganizationId" json:"organization,omitempty"`
	DefaultProfile *Profile      `gorm:"foreignKey:DefaultProfileId" json:"default_profile,omitempty"`

	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null"`
}

func (UsersPool) TableName() string {
	return "users_pool"
}
