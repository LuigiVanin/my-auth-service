package services

type ICipherService interface {
	EncryptUuidIntoToken(uuid string) (string, error)
	DecryptUuidToken(token string) (string, error)
}
