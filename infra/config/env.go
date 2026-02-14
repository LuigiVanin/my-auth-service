package config

import (
	"auth_service/common/constants"
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/joho/godotenv"
)

type ServerConfig struct {
	Port string
}

type AppConfig struct {
	Name          string `yaml:"name"`
	EncryptionKey string `yaml:"encryption_key"`
	Env           string `yaml:"env"`
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
	ResendApiKey string `yaml:"resend_api_key"`
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

func NewConfigFromYaml(envMode string) (*Config, error) {

	// Default filename
	envFileName := ".env.yml"
	var config Config

	if envMode != "" {
		// e.g. .env.development.yaml
		envFileName = ".env." + envMode + ".yaml"
	}

	rootPath, err := findProjectRoot()
	if err != nil {
		// If root not found, try current directory
		rootPath = "."
	}

	fullPath := filepath.Join(rootPath, envFileName)
	yamlFileContent, err := os.ReadFile(fullPath)

	if err != nil {
		return nil, fmt.Errorf("error reading %s from %s: %w", envFileName, rootPath, err)
	}

	err = yaml.Unmarshal(yamlFileContent, &config)

	if err != nil {
		return nil, err
	}

	return &config, nil
}

func findProjectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd, nil
		}

		parent := filepath.Dir(wd)
		if parent == wd {
			return "", fmt.Errorf("go.mod not found")
		}
		wd = parent
	}
}

func ReadEnvArg() string {
	args := os.Args[1:]

	if len(args) > 0 {
		return args[0]
	}

	return ""
}

func NewConfigFromSysEnv(envMode string) (*Config, error) {
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
	}, nil
}

func NewConfigFromEnvFile(envMode string) (*Config, error) {

	envFileName := ".env"

	if envMode != "" {
		envFileName = envFileName + "." + envMode
	}

	if err := LoadEnv(envFileName); err != nil {
		fmt.Printf("Error loading environment variables: %v", err)
		return nil, err
	}

	if envMode == "" {
		envMode = os.Getenv("APP_ENV")
	}

	if envMode == "" {
		envMode = "development"
	}

	return NewConfigFromSysEnv(envMode)
}

func CreateEnvConfig() *Config {
	envMode := ReadEnvArg()

	// Try YAML first
	cfg, err := NewConfigFromYaml(envMode)
	if err == nil {
		fmt.Printf(
			"-------------------------------------\n%s✔ Loaded YAML Configuration (%s)%s\n-------------------------------------\n",
			constants.ColorGreen,
			envMode,
			constants.ColorReset,
		)
		return cfg
	}

	// If YAML fails (file not found or other error), try .env file
	// fmt.Printf("YAML configuration not found or invalid: %v. Trying .env file...\n", err)

	cfg, err = NewConfigFromEnvFile(envMode)
	if err == nil {
		fmt.Printf(
			"-------------------------------------\n%s✔ Loaded .env Configuration (%s)%s\n-------------------------------------\n",
			constants.ColorGreen,
			envMode,
			constants.ColorReset,
		)
		return cfg
	}

	// If .env fails, try system environment variables
	// fmt.Printf(".env configuration not found or invalid: %v. Trying system environment variables...\n", err)

	fmt.Printf("Loading from system environment variables...\n")
	cfg, err = NewConfigFromSysEnv(envMode)

	if err != nil {
		panic("No valid configuration found (checked YAML, .env, and system environment variables).")
	}

	fmt.Printf(
		"-------------------------------------\n%s✔ Loaded System Environment Variables (%s)%s\n-------------------------------------\n",
		constants.ColorGreen,
		envMode,
		constants.ColorReset,
	)
	return cfg
}

func (cfg *Config) FormatDatabaseUrl() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.Port,
		cfg.Database.Sslmode,
	)

}
