package config

import (
	"os"

	"github.com/bsthun/gut"
	"github.com/sunthewhat/secure-docs-api/common"
	"github.com/sunthewhat/secure-docs-api/type/shared"
	"gopkg.in/yaml.v3"
)

func LoadConfig() {
	config := new(shared.Config)

	yml, readErr := os.ReadFile("config.yml")

	if readErr != nil {
		gut.Fatal("Failed to read config.yml", readErr)
	}

	if unmarshalErr := yaml.Unmarshal(yml, config); unmarshalErr != nil {
		gut.Fatal("Failed to unmarshal config.yml", unmarshalErr)
	}

	if validateErr := gut.Validate(config); validateErr != nil {
		gut.Fatal("Invalid config.yml", validateErr)
	}

	common.Config = config
}
