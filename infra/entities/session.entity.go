package entity

import (
	"encoding/json"
	"time"
)

type Session struct {
	ID           string `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserId       string `gorm:"type:uuid;not null"`
	AppId        string `gorm:"type:uuid;not null"`
	Token        string `gorm:"not null;type:uuid;default:uuid_generate_v4()" json:"-"`
	RefreshToken string `gorm:"not null;type:uuid;default:uuid_generate_v4()" json:"-"`
	Invalidated  bool   `gorm:"not null;default:false" json:"-"`

	IpAddress string `gorm:"not null"`
	LoginType string `gorm:"type:AUTH_METHOD;not null;default:'WITH_LOGIN'"`
	UserAgent string

	ExpiresAt        time.Time `gorm:"not null"`
	RefreshExpiresAt time.Time `gorm:"not null"`

	LastUsedAt time.Time `gorm:"not null"`

	Metadata json.RawMessage `gorm:"type:jsonb;default:'{}';not null"`

	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null"`

	User User `gorm:"foreignKey:UserId"`
	App  App  `gorm:"foreignKey:AppId"`
}

func (Session) TableName() string {
	return "sessions"
}
