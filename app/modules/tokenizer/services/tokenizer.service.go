package services

import (
	"auth_service/common/utils"
	"errors"
	"fmt"
	"strings"
)

type TokenizerService struct{}

func NewTokenizerService() ITokenizerService {
	return &TokenizerService{}
}

func (s *TokenizerService) CreateToken(sessionId string, token string, secretKey string) (string, error) {
	// 1. Prepare Data
	rawData := fmt.Sprintf("%s|%s", sessionId, token)

	// 2. Initialize Cipher with user's secret key
	cipher := utils.NewCipher(secretKey)

	// 3. Encrypt using the new method
	return cipher.EncryptText(rawData)
}

func (s *TokenizerService) ParseToken(tokenString string, secretKey string) (string, string, error) {
	// 1. Initialize Cipher with user's secret key
	cipher := utils.NewCipher(secretKey)

	// 2. Decrypt using the new method
	plaintext, err := cipher.DecryptText(tokenString)
	if err != nil {
		return "", "", err
	}

	// 3. Parse Data
	parts := strings.Split(plaintext, "|")
	if len(parts) != 2 {
		return "", "", errors.New("invalid token format")
	}

	return parts[0], parts[1], nil
}
