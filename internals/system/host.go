package system

import (
	"fmt"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

func GetCPUInfo() (string, error) {
	cpuInfo, err := cpu.Info()
	if err != nil {
		return "", fmt.Errorf("failed to get CPU info: %w", err)
	}
	return cpuInfo[0].ModelName, nil
}

func GetRAMInfo() (uint64, uint64, error) {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get RAM info: %w", err)
	}
	return vmStat.Total, vmStat.Used, nil
}

func GetHostInfo() (string, string, string, error) {
	hostInfo, err := host.Info()
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get host info: %w", err)
	}
	return hostInfo.OS, hostInfo.Platform, hostInfo.KernelVersion, nil
}

func GetKernelVersion() (string, error) {
	hostInfo, err := host.Info()
	if err != nil {
		return "", fmt.Errorf("failed to get host info: %w", err)
	}
	return hostInfo.KernelVersion, nil
}
