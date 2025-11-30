package utils

import (
	"crypto/aes"
	"crypto/sha256"
	"encoding/base64"
	"errors"

	"github.com/google/uuid"
)

type Cipher struct {
	key []byte
}

func NewCipher(secret string) *Cipher {
	hash := sha256.Sum256([]byte(secret))
	return &Cipher{key: hash[:]}
}

func (c *Cipher) EncryptUuid(uuidString string) (string, error) {
	// Parse UUID to ensure it's valid and get bytes
	u, err := uuid.Parse(uuidString)
	if err != nil {
		return "", err
	}
	uuidBytes := u[:] // 16 bytes

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}

	encrypted := make([]byte, len(uuidBytes))
	block.Encrypt(encrypted, uuidBytes)

	formattedEncryptedUuid := base64.RawURLEncoding.EncodeToString(encrypted)

	return formattedEncryptedUuid, nil
}

func (c *Cipher) DecryptUuid(token string) (string, error) {
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", err
	}

	if len(data) != 16 {
		return "", errors.New("invalid token length")
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}

	decrypted := make([]byte, 16)
	block.Decrypt(decrypted, data)

	// Parse bytes back to UUID to ensure validity and formatting
	u, err := uuid.FromBytes(decrypted)
	if err != nil {
		return "", err
	}

	return u.String(), nil
}
