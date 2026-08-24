package errors

import (
	"auth_service/shared/utils"

	"github.com/gofiber/fiber/v3"
)

type ErrorCodePair = utils.Pair[AppErrorCode, int]

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

	InvalidOtpCodeErrorCode ErrorCodePair = ErrorCodePair{
		First:  "INVALID_OTP_CODE",
		Second: fiber.StatusBadRequest,
	}

	// Unauthorized 401
	UnauthorizedErrorCode ErrorCodePair = ErrorCodePair{
		First:  "UNAUTHORIZED",
		Second: fiber.StatusUnauthorized,
	}

	PermissionDeniedErrorCode ErrorCodePair = ErrorCodePair{
		First:  "PERMISSION_DENIED",
		Second: fiber.StatusForbidden,
	}

	// Distinct from PERMISSION_DENIED because the client reacts differently: the
	// scope is the problem, not the profile, so the fix is to switch organization.
	NotAParticipantErrorCode ErrorCodePair = ErrorCodePair{
		First:  "NOT_A_PARTICIPANT",
		Second: fiber.StatusForbidden,
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

	// Too Many Requests 429
	TooManyRequestsErrorCode ErrorCodePair = ErrorCodePair{
		First:  "TOO_MANY_REQUESTS",
		Second: fiber.StatusTooManyRequests,
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
)
