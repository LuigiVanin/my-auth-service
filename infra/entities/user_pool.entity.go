package entity

import "time"

type UsersPool struct {
	ID   string `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Name string `gorm:"not null"`

	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null"`
}

func (UsersPool) TableName() string {
	return "users_pool"
}
