package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port           string
	ProductionType string
	LogPath        string

	Database Database
}

type Database struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

func NewEnvConfig() *Config {
	return &Config{
		Port:           os.Getenv("APP_PORT"),
		ProductionType: os.Getenv("APP_PRODUCTION_TYPE"),
		LogPath:        os.Getenv("APP_LOG_PATH"),

		Database: Database{
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     os.Getenv("DB_NAME"),
			SSLMode:  os.Getenv("DB_SSLMODE"),
		},
	}
}

func (config *Config) PrintConfigWithHiddenSecrets() {
	// Функция для маскировки секретов
	mask := func(s string) string {
		if s == "" {
			return ""
		}
		return strings.Repeat("*", len(s))
	}

	fmt.Println("========== Configuration ==========\n")

	fmt.Println("App Configuration:")
	fmt.Printf("\tPort: %s\n", config.Port)
	fmt.Printf("\tProductionType: %s\n", config.ProductionType)
	fmt.Printf("\tLogPath: %s\n", config.LogPath)

	fmt.Println("\nDatabase Configuration:")
	fmt.Printf("\tHost: %s\n", config.Database.Host)
	fmt.Printf("\tPort: %s\n", config.Database.Port)
	fmt.Printf("\tUser: %s\n", config.Database.User)
	fmt.Printf("\tPassword: %s\n", mask(config.Database.Password))
	fmt.Printf("\tName: %s\n", config.Database.Name)
	fmt.Printf("\tSSLMode: %s\n", config.Database.SSLMode)

	fmt.Println("\n===================================")
}
