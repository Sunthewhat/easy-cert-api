package config

import (
	"log/slog"
	"os"

	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/common/util"
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

	if err := util.ValidateStruct(config); err != nil {
		slog.Error("Invalid config.yml", "error", err)
		os.Exit(1)
	}

	slog.Info("Configuration loaded successfully")

	common.Config = config
}
