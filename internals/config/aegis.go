package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Must be defined in globals.yaml or via environment variables

type Currency struct {
	Symbol     string   `yaml:"symbol"`
	Name       string   `yaml:"name"`
	Type       string   `yaml:"type"` // spot, futures, options, etc.
	Timeframes []string `yaml:"timeframes"`
}

type Backtesting struct {
	StartDate string `yaml:"start_date"`
	EndDate   string `yaml:"end_date"`

	DispatchSocket string `yaml:"dispatch_socket"`
}

type Live struct {
	GraphqlEndpoint string `yaml:"graphql_endpoint"`

	DispatchSocket string `yaml:"dispatch_socket"`
}

type AegisConfig struct {
	DataPath   string     `yaml:"data_path"`
	Currencies []Currency `yaml:"currencies"`

	Backtesting Backtesting `yaml:"backtesting"`
	Live        Live        `yaml:"live"`
}

func LoadAegis() (*AegisConfig, error) {
	var aegis AegisConfig

	filePath := "config/aegis.yaml"

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, &aegis); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}

	if aegis.DataPath == "" {
		return nil, fmt.Errorf("data_path not defined in aegis.yaml")
	}

	if aegis.Currencies == nil {
		return nil, fmt.Errorf("currencies not defined in aegis.yaml")
	}

	if aegis.Backtesting.DispatchSocket == "" {
		return nil, fmt.Errorf("backtesting.dispatch_socket not defined in aegis.yaml")
	}

	if aegis.Live.GraphqlEndpoint == "" {
		return nil, fmt.Errorf("live.graphql_endpoint not defined in aegis.yaml")
	}

	// First check if the environment variable is set, if so, use it directly
	aegis_data_path_env := os.Getenv("DATA_PATH")

	if aegis.DataPath == "" && aegis_data_path_env == "" {
		return nil, fmt.Errorf("data_path not defined in aegis.yaml or DATA_PATH environment variable")
	}

	// Override with environment variable if set
	if aegis_data_path_env != "" {
		aegis.DataPath = aegis_data_path_env
	}

	return &AegisConfig{
		DataPath:    aegis.DataPath,
		Currencies:  aegis.Currencies,
		Backtesting: aegis.Backtesting,
		Live:        aegis.Live,
	}, nil
}
