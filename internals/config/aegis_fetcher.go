package config

import (
	"os"

	"github.com/eichiarakaki/aegis/internals/logger"
	"gopkg.in/yaml.v3"
)

type Cryptocurrency struct {
	Symbol    string   `yaml:"symbol"`
	DataTypes []string `yaml:"datatypes"`
	Intervals []string `yaml:"intervals"`
}

type Download struct {
	Enable                   bool `yaml:"enable"`
	MaxConcurrentDownloads   int  `yaml:"max_concurrent_downloads"`
	OverwriteDownloadedFiles bool `yaml:"overwrite_downloaded_files"`
}

type Extraction struct {
	Enable                 bool `yaml:"enable"`
	RemoveAfterExtraction  bool `yaml:"remove_after_extraction"`
	OverrideExtractedFiles bool `yaml:"override_extracted_files"`
}

type AegisFetcherConfig struct {
	SkipChecksumVerification bool             `yaml:"skip_checksum_verification"`
	Extraction               Extraction       `yaml:"extraction"`
	Download                 Download         `yaml:"download"`
	Cryptocurrencies         []Cryptocurrency `yaml:"cryptocurrencies"`
}

func DefaultAegisFetcherConfig() *AegisFetcherConfig {
	return &AegisFetcherConfig{
		SkipChecksumVerification: false,
		Extraction: Extraction{
			Enable:                 true,
			RemoveAfterExtraction:  false,
			OverrideExtractedFiles: false,
		},
		Download: Download{
			Enable:                   true,
			MaxConcurrentDownloads:   5,
			OverwriteDownloadedFiles: false,
		},
		Cryptocurrencies: []Cryptocurrency{},
	}
}

func LoadAegisFetcher() *AegisFetcherConfig {
	var aegis AegisFetcherConfig

	filePath := "config/aegis-fetcher.yaml"

	data, err := os.ReadFile(filePath)
	if err != nil {
		logger.Errorf("Failed to read config file: %v\n", err)
		return DefaultAegisFetcherConfig()
	}

	if err := yaml.Unmarshal(data, &aegis); err != nil {
		logger.Errorf("Failed to parse yaml: %v\n", err)
		return DefaultAegisFetcherConfig()
	}

	if aegis.Cryptocurrencies == nil {
		logger.Warn("No cryptocurrencies defined in config — no data will be fetched")
		aegis.Cryptocurrencies = []Cryptocurrency{}
	}

	return &AegisFetcherConfig{
		SkipChecksumVerification: aegis.SkipChecksumVerification,
		Extraction:               aegis.Extraction,
		Download:                 aegis.Download,
		Cryptocurrencies:         aegis.Cryptocurrencies,
	}
}
