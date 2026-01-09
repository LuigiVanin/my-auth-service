package entity

import (
	"encoding/json"
	"time"
)

type Otp struct {
	ID     string `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserId string `gorm:"type:uuid"`
	AppId  string `gorm:"type:uuid"`

	Code  string `gorm:"not null" json:"-"`
	Token string `gorm:"not null;type:uuid;default:uuid_generate_v4()" json:"-"`

	Action string `gorm:"type:ACTION"`
	Method string `gorm:"type:OTP_METHOD;not null;default:'EMAIL'"`

	Invalidated bool            `gorm:"not null;default:false"`
	Metadata    json.RawMessage `gorm:"type:jsonb;default:'{}';not null"`

	User User `gorm:"foreignKey:UserId"`
	App  App  `gorm:"foreignKey:AppId"`

	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null"`
}

func (Otp) TableName() string {
	return "otps"
}
