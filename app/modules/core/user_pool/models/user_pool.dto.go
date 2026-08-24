package models

type CreateUserPool struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`

	// Ceiling of the organizations born in this pool, capped by the ceiling of the
	// one creating it. Defaults to LOGIN_PROFILE, so a new pool is born closed.
	DefaultProfileId string `json:"default_profile_id" validate:"omitempty,uuid4"`
}
