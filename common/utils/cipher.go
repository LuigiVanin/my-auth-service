package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"

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

func (c *Cipher) EncryptText(text string) (string, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(text), nil)

	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func (c *Cipher) DecryptText(token string) (string, error) {
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("invalid token length")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
