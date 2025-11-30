package main

import (
	"auth_service/app/modules/cipher/services"
	"auth_service/infra/config"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/encrypt/main.go <uuid>")
		os.Exit(1)
	}

	uuidStr := os.Args[1]
	command := "encrypt"

	if len(os.Args) > 2 && len(os.Args) < 4 {
		command = os.Args[2]
	}

	cfg := config.NewConfigFromEnv()
	cipher := services.NewCipherService(cfg)

	switch command {

	case "encrypt":
		encrypted, err := cipher.EncryptUuidIntoToken(uuidStr)
		if err != nil {
			fmt.Printf("Error encrypting UUID: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nOriginal UUID: %s\n", uuidStr)
		fmt.Printf("Encrypted Key: %s\n", encrypted)
		fmt.Println("\nUse this 'Encrypted Key' in your X-Public-Key or X-Pool-Key header.")

	case "decrypt":
		decrypted, err := cipher.DecryptUuidToken(uuidStr)
		if err != nil {
			fmt.Printf("Error decrypting UUID: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nEncrypted Key: %s\n", uuidStr)
		fmt.Printf("Decrypted Key: %s\n", decrypted)
	default:
		fmt.Println("Invalid command")
		os.Exit(1)
	}
}
