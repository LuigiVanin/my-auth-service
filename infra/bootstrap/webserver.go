package bootstrap

import (
	middleware "auth_service/app/middlewares"
	"auth_service/infra/config"
	"context"
	"fmt"
	"net"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewHttpServer(cfg *config.Config, logger *zap.Logger) *fiber.App {

	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.NewErrorHandler(logger),

		AppName:      cfg.App.Name,
		ServerHeader: cfg.App.Name,
		BodyLimit:    1024 * 1024 * 5, // 5MB
		ReadTimeout:  15 * time.Minute,
		WriteTimeout: 15 * time.Minute,
		IdleTimeout:  15 * time.Minute,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
	}))

	app.Use(middleware.Json)

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
				addr := fmt.Sprintf(":%s", config.Server.Port)
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
