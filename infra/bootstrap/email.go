package bootstrap

import (
	"auth_service/infra/config"
	"auth_service/shared/email"
	"auth_service/shared/email/adapters"

	"go.uber.org/zap"
)

func NewEmailManager(cfg *config.Config, logger *zap.Logger) *email.EmailManager {
	apiKey := cfg.Email.ResendApiKey

	if apiKey == "" {
		logger.Warn("RESEND_API_KEY is not set")
	}

	return email.NewEmailManager(
		adapters.NewResendAdapter(adapters.ResendOptions{
			ApiKey: apiKey,
		}),
	)
}
