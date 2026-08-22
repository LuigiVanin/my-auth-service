package errors

import (
	"auth_service/shared/utils"
	"fmt"
	"maps"

	"github.com/gofiber/fiber/v3"
)

type AppErrorCode string

type AppError struct {
	Title  string
	Code   ErrorCodePair
	Detail string
	Type   string
	Extra  utils.JSON
}

func NewAppError(title string, detail string, code ErrorCodePair, type_ string, extra ...utils.JSON) *AppError {

	extraInfo := make(utils.JSON)

	for _, extra := range extra {
		maps.Copy(extraInfo, extra)
	}

	return &AppError{
		Title:  title,
		Code:   code,
		Detail: detail,
		Extra:  extraInfo,
		Type:   type_,
	}
}

func ThrowNotAllowed(detail string, extra ...utils.JSON) *AppError {
	return NewAppError(
		"Not Allowed",
		detail,
		NotAllowedErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/405",
		extra...,
	)
}

func ThrowTooManyRequests(detail string, extra ...utils.JSON) *AppError {
	return NewAppError(
		"Too Many Requests",
		detail,
		TooManyRequestsErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/429",
		extra...,
	)
}

func ThrowBadRequest(detail string, extra ...utils.JSON) *AppError {
	return NewAppError(
		"Bad Request",
		detail,
		BadRequestCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/400",
		extra...,
	)
}

func ThrowValidationError(detail string, extra ...utils.JSON) *AppError {
	return NewAppError(
		"Bad Request",
		detail,
		BadRequestCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/400",
		extra...,
	)
}

func ThrowInvalidOtpCode(detail string, extra ...utils.JSON) *AppError {
	return NewAppError(
		"Invalid OTP Code",
		detail,
		InvalidOtpCodeErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/400",
		extra...,
	)
}

func ThrowConflict(detail string, extra ...utils.JSON) *AppError {
	return NewAppError(
		"Conflict",
		detail,
		ConflictErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/409",
		extra...,
	)
}

func ThrowUserAlreadyExists(detail string, extra ...utils.JSON) *AppError {
	return NewAppError(
		"Conflict",
		detail,
		UserAlreadyExistsCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/409",
		extra...,
	)
}

func ThrowNotFound(detail string, extra ...utils.JSON) *AppError {
	return NewAppError(
		"Not Found",
		detail,
		NotFoundErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/404",
		extra...,
	)
}

func ThrowUnauthorizedError(detail string, extra ...utils.JSON) *AppError {
	return NewAppError(
		"Unauthorized",
		detail,
		UnauthorizedErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/401",
		extra...,
	)
}

func ThrowPermissionDeniedError(detail string, extra ...utils.JSON) *AppError {
	return NewAppError(
		"Permission Denied",
		detail,
		PermissionDeniedErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/401",
		extra...,
	)
}

func ThrowTokenExpiredError(detail string, extra ...utils.JSON) *AppError {
	return NewAppError(
		"Token Expired",
		detail,
		TokenExpiredErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/401",
		extra...,
	)
}

func ThrowInvalidFormatError(detail string, extra ...utils.JSON) *AppError {
	return NewAppError(
		"Invalid Format",
		detail,
		InvalidFormatErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/401",
		extra...,
	)
}

func ThrowSignatureFaildError(detail string, extra ...utils.JSON) *AppError {
	return NewAppError(
		"Signature Failed",
		detail,
		SignatureFailErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/401",
		extra...,
	)
}

func ThrowUnprocessableEntity(detail string, extra ...utils.JSON) *AppError {
	return NewAppError(
		"Unprocessable Entity",
		detail,
		UnprocessableEntityErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/422",
		extra...,
	)
}

func ThrowInternalServerError(detail string, extra ...utils.JSON) *AppError {
	return NewAppError(
		"Internal Server Error",
		detail,
		InternalServerErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/500",
		extra...,
	)
}

func ThrowNotImplementedError(detail string, extra ...utils.JSON) *AppError {
	return NewAppError(
		"Not Implemented",
		detail,
		NotImplementedErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/501",
		extra...,
	)
}

func (e *AppError) Error() string {

	return fmt.Sprintf("AppError: %s, Code: %s, Detail: %s", e.Title, e.Code.First, e.Detail)
}

func (e *AppError) IntoProblemDetail(instance string) *ProblemDetail {
	status := e.Code.Second

	if status == 0 {
		status = fiber.StatusInternalServerError
	}

	return &ProblemDetail{
		Type:     e.Type,
		Title:    e.Title,
		Detail:   e.Detail,
		Instance: instance,
		Code:     string(e.Code.First),
		Status:   status,
		Data:     e.Extra,
	}
}
