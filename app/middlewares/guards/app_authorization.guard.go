package guards

import (
	ar "auth_service/app/modules/app/repository"
	cs "auth_service/app/modules/cipher/services"
	e "auth_service/common/errors"
	"errors"
	"fmt"
	"regexp"

	"github.com/gofiber/fiber/v2"
)

type AppGuard struct {
	appRepository ar.IAppRepository
	cipherService cs.ICipherService
}

func NewAppGuard(appRepository ar.IAppRepository, cipherService cs.ICipherService) *AppGuard {
	return &AppGuard{
		appRepository: appRepository,
		cipherService: cipherService,
	}
}

func validateKeyFormat(key string) error {
	matched, err := regexp.MatchString(`^as_[a-zA-Z0-9._-]+$`, key)

	if err != nil || !matched {
		return errors.New("invalid key format")
	}
	return nil
}

func (guard *AppGuard) Act(ctx *fiber.Ctx) error {
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

	appUuid, err := guard.cipherService.DecryptUuidToken(appKey)

	if err != nil {
		return e.ThrowUnauthorizedError("Invalid X-Public-Key")
	}

	poolUuid, err := guard.cipherService.DecryptUuidToken(poolKey)

	if err != nil {
		return e.ThrowUnauthorizedError("Invalid X-Pool-Key")
	}

	appWithPool, err := guard.appRepository.FindAppbyIdWithPool(appUuid)

	if err != nil || appWithPool == nil {
		return e.ThrowNotFound(fmt.Sprintf("Error Searching for App: `%s`", err.Error()))
	}

	if appWithPool.Pool.ID != poolUuid {
		return e.ThrowUnauthorizedError("Invalid X-Pool-Key: `Mismatching pool key and app public key`")
	}

	if appWithPool.App.Private {
		if secretKey == "" {
			return e.ThrowUnauthorizedError("X-Secret-Key is required for private apps")
		}

		if appWithPool.App.SecretKey != secretKey {
			return e.ThrowUnauthorizedError("Invalid X-Secret-Key")
		}

		ctx.Locals("secretKey", secretKey)
	}

	ctx.Locals("app", appWithPool.App)
	ctx.Locals("pool", appWithPool.Pool)

	return ctx.Next()
}
