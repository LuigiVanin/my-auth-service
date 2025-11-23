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
	Name string
}

type Config struct {
	Env string
	App AppConfig

	Server ServerConfig
}

func LoadEnv() error {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found or error loading .env file")
		return err
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

	return &Config{

		Env: env,

		Server: ServerConfig{
			Port: os.Getenv("SERVER_PORT"),
		},
		App: AppConfig{
			Name: os.Getenv("APP_NAME"),
		},
	}
}
