package plugins

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/flowchartsman/swaggerui"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type SwaggerPluginOptions struct {
	Path string

	Info openapi.Info
}

type Swagger struct {
	Handler fiber.Handler
	adapter fiber.Handler
}

func NewSwagger(url string, document openapi.Document) Swagger {
	content, err := document.Output("json")

	if err != nil {
		panic(err.Error())
	}

	swagger := Swagger{
		adapter: adaptor.HTTPHandler(
			http.StripPrefix(url, swaggerui.Handler([]byte(content))),
		),
	}

	swagger.Handler = func(ctx fiber.Ctx) error {
		if ctx.Path() == url {
			return ctx.
				Redirect().
				Status(fiber.StatusFound).
				To(fmt.Sprintf("%s/", url))
		}

		return swagger.adapter(ctx)
	}

	return swagger
}

func SwaggerPlugin(options ...SwaggerPluginOptions) fx.Option {
	option := SwaggerPluginOptions{
		Path: "/docs/",
		Info: openapi.Info{
			Title:       "Auth Service",
			Description: "Api documentation for auth service",
			Version:     "0.1.0",
		},
	}

	if len(options) > 0 {
		option = options[0]
	}

	return fx.Options(
		fx.Provide(
			func() *openapi.Builder {
				return openapi.NewBuilder(
					option.Info.Title,
					option.Info.Description,
					option.Info.Version,
				)
			},
		),
		fx.Invoke(
			func(server *fiber.App, lifecycle fx.Lifecycle, logger *zap.Logger, swagger *openapi.Builder) {
				document := swagger.Build()
				url := strings.TrimSuffix(option.Path, "/")

				swaggerAdapter := NewSwagger(url, *document)

				server.Get(fmt.Sprintf("%s/*", url), swaggerAdapter.Handler)
			},
		),
	)
}
