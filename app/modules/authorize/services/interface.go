package services

import (
	dto "auth_service/app/modules/authorize/models"
	entity "auth_service/infra/entities"
)

type AuthorizationCredentials struct {
	AccessToken  string
	RefreshToken string
}

type IAuthorizeService interface {
	Authorize(
		app *entity.App,
		token string,
		ip string,
	) (*dto.AuthorizeResponse, error)

	Refresh(
		app *entity.App,
		token string,
		ip string,
	) (*dto.RefreshResponse, error)

	CreateAuthorizationCredentials(
		app *entity.App,
		session *entity.Session,
	) (*AuthorizationCredentials, error)

	ResetPassword(app *entity.App, payload dto.ResetPasswordPayload) (*entity.User, error)
}
