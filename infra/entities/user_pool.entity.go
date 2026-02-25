package entity

import "time"

type UsersPool struct {
	ID   string `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Name string `gorm:"not null"`

	OwnerUserId *uint `gorm:"type:bigint;default:null" json:"owner_user_id,omitempty"`

	OwnerUser *User `gorm:"foreignKey:OwnerUserId" json:"owner_user,omitempty"`

	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null"`
}

func (UsersPool) TableName() string {
	return "users_pool"
}
