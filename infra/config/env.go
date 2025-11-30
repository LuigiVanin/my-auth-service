package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type ServerConfig struct {
	Port string
}

type AppConfig struct {
	Name          string
	EncryptionKey string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	Sslmode  string
}

type Config struct {
	Env      string
	App      AppConfig
	Database DatabaseConfig
	Server   ServerConfig
}

func LoadEnv() error {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found or error loading .env file. Using system environment variables.")
	}
	return nil
}

func NewConfigFromEnv() *Config {
	if err := LoadEnv(); err != nil {
		fmt.Printf("Error loading environment variables: %v", err)
		panic(err)
	}

	env := os.Getenv("ENV")

	if env == "" {
		env = "dev"
	}

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_NAME")
	sllmode := os.Getenv("DB_SSLMODE")

	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "5432"
	}
	if user == "" {
		user = "postgres"
	}
	if sllmode == "" {
		sllmode = "disable"
	}

	if name == "" || password == "" {
		panic("DATABASE_NAME and DATABASE_PASSWORD are required - Cannot define database")
	}

	return &Config{

		Env: env,

		Server: ServerConfig{
			Port: os.Getenv("SERVER_PORT"),
		},
		App: AppConfig{
			Name:          os.Getenv("APP_NAME"),
			EncryptionKey: os.Getenv("APP_ENCRYPTION_KEY"),
		},

		Database: DatabaseConfig{
			Host:     host,
			Port:     port,
			User:     user,
			Password: password,
			Name:     name,
			Sslmode:  sllmode,
		},
	}
}

func (cfg *Config) FormatDatabaseUrl() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.Sslmode,
	)

}
