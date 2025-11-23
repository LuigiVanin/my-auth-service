package errors

import "github.com/gofiber/fiber/v2"

var HttpErrorMap = map[GlobalErrorCode]int{

	// 400
	BadRequestCode: fiber.StatusBadRequest,

	// 401
	UnauthorizedErrorCode: fiber.StatusUnauthorized,
	TokenExpiredErrorCode: fiber.StatusUnauthorized,

	// 409
	ConflictErrorCode:     fiber.StatusConflict,
	UserAlreadyExistsCode: fiber.StatusConflict,

	// 422
	UnprocessableEntityErrorCode: fiber.StatusUnprocessableEntity,
	ValidationErrorCode:          fiber.StatusUnprocessableEntity,

	// 500
	InternalServerErrorCode: fiber.StatusInternalServerError,
}
