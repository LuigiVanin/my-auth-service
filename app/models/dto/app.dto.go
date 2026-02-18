package dto

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
