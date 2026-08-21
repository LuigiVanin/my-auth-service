package utils

import (
	"math/rand"
	"strconv"
	"strings"
)

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
