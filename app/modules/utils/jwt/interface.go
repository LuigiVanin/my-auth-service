package jwt

import "auth_service/app/models/dto"

type IJwtService interface {
	CreateAuthToken(paylaod dto.AuthPayload, key string) (string, error)
	ParseAuthToken(jwt string, key string) (*dto.AuthPayload, error)
}
