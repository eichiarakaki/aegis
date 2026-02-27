package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Cryptocurrency struct {
	Symbol    string   `yaml:"symbol"`
	DataTypes []string `yaml:"datatypes"`
	Intervals []string `yaml:"intervals"`
}

type AegisFetcherConfig struct {
	Cryptocurrencies []Cryptocurrency `yaml:"cryptocurrencies"`
}

func LoadAegisFetcher() (*AegisFetcherConfig, error) {
	var aegis AegisFetcherConfig

	filePath := "config/aegis-fetcher.yaml"

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, &aegis); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}

	if aegis.Cryptocurrencies == nil {
		return nil, fmt.Errorf("cryptocurrencies not defined in aegis-fetcher.yaml")
	}

	return &AegisFetcherConfig{
		Cryptocurrencies: aegis.Cryptocurrencies,
	}, nil
}
