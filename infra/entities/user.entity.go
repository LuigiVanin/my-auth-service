package entity

import (
	"encoding/json"
	"time"
)

type User struct {
	ID               uint            `gorm:"primaryKey;autoIncrement"`
	Uuid             string          `gorm:"type:uuid;default:uuid_generate_v4();unique;not null"`
	UsersPoolId      string          `gorm:"type:uuid;not null;uniqueIndex:users_email_users_pool_unique,priority:2"`
	ProfileId        string          `gorm:"type:uuid;not null"`
	Name             string          `gorm:"not null"`
	Email            string          `gorm:"not null;uniqueIndex:users_email_users_pool_unique,priority:1"`
	Phone            string          `gorm:"default:null"`
	VerifyEmail      bool            `gorm:"not null;default:false"`
	TwoFactorEnabled bool            `gorm:"not null;default:false"`
	PasswordHash     string          `gorm:"not null"`
	Metadata         json.RawMessage `gorm:"type:jsonb;default:'{}';not null"`
	CreatedAt        time.Time       `gorm:"default:CURRENT_TIMESTAMP;not null"`
	UpdatedAt        time.Time       `gorm:"default:CURRENT_TIMESTAMP;not null"`

	UsersPool *UserPool `gorm:"foreignKey:UsersPoolId"`
	Profile   *Profile  `gorm:"foreignKey:ProfileId"`
}

func (User) TableName() string {
	return "users"
}
