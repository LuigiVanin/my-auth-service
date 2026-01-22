package main

import (
	"context"

	"auth_service/app/middlewares/guards"
	"auth_service/app/modules/api/authorize"
	"auth_service/app/modules/api/login"
	"auth_service/app/modules/api/register"
	"auth_service/app/modules/core/app"
	"auth_service/app/modules/core/otp"
	"auth_service/app/modules/core/profile"
	"auth_service/app/modules/core/session"
	"auth_service/app/modules/core/user_pool"
	"auth_service/app/modules/utils/cipher"
	"auth_service/app/modules/utils/hash"
	"auth_service/app/modules/utils/jwt"
	"auth_service/infra/bootstrap"
	"auth_service/infra/config"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {

	fx.New(
		fx.Provide(config.NewConfigFromEnv),
		fx.Provide(bootstrap.NewZapLogger),
		fx.Provide(bootstrap.NewDatabase),
		fx.Provide(bootstrap.NewHttpServer),

		fx.Provide(guards.NewAppGuard),

		// Utils
		cipher.Module,
		hash.Module,
		jwt.Module,

		// Core
		app.Module,
		user_pool.Module,
		session.Module,
		profile.Module,
		otp.Module,

		// API
		register.Module,
		login.Module,
		authorize.Module,

		fx.Invoke(bootstrap.StartServer),

		fx.Invoke(func(lifecycle fx.Lifecycle, logger *zap.Logger) {
			lifecycle.Append(fx.Hook{
				OnStop: func(_ context.Context) error {
					return logger.Sync()
				},
			})
		}),
	).Run()
}
