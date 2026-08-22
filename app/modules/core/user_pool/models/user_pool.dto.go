package models

type CreateUserPool struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}
