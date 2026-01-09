package dto

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JwtUser struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Id    uint   `json:"id"`
}

type AuthPayload struct {
	User JwtUser `json:"user"`

	AppId      string    `json:"app_id"`
	UserPoolId string    `json:"user_pool_id"`
	SessionId  string    `json:"session_id"`
	Token      string    `json:"token"`
	Time       time.Time `json:"time"`

	ExpireTime uint `json:"expire_time"`

	jwt.RegisteredClaims
}
