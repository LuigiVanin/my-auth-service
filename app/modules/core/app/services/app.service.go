package services

import (
	e "auth_service/app/errors"
	dto "auth_service/app/modules/core/app/models"
	ar "auth_service/app/modules/core/app/repository"
	ur "auth_service/app/modules/core/user_pool/repository"
	"auth_service/app/modules/utils/cipher"
	"errors"

	entity "auth_service/infra/entities"

	"github.com/google/uuid"
	"gorm.io/gorm"
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

	app, err := this.appRepository.Create(&entity.App{
		UsersPoolId:                targetUserPoolid,
		Name:                       payload.Name,
		LoginTypes:                 payload.LoginTypes,
		TokenType:                  payload.TokenType,
		TokenExpirationTime:        payload.TokenExpirationTime,
		RefreshTokenExpirationTime: payload.RefreshTokenExpirationTime,
		Private:                    payload.Private,
		VerifyEmail:                payload.VerifyEmail,
		// PublicKey:                  secretKey,
		SecretKey: secretKey,

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

	encryptKey, err := this.cipherService.EncryptUuidIntoToken(app.ID)
	app.PublicKey = encryptKey

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to encrypt public key")
	}

	_, err = this.appRepository.Update(
		app.ID,
		*app,
	)

	if err != nil {
		return nil, e.ThrowInternalServerError("Unable to update app after creation")
	}

	// app.PublicKey, err = this.cipherService.EncryptUuidIntoToken(secretKey)
	// if err != nil {
	// 	return nil, e.ThrowInternalServerError("Failed to encrypt public key")
	// }
	// app.SecretKey, err = this.cipherService.EncryptUuidIntoToken(publicKey)
	// if err != nil {
	// 	return nil, e.ThrowInternalServerError("Failed to encrypt secret key")
	// }

	return app, nil
}

func (this *AppService) FindAll(currentUser *entity.User, currentApp *entity.App, query *dto.GetAppsQuery) (*dto.GetAppsResponse, error) {
	if currentUser == nil || currentApp == nil {
		return nil, e.ThrowInternalServerError("Current user or current app is required")
	}

	var queryString string
	var args []interface{}

	if currentUser.Profile != nil && currentUser.Profile.Key == "ADMIN" {

		// If there is a "OwnerUserId" on the query string it should be used if the currentUser is an ADMIN user
		if query.OwnerUserId != 0 {
			queryString = "owner_user_id = ?"
			args = append(args, query.OwnerUserId)
		} else {
			queryString = "(parent_app_id = ? OR owner_user_id = ?)"
			args = append(args, currentApp.ID, currentUser.ID)
		}

		// If the user is manager, the OwnerUserId field will always be equal to the current user
	} else if currentUser.Profile != nil && currentUser.Profile.Key == "MANAGER" {
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

	apps, count, err := this.appRepository.FindManyWhereAndCount(queryString, args, skip, limit, "UsersPool")

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to fetch apps")
	}

	for idx := range apps {
		// Omiting Public and Private Key from listing visualization
		apps[idx].SecretKey = ""
		apps[idx].PublicKey = ""
	}

	return &dto.GetAppsResponse{
		Total:  count,
		Amount: len(apps),
		Skip:   skip,
		Limit:  limit,
		Data:   apps,
	}, nil
}

func (this *AppService) FindById(id string) (*entity.App, error) {
	return nil, e.ThrowInternalServerError("Not Implemented Yet")
}

func (this *AppService) FindAllUserApps(userId string) ([]entity.App, error) {
	return []entity.App{}, e.ThrowInternalServerError("Not Implemented Yet")
}

func (this *AppService) Update(user *entity.User, app *entity.App) (*entity.App, error) {
	app, err := this.appRepository.FindWhere(entity.App{ID: app.ID})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.ThrowNotFound("App with this Id was not found")
		}

		return nil, e.ThrowInternalServerError("Unable to query app")
	}

	if *app.OwnerUserId != user.ID && user.Profile.Key != "ADMIN" {
		return nil, e.ThrowUnauthorizedError("User does not own the app")
	}

	_, err = this.appRepository.Update(app.ID, *app)

	if err != nil {
		return nil, e.ThrowInternalServerError("Could not update app")
	}

	return app, e.ThrowInternalServerError("Not Implemented Yet")
}
