package middlewares_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	e "auth_service/app/errors"
	middleware "auth_service/app/middlewares"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// newTestApp builds an app wired with the real error handler plus the handful of
// routes needed to reach each of its branches.
func newTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.NewErrorHandler(zap.NewNop()),
	})

	app.Get("/ok", func(ctx fiber.Ctx) error {
		return ctx.SendString("ok")
	})

	// A handler that raises a 404 on its own - it has to keep answering the
	// message of the error instead of the router wording.
	app.Get("/gone", func(ctx fiber.Ctx) error {
		return fiber.ErrNotFound
	})

	app.Get("/unauthorized", func(ctx fiber.Ctx) error {
		return e.ThrowUnauthorizedError("Invalid X-Public-Key")
	})

	app.Get("/bind-query", func(ctx fiber.Ctx) error {
		var query struct {
			Skip int `query:"skip"`
		}

		if err := ctx.Bind().Query(&query); err != nil {
			return err
		}

		return ctx.SendString("bound")
	})

	app.Post("/bind", func(ctx fiber.Ctx) error {
		var payload struct {
			Email string `json:"email"`
		}

		if err := ctx.Bind().Body(&payload); err != nil {
			return err
		}

		return ctx.SendString("bound")
	})

	return app
}

func request(t *testing.T, app *fiber.App, method, url, body, contentType string) (*http.Response, e.ProblemDetail) {
	t.Helper()

	var req *http.Request

	if body == "" {
		req = httptest.NewRequest(method, url, nil)
	} else {
		req = httptest.NewRequest(method, url, strings.NewReader(body))
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := app.Test(req)
	assert.NoError(t, err)

	var problemDetail e.ProblemDetail
	json.NewDecoder(resp.Body).Decode(&problemDetail)

	return resp, problemDetail
}

func TestUnknownRouteAnswersProblemDetail(t *testing.T) {
	app := newTestApp()

	resp, problemDetail := request(t, app, "GET", "/nope", "", "")

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, http.StatusNotFound, problemDetail.Status)
	assert.Equal(t, "Not Found", problemDetail.Title)
	assert.Equal(t, string(e.NotFoundErrorCode.First), problemDetail.Code)
	assert.Equal(t, "Route `GET /nope` does not exist", problemDetail.Detail)
	assert.Equal(t, "/nope", problemDetail.Instance)
	assert.Contains(t, problemDetail.Type, "/404")
}

func TestUnknownRouteKeepsQueryOutOfTheDetail(t *testing.T) {
	app := newTestApp()

	// The instance carries the original url, the detail only the path - a query
	// string is caller input and has no business being echoed back.
	_, problemDetail := request(t, app, "GET", "/core/apps/9/nope?token=secret", "", "")

	assert.Equal(t, "Route `GET /core/apps/9/nope` does not exist", problemDetail.Detail)
	assert.Equal(t, "/core/apps/9/nope?token=secret", problemDetail.Instance)
}

func TestUnregisteredMethodAnswersMethodNotAllowed(t *testing.T) {
	app := newTestApp()

	resp, problemDetail := request(t, app, "POST", "/ok", "", "")

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	assert.Equal(t, string(e.NotAllowedErrorCode.First), problemDetail.Code)
	assert.Equal(t, "Method Not Allowed", problemDetail.Title)
}

func TestNotFoundRaisedByHandlerKeepsItsOwnMessage(t *testing.T) {
	app := newTestApp()

	resp, problemDetail := request(t, app, "GET", "/gone", "", "")

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, string(e.NotFoundErrorCode.First), problemDetail.Code)
	// The route exists, so the router wording must not be used.
	assert.NotContains(t, problemDetail.Detail, "does not exist")
}

func TestAppErrorIsStillHandled(t *testing.T) {
	app := newTestApp()

	resp, problemDetail := request(t, app, "GET", "/unauthorized", "", "")

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Equal(t, string(e.UnauthorizedErrorCode.First), problemDetail.Code)
	assert.Equal(t, "Invalid X-Public-Key", problemDetail.Detail)
}

func TestUnbindableContentTypeAnswersUnprocessableEntity(t *testing.T) {
	app := newTestApp()

	resp, problemDetail := request(t, app, "POST", "/bind", "email,name", "text/csv")

	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	assert.Equal(t, string(e.UnprocessableEntityErrorCode.First), problemDetail.Code)
}

func TestValidRouteIsUntouched(t *testing.T) {
	app := newTestApp()

	resp, _ := request(t, app, "GET", "/ok", "", "")

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestMalformedBodyAnswersBadRequest(t *testing.T) {
	app := newTestApp()

	resp, problemDetail := request(t, app, "POST", "/bind", `{"email":`, "application/json")

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, http.StatusBadRequest, problemDetail.Status)
	assert.Equal(t, string(e.BadRequestCode.First), problemDetail.Code)
	assert.Contains(t, problemDetail.Detail, "Request body could not be parsed")
	assert.Equal(t, "body", problemDetail.Data["source"])
}

func TestUnbindableQueryAnswersBadRequest(t *testing.T) {
	app := newTestApp()

	// skip binds to an int, so a word cannot be read into it
	resp, problemDetail := request(t, app, "GET", "/bind-query?skip=abc", "", "")

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, string(e.BadRequestCode.First), problemDetail.Code)
	assert.Equal(t, "query", problemDetail.Data["source"])
}
