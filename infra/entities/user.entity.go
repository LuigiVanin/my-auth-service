package entity

import (
	"encoding/json"
	"time"
)

type User struct {
	ID               uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	Uuid             string          `gorm:"type:uuid;default:uuid_generate_v4();unique;not null" json:"-"`
	UsersPoolId      string          `gorm:"type:uuid;not null;uniqueIndex:users_email_users_pool_unique,priority:2" json:"-"`
	ProfileId        string          `gorm:"type:uuid;not null" json:"profileId"`
	Name             string          `gorm:"not null" json:"name"`
	Email            string          `gorm:"not null;uniqueIndex:users_email_users_pool_unique,priority:1" json:"email"`
	Phone            string          `gorm:"default:null" json:"phone"`
	VerifyEmail      bool            `gorm:"not null;default:false" json:"verifyEmail"`
	TwoFactorEnabled bool            `gorm:"not null;default:false" json:"twoFactorEnabled"`
	PasswordHash     string          `gorm:"not null" json:"-"`
	Metadata         json.RawMessage `gorm:"type:jsonb;default:'{}';not null" json:"metadata"`
	CreatedAt        time.Time       `gorm:"default:CURRENT_TIMESTAMP;not null" json:"createdAt"`
	UpdatedAt        time.Time       `gorm:"default:CURRENT_TIMESTAMP;not null" json:"updatedAt"`

	UsersPool *UsersPool `gorm:"foreignKey:UsersPoolId" json:"-"`
	Profile   *Profile   `gorm:"foreignKey:ProfileId" json:"profile,omitempty"`
}

func (User) TableName() string {
	return "users"
}
