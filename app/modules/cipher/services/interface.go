package services

type ICipherService interface {
	EncryptUuidIntoToken(uuid string, options ...CipherServiceOptions) (string, error)
	DecryptUuidToken(token string, options ...CipherServiceOptions) (string, error)
}
