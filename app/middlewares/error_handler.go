package middleware

import (
	e "auth_service/common/errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func logError(logger *zap.Logger, ctx *fiber.Ctx, message string, fields ...zap.Field) {
	defaultFields := []zap.Field{
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.OriginalURL()),
		zap.String("host", ctx.Hostname()),
		zap.String("ip", ctx.IP()),
	}

	allFields := append(defaultFields, fields...)
	logger.Error(message, allFields...)
}

func NewErrorHandler(logger *zap.Logger) fiber.ErrorHandler {
	return func(ctx *fiber.Ctx, requestError error) error {
		instance := ctx.OriginalURL()

		if appErr, ok := requestError.(*e.GlobalError); ok {
			problemDetail := appErr.IntoProblemDetail(instance)

			logError(logger, ctx, fmt.Sprintf("Request error: %s", problemDetail.Detail),
				zap.String("code", string(appErr.Code.First)),
				zap.Int("status", problemDetail.Status),
				zap.String("detail", appErr.Detail),
			)

			return ctx.
				Status(problemDetail.Status).
				JSON(problemDetail)
		}

		if validationErr, ok := requestError.(ValidationError); ok {
			logError(logger, ctx, "Validation error",
				zap.String("detail", validationErr.Error()),
			)

			return ctx.Status(fiber.StatusUnprocessableEntity).JSON(
				fiber.Map{
					"title":    "Validation error",
					"status":   fiber.StatusUnprocessableEntity,
					"detail":   validationErr.Error(),
					"instance": instance,
					"code":     e.UnprocessableEntityErrorCode.First,
					"data": fiber.Map{
						"errors": validationErr.List,
					},
				},
			)
		}

		logError(logger, ctx, "Unexpected error",
			zap.Error(requestError),
		)

		return ctx.Status(fiber.StatusInternalServerError).JSON(
			e.NewProblemDetail(
				"about:blank",
				"Unexpected Internal Error",
				fiber.StatusInternalServerError,
				requestError.Error(),
				instance,
				"",
			),
		)
	}
}
