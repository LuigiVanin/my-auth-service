package models

import (
	"time"

	entity "auth_service/infra/entities"
	smodels "auth_service/shared/models"
)

type InviteStatus string

const (
	InviteStatusPending  InviteStatus = "pending"
	InviteStatusAccepted InviteStatus = "accepted"
	InviteStatusRevoked  InviteStatus = "revoked"
	InviteStatusExpired  InviteStatus = "expired"
)

// TtlSeconds bounds are the same as UpdateOrganization.InviteTtlSeconds.
type CreateInvite struct {
	Email      string `json:"email" validate:"required,email"`
	ProfileId  string `json:"profile_id" validate:"required,uuid4"`
	TtlSeconds *int64 `json:"ttl_seconds" validate:"omitnil,min=3600,max=2592000"`
}

type GetInvitesQuery struct {
	Skip   int    `query:"skip"`
	Limit  int    `query:"limit"`
	Status string `query:"status"`
}

type DescribeInviteQuery struct {
	Token string `query:"token"`
}

type AcceptInvite struct {
	Token string `json:"token" validate:"required"`
}

type InviteResponse struct {
	entity.Invite
	Status InviteStatus `json:"status"`
}

type GetInvitesResponse struct {
	Total  int64            `json:"total"`
	Amount int              `json:"amount"`
	Skip   int              `json:"skip"`
	Limit  int              `json:"limit"`
	Data   []InviteResponse `json:"data"`
}

// InviteDescription is what the bearer of a token sees before signing in, so it
// carries names and nothing else of the organization, the profile or the inviter.
type InviteDescription struct {
	Organization  InvitedOrganization `json:"organization"`
	InvitedBy     InvitedBy           `json:"invited_by"`
	Profile       InvitedProfile      `json:"profile"`
	Email         string              `json:"email"`
	ExpiresAt     time.Time           `json:"expires_at"`
	AccountExists bool                `json:"account_exists"`
}

type InvitedOrganization struct {
	Id        string  `json:"id"`
	Name      string  `json:"name"`
	AvatarUrl *string `json:"avatar_url"`
}

type InvitedBy struct {
	Name string `json:"name"`
}

type InvitedProfile struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"`
}

type AcceptInviteResponse struct {
	Organization smodels.OrganizationView `json:"organization"`
	Participant  smodels.ParticipantView  `json:"participant"`
}
