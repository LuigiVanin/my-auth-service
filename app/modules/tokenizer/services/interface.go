package services

type ITokenizerService interface {
	CreateToken(sessionId string, token string, secretKey string) (string, error)
	ParseToken(tokenString string, secretKey string) (string, string, error)
}
