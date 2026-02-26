package health

import (
	"fmt"
	"os"

	"github.com/eichiarakaki/aegis/internals/config"
	"github.com/eichiarakaki/aegis/internals/logger"
)

func DataHealthCheck() error {
	logger.Info("Checking data health...")

	cfg, err := config.LoadAegis()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	dataPath := cfg.DataPath

	if dataPath == "" {
		return fmt.Errorf("data path is empty")
	}

	info, err := os.Stat(dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("data path does not exist: %s", dataPath)
		}
		return fmt.Errorf("cannot access data path: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("data path is not a directory: %s", dataPath)
	}

	return nil
}

func SessionsHealthCheck() error {
	logger.Info("Checking session manager health...")

	// Placeholder for session manager health check logic
	return nil
}

func CheckAll() error {
	logger.Info("Performing health check...")

	return nil
}
