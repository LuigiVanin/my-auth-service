package errors

import (
	"auth_service/common/utils"
	"fmt"
	"maps"

	"github.com/gofiber/fiber/v2"
)

type GlobalErrorCode string

type GlobalError struct {
	Title  string
	Code   ErrorCodePair
	Detail string
	Type   string
	Extra  utils.JSON
}

func NewGlobalError(title string, detail string, code ErrorCodePair, type_ string, extra ...utils.JSON) *GlobalError {

	extraInfo := make(utils.JSON)

	for _, extra := range extra {
		maps.Copy(extraInfo, extra)
	}

	return &GlobalError{
		Title:  title,
		Code:   code,
		Detail: detail,
		Extra:  extraInfo,
		Type:   type_,
	}
}

func ThrowNotAllowed(detail string, extra ...utils.JSON) *GlobalError {
	return NewGlobalError(
		"Not Allowed",
		detail,
		NotAllowedErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/405",
		extra...,
	)
}

func ThrowBadRequest(detail string, extra ...utils.JSON) *GlobalError {
	return NewGlobalError(
		"Bad Request",
		detail,
		BadRequestCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/400",
		extra...,
	)
}

func ThrowConflict(detail string, extra ...utils.JSON) *GlobalError {
	return NewGlobalError(
		"Conflict",
		detail,
		ConflictErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/409",
		extra...,
	)
}

func ThrowUserAlreadyExists(detail string, extra ...utils.JSON) *GlobalError {
	return NewGlobalError(
		"Conflict",
		detail,
		UserAlreadyExistsCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/409",
		extra...,
	)
}

func ThrowNotFound(detail string, extra ...utils.JSON) *GlobalError {
	return NewGlobalError(
		"Not Found",
		detail,
		NotFoundErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/404",
		extra...,
	)
}

func ThrowUnauthorizedError(detail string, extra ...utils.JSON) *GlobalError {
	return NewGlobalError(
		"Unauthorized",
		detail,
		UnauthorizedErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/401",
		extra...,
	)
}

func ThrowTokenExpiredError(detail string, extra ...utils.JSON) *GlobalError {
	return NewGlobalError(
		"Token Expired",
		detail,
		TokenExpiredErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/401",
		extra...,
	)
}

func ThrowInvalidFormatError(detail string, extra ...utils.JSON) *GlobalError {
	return NewGlobalError(
		"Invalid Format",
		detail,
		InvalidFormatErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/401",
		extra...,
	)
}

func ThrowSignatureFaildError(detail string, extra ...utils.JSON) *GlobalError {
	return NewGlobalError(
		"Signature Failed",
		detail,
		SignatureFailErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/401",
		extra...,
	)
}

func ThrowUnprocessableEntity(detail string, extra ...utils.JSON) *GlobalError {
	return NewGlobalError(
		"Unprocessable Entity",
		detail,
		UnprocessableEntityErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/422",
		extra...,
	)
}

func ThrowInternalServerError(detail string, extra ...utils.JSON) *GlobalError {
	return NewGlobalError(
		"Internal Server Error",
		detail,
		InternalServerErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/500",
		extra...,
	)
}

func ThrowNotImplementedError(detail string, extra ...utils.JSON) *GlobalError {
	return NewGlobalError(
		"Not Implemented",
		detail,
		NotImplementedErrorCode,
		"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/501",
		extra...,
	)
}

func (e *GlobalError) Error() string {

	return fmt.Sprintf("GlobalError: %s, Code: %s, Detail: %s", e.Title, e.Code.First, e.Detail)
}

func (e *GlobalError) IntoProblemDetail(instance string) *ProblemDetail {
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
