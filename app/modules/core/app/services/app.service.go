package services

import (
	e "auth_service/app/errors"
	dto "auth_service/app/modules/core/app/models"
	ar "auth_service/app/modules/core/app/repository"
	ups "auth_service/app/modules/core/user_pool/services"
	"auth_service/app/modules/utils/cipher"
	repo "auth_service/shared/repository"
	"auth_service/shared/utils"

	entity "auth_service/infra/entities"

	"github.com/google/uuid"
	"github.com/lib/pq"
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

func (this *AppService) CreateWithUserPool(
	currentUser *entity.User,
	currentApp *entity.App,
	currentOrganization *entity.Organization,
	participant *entity.Participant,
	payload *dto.CreateAppPayload,
) (*entity.App, error) {

	if currentUser == nil || currentApp == nil || currentOrganization == nil {
		return nil, e.ThrowInternalServerError("Current user, app and organization are required")
	}

	var targetUserPoolid string

	if payload.UserPool.Id != "" {
		// NOTE: scoped. Unscoped, an id from the payload let a caller attach its app
		// to any pool in the database, including pools of other organizations.
		userPool, err := this.userPoolService.FindByIdInOrganization(
			payload.UserPool.Id,
			currentOrganization.ID,
		)

		if err != nil {
			return nil, e.ThrowInternalServerError("Failed to find user pool")
		}

		if userPool == nil {
			return nil, e.ThrowNotFound("User pool not found")
		}

		targetUserPoolid = userPool.ID
	} else if payload.UserPool.Name != "" {
		userPool, err := this.userPoolService.Create(ups.CreateUserPoolData{
			Name:             payload.UserPool.Name,
			OwnerUserId:      &currentUser.ID,
			OrganizationId:   &currentOrganization.ID,
			DefaultProfileId: payload.UserPool.DefaultProfileId,
		}, currentOrganization, participant)

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

		OwnerUserId:    &currentUser.ID,
		OrganizationId: &currentOrganization.ID,
		ParentAppId:    &currentApp.ID,
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

// NOTE: no branch per profile any more. It used to be "apps I own", plus the
// children of the current app for an ADMIN, which approximated ownership before
// organizations existed. An ADMIN now looks elsewhere by switching organization.
func (this *AppService) FindAll(
	currentUser *entity.User,
	currentApp *entity.App,
	currentOrganization *entity.Organization,
	query *dto.GetAppsQuery,
) (*dto.GetAppsResponse, error) {
	if currentUser == nil || currentApp == nil || currentOrganization == nil {
		return nil, e.ThrowInternalServerError("Current user, app and organization are required")
	}

	search := ar.AppSearch{OrganizationId: currentOrganization.ID}

	option := repo.Option{With: []string{"UsersPool"}}

	if query != nil {
		search.Name = query.Name
		option = option.Paginate(query.Skip, query.Limit)

		if query.PoolId != "" {
			pool, err := this.userPoolService.FindByIdInOrganization(
				query.PoolId,
				currentOrganization.ID,
			)

			if err != nil {
				return nil, e.ThrowInternalServerError("Failed to find the users pool")
			}

			if pool == nil {
				return nil, e.ThrowNotFound("Users pool not found in the current organization")
			}

			search.UsersPoolId = pool.ID
		}
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

func (this *AppService) FindById(id string, currentOrganization *entity.Organization) (*entity.App, error) {
	if currentOrganization == nil {
		return nil, e.ThrowInternalServerError("Current organization is required")
	}

	app, err := this.appRepository.FindOne(
		entity.App{ID: id},
		repo.Option{With: []string{"UsersPool"}},
	)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to fetch the app")
	}

	if app == nil || app.OrganizationId == nil || *app.OrganizationId != currentOrganization.ID {
		return nil, e.ThrowNotFound("App not found in the current organization")
	}

	// The public key is how a client addresses the app, so it belongs to the
	// detail view; the secret is write once at creation and never read back.
	app.SecretKey = ""

	return app, nil
}

func (this *AppService) FindAllUserApps(userId string, currentOrganization *entity.Organization) ([]entity.App, error) {
	return []entity.App{}, e.ThrowInternalServerError("Not Implemented Yet")
}

// FindById is what carries the scope rule, so the update inherits it: an app of
// another organization answers 404 and never 403, or the id becomes an oracle.
func (this *AppService) Update(
	id string,
	currentOrganization *entity.Organization,
	payload *dto.UpdateApp,
) (*entity.App, error) {
	if currentOrganization == nil || payload == nil {
		return nil, e.ThrowInternalServerError("Current organization and payload are required")
	}

	stored, err := this.FindById(id, currentOrganization)

	if err != nil {
		return nil, err
	}

	dao := dto.AppUpdateDao{
		Name:                       payload.Name,
		TokenType:                  payload.TokenType,
		TokenExpirationTime:        payload.TokenExpirationTime,
		RefreshTokenExpirationTime: payload.RefreshTokenExpirationTime,
		Private:                    payload.Private,
		VerifyEmail:                payload.VerifyEmail,
		Enabled2FA:                 payload.Enabled2FA,
	}

	if payload.LoginTypes != nil {
		loginTypes := pq.StringArray(*payload.LoginTypes)
		dao.LoginTypes = &loginTypes
	}

	if payload.Metadata != nil {
		merged, err := utils.MergeJsonPatch(stored.Metadata, *payload.Metadata)

		if err != nil {
			return nil, e.ThrowBadRequest("`metadata` has to be a JSON object", utils.JSON{"field": "metadata"})
		}

		dao.Metadata = &merged
	}

	if !repo.HasChanges(dao) {
		return stored, nil
	}

	affected, err := this.appRepository.Update(entity.App{ID: id}, dao)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to update the app")
	}

	if affected == 0 {
		return nil, e.ThrowNotFound("App not found in the current organization")
	}

	return this.FindById(id, currentOrganization)
}
