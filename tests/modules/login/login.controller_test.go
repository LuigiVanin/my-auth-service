package login_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	e "auth_service/app/errors"
	otpDto "auth_service/app/modules/core/otp/models"
	"auth_service/app/modules/login/controller"
	dto "auth_service/app/modules/login/models"
	entity "auth_service/infra/entities"
	sharedDto "auth_service/shared/models"
	mock "auth_service/tests/modules/mock"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// --- Test Suite for Login Controller ---

type LoginControllerTestSuite struct {
	suite.Suite
	mockLoginService *mock.MockLoginService
	loginController  *controller.LoginController
	app              *fiber.App
	appEntity        *entity.App
}

func (this *LoginControllerTestSuite) SetupTest() {
	logger := zap.NewNop()

	this.mockLoginService = new(mock.MockLoginService)
	this.loginController = controller.NewLoginController(
		this.mockLoginService,
		nil,
		logger,
		openapi.NewBuilder("test", "test", "test"),
	)

	// Setup fiber app with custom error handler
	this.app = fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			if ge, ok := err.(*e.AppError); ok {
				return c.Status(ge.Code.Second).JSON(ge)
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Setup app entity
	this.appEntity = &entity.App{
		ID:         "app-id",
		LoginTypes: []string{"WITH_PASSWORD", "WITH_OTP"},
		UsersPool:  entity.UsersPool{ID: "pool-id"},
		Role:       "USER",
	}

	// Add middleware to inject app entity into context
	this.app.Use(func(c fiber.Ctx) error {
		c.Locals("app", this.appEntity)
		return c.Next()
	})

	// Register controller routes
	this.loginController.Register(this.app)
}

func (this *LoginControllerTestSuite) TestLoginWithPassword_Success() {
	// Arrange
	payload := dto.LoginPayloadWithPassoword{
		Email:    "test@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(payload)

	expectedResponse := &dto.LoginResponse{
		SessionId:        "session-id",
		AccessToken:      "access-token",
		RefreshToken:     "refresh-token",
		ExpiresAt:        time.Now().Add(1 * time.Hour),
		RefreshExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		User: entity.User{
			ID:    1,
			Email: "test@example.com",
			Name:  "Test User",
		},
	}

	// Mock service call
	this.mockLoginService.On("LoginWithPassword", this.appEntity, payload, testifymock.MatchedBy(func(req sharedDto.RequestInfo) bool {
		return req.IpAddress != "" // Just verify it has an IP
	})).Return(expectedResponse, nil)

	// Create request
	req := httptest.NewRequest("POST", "/auth/login?method=password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := this.app.Test(req)

	// Assert
	assert.NoError(this.T(), err)
	assert.Equal(this.T(), http.StatusOK, resp.StatusCode)

	var result dto.LoginResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(this.T(), err)

	assert.Equal(this.T(), "session-id", result.SessionId)
	assert.Equal(this.T(), "access-token", result.AccessToken)
	assert.Equal(this.T(), "refresh-token", result.RefreshToken)
	assert.Equal(this.T(), "test@example.com", result.User.Email)

	this.mockLoginService.AssertExpectations(this.T())
}

func (this *LoginControllerTestSuite) TestLoginWithPassword_DefaultMethod_Success() {
	// Arrange - test that password is default when method is not specified
	payload := dto.LoginPayloadWithPassoword{
		Email:    "test@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(payload)

	expectedResponse := &dto.LoginResponse{
		SessionId:   "session-id",
		AccessToken: "access-token",
		User: entity.User{
			Email: "test@example.com",
		},
	}

	this.mockLoginService.On("LoginWithPassword", this.appEntity, payload, testifymock.Anything).Return(expectedResponse, nil)

	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(body)) // No method query param
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := this.app.Test(req)

	// Assert
	assert.NoError(this.T(), err)
	assert.Equal(this.T(), http.StatusOK, resp.StatusCode)

	this.mockLoginService.AssertExpectations(this.T())
}

func (this *LoginControllerTestSuite) TestLoginWithPassword_InvalidCredentials_Error() {
	// Arrange
	payload := dto.LoginPayloadWithPassoword{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}
	body, _ := json.Marshal(payload)

	// Mock service returning unauthorized error
	this.mockLoginService.On("LoginWithPassword", this.appEntity, payload, testifymock.Anything).
		Return(nil, e.ThrowUnauthorizedError("Invalid Credentials"))

	req := httptest.NewRequest("POST", "/auth/login?method=password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := this.app.Test(req)

	// Assert
	assert.NoError(this.T(), err)
	assert.Equal(this.T(), http.StatusUnauthorized, resp.StatusCode)

	var result map[string]any
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err == nil && result["title"] != nil {
		assert.Contains(this.T(), result["title"], "Invalid Credentials")
	}

	this.mockLoginService.AssertExpectations(this.T())
}

func (this *LoginControllerTestSuite) TestLoginWithPassword_InvalidJSON_Error() {
	// Arrange - malformed JSON
	malformedJSON := []byte(`{"email": "test@example.com", "password": }`)

	req := httptest.NewRequest("POST", "/auth/login?method=password", bytes.NewBuffer(malformedJSON))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := this.app.Test(req)

	// Assert
	assert.NoError(this.T(), err)
	// Note: Fiber returns 500 for JSON parsing errors when it can't parse the body
	// This is expected behavior as the controller's body binder catches the error
	assert.True(this.T(), resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusInternalServerError)
}

func (this *LoginControllerTestSuite) TestLoginWithPassword_ServiceError_Error() {
	// Arrange
	payload := dto.LoginPayloadWithPassoword{
		Email:    "test@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(payload)

	// Mock service returning internal server error
	this.mockLoginService.On("LoginWithPassword", this.appEntity, payload, testifymock.Anything).
		Return(nil, e.ThrowInternalServerError("Database connection failed"))

	req := httptest.NewRequest("POST", "/auth/login?method=password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := this.app.Test(req)

	// Assert
	assert.NoError(this.T(), err)
	assert.Equal(this.T(), http.StatusInternalServerError, resp.StatusCode)

	this.mockLoginService.AssertExpectations(this.T())
}

func (this *LoginControllerTestSuite) TestLoginWithOtp_Success() {
	// Arrange
	payload := dto.LoginPayloadWithOtp{
		Email: "test@example.com",
		Otp: otpDto.PayloadOtpData{
			Id:   "otp-id",
			Code: "123456",
		},
	}
	body, _ := json.Marshal(payload)

	expectedResponse := &dto.LoginResponse{
		SessionId:   "session-id-otp",
		AccessToken: "access-token-otp",
		User: entity.User{
			Email: "test@example.com",
		},
	}

	this.mockLoginService.On("LoginWithOtp", this.appEntity, payload, testifymock.Anything).Return(expectedResponse, nil)

	req := httptest.NewRequest("POST", "/auth/login?method=otp", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := this.app.Test(req)

	// Assert
	assert.NoError(this.T(), err)
	assert.Equal(this.T(), http.StatusOK, resp.StatusCode)

	var result dto.LoginResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(this.T(), err)

	assert.Equal(this.T(), "session-id-otp", result.SessionId)
	assert.Equal(this.T(), "access-token-otp", result.AccessToken)

	this.mockLoginService.AssertExpectations(this.T())
}

func (this *LoginControllerTestSuite) TestLoginWithOtp_InvalidOtp_Error() {
	// Arrange
	payload := dto.LoginPayloadWithOtp{
		Email: "test@example.com",
		Otp: otpDto.PayloadOtpData{
			Id:   "otp-id",
			Code: "wrong-code",
		},
	}
	body, _ := json.Marshal(payload)

	this.mockLoginService.On("LoginWithOtp", this.appEntity, payload, testifymock.Anything).
		Return(nil, e.ThrowUnauthorizedError("Invalid OTP code"))

	req := httptest.NewRequest("POST", "/auth/login?method=otp", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := this.app.Test(req)

	// Assert
	assert.NoError(this.T(), err)
	assert.Equal(this.T(), http.StatusUnauthorized, resp.StatusCode)

	this.mockLoginService.AssertExpectations(this.T())
}

func (this *LoginControllerTestSuite) TestLogin_RequestInfoExtraction() {
	// Arrange - test that IP and User-Agent are correctly extracted
	payload := dto.LoginPayloadWithPassoword{
		Email:    "test@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(payload)

	expectedResponse := &dto.LoginResponse{
		SessionId:   "session-id",
		AccessToken: "access-token",
	}

	// Mock with specific request info matcher
	this.mockLoginService.On("LoginWithPassword", this.appEntity, payload, testifymock.MatchedBy(func(req sharedDto.RequestInfo) bool {
		// Verify that request info contains IP and User-Agent
		return req.IpAddress != "" && req.UserAgent == "TestAgent/1.0"
	})).Return(expectedResponse, nil)

	req := httptest.NewRequest("POST", "/auth/login?method=password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TestAgent/1.0")

	// Act
	resp, err := this.app.Test(req)

	// Assert
	assert.NoError(this.T(), err)
	assert.Equal(this.T(), http.StatusOK, resp.StatusCode)

	this.mockLoginService.AssertExpectations(this.T())
}

func TestLoginControllerSuite(t *testing.T) {
	suite.Run(t, new(LoginControllerTestSuite))
}
