package config

import (
	"errors"
	"log/slog"
	"os"

	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/type/shared"
	"gopkg.in/yaml.v3"
)

func LoadConfig() {
	config := new(shared.Config)

	yml, readErr := os.ReadFile("config.yml")

	if readErr != nil {
		slog.Error("Failed to read config.yml", "error", readErr)
		os.Exit(1)
	}

	if unmarshalErr := yaml.Unmarshal(yml, config); unmarshalErr != nil {
		slog.Error("Failed to unmarshal config.yml", "error", unmarshalErr)
		os.Exit(1)
	}

	if err := validateConfig(config); err != nil {
		slog.Error("Invalid config.yml", "error", err)
		os.Exit(1)
	}

	slog.Info("Configuration loaded successfully")

	common.Config = config
}

func validateConfig(config *shared.Config) error {
	if config.Environment == nil {
		return errors.New("environment is required")
	}
	if config.Port == nil {
		return errors.New("port is required")
	}
	if len(config.Cors) == 0 {
		return errors.New("cors is required")
	}
	if config.JWTSecret == nil {
		return errors.New("jwt_secret is required")
	}
	if config.Postgres == nil {
		return errors.New("postgres is required")
	}
	if config.Mongo == nil {
		return errors.New("mongo is required")
	}
	if config.MongoDatabase == nil {
		return errors.New("mongo_database is required")
	}
	return nil
}
