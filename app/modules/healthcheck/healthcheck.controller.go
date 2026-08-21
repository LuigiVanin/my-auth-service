package healthcheck

import (
	"auth_service/app/apidocs"
	"auth_service/shared/interfaces"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/gofiber/fiber/v2"
)

var _ interfaces.IController = &HealthCheckController{}

// healthCheckResponse mirrors the body returned by Status - it only exists to
// describe the payload in the OpenAPI document.
type healthCheckResponse struct {
	Status string `json:"status"`
}

type HealthCheckController struct {
	swagger *openapi.Builder
}

func NewHealthCheckController(builder *openapi.Builder) *HealthCheckController {
	return &HealthCheckController{
		swagger: builder,
	}
}

func (this *HealthCheckController) Status(ctx *fiber.Ctx) error {

	return ctx.JSON(fiber.Map{
		"status": "ok",
	})
}

func (this *HealthCheckController) Register(ctx *fiber.App) {
	this.swagger.Add(
		apidocs.PublicRoute(this.swagger, "GET", "/health-check", openapi.Options{
			Summary:     "Health check",
			Description: "Tells whether the service is up. Open endpoint - no application credentials required.",
			Tags:        []string{apidocs.TagHealth},
		}).
			AddResponse(fiber.StatusOK, healthCheckResponse{}, openapi.Options{
				Description: "Service is up",
			}),
	)

	ctx.Get("/health-check", this.Status)
}
