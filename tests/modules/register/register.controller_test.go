package register_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	e "auth_service/app/errors"
	"auth_service/app/models/dto"
	"auth_service/app/modules/register/controller"
	entity "auth_service/infra/entities"
	mock "auth_service/tests/modules/mock"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// --- Test Suite for Register Controller ---

type RegisterControllerTestSuite struct {
	suite.Suite
	mockRegisterService *mock.MockRegisterService
	registerController  *controller.RegisterController
	app                 *fiber.App
	appEntity           *entity.App
}

func (this *RegisterControllerTestSuite) SetupTest() {
	logger := zap.NewNop()

	this.mockRegisterService = new(mock.MockRegisterService)
	this.registerController = controller.NewRegisterController(
		this.mockRegisterService,
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
	this.registerController.Register(this.app)
}

func (this *RegisterControllerTestSuite) TestRegisterWithPassword_Success() {
	// Arrange
	metadata := json.RawMessage(`{"plan":"free"}`) // No spaces in JSON
	payload := dto.RegisterPayloadWithPassoword{
		Email:    "newuser@example.com",
		Password: "password123",
		Name:     "New User",
		Metadata: metadata,
	}
	body, _ := json.Marshal(payload)

	expectedResponse := &dto.RegisterResponse{
		SessionId:        "session-id",
		AccessToken:      "access-token",
		RefreshToken:     "refresh-token",
		ExpiresAt:        time.Now().Add(1 * time.Hour),
		RefreshExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		User: entity.User{
			ID:    1,
			Email: "newuser@example.com",
			Name:  "New User",
		},
	}

	// Mock service call
	this.mockRegisterService.On("RegisterWithPassword", this.appEntity, payload, testifymock.MatchedBy(func(req dto.RequestInfo) bool {
		return req.IpAddress != ""
	})).Return(expectedResponse, nil)

	// Create request
	req := httptest.NewRequest("POST", "/auth/register?method=password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := this.app.Test(req)

	// Assert
	assert.NoError(this.T(), err)
	assert.Equal(this.T(), http.StatusCreated, resp.StatusCode)

	var result dto.RegisterResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(this.T(), err)

	assert.Equal(this.T(), "session-id", result.SessionId)
	assert.Equal(this.T(), "access-token", result.AccessToken)
	assert.Equal(this.T(), "refresh-token", result.RefreshToken)
	assert.Equal(this.T(), "newuser@example.com", result.User.Email)
	assert.Equal(this.T(), "New User", result.User.Name)

	this.mockRegisterService.AssertExpectations(this.T())
}

func (this *RegisterControllerTestSuite) TestRegisterWithPassword_UserAlreadyExists_Error() {
	// Arrange
	metadata := json.RawMessage(`{"plan":"free"}`) // No spaces in JSON
	payload := dto.RegisterPayloadWithPassoword{
		Email:    "existing@example.com",
		Password: "password123",
		Name:     "Existing User",
		Metadata: metadata,
	}
	body, _ := json.Marshal(payload)

	// Mock service returning bad request error
	this.mockRegisterService.On("RegisterWithPassword", this.appEntity, payload, testifymock.Anything).
		Return(nil, e.ThrowBadRequest("User already exists"))

	req := httptest.NewRequest("POST", "/auth/register?method=password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := this.app.Test(req)

	// Assert
	assert.NoError(this.T(), err)
	assert.Equal(this.T(), http.StatusBadRequest, resp.StatusCode)

	var result map[string]any
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err == nil && result["title"] != nil {
		assert.Contains(this.T(), result["title"], "User already exists")
	}

	this.mockRegisterService.AssertExpectations(this.T())
}

func (this *RegisterControllerTestSuite) TestRegisterWithPassword_InvalidJSON_Error() {
	// Arrange - malformed JSON
	malformedJSON := []byte(`{"email": "test@example.com", "password": "pass", "name": }`)

	req := httptest.NewRequest("POST", "/auth/register?method=password", bytes.NewBuffer(malformedJSON))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := this.app.Test(req)

	// Assert
	assert.NoError(this.T(), err)
	// Note: Fiber returns 500 for JSON parsing errors when it can't parse the body
	assert.True(this.T(), resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusInternalServerError)
}

func (this *RegisterControllerTestSuite) TestRegisterWithPassword_ServiceError_Error() {
	// Arrange
	metadata := json.RawMessage(`{"plan":"free"}`) // No spaces in JSON
	payload := dto.RegisterPayloadWithPassoword{
		Email:    "newuser@example.com",
		Password: "password123",
		Name:     "New User",
		Metadata: metadata,
	}
	body, _ := json.Marshal(payload)

	// Mock service returning internal server error
	this.mockRegisterService.On("RegisterWithPassword", this.appEntity, payload, testifymock.Anything).
		Return(nil, e.ThrowInternalServerError("Failed to create user"))

	req := httptest.NewRequest("POST", "/auth/register?method=password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := this.app.Test(req)

	// Assert
	assert.NoError(this.T(), err)
	assert.Equal(this.T(), http.StatusInternalServerError, resp.StatusCode)

	this.mockRegisterService.AssertExpectations(this.T())
}

func (this *RegisterControllerTestSuite) TestRegisterWithOtp_Success() {
	// Arrange
	phone := "+1234567890"
	payload := dto.RegisterPayloadWithOtp{
		Email:    "newuser@example.com",
		Name:     "New User",
		Phone:    &phone,
		Password: "password123",
		Metadata: map[string]any{"plan": "premium"},
		Otp: dto.PayloadOtpData{
			Id:   "otp-id",
			Code: "123456",
		},
	}
	body, _ := json.Marshal(payload)

	expectedResponse := &dto.RegisterResponse{
		SessionId:   "session-id-otp",
		AccessToken: "access-token-otp",
		User: entity.User{
			Email:       "newuser@example.com",
			Name:        "New User",
			VerifyEmail: true,
		},
	}

	this.mockRegisterService.On("RegisterWithOtp", this.appEntity, payload, testifymock.Anything).Return(expectedResponse, nil)

	req := httptest.NewRequest("POST", "/auth/register?method=otp", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := this.app.Test(req)

	// Assert
	assert.NoError(this.T(), err)
	assert.Equal(this.T(), http.StatusCreated, resp.StatusCode)

	var result dto.RegisterResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(this.T(), err)

	assert.Equal(this.T(), "session-id-otp", result.SessionId)
	assert.Equal(this.T(), "access-token-otp", result.AccessToken)
	assert.True(this.T(), result.User.VerifyEmail)

	this.mockRegisterService.AssertExpectations(this.T())
}

func (this *RegisterControllerTestSuite) TestRegisterWithOtp_DefaultMethod_Success() {
	// Arrange - test that OTP is default when method is not specified
	payload := dto.RegisterPayloadWithOtp{
		Email:    "newuser@example.com",
		Name:     "New User",
		Metadata: map[string]any{"plan": "free"},
		Otp: dto.PayloadOtpData{
			Id:   "otp-id",
			Code: "123456",
		},
	}
	body, _ := json.Marshal(payload)

	expectedResponse := &dto.RegisterResponse{
		SessionId:   "session-id",
		AccessToken: "access-token",
		User:        entity.User{Email: "newuser@example.com"},
	}

	this.mockRegisterService.On("RegisterWithOtp", this.appEntity, payload, testifymock.Anything).Return(expectedResponse, nil)

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body)) // No method query param
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := this.app.Test(req)

	// Assert
	assert.NoError(this.T(), err)
	assert.Equal(this.T(), http.StatusCreated, resp.StatusCode)

	this.mockRegisterService.AssertExpectations(this.T())
}

func (this *RegisterControllerTestSuite) TestRegisterWithOtp_InvalidOtp_Error() {
	// Arrange
	payload := dto.RegisterPayloadWithOtp{
		Email:    "newuser@example.com",
		Name:     "New User",
		Metadata: map[string]any{"plan": "free"},
		Otp: dto.PayloadOtpData{
			Id:   "otp-id",
			Code: "wrong-code",
		},
	}
	body, _ := json.Marshal(payload)

	this.mockRegisterService.On("RegisterWithOtp", this.appEntity, payload, testifymock.Anything).
		Return(nil, e.ThrowUnauthorizedError("Invalid OTP code"))

	req := httptest.NewRequest("POST", "/auth/register?method=otp", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := this.app.Test(req)

	// Assert
	assert.NoError(this.T(), err)
	assert.Equal(this.T(), http.StatusUnauthorized, resp.StatusCode)

	this.mockRegisterService.AssertExpectations(this.T())
}

func (this *RegisterControllerTestSuite) TestRegisterWithOtp_OtpExpired_Error() {
	// Arrange
	payload := dto.RegisterPayloadWithOtp{
		Email:    "newuser@example.com",
		Name:     "New User",
		Metadata: map[string]any{"plan": "free"},
		Otp: dto.PayloadOtpData{
			Id:   "expired-otp-id",
			Code: "123456",
		},
	}
	body, _ := json.Marshal(payload)

	this.mockRegisterService.On("RegisterWithOtp", this.appEntity, payload, testifymock.Anything).
		Return(nil, e.ThrowBadRequest("OTP has expired"))

	req := httptest.NewRequest("POST", "/auth/register?method=otp", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := this.app.Test(req)

	// Assert
	assert.NoError(this.T(), err)
	assert.Equal(this.T(), http.StatusBadRequest, resp.StatusCode)

	this.mockRegisterService.AssertExpectations(this.T())
}

func (this *RegisterControllerTestSuite) TestRegister_RequestInfoExtraction() {
	// Arrange - test that IP and User-Agent are correctly extracted
	metadata := json.RawMessage(`{"plan":"free"}`) // No spaces in JSON
	payload := dto.RegisterPayloadWithPassoword{
		Email:    "newuser@example.com",
		Password: "password123",
		Name:     "New User",
		Metadata: metadata,
	}
	body, _ := json.Marshal(payload)

	expectedResponse := &dto.RegisterResponse{
		SessionId:   "session-id",
		AccessToken: "access-token",
	}

	// Mock with specific request info matcher
	this.mockRegisterService.On("RegisterWithPassword", this.appEntity, payload, testifymock.MatchedBy(func(req dto.RequestInfo) bool {
		// Verify that request info contains IP and User-Agent
		return req.IpAddress != "" && req.UserAgent == "TestAgent/1.0"
	})).Return(expectedResponse, nil)

	req := httptest.NewRequest("POST", "/auth/register?method=password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TestAgent/1.0")

	// Act
	resp, err := this.app.Test(req)

	// Assert
	assert.NoError(this.T(), err)
	assert.Equal(this.T(), http.StatusCreated, resp.StatusCode)

	this.mockRegisterService.AssertExpectations(this.T())
}

func (this *RegisterControllerTestSuite) TestRegister_WithPhoneNumber_Success() {
	// Arrange
	phone := "+1234567890"
	metadata := json.RawMessage(`{"plan":"free"}`) // No spaces in JSON
	payload := dto.RegisterPayloadWithPassoword{
		Email:    "newuser@example.com",
		Password: "password123",
		Name:     "New User",
		Phone:    &phone,
		Metadata: metadata,
	}
	body, _ := json.Marshal(payload)

	expectedResponse := &dto.RegisterResponse{
		SessionId:   "session-id",
		AccessToken: "access-token",
		User: entity.User{
			Email: "newuser@example.com",
			Phone: "+1234567890",
		},
	}

	this.mockRegisterService.On("RegisterWithPassword", this.appEntity, payload, testifymock.Anything).Return(expectedResponse, nil)

	req := httptest.NewRequest("POST", "/auth/register?method=password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := this.app.Test(req)

	// Assert
	assert.NoError(this.T(), err)
	assert.Equal(this.T(), http.StatusCreated, resp.StatusCode)

	var result dto.RegisterResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(this.T(), "+1234567890", result.User.Phone)

	this.mockRegisterService.AssertExpectations(this.T())
}

func TestRegisterControllerSuite(t *testing.T) {
	suite.Run(t, new(RegisterControllerTestSuite))
}
