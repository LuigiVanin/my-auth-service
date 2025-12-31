package entity

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

type App struct {
	ID          string `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UsersPoolId string `gorm:"type:uuid;not null"`

	PublicKey string `gorm:"not null"`
	SecretKey string `gorm:"not null"`

	Name string `gorm:"not null"`

	Role string `gorm:"type:APP_ROLE;default:'USER';not null"`

	LoginTypes                 pq.StringArray `gorm:"type:AUTH_METHOD[];not null"`
	TokenType                  string         `gorm:"type:TOKEN_TYPE;not null"`
	TokenExpirationTime        int64          `gorm:"not null"`
	RefreshTokenExpirationTime int64          `gorm:"not null;default:1296000"` // 15 days

	Private           bool       `gorm:"default:false;not null"`
	VerifiedEmailDate *time.Time `gorm:"column:verified_email_date"`
	VerifyEmail       bool       `gorm:"-"`

	Metadata json.RawMessage `gorm:"type:jsonb;default:'{}';not null"`

	UsersPool UsersPool `gorm:"foreignKey:UsersPoolId"`

	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null"`
}

func (App) TableName() string {
	return "apps"
}
