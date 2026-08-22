package bootstrap_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	e "auth_service/app/errors"
	"auth_service/infra/bootstrap"
	"auth_service/infra/config"
	entity "auth_service/infra/entities"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// newServer builds the real http server so that the middleware order of
// NewHttpServer is what gets exercised. appGuard is only turned into a handler
// value by RegisterGuards, and no test here touches /auth, /otp or /core, so it
// is never invoked.
func newServer(t *testing.T) (*fiber.App, *observer.ObservedLogs) {
	t.Helper()

	core, logs := observer.New(zapcore.ErrorLevel)

	cfg := &config.Config{
		Env:    "test",
		App:    config.AppConfig{Name: "auth_service_test", Env: "test"},
		Server: config.ServerConfig{Port: "0"},
	}

	return bootstrap.NewHttpServer(cfg, zap.New(core), nil), logs
}

func TestPanicInHandlerAnswersProblemDetail(t *testing.T) {
	server, logs := newServer(t)

	// The shape every controller uses to read what a guard left behind. On a route
	// the guard does not cover, the assertion panics.
	server.Get("/panics", func(ctx fiber.Ctx) error {
		app := ctx.Locals("app").(*entity.App)
		return ctx.SendString(app.ID)
	})

	resp, err := server.Test(httptest.NewRequest("GET", "/panics", nil))
	assert.NoError(t, err, "the panic must not escape the middleware")

	var problemDetail e.ProblemDetail
	json.NewDecoder(resp.Body).Decode(&problemDetail)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, string(e.InternalServerErrorCode.First), problemDetail.Code)
	assert.Equal(t, "Unexpected internal error", problemDetail.Detail)

	// The panic value names internal types and file paths - it belongs in the log
	// and nowhere near the caller.
	assert.NotContains(t, problemDetail.Detail, "interface conversion")
	assert.NotContains(t, problemDetail.Detail, "entity.App")

	recovered := logs.FilterMessage("Recovered from panic")
	assert.Equal(t, 1, recovered.Len(), "the panic has to be logged")

	if recovered.Len() > 0 {
		fields := recovered.All()[0].ContextMap()
		assert.Contains(t, fields, "stack", "the stack trace has to reach the log")
		assert.Contains(t, fields, "panic")
		assert.Equal(t, "/panics", fields["path"])
	}
}

func TestServerSurvivesAPanic(t *testing.T) {
	server, _ := newServer(t)

	server.Get("/panics", func(ctx fiber.Ctx) error {
		panic("boom")
	})
	server.Get("/healthy", func(ctx fiber.Ctx) error {
		return ctx.SendString("ok")
	})

	_, err := server.Test(httptest.NewRequest("GET", "/panics", nil))
	assert.NoError(t, err)

	// One bad request must not take the rest of the server with it.
	resp, err := server.Test(httptest.NewRequest("GET", "/healthy", nil))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPanicWithAnErrorValueIsStillGeneric(t *testing.T) {
	server, _ := newServer(t)

	// DefaultPanicHandler would forward an error value as is, leaking it. The
	// configured handler must answer the same generic detail either way.
	server.Get("/panics", func(ctx fiber.Ctx) error {
		panic(assert.AnError)
	})

	resp, err := server.Test(httptest.NewRequest("GET", "/panics", nil))
	assert.NoError(t, err)

	var problemDetail e.ProblemDetail
	json.NewDecoder(resp.Body).Decode(&problemDetail)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, "Unexpected internal error", problemDetail.Detail)
	assert.NotContains(t, problemDetail.Detail, assert.AnError.Error())
}
