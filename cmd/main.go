package main

import (
	"auth_service/app/middlewares/guards"
	"auth_service/app/modules/app"
	"auth_service/app/modules/cipher"
	"auth_service/app/modules/hash"
	"auth_service/app/modules/register"
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

		cipher.Module,
		hash.Module,
		app.Module,
		user_pool.Module,
		register.Module,

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
