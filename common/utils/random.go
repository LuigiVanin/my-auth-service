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

	for range length {
		count := len(characters)
		character := strconv.Itoa(rand.Intn(count))
		result.WriteString(character)
	}

	return result.String()
}
