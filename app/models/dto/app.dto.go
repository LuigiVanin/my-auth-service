package dto

import entity "auth_service/infra/entities"

type CreateAppPayloadUserPool struct {
	Id   string `json:"id" validate:"required_without=Name"`
	Name string `json:"name" validate:"required_without=Id"`
}

type CreateAppPayload struct {
	Name       string   `json:"name" validate:"required"`
	LoginTypes []string `json:"login_types" validate:"required,dive,oneof=WITH_LOGIN WITH_OTP WITH_PASSWORD"`
	TokenType  string   `json:"token_type" validate:"required,oneof=JWT FAST_JWT SESSION_UUID"`

	TokenExpirationTime        int64 `json:"token_expiration_time" validate:"required,numeric,gt=0"`
	RefreshTokenExpirationTime int64 `json:"refresh_token_expiration_time" validate:"required,numeric,gt=0"`

	Private bool `json:"private"`

	VerifyEmail bool `json:"verify_email" validate:"required"`

	UserPool CreateAppPayloadUserPool `json:"user_pool" validate:"required"`
}

type GetAppsQuery struct {
	Skip  int    `query:"skip"`
	Limit int    `query:"limit"`
	Name  string `query:"name"`

	UserOwnerId int64 `query:"user_owner_id"`
}

type GetAppsResponse struct {
	Total  int64        `json:"total"`
	Amount int          `json:"amount"`
	Skip   int          `json:"skip"`
	Data   []entity.App `json:"data"`
}
