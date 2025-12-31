package services

import (
	"auth_service/common/utils"
	"auth_service/infra/config"
	"regexp"
)

var _ ICipherService = &CipherService{}

type CipherService struct {
	cfg *config.Config
}

type CipherServiceOptions struct {
	Prefix      string
	OverrideKey string
}

func NewCipherService(cfg *config.Config) *CipherService {
	return &CipherService{
		cfg: cfg,
	}
}

func (service *CipherService) EncryptUuidIntoToken(uuid string, options ...CipherServiceOptions) (string, error) {
	cipher := utils.NewCipher(service.cfg.App.EncryptionKey)

	token, err := cipher.EncryptUuid(uuid)

	if err != nil {
		return "", err
	}

	formattedToken := "as_" + token

	return formattedToken, nil
}

func (service *CipherService) DecryptUuidToken(token string, options ...CipherServiceOptions) (string, error) {
	token = regexp.MustCompile("^as_").ReplaceAllString(token, "")

	cipher := utils.NewCipher(service.cfg.App.EncryptionKey)

	return cipher.DecryptUuid(token)
}
