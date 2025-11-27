package main

import (
	"auth_service/infra/config"
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func RunMigrations(db *sql.DB, url string) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})

	if err != nil {
		fmt.Println("Failed to create driver: ", err.Error())
		return err
	}

	fmt.Print("Created Driver Successfully(postgres)!\n\n")

	fmt.Println("Running migrations...")
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)

	if err != nil {
		fmt.Println("Failed to create migration instance: ", err.Error())
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		fmt.Println("Failed to run migrations: ", err.Error())
		return err
	}

	fmt.Print("Created Migration Instance Successfully!\n\n")
	return nil
}

func main() {
	fmt.Println("Running migrations...")

	cfg := config.NewConfigFromEnv()
	url := cfg.FormatDatabaseUrl()

	db, err := sql.Open("postgres", url)
	if err != nil {
		panic(fmt.Sprintf("Failed to open database: %s", err.Error()))
	}
	defer db.Close()

	if err := RunMigrations(db, url); err != nil {
		panic(fmt.Sprintf("Failed to run migrations: %s", err.Error()))
	}

	fmt.Println("Migrations Finished ✅")
}
