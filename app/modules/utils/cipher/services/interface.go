package services

type ICipherService interface {
	EncryptUuidIntoToken(uuid string, options ...CipherOptions) (string, error)
	DecryptUuidToken(token string, options ...CipherOptions) (string, error)

	DecryptTokenIntoText(token string, options ...CipherOptions) (string, error)
	EncryptTextIntoToken(text string, options ...CipherOptions) (string, error)
}
