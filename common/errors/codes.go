package errors

const (
	InternalServerErrorCode GlobalErrorCode = "INTERNAL_SERVER_ERROR"

	BadRequestCode GlobalErrorCode = "BAD_REQUEST"

	UnauthorizedErrorCode GlobalErrorCode = "UNAUTHORIZED"
	TokenExpiredErrorCode GlobalErrorCode = "TOKEN_EXPIRED"

	ConflictErrorCode     GlobalErrorCode = "CONFLICT"
	UserAlreadyExistsCode GlobalErrorCode = "USER_ALREADY_EXISTS"

	UnprocessableEntityErrorCode GlobalErrorCode = "UNPROCESSABLE_ENTITY"
	ValidationErrorCode          GlobalErrorCode = "VALIDATION_ERROR"

	NotFoundErrorCode GlobalErrorCode = "NOT_FOUND"
)
