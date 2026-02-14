package main

import (
	"fmt"
	"os"

	"auth_service/app/modules/utils/cipher/services"
	"auth_service/infra/config"
)

func main() {
	// Capture original args
	originalArgs := os.Args

	if len(originalArgs) < 2 {
		fmt.Println("Usage: make cipher <uuid> [command: encrypt|decrypt] [env: dev|prod]")
		os.Exit(1)
	}

	uuidStr := originalArgs[1]
	command := "encrypt"
	env := ""

	// Handle optional arguments for command and env
	if len(originalArgs) > 2 {
		arg2 := originalArgs[2]
		if arg2 == "encrypt" || arg2 == "decrypt" {
			command = arg2
			// If command is specified, env must be the 3rd argument if present
			if len(originalArgs) > 3 {
				env = originalArgs[3]
			}
		} else {
			// If arg2 is not a command, assume it's the environment
			env = arg2
		}
	}

	// Prepare os.Args for config.NewConfigFromEnv
	// NewConfigFromEnv expects os.Args[1] to be the environment name if provided.
	// We temporarily modify os.Args to match this expectation so the config loads the correct .env file.
	if env != "" {
		os.Args = []string{originalArgs[0], env}
	} else {
		os.Args = []string{originalArgs[0]}
	}

	cfg := config.CreateEnvConfig()
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
		// This should be unreachable given the logic above, but good for safety
		fmt.Println("Invalid command. Available commands: encrypt, decrypt")
		os.Exit(1)
	}
}
