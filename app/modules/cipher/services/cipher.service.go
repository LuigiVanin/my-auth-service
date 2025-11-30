package services

import (
	"auth_service/common/utils"
	"auth_service/infra/config"
	"regexp"
)

type CipherService struct {
	cfg *config.Config
}

func NewCipherService(cfg *config.Config) ICipherService {
	return &CipherService{
		cfg: cfg,
	}
}

func (service *CipherService) EncryptUuidIntoToken(uuid string) (string, error) {
	cipher := utils.NewCipher(service.cfg.App.EncryptionKey)

	token, err := cipher.EncryptUuid(uuid)

	if err != nil {
		return "", err
	}

	formattedToken := "as_" + token

	return formattedToken, nil
}

func (service *CipherService) DecryptUuidToken(token string) (string, error) {
	token = regexp.MustCompile("^as_").ReplaceAllString(token, "")

	cipher := utils.NewCipher(service.cfg.App.EncryptionKey)

	return cipher.DecryptUuid(token)
}
