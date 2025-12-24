package entity

import "time"

type UserPool struct {
	ID   string `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Name string `gorm:"not null"`

	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null"`
}

func (UserPool) TableName() string {
	return "users_pool"
}
