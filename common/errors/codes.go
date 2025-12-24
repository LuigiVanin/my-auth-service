package errors

import (
	"auth_service/common/utils"

	"github.com/gofiber/fiber/v2"
)

type ErrorCodePair = utils.Pair[GlobalErrorCode, int]

var (
	InternalServerErrorCode ErrorCodePair = ErrorCodePair{
		First:  "INTERNAL_SERVER_ERROR",
		Second: fiber.StatusInternalServerError,
	}
	NotImplementedErrorCode ErrorCodePair = ErrorCodePair{
		First:  "NOT_IMPLEMENTED",
		Second: fiber.StatusNotImplemented,
	}

	BadRequestCode ErrorCodePair = ErrorCodePair{
		First:  "BAD_REQUEST",
		Second: fiber.StatusBadRequest,
	}

	UnauthorizedErrorCode ErrorCodePair = ErrorCodePair{
		First:  "UNAUTHORIZED",
		Second: fiber.StatusUnauthorized,
	}
	TokenExpiredErrorCode ErrorCodePair = ErrorCodePair{
		First:  "TOKEN_EXPIRED",
		Second: fiber.StatusUnauthorized,
	}

	ConflictErrorCode ErrorCodePair = ErrorCodePair{
		First:  "CONFLICT",
		Second: fiber.StatusConflict,
	}
	UserAlreadyExistsCode ErrorCodePair = ErrorCodePair{
		First:  "USER_ALREADY_EXISTS",
		Second: fiber.StatusConflict,
	}

	UnprocessableEntityErrorCode ErrorCodePair = ErrorCodePair{
		First:  "UNPROCESSABLE_ENTITY",
		Second: fiber.StatusUnprocessableEntity,
	}
	ValidationErrorCode ErrorCodePair = ErrorCodePair{
		First:  "VALIDATION_ERROR",
		Second: fiber.StatusUnprocessableEntity,
	}

	NotFoundErrorCode ErrorCodePair = ErrorCodePair{
		First:  "NOT_FOUND",
		Second: fiber.StatusNotFound,
	}
)
