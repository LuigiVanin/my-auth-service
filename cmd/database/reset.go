package main

import (
	"fmt"
	"strings"

	entity "auth_service/infra/entities"
	"auth_service/shared/utils"

	"gorm.io/gorm"
)

// seededTables lists every table written by `init`, ordered from the most
// dependent one to the least, so the truncate reads top-down like the seed does
// bottom-up. `CASCADE` covers any table added later that references them.
var seededTables = []any{
	&entity.Otp{},
	&entity.Session{},
	&entity.AppRoleProfile{},
	&entity.App{},
	&entity.User{},
	&entity.Profile{},
	&entity.UsersPool{},
}

// reset wipes the seeded data while keeping the schema (tables, enums and
// extensions) in place, so `make seed init` can run from scratch again without
// re-running the migrations.
func reset(db *gorm.DB) error {
	fmt.Println()
	utils.PrintHeader("Resetting Database Seed")

	tables := make([]string, 0, len(seededTables))

	for _, model := range seededTables {
		stmt := &gorm.Statement{DB: db}

		if err := stmt.Parse(model); err != nil {
			return fmt.Errorf("failed to resolve table name: %w", err)
		}

		table := stmt.Schema.Table

		if !db.Migrator().HasTable(table) {
			utils.PrintWarning(fmt.Sprintf("Table `%s` does not exist, skipping", table))
			continue
		}

		tables = append(tables, fmt.Sprintf("%q", table))
	}

	if len(tables) == 0 {
		utils.PrintWarning("No seeded table found, nothing to reset (did you run `make migrate up`?)")
		return nil
	}

	// RESTART IDENTITY resets the `users.id` sequence so a fresh `init` starts at 1 again.
	query := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", strings.Join(tables, ", "))

	if err := db.Exec(query).Error; err != nil {
		return fmt.Errorf("failed to truncate seeded tables: %w", err)
	}

	for _, table := range tables {
		utils.PrintSuccess(fmt.Sprintf("Table %s truncated", table))
	}

	fmt.Println()
	utils.PrintWarning(fmt.Sprintf("Credentials from previous runs are still listed in %s", credentialsFile))
	utils.PrintSuccess("Database reset, you can run `make seed init` again")
	fmt.Println()

	return nil
}
