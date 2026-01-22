package services

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type HashService struct{}

func NewHashService() IHashService {
	return &HashService{}
}

const (
	// Argon2 parameters
	time    = 1
	memory  = 64 * 1024 // 64 MB
	threads = 4
	keyLen  = 32
)

func (*HashService) HashText(text string, salt string) (string, error) {
	if salt == "" {
		return "", errors.New("salt is required")
	}

	saltBytes := []byte(salt)

	hash := argon2.IDKey([]byte(text), saltBytes, time, memory, threads, keyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(saltBytes)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Format: $argon2id$v=19$m=65536,t=1,p=4$salt$hash
	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, memory, time, threads, b64Salt, b64Hash)

	return encoded, nil
}

func (*HashService) Compare(text string, hash string) (bool, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return false, errors.New("invalid hash format")
	}

	if parts[1] != "argon2id" {
		return false, errors.New("unsupported algorithm")
	}

	var v int
	_, err := fmt.Sscanf(parts[2], "v=%d", &v)
	if err != nil {
		return false, err
	}

	if v != argon2.Version {
		return false, errors.New("incompatible version")
	}

	var m, t uint32
	var p uint8

	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p)

	if err != nil {
		return false, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	newHash := argon2.IDKey([]byte(text), salt, t, m, p, uint32(len(decodedHash)))

	if subtle.ConstantTimeCompare(decodedHash, newHash) == 1 {
		return true, nil
	}

	return false, nil
}
