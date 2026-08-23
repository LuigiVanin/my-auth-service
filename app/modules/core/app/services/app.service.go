package services

import (
	e "auth_service/app/errors"
	dto "auth_service/app/modules/core/app/models"
	ar "auth_service/app/modules/core/app/repository"
	ups "auth_service/app/modules/core/user_pool/services"
	"auth_service/app/modules/utils/cipher"
	repo "auth_service/shared/repository"

	entity "auth_service/infra/entities"

	"github.com/google/uuid"
)

type AppService struct {
	appRepository   ar.IAppRepository
	userPoolService ups.IUserPoolService
	cipherService   cipher.ICipherService
}

var _ IAppService = &AppService{}

func NewAppService(appRepository ar.IAppRepository, userPoolService ups.IUserPoolService, cipherService cipher.ICipherService) *AppService {
	return &AppService{
		appRepository:   appRepository,
		userPoolService: userPoolService,
		cipherService:   cipherService,
	}
}

func (this *AppService) CreateWithUserPool(currentUser *entity.User, currentApp *entity.App, payload *dto.CreateAppPayload) (*entity.App, error) {

	if currentUser == nil || currentApp == nil {
		return nil, e.ThrowInternalServerError("Current user or current app is required")
	}

	var targetUserPoolid string

	if payload.UserPool.Id != "" {
		userPool, err := this.userPoolService.FindById(payload.UserPool.Id)

		if err != nil {
			return nil, e.ThrowInternalServerError("Failed to find user pool")
		}

		if userPool == nil {
			return nil, e.ThrowNotFound("User pool not found")
		}

		targetUserPoolid = userPool.ID
	} else if payload.UserPool.Name != "" {
		userPool, err := this.userPoolService.Create(payload.UserPool.Name, &currentUser.ID)

		if err != nil {
			return nil, err
		}

		targetUserPoolid = userPool.ID
	} else {
		return nil, e.ThrowBadRequest("User pool id or name is required")
	}

	secretKey := uuid.New().String()

	app, err := this.appRepository.Create(entity.App{
		UsersPoolId:                targetUserPoolid,
		Name:                       payload.Name,
		LoginTypes:                 payload.LoginTypes,
		TokenType:                  payload.TokenType,
		TokenExpirationTime:        payload.TokenExpirationTime,
		RefreshTokenExpirationTime: payload.RefreshTokenExpirationTime,
		Private:                    payload.Private,
		VerifyEmail:                payload.VerifyEmail,
		SecretKey:                  secretKey,

		OwnerUserId: &currentUser.ID,
		ParentAppId: &currentApp.ID,
	})

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to create app")
	}

	// NOTE: the public key is derived from the id, so it can only be written
	// after the insert.
	publicKey, err := this.cipherService.EncryptUuidIntoToken(app.ID)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to encrypt public key")
	}

	if _, err := this.appRepository.Update(
		entity.App{ID: app.ID},
		dto.AppUpdateDao{PublicKey: &publicKey},
	); err != nil {
		return nil, e.ThrowInternalServerError("Unable to update app after creation")
	}

	app.PublicKey = publicKey

	return app, nil
}

func (this *AppService) FindAll(currentUser *entity.User, currentApp *entity.App, query *dto.GetAppsQuery) (*dto.GetAppsResponse, error) {
	if currentUser == nil || currentApp == nil {
		return nil, e.ThrowInternalServerError("Current user or current app is required")
	}

	profileKey := ""

	if currentUser.Profile != nil {
		profileKey = currentUser.Profile.Key
	}

	search := ar.AppSearch{}

	switch profileKey {
	case "ADMIN":
		// An ADMIN may target another owner explicitly, otherwise it sees its
		// own apps plus everything under the current app.
		if query != nil && query.OwnerUserId != 0 {
			search.OwnerUserId = uint(query.OwnerUserId)
		} else {
			search.OwnerUserId = currentUser.ID
			search.OrChildrenOfAppId = &currentApp.ID
		}

	case "MANAGER":
		search.OwnerUserId = currentUser.ID

	default:
		// NOTE: the previous version left the condition empty here, which
		// listed every app in the database to any authenticated user.
		return nil, e.ThrowUnauthorizedError("User does not have permission to list apps")
	}

	option := repo.Option{With: []string{"UsersPool"}}

	if query != nil {
		search.Name = query.Name
		option = option.Paginate(query.Skip, query.Limit)
	}

	apps, err := this.appRepository.FindSearch(search, option)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to fetch apps")
	}

	total, err := this.appRepository.FindSearchCount(search, option)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to count apps")
	}

	for idx := range apps {
		// Omiting Public and Private Key from listing visualization
		apps[idx].SecretKey = ""
		apps[idx].PublicKey = ""
	}

	return &dto.GetAppsResponse{
		Total:  total,
		Amount: len(apps),
		Skip:   option.Skip,
		Limit:  option.Size(),
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
	stored, err := this.appRepository.FindOne(entity.App{ID: app.ID})

	if err != nil {
		return nil, e.ThrowInternalServerError("Unable to query app")
	}

	if stored == nil {
		return nil, e.ThrowNotFound("App with this Id was not found")
	}

	if stored.OwnerUserId == nil || (*stored.OwnerUserId != user.ID && user.Profile.Key != "ADMIN") {
		return nil, e.ThrowUnauthorizedError("User does not own the app")
	}

	return stored, e.ThrowInternalServerError("Not Implemented Yet")
}
