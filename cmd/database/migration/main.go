package main

import (
	"auth_service/infra/config"
	entity "auth_service/infra/entities"
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := config.NewConfigFromEnv()
	dsn := cfg.FormatDatabaseUrl()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	fmt.Println("Running migrations with GORM...")

	// 1. Create Extensions
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";").Error; err != nil {
		log.Printf("Failed to create extension: %v", err)
	}

	// 2. Create Enums
	createEnum(db, "auth_method", "'WITH_LOGIN', 'WITH_OTP'")
	createEnum(db, "token_type", "'JWT', 'FAST_JWT', 'SESSION_UUID'")
	createEnum(db, "app_role", "'ADMIN', 'USER'")
	// Postgres types are case-insensitive usually, but typically lowercase in pg_type.
	// The previous migration used caps: CREATE TYPE AUTH_METHOD ...
	// Postgres will store it as `auth_method` usually unless quoted.
	// I'll stick to what the SQL used: caps in creation query if unquoted -> lowercase in DB?
	// SQL: CREATE TYPE AUTH_METHOD ... -> creates auth_method.
	// My createEnum uses name as provided.
	// I will use lowercase names in createEnum to match Postgres behavior and avoid issues?
	// Wait, the original migration used `AUTH_METHOD` (unquoted).
	// `SELECT typname FROM pg_type` returns lowercase.
	// So I should check for lowercase.

	// 3. AutoMigrate
	err = db.AutoMigrate(
		&entity.UsersPool{},
		&entity.Profile{},
		&entity.App{},
		&entity.User{},
		&entity.Session{},
		&entity.AppRoleProfile{},
	)
	if err != nil {
		log.Fatal("Migration failed:", err)
	}

	fmt.Println("Migrations Finished ✅")
}

func createEnum(db *gorm.DB, name string, values string) {
	var exists bool
	// Check for lower case version of name as Postgres normalizes unquoted identifiers
	db.Raw("SELECT EXISTS (SELECT 1 FROM pg_type WHERE typname = lower(?))", name).Scan(&exists)
	if !exists {
		if err := db.Exec(fmt.Sprintf("CREATE TYPE %s AS ENUM (%s);", name, values)).Error; err != nil {
			log.Printf("Failed to create enum %s: %v", name, err)
		}
	}
}
