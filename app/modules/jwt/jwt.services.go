package jwt

import (
	"auth_service/app/models/dto"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var _ IJwtService = &JwtService{}

type JwtService struct {
}

func NewJwtService() *JwtService {
	return &JwtService{}
}

func (this *JwtService) CreateAuthToken(payload dto.AuthPayload, key string) (string, error) {
	expireTime := time.Now().Add(time.Duration(payload.ExpireTime) * time.Second)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, dto.AuthPayload{
		User: payload.User,

		AppId:      payload.AppId,
		UserPoolId: payload.UserPoolId,
		SessionId:  payload.SessionId,
		Token:      payload.Token,
		ExpireTime: payload.ExpireTime,

		Time: time.Now(),

		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    payload.AppId,
			IssuedAt:  jwt.NewNumericDate(payload.Time),
			ExpiresAt: jwt.NewNumericDate(expireTime),
		},
	})

	tokenString, err := token.SignedString([]byte(key))

	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (this *JwtService) ParseAuthToken(token string, key string) (*dto.AuthPayload, error) {
	var keyParser jwt.Keyfunc = func(t *jwt.Token) (any, error) {
		return []byte(key), nil
	}

	data, err := jwt.ParseWithClaims(token, &dto.AuthPayload{}, keyParser)

	if err != nil {
		return nil, err
	}

	if claims, ok := data.Claims.(*dto.AuthPayload); ok && data.Valid {
		return claims, nil
	}

	return nil, errors.New("Token is not Valid")
}
