package main

import (
	"auth_service/app/middlewares/guards"
	"auth_service/app/modules/app"
	"auth_service/app/modules/authorize"
	"auth_service/app/modules/cipher"
	"auth_service/app/modules/hash"
	"auth_service/app/modules/jwt"
	"auth_service/app/modules/login"
	"auth_service/app/modules/register"
	"auth_service/app/modules/router"

	"auth_service/app/modules/profile"
	"auth_service/app/modules/session"
	"auth_service/app/modules/user_pool"
	"auth_service/infra/bootstrap"
	"auth_service/infra/config"
	"context"

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

		// Entities
		app.Module,
		user_pool.Module,
		session.Module,
		profile.Module,

		// API
		router.Module,
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
