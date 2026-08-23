package healthcheck

import (
	"auth_service/app/docs"
	"auth_service/shared/interfaces"
	"auth_service/shared/utils"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

var _ interfaces.IController = &HealthCheckController{}

type healthCheckResponse struct {
	Status string `json:"status"`

	Database utils.DatabaseInfo `json:"database"`
}

type HealthCheckController struct {
	swagger *openapi.Builder
	client  *gorm.DB
}

func NewHealthCheckController(builder *openapi.Builder, client *gorm.DB) *HealthCheckController {
	return &HealthCheckController{
		swagger: builder,
		client:  client,
	}
}

func (this *HealthCheckController) Status(ctx fiber.Ctx) error {

	dbInfo := utils.GetDatabaseInfo(this.client)

	return ctx.JSON(utils.JSON{
		"status":   utils.OK,
		"database": dbInfo.ToJSON(),
	})
}

func (this *HealthCheckController) Register(ctx *fiber.App) {
	this.swagger.Add(
		docs.PublicRoute(this.swagger, "GET", "/health-check", openapi.Options{
			Summary:     "Health check",
			Description: "Tells whether the service is up. Open endpoint - no application credentials required.",
			Tags:        []string{docs.TagHealth},
		}).
			AddResponse(fiber.StatusOK, healthCheckResponse{}, openapi.Options{
				Description: "Service is up",
			}),
	)

	ctx.Get("/health-check", this.Status)
}
