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

type CipherOptions struct {
	Prefix      string
	OverrideKey string
}

func NewCipherService(cfg *config.Config) *CipherService {
	return &CipherService{
		cfg: cfg,
	}
}

func (this *CipherService) resolveOptions(options ...CipherOptions) CipherOptions {
	defaultOptions := CipherOptions{
		Prefix:      "as_",
		OverrideKey: this.cfg.App.EncryptionKey,
	}

	if len(options) > 0 {
		if options[0].Prefix != "" {
			defaultOptions.Prefix = options[0].Prefix
		}
		if options[0].OverrideKey != "" {
			defaultOptions.OverrideKey = options[0].OverrideKey
		}
	}

	return defaultOptions
}

func (this *CipherService) EncryptUuidIntoToken(uuid string, options ...CipherOptions) (string, error) {
	opts := this.resolveOptions(options...)
	prefix := opts.Prefix
	secretKey := opts.OverrideKey

	cipher := utils.NewCipher(secretKey)

	token, err := cipher.EncryptUuid(uuid)

	if err != nil {
		return "", err
	}

	formattedToken := prefix + token

	return formattedToken, nil
}

func (this *CipherService) DecryptUuidToken(token string, options ...CipherOptions) (string, error) {
	opts := this.resolveOptions(options...)
	prefix := opts.Prefix
	secretKey := opts.OverrideKey

	token = regexp.MustCompile("^"+prefix).ReplaceAllString(token, "")

	cipher := utils.NewCipher(secretKey)

	return cipher.DecryptUuid(token)
}

func (this *CipherService) EncryptTextIntoToken(text string, options ...CipherOptions) (string, error) {
	opts := this.resolveOptions(options...)
	prefix := opts.Prefix
	secretKey := opts.OverrideKey

	cipher := utils.NewCipher(secretKey)

	tokenString, err := cipher.EncryptText(text)
	if err != nil {
		return "", err
	}

	return prefix + tokenString, nil
}

func (this *CipherService) DecryptTokenIntoText(token string, options ...CipherOptions) (string, error) {
	opts := this.resolveOptions(options...)
	prefix := opts.Prefix
	secretKey := opts.OverrideKey

	token = regexp.MustCompile("^"+prefix).ReplaceAllString(token, "")

	cipher := utils.NewCipher(secretKey)

	return cipher.DecryptText(token)
}
