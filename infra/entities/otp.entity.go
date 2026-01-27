package entity

import (
	"encoding/json"
	"time"
)

type Otp struct {
	ID     string `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserId *uint  `gorm:"type:bigint"` // nullable: OTP may be generated for unknown users
	AppId  string `gorm:"type:uuid;index:idx_otp_app_contact"`

	Code  string `gorm:"not null" json:"-"`
	Token string `gorm:"not null;type:uuid;default:uuid_generate_v4()" json:"-"`

	Action string `gorm:"type:auth_action"`
	// NOTE: In he future add column to chose the sender
	// Method string `gorm:"type:OTP_METHOD;not null;default:'EMAIL'"`

	Contact string `gorm:"not null;index:idx_otp_app_contact"`

	Invalidated bool            `gorm:"not null;default:false"`
	Metadata    json.RawMessage `gorm:"type:jsonb;default:'{}';not null"`

	User *User `gorm:"foreignKey:UserId"`
	App  App   `gorm:"foreignKey:AppId"`

	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null"`
}

func (Otp) TableName() string {
	return "otps"
}
