package errors

import (
	"auth_service/common/utils"

	"github.com/gofiber/fiber/v2"
)

type ErrorCodePair = utils.Pair[GlobalErrorCode, int]

var (
	// Internal Server 500
	InternalServerErrorCode ErrorCodePair = ErrorCodePair{
		First:  "INTERNAL_SERVER_ERROR",
		Second: fiber.StatusInternalServerError,
	}

	// Not Implemented 501
	NotImplementedErrorCode ErrorCodePair = ErrorCodePair{
		First:  "NOT_IMPLEMENTED",
		Second: fiber.StatusNotImplemented,
	}

	// Bad Request 400
	BadRequestCode ErrorCodePair = ErrorCodePair{
		First:  "BAD_REQUEST",
		Second: fiber.StatusBadRequest,
	}

	// Unauthorized 401
	UnauthorizedErrorCode ErrorCodePair = ErrorCodePair{
		First:  "UNAUTHORIZED",
		Second: fiber.StatusUnauthorized,
	}

	TokenExpiredErrorCode ErrorCodePair = ErrorCodePair{
		First:  "TOKEN_EXPIRED",
		Second: fiber.StatusUnauthorized,
	}

	InvalidFormatErrorCode ErrorCodePair = ErrorCodePair{
		First:  "INVALID_FORMAT",
		Second: fiber.StatusUnauthorized,
	}

	SignatureFailErrorCode ErrorCodePair = ErrorCodePair{
		First:  "SIGNATURE_FAIL",
		Second: fiber.StatusUnauthorized,
	}

	// Not Found 404
	NotFoundErrorCode ErrorCodePair = ErrorCodePair{
		First:  "NOT_FOUND",
		Second: fiber.StatusNotFound,
	}

	// Method Not Allowed 405
	NotAllowedErrorCode ErrorCodePair = ErrorCodePair{
		First:  "NOT_ALLOWED",
		Second: fiber.StatusMethodNotAllowed,
	}

	// Conflict 409
	ConflictErrorCode ErrorCodePair = ErrorCodePair{
		First:  "CONFLICT",
		Second: fiber.StatusConflict,
	}

	// Conflict 409
	UserAlreadyExistsCode ErrorCodePair = ErrorCodePair{
		First:  "USER_ALREADY_EXISTS",
		Second: fiber.StatusConflict,
	}

	// Unprocessable Entity 422
	UnprocessableEntityErrorCode ErrorCodePair = ErrorCodePair{
		First:  "UNPROCESSABLE_ENTITY",
		Second: fiber.StatusUnprocessableEntity,
	}

	ValidationErrorCode ErrorCodePair = ErrorCodePair{
		First:  "VALIDATION_ERROR",
		Second: fiber.StatusUnprocessableEntity,
	}
)
