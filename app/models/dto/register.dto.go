package dto

import "encoding/json"

type RegisterRequestWithPassoword struct {
	Email    string  `json:"email" validate:"required,email"`
	Password string  `json:"password" validate:"required,min=8"`
	Name     string  `json:"name" validate:"required"`
	Phone    *string `json:"phone"`

	Metadata json.RawMessage `json:"metadata" validate:"required"`
}

type RegisterRequestWithOtp struct {
	Email string  `json:"email" validate:"required,email"`
	Phone *string `json:"phone"`
	Name  string  `json:"name" validate:"required"`

	Metadata json.RawMessage `json:"metadata" validate:"required"`
}
