package main

import (
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
		fx.Provide(bootstrap.NewHttpServer),

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
