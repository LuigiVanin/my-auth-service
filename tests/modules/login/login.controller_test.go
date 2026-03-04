package login_test

// import (
// 	"auth_service/app/middlewares/guards"
// 	"auth_service/app/models/dto"
// 	"auth_service/app/modules/login/controller"
// 	entity "auth_service/infra/entities"
// 	mockpkg "auth_service/tests/modules/mock"
// 	"bytes"
// 	"encoding/json"
// 	"net/http"
// 	"testing"

// 	"github.com/gofiber/fiber/v2"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/mock"
// 	"github.com/stretchr/testify/suite"
// 	"go.uber.org/zap"
// )

// type LoginWithPasswordControllerTestSuite struct {
// 	suite.Suite
// 	mockLoginService  *mockpkg.MockLoginService
// 	mockAppRepository *mockpkg.MockAppRepository
// 	mockCipherService *mockpkg.MockCipherService
// 	appGuard          *guards.AppGuard
// 	loginController   *controller.LoginController
// 	app               *fiber.App
// }

// func (this *LoginWithPasswordControllerTestSuite) SetupTest() {
// 	logger := zap.NewNop()

// 	this.mockLoginService = new(mockpkg.MockLoginService)
// 	this.mockAppRepository = new(mockpkg.MockAppRepository)
// 	this.mockCipherService = new(mockpkg.MockCipherService)

// 	// AppGuard is a struct, so we initialize a real one with our mocks
// 	this.appGuard = guards.NewAppGuard(this.mockAppRepository, this.mockCipherService, logger)

// 	this.loginController = controller.NewLoginController(this.mockLoginService, this.appGuard, logger)

// 	this.app = fiber.New()
// }

// func (this *LoginWithPasswordControllerTestSuite) TestLoginController_Success() {
// 	// Arrange
// 	appEntity := &entity.App{
// 		ID: "app-id",
// 	}

// 	// Add middleware BEFORE registering routes to ensure ctx.Locals is populated
// 	this.app.Use(func(c *fiber.Ctx) error {
// 		c.Locals("app", appEntity)
// 		return c.Next()
// 	})

// 	this.loginController.Register(this.app)

// 	payload := dto.LoginPayloadWithPassoword{
// 		Email:    "test@example.com",
// 		Password: "password",
// 	}
// 	body, _ := json.Marshal(payload)

// 	loginResponse := &dto.LoginResponse{
// 		SessionId:   "session-id",
// 		AccessToken: "access-token",
// 	}

// 	this.mockLoginService.
// 		On("LoginWithPassword", appEntity, payload, mock.Anything).
// 		Return(loginResponse, nil)

// 	req, _ := http.NewRequest("POST", "/auth/login?method=password", bytes.NewBuffer(body))
// 	req.Header.Set("Content-Type", "application/json")

// 	// Act
// 	resp, err := this.app.Test(req)

// 	// Assert
// 	assert.NoError(this.T(), err)
// 	assert.Equal(this.T(), http.StatusOK, resp.StatusCode)

// 	var result dto.LoginResponse
// 	err = json.NewDecoder(resp.Body).Decode(&result)

// 	if err != nil {
// 		this.T().Fatalf("Failed to decode response body: %v", err)
// 	}

// 	assert.Equal(this.T(), "session-id", result.SessionId)
// 	assert.Equal(this.T(), "access-token", result.AccessToken)
// }

// func TestControllerSuite(t *testing.T) {
// 	suite.Run(t, new(LoginWithPasswordControllerTestSuite))
// }
