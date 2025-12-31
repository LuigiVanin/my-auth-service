package dto

type LoginPayloadWithPassoword struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type LoginPayloadWithOtp struct {
	Email string `json:"email" validate:"required,email"`

	Otp OtpPayload `json:"otp"`
}

type OtpPayload struct {
	OtpCode string `json:"otpCode" validate:"required,min=6,max=6"`
	OtpId   string `json:"otpId" validate:"required"`
}
