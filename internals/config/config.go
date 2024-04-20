package config

import (
	"encoding/json"
	"fmt"
	"os"
)

const DefaultConfigFile = "/.gorp/gorp-config.json"

type ServerConfig struct {
	Port int `json:"port"`
}

type NodeConfig struct {
	Registry               string            `json:"registry"`
	Fallback               []string          `json:"fallback"`
	UseFallbackForMappings bool              `json:"useFallbackForMappings"`
	Mappings               map[string]string `json:"mappings"`
}

type GorpConfig struct {
	Server ServerConfig `json:"server"`
	Node   NodeConfig   `json:"node"`
}

type Overrides struct {
	Port int
}

func HandleOverrides(config GorpConfig, overrides Overrides) (GorpConfig, error) {
	if overrides.Port > 0 {
		config.Server.Port = overrides.Port
	}
	return config, nil
}

func GetConfigFile(configFile string) (string, error) {

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	if configFile == "" {
		configFile = DefaultConfigFile
	}
	configFile = homeDir + configFile
	return configFile, nil
}

func LoadConfigFile(configFile string, overrides Overrides) (GorpConfig, error) {
	var config GorpConfig = GorpConfig{}

	file, err := os.Open(configFile)
	if err != nil {
		return config, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)

	if err != nil {
		return config, err
	}
	config, err = HandleOverrides(config, overrides)
	if err != nil {
		return config, fmt.Errorf(`error handling overrides: %s`, err)
	}
	return config, nil
}
