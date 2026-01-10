package services

import entity "auth_service/infra/entities"

type IAuthorizeService interface {
	Authorize(
		app *entity.App,
		token string,
		// user *entity.User,
		// request dto.RequestInfo,
	) (string, error)
}
