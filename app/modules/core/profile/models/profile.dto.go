package models

import (
	entity "auth_service/infra/entities"
)

// An `api` key in the body is dropped, not refused: the endpoint does not speak
// that half yet.
type CreateProfile struct {
	Name        string             `json:"name" validate:"required,min=1,max=120"`
	Permissions ProfilePermissions `json:"permissions" validate:"required"`
}

type UpdateProfile struct {
	Name        *string             `json:"name" validate:"omitempty,min=1,max=120"`
	Permissions *ProfilePermissions `json:"permissions"`
}

type ProfilePermissions struct {
	Grants []string `json:"grants" validate:"required,min=1,dive,required"`
}

// OrganizationId narrows to the rows that organization owns; it can never widen
// what is visible, so it is not a lens into another organization.
type GetProfilesQuery struct {
	Skip           int    `query:"skip"`
	Limit          int    `query:"limit"`
	Name           string `query:"name"`
	OrganizationId string `query:"organization_id"`
}

type GetProfilesResponse struct {
	Total  int64            `json:"total"`
	Amount int              `json:"amount"`
	Skip   int              `json:"skip"`
	Limit  int              `json:"limit"`
	Data   []entity.Profile `json:"data"`
}
