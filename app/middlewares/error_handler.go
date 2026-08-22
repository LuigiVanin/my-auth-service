package middleware

import (
	e "auth_service/app/errors"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

func logError(logger *zap.Logger, ctx fiber.Ctx, message string, fields ...zap.Field) {
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
	return func(ctx fiber.Ctx, requestError error) error {
		instance := ctx.OriginalURL()

		if appErr, ok := requestError.(*e.AppError); ok {
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

		// LEGACY: Now the validator middleware throw an AppError, there is no need to
		// treat this differentally
		// if validationErr, ok := requestError.(ValidationError); ok {
		// 	logError(logger, ctx, "Validation error",
		// 		zap.String("detail", validationErr.Error()),
		// 	)

		// 	return ctx.Status(fiber.StatusUnprocessableEntity).JSON(
		// 		fiber.Map{
		// 			"title":    "Validation error",
		// 			"status":   fiber.StatusUnprocessableEntity,
		// 			"detail":   validationErr.Error(),
		// 			"instance": instance,
		// 			"code":     e.UnprocessableEntityErrorCode.First,
		// 			"data": fiber.Map{
		// 				"errors": validationErr.List,
		// 			},
		// 		},
		// 	)
		// }

		// A payload the binder could not read - malformed json body, a query value
		// that does not fit the field it binds to. Checked before *fiber.Error
		// because it is the more specific type of the two.
		var bindError *fiber.BindError

		if errors.As(requestError, &bindError) {
			problemDetail := e.FromBindError(bindError).IntoProblemDetail(instance)

			logError(logger, ctx, fmt.Sprintf("Request error: %s", problemDetail.Detail),
				zap.String("code", problemDetail.Code),
				zap.Int("status", problemDetail.Status),
				zap.String("source", bindError.Source),
			)

			return ctx.
				Status(problemDetail.Status).
				JSON(problemDetail)
		}

		// Errors Fiber raises on its own - an unregistered route or method, a body
		// over BodyLimit, a content type no binder handles - arrive here as
		// *fiber.Error. They carry no code of the service, so they are mapped onto
		// one to answer the same problem detail body as every other error.
		var fiberError *fiber.Error

		if errors.As(requestError, &fiberError) {
			detail := ""

			// A request no route matched reaches this handler as a bare 404 whose
			// message is just the status text. Naming the route the caller asked for
			// makes the answer say what was actually wrong.
			if fiberError.Code == fiber.StatusNotFound && !ctx.Matched() {
				detail = fmt.Sprintf(
					"Route `%s %s` does not exist",
					ctx.Method(),
					ctx.Path(),
				)
			}

			problemDetail := e.FromFiberError(fiberError, detail).IntoProblemDetail(instance)

			logError(logger, ctx, fmt.Sprintf("Request error: %s", problemDetail.Detail),
				zap.String("code", problemDetail.Code),
				zap.Int("status", problemDetail.Status),
				zap.String("detail", problemDetail.Detail),
			)

			return ctx.
				Status(problemDetail.Status).
				JSON(problemDetail)
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
