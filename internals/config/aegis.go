package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ─── Types ────────────────────────────────────────────────────────────────────

type Cryptocurrency struct {
	Symbol    string   `yaml:"symbol"`
	DataTypes []string `yaml:"datatypes"`
	Intervals []string `yaml:"intervals"`
}

type Download struct {
	Enable                   bool   `yaml:"enable"`
	StartDate                string `yaml:"start_date"`
	EndDate                  string `yaml:"end_date"`
	MaxConcurrentDownloads   int    `yaml:"max_concurrent_downloads"`
	OverwriteDownloadedFiles bool   `yaml:"overwrite_downloaded_files"`
}

type Extraction struct {
	Enable                  bool `yaml:"enable"`
	RemoveAfterExtraction   bool `yaml:"remove_after_extraction"`
	OverwriteExtractedFiles bool `yaml:"overwrite_extracted_files"`
}

type Fetcher struct {
	SkipChecksumVerification bool             `yaml:"skip_checksum_verification"`
	Download                 Download         `yaml:"download"`
	Extraction               Extraction       `yaml:"extraction"`
	Cryptocurrencies         []Cryptocurrency `yaml:"cryptocurrencies"`
}

type AegisConfig struct {
	AegisPIDFile     string
	AegisCTLSocket   string  `yaml:"aegis_ctl_socket"`
	ComponentsSocket string  `yaml:"components_socket"`
	DataPath         string  `yaml:"data_path"`
	Fetcher          Fetcher `yaml:"fetcher"`
}

// ─── Defaults ─────────────────────────────────────────────────────────────────

func DefaultAegisConfig() *AegisConfig {
	return &AegisConfig{
		AegisPIDFile:     defaultPIDFile(),
		AegisCTLSocket:   "/tmp/aegis.sock",
		ComponentsSocket: "/tmp/aegis-components.sock",
		DataPath:         "~/aegis/data",
		Fetcher: Fetcher{
			SkipChecksumVerification: false,
			Extraction: Extraction{
				Enable:                  true,
				RemoveAfterExtraction:   false,
				OverwriteExtractedFiles: false,
			},
			Download: Download{
				Enable:                   true,
				StartDate:                "2023-01-01",
				EndDate:                  "2023-02-01",
				MaxConcurrentDownloads:   5,
				OverwriteDownloadedFiles: false,
			},
			Cryptocurrencies: []Cryptocurrency{},
		},
	}
}

// ─── Loaders ──────────────────────────────────────────────────────────────────

// LoadAegis loads and validates the full aegis.yaml config.
func LoadAegis() (*AegisConfig, error) {
	path, err := findConfig("aegis.yaml")
	if err != nil {
		return nil, err
	}

	cfg, err := parseConfig(path)
	if err != nil {
		return nil, err
	}

	// Environment variable takes precedence over data_path in yaml
	if v := os.Getenv("AEGIS_DATA_PATH"); v != "" {
		cfg.DataPath = v
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	cfg.DataPath = expandHome(cfg.DataPath)
	return cfg, nil
}

// LoadGlobals loads only the socket paths from aegis.yaml.
// Used by aegisd and aegisctl which only need the socket addresses.
func LoadGlobals() (*AegisConfig, error) {
	path, err := findConfig("aegis.yaml")
	if err != nil {
		return nil, err
	}

	cfg, err := parseConfig(path)
	if err != nil {
		return nil, err
	}
	
	if cfg.ComponentsSocket == "" {
		return nil, fmt.Errorf("aegis.yaml: components_socket is required")
	}
	if cfg.AegisCTLSocket == "" {
		return nil, fmt.Errorf("aegis.yaml: aegis_ctl_socket is required")
	}

	return cfg, nil
}

// ─── Internal ─────────────────────────────────────────────────────────────────

// findConfig looks for filename in ~/.config/aegis/ first, then config/ locally.
func findConfig(filename string) (string, error) {
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, ".config", "aegis", filename)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	p := filepath.Join("config", filename)
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}

	return "", fmt.Errorf("%s not found in ~/.config/aegis/ or config/", filename)
}

func parseConfig(path string) (*AegisConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var cfg AegisConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	cfg.AegisPIDFile = defaultPIDFile()

	return &cfg, nil
}

func (c *AegisConfig) validate() error {
	var errs []string

	if c.AegisCTLSocket == "" {
		errs = append(errs, "aegis_ctl_socket is required")
	}
	if c.ComponentsSocket == "" {
		errs = append(errs, "components_socket is required")
	}
	if c.DataPath == "" {
		errs = append(errs, "data_path is required (or set AEGIS_DATA_PATH)")
	}

	f := c.Fetcher
	if f.Download.Enable {
		if f.Download.StartDate == "" {
			errs = append(errs, "fetcher.download.start_date is required when download is enabled")
		}
		if f.Download.EndDate == "" {
			errs = append(errs, "fetcher.download.end_date is required when download is enabled")
		}
		if f.Download.MaxConcurrentDownloads <= 0 {
			errs = append(errs, "fetcher.download.max_concurrent_downloads must be greater than 0")
		}
		if len(f.Cryptocurrencies) == 0 {
			errs = append(errs, "fetcher.cryptocurrencies must have at least one entry when download is enabled")
		}
	}

	for i, crypto := range f.Cryptocurrencies {
		if crypto.Symbol == "" {
			errs = append(errs, fmt.Sprintf("fetcher.cryptocurrencies[%d].symbol is required", i))
		}
		if len(crypto.DataTypes) == 0 {
			errs = append(errs, fmt.Sprintf("fetcher.cryptocurrencies[%d].datatypes is required", i))
		}
		if len(crypto.Intervals) == 0 {
			errs = append(errs, fmt.Sprintf("fetcher.cryptocurrencies[%d].intervals is required", i))
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
}

func expandHome(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}

// defaultPIDFile returns $XDG_RUNTIME_DIR/aegis.pid if available,
// otherwise falls back to os.TempDir().
func defaultPIDFile() string {
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		return filepath.Join(xdg, "aegis.pid")
	}
	return filepath.Join(os.TempDir(), "aegis.pid")
}
