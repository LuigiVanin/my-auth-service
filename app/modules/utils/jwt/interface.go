package jwt

import dto "auth_service/app/modules/utils/jwt/models"

type IJwtService interface {
	CreateAuthToken(paylaod dto.AuthPayload, key string) (string, error)
	ParseAuthToken(jwt string, key string) (*dto.AuthPayload, error)
}
