package entity

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

type App struct {
	ID          string `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()" json:"id"`
	UsersPoolId string `gorm:"type:uuid;not null" json:"users_pool_id"`
	OwnerUserId *uint  `gorm:"type:bigint;default:null" json:"owner_user_id,omitempty"`

	// Nullable for the same reason as UsersPool.OrganizationId.
	OrganizationId *string `gorm:"type:uuid;default:null;index" json:"organization_id,omitempty"`

	// this will come out
	ParentAppId *string `gorm:"type:uuid;default:null" json:"parent_app_id,omitempty"`

	PublicKey string `gorm:"not null" json:"public_key,omitempty"`
	SecretKey string `gorm:"not null" json:"secret_key,omitempty"`

	Name string `gorm:"not null" json:"name"`

	// NOTE: no consumer left - GetProfileByAppRole was the only one. Kept as
	// descriptive metadata rather than dropped.
	Role string `gorm:"type:APP_ROLE;default:'USER';not null" json:"role"`

	LoginTypes                 pq.StringArray `gorm:"type:AUTH_METHOD[];not null" json:"login_types"`
	TokenType                  string         `gorm:"type:TOKEN_TYPE;not null" json:"token_type"`
	TokenExpirationTime        int64          `gorm:"not null" json:"token_expiration_time"`
	RefreshTokenExpirationTime int64          `gorm:"not null;default:1296000" json:"refresh_token_expiration_time"` // 15 days

	Private bool `gorm:"default:false;not null" json:"private"`

	VerifiedEmailDate *time.Time `gorm:"column:verified_email_date"`
	VerifyEmail       bool       `gorm:"default:true;not null" json:"verify_email"`

	Enabled2FA bool `gorm:"default:true;not null" json:"enabled_2fa"`

	Metadata json.RawMessage `gorm:"type:jsonb;default:'{}';not null" json:"metadata"`

	UsersPool    UsersPool     `gorm:"foreignKey:UsersPoolId" json:"users_pool,omitempty"`
	OwnerUser    *User         `gorm:"foreignKey:OwnerUserId" json:"owner_user,omitempty"`
	ParentApp    *App          `gorm:"foreignKey:ParentAppId" json:"parent_app,omitempty"`
	Organization *Organization `gorm:"foreignKey:OrganizationId" json:"organization,omitempty"`

	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null" json:"updated_at"`
}

func (App) TableName() string {
	return "apps"
}
