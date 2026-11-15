package entity

import "time"

// OtpId, ExpiresAt, AcceptedAt and RevokedAt are written only by the invite
// service and its dedicated repository methods - there is no update dao. The
// secret of the link lives in the OTP, never here.
// See docs/features/2026-10-13-organization-invites/.
type Invite struct {
	ID              string `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()" json:"id"`
	OrganizationId  string `gorm:"type:uuid;not null;index" json:"organization_id"`
	ProfileId       string `gorm:"type:uuid;not null" json:"profile_id"`
	InvitedByUserId uint   `gorm:"type:bigint;not null" json:"invited_by_user_id"`

	Email string `gorm:"not null;index" json:"email"`
	OtpId string `gorm:"type:uuid;not null;uniqueIndex" json:"-"`

	// The same value as the OTP's, written once at issue: the OTP enforces it, this
	// copy is what a listing reads without joining otps.
	ExpiresAt  time.Time  `gorm:"not null" json:"expires_at"`
	AcceptedAt *time.Time `gorm:"default:null" json:"accepted_at"`
	RevokedAt  *time.Time `gorm:"default:null" json:"revoked_at"`

	Organization *Organization `gorm:"foreignKey:OrganizationId" json:"-"`
	Profile      *Profile      `gorm:"foreignKey:ProfileId" json:"profile,omitempty"`
	InvitedBy    *User         `gorm:"foreignKey:InvitedByUserId" json:"-"`
	Otp          *Otp          `gorm:"foreignKey:OtpId" json:"-"`

	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;not null" json:"updated_at"`
}

func (Invite) TableName() string {
	return "invites"
}
