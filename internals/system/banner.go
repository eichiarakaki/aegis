package system

import (
	"github.com/eichiarakaki/aegis/internals/logger"
)

func Print() {
	logger.Info("=========================================================")
	logger.Info("Aegis Daemon - Deterministic Event Distribution Engine")
	logger.Info("Version: 0.1.0")
	logger.Info("Author: Eichi Arakaki")
	logger.Info("License: GNU GENERAL PUBLIC LICENSE Version 3 (GPLv3)")
	logger.Info("Repository: https://github.com/eichiarakaki/aegis")
	logger.Info("========================================================")
	cpuInfo, err := GetCPUInfo()
	if err != nil {
		logger.Error("Error getting CPU info:", err)
	} else {
		logger.Info("CPU:", cpuInfo)
	}

	totalRAM, usedRAM, err := GetRAMInfo()
	if err != nil {
		logger.Error("Error getting RAM info:", err)
	} else {
		logger.Infof("RAM: Total: %.2f GB | Used: %.2f GB", float64(totalRAM)/1e9, float64(usedRAM)/1e9)
	}

	os, platform, kernel, err := GetHostInfo()
	if err != nil {
		logger.Error("Error getting host info:", err)
	} else {
		logger.Infof("OS: %s", os)
		logger.Infof("Platform: %s", platform)
		logger.Infof("Kernel Version: %s", kernel)

	}

	logger.Info("=========================================================\n")
}
