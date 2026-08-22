package errors

import (
	"fmt"
	"net/http"
	"strings"

	"auth_service/shared/utils"

	"github.com/gofiber/fiber/v3"
)

// fiberErrorCodes maps the status of a *fiber.Error onto the error code pair of
// the service. Fiber raises these around the handlers instead of inside them -
// a request to a path no route matches, a method a route does not register, a
// body over BodyLimit - so they never carry a GlobalError and would otherwise
// fall into the unexpected error branch of the handler and answer 500.
var fiberErrorCodes = map[int]ErrorCodePair{
	fiber.StatusBadRequest:          BadRequestCode,
	fiber.StatusUnauthorized:        UnauthorizedErrorCode,
	fiber.StatusForbidden:           PermissionDeniedErrorCode,
	fiber.StatusNotFound:            NotFoundErrorCode,
	fiber.StatusMethodNotAllowed:    NotAllowedErrorCode,
	fiber.StatusConflict:            ConflictErrorCode,
	fiber.StatusUnprocessableEntity: UnprocessableEntityErrorCode,
	fiber.StatusTooManyRequests:     TooManyRequestsErrorCode,
	fiber.StatusInternalServerError: InternalServerErrorCode,
	fiber.StatusNotImplemented:      NotImplementedErrorCode,
}

func codeFromStatus(status int) ErrorCodePair {
	text := http.StatusText(status)

	if text == "" {
		return ErrorCodePair{First: "UNEXPECTED_ERROR", Second: status}
	}

	return ErrorCodePair{
		First:  GlobalErrorCode(strings.ToUpper(strings.ReplaceAll(text, " ", "_"))),
		Second: status,
	}
}

func FromFiberError(fiberError *fiber.Error, detail string) *GlobalError {
	code, mapped := fiberErrorCodes[fiberError.Code]

	if !mapped {
		code = codeFromStatus(fiberError.Code)
	}

	if detail == "" {
		detail = fiberError.Message
	}

	title := http.StatusText(fiberError.Code)

	if title == "" {
		title = "Unexpected Error"
	}

	return NewGlobalError(
		title,
		detail,
		code,
		fmt.Sprintf(
			"https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/%d",
			fiberError.Code,
		),
	)
}

// FromBindError turns a binding failure into a GlobalError. Every binding source
// - body, query, header, cookie, uri - is caller input, so a payload the binder
// cannot read is a bad request and not an internal error. Without this the raw
// *fiber.BindError would reach the unexpected error branch of the handler and
// answer 500 to a caller that simply sent a malformed body.
func FromBindError(bindError *fiber.BindError) *GlobalError {
	source := bindError.Source

	if source == "" {
		source = fiber.BindSourceBody
	}

	extra := utils.JSON{"source": source}

	// Field is best effort - the binder fills it only when it can tell which key
	// of the payload broke.
	if bindError.Field != "" {
		extra["field"] = bindError.Field
	}

	detail := fmt.Sprintf("Request %s could not be parsed", source)

	if bindError.Err != nil {
		detail = fmt.Sprintf("%s: %v", detail, bindError.Err)
	}

	return ThrowBadRequest(detail, extra)
}
