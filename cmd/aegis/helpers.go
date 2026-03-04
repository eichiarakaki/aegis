package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func resolveDaemonBinary() (string, error) {
	// 1. Same directory as the CLI binary
	self, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(self), "aegisd")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	// 2. PATH
	return exec.LookPath("aegisd")
}

func writePID(path string, pid int) error {
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0644)
}

func readPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func isProcessRunning(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 checks existence without actually sending anything
	return proc.Signal(syscall.Signal(0)) == nil
}
