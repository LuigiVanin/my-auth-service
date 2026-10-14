package models

type RequestUpload struct {
	ContentType string `json:"content_type" validate:"required,oneof=image/png image/jpeg image/webp"`
}

type ConfirmUpload struct {
	Key string `json:"key" validate:"required"`
}
