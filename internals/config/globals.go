package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Globals struct {
	AegisCLISocket   string `yaml:"aegis_cli_socket"`
	ComponentsSocket string `yaml:"components_socket"`
}

type Config struct {
	AegisCLISocket   string
	ComponentsSocket string
}

func LoadGlobals() (*Config, error) {
	var globals Globals
	// First check if the environment variable is set, if so, use it directly
	aegis_env := os.Getenv("AEGIS_CLI_SOCKET")
	components_env := os.Getenv("COMPONENTS_SOCKET")

	if aegis_env != "" && components_env != "" {
		globals.AegisCLISocket = aegis_env
		globals.ComponentsSocket = components_env
		return &Config{
			AegisCLISocket:   globals.AegisCLISocket,
			ComponentsSocket: globals.ComponentsSocket,
		}, nil
	}

	// If not, read from the config file
	filePath := "config/globals.yaml"

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, &globals); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}

	if globals.AegisCLISocket == "" {
		return nil, fmt.Errorf("aegis_cli_socket not defined in globals.yaml")
	}

	return &Config{
		AegisCLISocket:   globals.AegisCLISocket,
		ComponentsSocket: globals.ComponentsSocket,
	}, nil
}
