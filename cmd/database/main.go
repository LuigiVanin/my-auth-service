package main

import (
	"fmt"
	"os"

	"auth_service/infra/config"
	"auth_service/shared/constants"
	"auth_service/shared/utils"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Usage: go run ./cmd/database <seed> [environment]
//
//	<seed>        : init | reset
//	[environment] : development (default) | prod | ...
func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	seedName := args[0]

	// `config.CreateEnvConfig` resolves the environment from `os.Args[1]`, so the
	// seed name is dropped from the arguments to keep `[environment]` in place.
	os.Args = append([]string{os.Args[0]}, args[1:]...)

	cfg := config.CreateEnvConfig()

	db, err := gorm.Open(postgres.Open(cfg.FormatDatabaseUrl()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		utils.PrintError("failed to connect to database", err)
		os.Exit(1)
	}

	switch seedName {
	case "init":
		if err := run(db, cfg); err != nil {
			utils.PrintError("seed `init` failed", err)
			os.Exit(1)
		}

	case "reset":
		if err := reset(db); err != nil {
			utils.PrintError("seed `reset` failed", err)
			os.Exit(1)
		}

	default:
		fmt.Printf("%s✘ unknown seed `%s`%s\n", constants.ColorRed, seedName, constants.ColorReset)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println()
	utils.PrintHeader("Database Seeds")
	fmt.Printf("Usage: make seed <seed> [environment]\n\n")
	utils.PrintKeyValue("make seed init", "populates the database with the main pool, app, profiles and admin user")
	utils.PrintKeyValue("make seed reset", "wipes every seeded table so `init` can run from scratch again")
	fmt.Println()
}
