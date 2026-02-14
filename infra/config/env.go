package config

import (
	"auth_service/common/constants"
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
	Env           string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	Sslmode  string
}

type EmailConfig struct {
	ResendApiKey string
}

type Config struct {
	Env string

	App      AppConfig
	Database DatabaseConfig
	Server   ServerConfig
	Email    EmailConfig
}

func LoadEnv(envFileName string) error {
	if err := godotenv.Load(envFileName); err != nil {
		fmt.Println("No .env file found or error loading .env file. Using system environment variables.")
	}
	return nil
}

func NewConfigFromYaml() *Config {
	return &Config{}
}

func ReadEnvArg() string {
	args := os.Args[1:]

	if len(args) > 0 {
		return args[0]
	}

	return ""
}

func NewConfigFromEnv() *Config {

	envMode := ReadEnvArg()

	envFileName := ".env"

	if envMode != "" {
		envFileName = envFileName + "." + envMode
	}

	if err := LoadEnv(envFileName); err != nil {
		fmt.Printf("Error loading environment variables: %v", err)
		panic(err)
	}

	if envMode == "" {
		envMode = os.Getenv("APP_ENV")
	}
	if envMode != "" {
		fmt.Printf(
			"-------------------------------------\n✔ Environment Mode Loaded:%s %s%s\n",
			constants.ColorGreen,
			envMode,
			constants.ColorReset,
		)
	}
	if envMode == "" {
		fmt.Printf(
			"-------------------------------------\n%sEnvironment Mode NOT FOUND%s - Using default: %s\n",
			constants.ColorRed,
			constants.ColorReset,
			"development",
		)

		envMode = "development"
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

		Server: ServerConfig{
			Port: os.Getenv("SERVER_PORT"),
		},

		App: AppConfig{
			Name:          os.Getenv("APP_NAME"),
			EncryptionKey: os.Getenv("APP_ENCRYPTION_KEY"),
			Env:           envMode,
		},

		Database: DatabaseConfig{
			Host:     host,
			Port:     port,
			User:     user,
			Password: password,
			Name:     name,
			Sslmode:  sllmode,
		},

		Email: EmailConfig{
			ResendApiKey: os.Getenv("RESEND_API_KEY"),
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
