package services

import (
	e "auth_service/app/errors"
	"auth_service/app/models/dto"
	ar "auth_service/app/modules/core/app/repository"
	ur "auth_service/app/modules/core/user_pool/repository"
	"auth_service/app/modules/utils/cipher"

	entity "auth_service/infra/entities"

	"github.com/google/uuid"
)

type AppService struct {
	appRepository      ar.IAppRepository
	userPoolRepository ur.IUserPoolRepository
	cipherService      cipher.ICipherService
}

var _ IAppService = &AppService{}

func NewAppService(appRepository ar.IAppRepository, userPoolRepository ur.IUserPoolRepository, cipherService cipher.ICipherService) *AppService {
	return &AppService{
		appRepository:      appRepository,
		userPoolRepository: userPoolRepository,
		cipherService:      cipherService,
	}
}

func (this *AppService) CreateWithUserPool(currentUser *entity.User, currentApp *entity.App, payload *dto.CreateAppPayload) (*entity.App, error) {

	if currentUser == nil || currentApp == nil {
		return nil, e.ThrowInternalServerError("Current user or current app is required")
	}

	var targetUserPoolid string

	if payload.UserPool.Id != "" {
		userPool, err := this.userPoolRepository.FindById(payload.UserPool.Id)

		if err != nil || userPool == nil {
			return nil, e.ThrowNotFound("User pool not found")
		}

		targetUserPoolid = userPool.ID
	} else if payload.UserPool.Name != "" {

		userPool, err := this.userPoolRepository.Create(&entity.UsersPool{
			Name:        payload.UserPool.Name,
			OwnerUserId: &currentUser.ID,
		})

		if err != nil {
			return nil, e.ThrowInternalServerError("Failed to create user pool")
		}

		if userPool == nil {
			return nil, e.ThrowInternalServerError("Failed to create user pool")
		}

		targetUserPoolid = userPool.ID
	} else {
		return nil, e.ThrowBadRequest("User pool id or name is required")
	}

	secretKey := uuid.New().String()
	publicKey := uuid.New().String()

	app, err := this.appRepository.Create(&entity.App{
		UsersPoolId:                targetUserPoolid,
		Name:                       payload.Name,
		LoginTypes:                 payload.LoginTypes,
		TokenType:                  payload.TokenType,
		TokenExpirationTime:        payload.TokenExpirationTime,
		RefreshTokenExpirationTime: payload.RefreshTokenExpirationTime,
		Private:                    payload.Private,
		VerifyEmail:                payload.VerifyEmail,
		PublicKey:                  secretKey,
		SecretKey:                  publicKey,

		OwnerUserId: &currentUser.ID,
		ParentAppId: &currentApp.ID,
	})

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to create app")
	}

	app, err = this.appRepository.FindWhere(entity.App{
		ID: app.ID,
	})

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to create app")
	}

	app.PublicKey, err = this.cipherService.EncryptUuidIntoToken(secretKey)
	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to encrypt public key")
	}
	app.SecretKey, err = this.cipherService.EncryptUuidIntoToken(publicKey)
	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to encrypt secret key")
	}

	return app, nil
}

func (this *AppService) FindAll(currentUser *entity.User, currentApp *entity.App, query *dto.GetAppsQuery) (*dto.GetAppsResponse, error) {
	if currentUser == nil || currentApp == nil {
		return nil, e.ThrowInternalServerError("Current user or current app is required")
	}

	var queryString string
	var args []interface{}

	if currentUser.Profile != nil && currentUser.Profile.Key == "ADMIN" {
		queryString = "(owner_user_id = ? OR parent_app_id = ?)"
		args = append(args, currentUser.ID, currentApp.ID)
	} else {
		queryString = "owner_user_id = ?"
		args = append(args, currentUser.ID)
	}

	if query != nil && query.Name != "" {
		queryString += " AND name ILIKE ?"
		args = append(args, "%"+query.Name+"%")
	}

	skip := 0
	limit := 10 // Default limit

	if query != nil {
		if query.Skip > 0 {
			skip = query.Skip
		}
		if query.Limit > 0 {
			limit = query.Limit
		}
	}

	apps, count, err := this.appRepository.FindManyWhereAndCount(queryString, args, skip, limit)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to fetch apps")
	}

	return &dto.GetAppsResponse{
		Total:  count,
		Amount: len(*apps),
		Skip:   skip,
		Data:   *apps,
	}, nil
}

func (this *AppService) FindById(id string) (*entity.App, error) {
	return nil, e.ThrowInternalServerError("Not Implemented Yet")
}

func (this *AppService) FindAllUserApps(userId string) ([]entity.App, error) {
	return []entity.App{}, e.ThrowInternalServerError("Not Implemented Yet")
}

func (this *AppService) Update(id string, app *entity.App) (*entity.App, error) {
	return nil, e.ThrowInternalServerError("Not Implemented Yet")
}
