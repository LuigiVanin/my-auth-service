package models

import entity "auth_service/infra/entities"

type CreateUserPool struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`

	// Ceiling of the organizations born in this pool, capped by the ceiling of the
	// one creating it. Defaults to LOGIN_PROFILE, so a new pool is born closed.
	DefaultProfileId string `json:"default_profile_id" validate:"omitempty,uuid4"`
}

// No owner filter: the listing is already scoped to the current organization.
type UserPoolListQuery struct {
	Skip  int    `query:"skip"`
	Limit int    `query:"limit"`
	Name  string `query:"name"`
}

type GetUserPoolsResponse struct {
	Total  int64              `json:"total"`
	Amount int                `json:"amount"`
	Skip   int                `json:"skip"`
	Limit  int                `json:"limit"`
	Data   []entity.UsersPool `json:"data"`
}
