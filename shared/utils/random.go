package utils

import (
	cryptorand "crypto/rand"
	"encoding/base64"
	"math/rand"
	"strconv"
	"strings"
)

// GenerateSecureToken is the only generator here fit for a secret: the two below
// use math/rand. The result is base64url without padding, safe in a URL as is.
func GenerateSecureToken(bytes int) (string, error) {
	buffer := make([]byte, bytes)

	if _, err := cryptorand.Read(buffer); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func GenerateRandomDigits(length int) string {
	var result strings.Builder

	for range length {
		digit := strconv.Itoa(rand.Intn(10))
		result.WriteString(digit)
	}

	return result.String()
}

func GenerateRandomCharacters(length int) string {
	var result strings.Builder
	characters := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	count := len(characters)

	for range length {
		index := rand.Intn(count)
		character := string(characters[index])
		result.WriteString(character)
	}

	return result.String()
}
