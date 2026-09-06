package main

import (
	"context"

	"auth_service/app/middlewares/guards"
	"auth_service/plugins"
	"auth_service/shared/interfaces"

	"auth_service/app/modules/authorize"
	"auth_service/app/modules/core/app"
	"auth_service/app/modules/core/organization"
	"auth_service/app/modules/core/otp"
	"auth_service/app/modules/core/participant"
	"auth_service/app/modules/core/profile"
	"auth_service/app/modules/core/session"
	"auth_service/app/modules/core/user"
	"auth_service/app/modules/core/user_pool"
	"auth_service/app/modules/healthcheck"
	"auth_service/app/modules/login"
	"auth_service/app/modules/register"
	"auth_service/app/modules/utils/cipher"
	"auth_service/app/modules/utils/hash"
	"auth_service/app/modules/utils/jwt"
	"auth_service/infra/bootstrap"
	"auth_service/infra/config"
	repo "auth_service/shared/repository"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	fx.New(AppBootstrap()).Run()
}

func AppBootstrap() fx.Option {
	swaggerOptions := plugins.SwaggerPluginOptions{
		Path: "/api/docs",
		Info: openapi.Info{
			Title:       "Auth Service",
			Description: "Description",
			Version:     "v1.0.0"},
	}

	return fx.Options(
		fx.Provide(config.CreateEnvConfig),
		fx.Provide(bootstrap.NewZapLogger),
		fx.Provide(bootstrap.NewDatabase),
		fx.Provide(bootstrap.NewHttpServer),
		fx.Provide(bootstrap.NewEmailManager),

		fx.Provide(
			fx.Annotate(
				repo.NewTransactionManager,
				fx.As(new(repo.ITransactionManager)),
			),
		),

		fx.Provide(guards.NewAppGuard),
		fx.Provide(guards.NewAuthGuard),
		fx.Provide(guards.NewOtpGuard),
		fx.Provide(guards.NewOrganizationGuard),
		fx.Provide(guards.NewPermissionsGuard),

		// Utils
		cipher.Module,
		hash.Module,
		jwt.Module,

		// Core
		user.Module,
		app.Module,
		user_pool.Module,
		session.Module,
		profile.Module,
		organization.Module,
		participant.Module,
		otp.Module,

		// API
		register.Module,
		login.Module,
		authorize.Module,
		healthcheck.Module,

		fx.Invoke(
			fx.Annotate(
				func(server *fiber.App, controllers []interfaces.IController) {
					for _, controller := range controllers {
						controller.Register(server)
					}
				},
				fx.ParamTags(``, `group:"controllers"`),
			),
		),
		plugins.SwaggerPlugin(swaggerOptions),
		fx.Invoke(func(lifecycle fx.Lifecycle, logger *zap.Logger) {
			lifecycle.Append(fx.Hook{
				OnStop: func(_ context.Context) error {
					return logger.Sync()
				},
			})
		}),
		fx.Invoke(bootstrap.StartServer),

		// Disabling Fx Logging - too verbose
		fx.NopLogger,
	)
}
