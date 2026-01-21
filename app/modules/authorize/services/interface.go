package services

import (
	"auth_service/app/models/dto"
	entity "auth_service/infra/entities"
)

type IAuthorizeService interface {
	Authorize(
		app *entity.App,
		token string,
		ip string,
	) (*dto.AuthorizeReponse, error)

	Refresh(
		app *entity.App,
		token string,
		ip string,
	) (*dto.RefreshResponse, error)
}
