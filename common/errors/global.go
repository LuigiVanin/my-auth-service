package errors

import "fmt"

type GlobalErrorCode string

type GlobalError struct {
	Title  string
	Code   GlobalErrorCode
	Detail string
	Type   string
}

func NewGlobalError(title string, code GlobalErrorCode, detail string) *GlobalError {

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
	}
}

func ThrowConflict(detail string) *GlobalError {
	return &GlobalError{
		Title:  "Conflict",
		Code:   ConflictErrorCode,
		Detail: detail,
	}
}

func ThrowUserAlreadyExists(detail string) *GlobalError {
	return &GlobalError{
		Title:  "Conflict",
		Code:   UserAlreadyExistsCode,
		Detail: detail,
		Type:   "https://example.com/errors/user-already-exists",
	}
}

func ThrowNotFound(detail string) *GlobalError {
	return &GlobalError{
		Title:  "Not Found",
		Code:   NotFoundErrorCode,
		Detail: detail,
		Type:   "https://example.com/errors/not-found",
	}
}

func ThorwUnauthorizedError(detail string) *GlobalError {
	return &GlobalError{
		Title:  "Unauthorized",
		Code:   UnauthorizedErrorCode,
		Detail: detail,
		Type:   "https://example.com/errors/unauthorized",
	}
}

func ThrowTokenExpiredError(detail string) *GlobalError {
	return &GlobalError{
		Title:  "Token Expired",
		Code:   TokenExpiredErrorCode,
		Detail: detail,
		Type:   "https://example.com/errors/token-expired",
	}
}

func ThrowUnprocessableEntity(detail string) *GlobalError {

	return &GlobalError{
		Title:  "Unprocessable Entity",
		Code:   UnprocessableEntityErrorCode,
		Detail: detail,
		Type:   "https://example.com/errors/unprocessable-entity",
	}
}

func ThrowInternalServerError(detail string) *GlobalError {
	return &GlobalError{
		Title:  "Internal Server Error",
		Code:   InternalServerErrorCode,
		Detail: detail,
	}
}

func (e *GlobalError) Error() string {

	return fmt.Sprintf("GlobalError: %s, Code: %s, Detail: %s", e.Title, e.Code, e.Detail)
}

func (e *GlobalError) IntoProblemDetail(instance string) *ProblemDetail {
	status := HttpErrorMap[e.Code]
	if status == 0 {
		status = 500
	}

	return &ProblemDetail{
		Type:     e.Type,
		Title:    e.Title,
		Status:   status,
		Detail:   e.Detail,
		Instance: instance,
		Code:     string(e.Code),
	}
}
