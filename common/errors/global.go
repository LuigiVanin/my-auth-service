package errors

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

type GlobalErrorCode string

type GlobalError struct {
	Title  string
	Code   ErrorCodePair
	Detail string
	Type   string
}

func NewGlobalError(title string, code ErrorCodePair, detail string) *GlobalError {

	return &GlobalError{
		Title:  title,
		Code:   code,
		Detail: detail,
	}
}

func ThrowBadRequest(detail string) *GlobalError {
	return &GlobalError{
		Title:  "Bad Request",
		Code:   BadRequestCode,
		Detail: detail,
		Type:   "https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/400",
	}
}

func ThrowConflict(detail string) *GlobalError {
	return &GlobalError{
		Title:  "Conflict",
		Code:   ConflictErrorCode,
		Detail: detail,
		Type:   "https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/409",
	}
}

func ThrowUserAlreadyExists(detail string) *GlobalError {
	return &GlobalError{
		Title:  "Conflict",
		Code:   UserAlreadyExistsCode,
		Detail: detail,
		Type:   "https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/401",
	}
}

func ThrowNotFound(detail string) *GlobalError {
	return &GlobalError{
		Title:  "Not Found",
		Code:   NotFoundErrorCode,
		Detail: detail,
		Type:   "https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/404",
	}
}

func ThrowUnauthorizedError(detail string) *GlobalError {
	return &GlobalError{
		Title:  "Unauthorized",
		Code:   UnauthorizedErrorCode,
		Detail: detail,
		Type:   "https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/401",
	}
}

func ThrowTokenExpiredError(detail string) *GlobalError {
	return &GlobalError{
		Title:  "Token Expired",
		Code:   TokenExpiredErrorCode,
		Detail: detail,
		Type:   "https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/401",
	}
}

func ThrowUnprocessableEntity(detail string) *GlobalError {

	return &GlobalError{
		Title:  "Unprocessable Entity",
		Code:   UnprocessableEntityErrorCode,
		Detail: detail,
		Type:   "https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/422",
	}
}

func ThrowInternalServerError(detail string) *GlobalError {
	return &GlobalError{
		Title:  "Internal Server Error",
		Code:   InternalServerErrorCode,
		Detail: detail,
		Type:   "https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/500",
	}
}

func ThrowNotImplementedError(detail string) *GlobalError {
	return &GlobalError{
		Title:  "Not Implemented",
		Code:   NotImplementedErrorCode,
		Detail: detail,
		Type:   "https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/501",
	}
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
	}
}
