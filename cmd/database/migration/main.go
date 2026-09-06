package main

import (
	"auth_service/infra/config"
	entity "auth_service/infra/entities"
	"auth_service/shared/constants"
	"fmt"
	"log"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := config.CreateEnvConfig()
	dsn := cfg.FormatDatabaseUrl()

	fmt.Println("DSN:", dsn)

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
	createEnum(db, "auth_method", "WITH_LOGIN", "WITH_OTP", "WITH_PASSWORD")
	createEnum(db, "auth_action", constants.AuthActions...)
	createEnum(db, "token_type", "JWT", "FAST_JWT", "SESSION_UUID")
	createEnum(db, "app_role", "ADMIN", "USER")
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

	// 3. Drop what the organization refactor removed.
	//
	// AutoMigrate never drops a column or a table. Without this step
	// `users.profile_id` stays behind as a NOT NULL column the entity no longer
	// fills, so the very next insert fails, and `app_role_profiles` lingers with a
	// foreign key into `profiles`.
	dropRemoved(db,
		`DROP TABLE IF EXISTS app_role_profiles`,
		`ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS profile_id`,
	)

	// 4. AutoMigrate
	// We run migration in two steps to handle circular dependencies (e.g., UsersPool <-> User)

	// Step 1: Create tables without foreign key constraints
	db.Config.DisableForeignKeyConstraintWhenMigrating = true
	err = db.AutoMigrate(
		&entity.UsersPool{},
		&entity.Profile{},
		&entity.User{},
		&entity.Organization{},
		&entity.Participant{},
		&entity.App{},
		&entity.Session{},
		&entity.Otp{},
	)
	if err != nil {
		log.Fatal("Migration failed during table creation:", err)
	}

	// Step 2: Create foreign key constraints
	db.Config.DisableForeignKeyConstraintWhenMigrating = false
	err = db.AutoMigrate(
		&entity.UsersPool{},
		&entity.Profile{},
		&entity.User{},
		&entity.Organization{},
		&entity.Participant{},
		&entity.App{},
		&entity.Session{},
		&entity.Otp{},
	)
	if err != nil {
		log.Fatal("Migration failed during constraint creation:", err)
	}

	fmt.Println("Migrations Finished ✅")
}

// Fatal on purpose: a leftover NOT NULL column breaks every insert afterwards, and
// a warning in the log is not enough to notice it.
func dropRemoved(db *gorm.DB, statements ...string) {
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			log.Fatalf("Failed to run `%s`: %v", statement, err)
		}
	}
}

func createEnum(db *gorm.DB, name string, enums ...string) {
	var exists bool

	for idx, e := range enums {
		enums[idx] = fmt.Sprintf("'%s'", e)
	}

	values := strings.Join(enums, ", ")

	// Check for lower case version of name as Postgres normalizes unquoted identifiers
	db.Raw("SELECT EXISTS (SELECT 1 FROM pg_type WHERE typname = lower(?))", name).Scan(&exists)
	if !exists {
		if err := db.Exec(fmt.Sprintf("CREATE TYPE %s AS ENUM (%s);", name, values)).Error; err != nil {
			log.Printf("Failed to create enum %s: %v", name, err)
		}
	}
}
