package config_test

import (
	"os"
	"testing"

	"avitoTechAutumn2025/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate_AllPresent(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Port:           "8080",
		ProductionType: "debug",
		Database: config.Database{
			Host:     "localhost",
			Port:     "5432",
			User:     "user",
			Password: "pass",
			Name:     "dbname",
			SSLMode:  "disable",
		},
	}

	err := cfg.Validate()
	require.NoError(t, err)
}

func TestValidate_MissingPort(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Port: "",
		Database: config.Database{
			Host:     "localhost",
			Port:     "5432",
			User:     "user",
			Password: "pass",
			Name:     "dbname",
		},
	}

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "APP_PORT")
}

func TestValidate_MissingDBHost(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Port: "8080",
		Database: config.Database{
			Host:     "",
			Port:     "5432",
			User:     "user",
			Password: "pass",
			Name:     "dbname",
		},
	}

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_HOST")
}

func TestValidate_MultipleMissing(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Port: "",
		Database: config.Database{
			Host:     "",
			Port:     "",
			User:     "",
			Password: "",
			Name:     "",
		},
	}

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "APP_PORT")
	assert.Contains(t, err.Error(), "DB_HOST")
	assert.Contains(t, err.Error(), "DB_PORT")
	assert.Contains(t, err.Error(), "DB_USER")
	assert.Contains(t, err.Error(), "DB_PASSWORD")
	assert.Contains(t, err.Error(), "DB_NAME")
}

func TestValidate_WhitespaceOnly(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Port: "   ",
		Database: config.Database{
			Host:     "localhost",
			Port:     "5432",
			User:     "user",
			Password: "pass",
			Name:     "dbname",
		},
	}

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "APP_PORT")
}

func TestValidate_DefaultSSLMode(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Port: "8080",
		Database: config.Database{
			Host:     "localhost",
			Port:     "5432",
			User:     "user",
			Password: "pass",
			Name:     "dbname",
			SSLMode:  "", // должен стать "disable"
		},
	}

	err := cfg.Validate()
	require.NoError(t, err)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
}

func TestValidate_DefaultProductionType(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Port:           "8080",
		ProductionType: "",
		Database: config.Database{
			Host:     "localhost",
			Port:     "5432",
			User:     "user",
			Password: "pass",
			Name:     "dbname",
		},
	}

	err := cfg.Validate()
	require.NoError(t, err)
	assert.Equal(t, "debug", cfg.ProductionType)
}

func TestNewEnvConfig_ReadsEnvVars(t *testing.T) {
	// Устанавливаем env-переменные
	envVars := map[string]string{
		"APP_PORT":            "9090",
		"APP_PRODUCTION_TYPE": "prod",
		"APP_LOG_PATH":        "/var/log/app.log",
		"DB_HOST":             "db.example.com",
		"DB_PORT":             "5433",
		"DB_USER":             "admin",
		"DB_PASSWORD":         "secret",
		"DB_NAME":             "proddb",
		"DB_SSLMODE":          "require",
	}

	for k, v := range envVars {
		_ = os.Setenv(k, v)
	}
	defer func() {
		for k := range envVars {
			_ = os.Unsetenv(k)
		}
	}()

	cfg := config.NewEnvConfig()

	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "prod", cfg.ProductionType)
	assert.Equal(t, "/var/log/app.log", cfg.LogPath)
	assert.Equal(t, "db.example.com", cfg.Database.Host)
	assert.Equal(t, "5433", cfg.Database.Port)
	assert.Equal(t, "admin", cfg.Database.User)
	assert.Equal(t, "secret", cfg.Database.Password)
	assert.Equal(t, "proddb", cfg.Database.Name)
	assert.Equal(t, "require", cfg.Database.SSLMode)
}
