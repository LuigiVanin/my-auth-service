package constants

type AuthAction string

const (
	ActionLogin             AuthAction = "LOGIN"
	ActionRegister          AuthAction = "REGISTER"
	ActionVerifyEmail       AuthAction = "VERIFY_EMAIL"
	ActionTwoFA             AuthAction = "TWO_FA"
	ActionForgotPassword    AuthAction = "FORGOT_PASSWORD"
	ActionChangeEmail       AuthAction = "CHANGE_EMAIL"
	ActionRegenAppSecretKey AuthAction = "REGEN_APP_SECRET_KEY"
)

var AuthActions = []string{
	string(ActionLogin),
	string(ActionRegister),
	string(ActionVerifyEmail),
	string(ActionTwoFA),
	string(ActionForgotPassword),
	string(ActionChangeEmail),
	string(ActionRegenAppSecretKey),
}
