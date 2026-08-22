package bootstrap

import (
	"context"
	"fmt"
	"net"
	"runtime/debug"
	"time"

	e "auth_service/app/errors"
	middleware "auth_service/app/middlewares"
	"auth_service/app/middlewares/guards"
	"auth_service/infra/config"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func newRecoverer(logger *zap.Logger) fiber.Handler {
	return recover.New(recover.Config{
		EnableStackTrace: true,

		StackTraceHandler: func(ctx fiber.Ctx, panicValue any) {
			logger.Error("Recovered from panic",
				zap.Any("panic", panicValue),
				zap.String("method", ctx.Method()),
				zap.String("path", ctx.OriginalURL()),
				zap.String("ip", ctx.IP()),
				zap.ByteString("stack", debug.Stack()),
			)
		},

		PanicHandler: func(_ fiber.Ctx, _ any) error {
			return e.ThrowInternalServerError("Unexpected internal error")
		},
	})
}

func NewHttpServer(cfg *config.Config, logger *zap.Logger, appGuard *guards.AppGuard) *fiber.App {

	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.NewErrorHandler(logger),

		AppName:      cfg.App.Name,
		ServerHeader: cfg.App.Name,
		BodyLimit:    1024 * 1024 * 5, // 5MB
		ReadTimeout:  15 * time.Minute,
		WriteTimeout: 15 * time.Minute,
		IdleTimeout:  15 * time.Minute,
	})

	// First in the chain, so that it wraps every middleware and handler
	// registered after it - cors, the guards and the controllers.
	app.Use(newRecoverer(logger))

	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			fiber.MethodGet,
			fiber.MethodPost,
			fiber.MethodPut,
			fiber.MethodDelete,
			fiber.MethodOptions,
		},
	}))
	app.Use(middleware.Json)

	RegisterGuards(app, appGuard)

	return app
}

func StartServer(
	lifecycle fx.Lifecycle,
	server *fiber.App,
	config *config.Config,
) {
	lifecycle.Append(
		fx.Hook{
			OnStart: func(_ context.Context) error {

				// Listen on all interfaces (0.0.0.0)
				addr := fmt.Sprintf("0.0.0.0:%s", config.Server.Port)
				ln, err := net.Listen("tcp", addr)

				if err != nil {
					fmt.Println("Failed to bind to port", err)
					return err
				}

				go func() {
					err := server.Listener(ln)
					if err != nil {
						fmt.Println("Error starting server", err)
					}
				}()

				return nil
			},

			OnStop: func(_ context.Context) error {
				return server.Shutdown()
			},
		},
	)
}
