package guards

import (
	"errors"
	"fmt"
	"regexp"

	e "auth_service/app/errors"
	ar "auth_service/app/modules/core/app/repository"
	cipher "auth_service/app/modules/utils/cipher"
	"auth_service/shared/global"

	i "auth_service/shared/interfaces"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

var _ i.IGuard = &AppGuard{}

type AppGuard struct {
	appRepository ar.IAppRepository
	cipherService cipher.ICipherService
	logger        *zap.Logger
}

func NewAppGuard(appRepository ar.IAppRepository, cipherService cipher.ICipherService, logger *zap.Logger) *AppGuard {
	return &AppGuard{
		appRepository: appRepository,
		cipherService: cipherService,
		logger:        logger,
	}
}

func validateKeyFormat(key string) error {
	matched, err := regexp.MatchString(`^as_[a-zA-Z0-9._-]+$`, key)

	if err != nil || !matched {
		return errors.New("invalid key format")
	}
	return nil
}

func (this *AppGuard) Act(ctx *fiber.Ctx) error {
	global.Logger.Info("App Guard Triggered")
	appKey := ctx.Get("X-Public-Key")
	poolKey := ctx.Get("X-Pool-Key")
	secretKey := ctx.Get("X-Secret-Key")

	if appKey == "" || poolKey == "" {
		return e.ThrowUnauthorizedError("X-Public-Key and X-Pool-Key are required")
	}

	if err := validateKeyFormat(appKey); err != nil {
		return e.ThrowUnauthorizedError("Invalid X-Public-Key format")
	}

	if err := validateKeyFormat(poolKey); err != nil {
		return e.ThrowUnauthorizedError("Invalid X-Pool-Key format")
	}

	appUuid, err := this.cipherService.DecryptUuidToken(appKey)

	if err != nil {
		return e.ThrowUnauthorizedError("Invalid X-Public-Key")
	}

	poolUuid, err := this.cipherService.DecryptUuidToken(poolKey)

	if err != nil {
		return e.ThrowUnauthorizedError("Invalid X-Pool-Key")
	}

	app, err := this.appRepository.FindAppbyIdWithPool(appUuid)

	if err != nil || app == nil {
		return e.ThrowNotFound(fmt.Sprintf("Error Searching for App: `%s`", err.Error()))
	}

	if app.UsersPoolId != poolUuid {
		return e.ThrowUnauthorizedError("Invalid X-Pool-Key: `Mismatching pool key and app public key`")
	}

	if app.Private {
		if secretKey == "" {
			return e.ThrowUnauthorizedError("X-Secret-Key is required for private apps")
		}

		if app.SecretKey != secretKey {
			return e.ThrowUnauthorizedError("Invalid X-Secret-Key")
		}

		ctx.Locals("secretKey", secretKey)
	}

	global.Logger.Info(
		"App found and User Pool found",
		zap.Any("app_id", app.ID),
		zap.Any("pool_id", app.UsersPool.ID),
	)

	ctx.Locals("app", app)
	ctx.Locals("pool", &app.UsersPool)

	this.logger.Info("Finished middleware app and pool")

	return ctx.Next()
}
