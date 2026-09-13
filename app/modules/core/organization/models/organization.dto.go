package models

import (
	entity "auth_service/infra/entities"
	"encoding/json"
)

// Service level input, not a transport shape: registration fills it with no
// owner, because the user that will own it does not exist yet.
type CreateOrganizationData struct {
	UsersPoolId string
	ProfileId   string
	OwnerUserId *uint

	Name        string
	Description string
	Metadata    json.RawMessage
}

type CreateOrganizationPayload struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`

	// Ceiling of the new organization, capped by the ceiling of the one the
	// request is made from. Falls back to the default profile of the pool.
	ProfileId string `json:"profile_id" validate:"omitempty,uuid4"`

	Metadata json.RawMessage `json:"metadata"`
}

// Every field is a pointer, and Metadata is merged rather than replaced - see
// steering/models-layer.md.
//
// `profile_id` is absent: it is the permission ceiling of the organization, and
// moving a ceiling from a route needs the clamp that
// docs/features/2026-08-23-scoped-profiles.md leaves open. `owner_user_id` is absent
// too - transferring an organization has to move the participation in the same
// transaction, so it is a flow of its own.
type UpdateOrganization struct {
	Name        *string          `json:"name" validate:"omitnil,min=1"`
	Description *string          `json:"description"`
	Metadata    *json.RawMessage `json:"metadata"`
}

type SwitchOrganizationPayload struct {
	OrganizationId string `json:"organization_id" validate:"required,uuid4"`
}

type GetOrganizationsQuery struct {
	Skip  int `query:"skip"`
	Limit int `query:"limit"`
}

type GetOrganizationsResponse struct {
	Total  int64                 `json:"total"`
	Amount int                   `json:"amount"`
	Skip   int                   `json:"skip"`
	Limit  int                   `json:"limit"`
	Data   []entity.Organization `json:"data"`
}
