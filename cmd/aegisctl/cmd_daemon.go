package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/eichiarakaki/aegis/internals/config"
	"github.com/eichiarakaki/aegis/internals/core"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the Aegis daemon process",
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Aegis daemon in the background",
	Run:   runDaemonStart,
}

var daemonShutdownCmd = &cobra.Command{
	Use:   "shutdown",
	Short: "Gracefully shut down the running Aegis daemon",
	Run:   runDaemonShutdown,
}

var daemonKillCmd = &cobra.Command{
	Use:   "kill",
	Short: "Forcefully kill the running Aegis daemon",
	Run:   runDaemonKill,
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show whether the Aegis daemon is running",
	Run:   runDaemonStatus,
}

// ---- handlers ---------------------------------------------------------------

func runDaemonStart(_ *cobra.Command, _ []string) {
	cfg, err := config.LoadGlobals()
	if err != nil {
		log.Fatal(err)
	}

	if pid, err := readPID(cfg.AegisPIDFile); err == nil {
		if isProcessRunning(pid) {
			fmt.Printf("Aegis daemon is already running (pid %d)\n", pid)
			return
		}
		os.Remove(cfg.AegisPIDFile)
	}

	daemonBin, err := resolveDaemonBinary()
	if err != nil {
		log.Fatalf("cannot find aegis-daemon binary: %v", err)
	}

	proc := exec.Command(daemonBin)
	proc.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := proc.Start(); err != nil {
		log.Fatalf("failed to start daemon: %v", err)
	}

	if err := writePID(cfg.AegisPIDFile, proc.Process.Pid); err != nil {
		log.Printf("warning: could not write PID file: %v", err)
	}

	fmt.Printf("Aegis daemon started (pid %d)\n", proc.Process.Pid)
}

func runDaemonShutdown(_ *cobra.Command, _ []string) {
	stopDaemon(core.CommandDaemonShutdown, "shutdown")
}

func runDaemonKill(_ *cobra.Command, _ []string) {
	stopDaemon(core.CommandDaemonKill, "kill")
}

// stopDaemon handles both "shutdown" and "kill" — they differ only in the
// fallback command sent over the socket when no PID file is present.
func stopDaemon(fallbackCmd core.CLICommandType, verb string) {
	cfg, err := config.LoadGlobals()
	if err != nil {
		log.Fatal(err)
	}

	pid, err := readPID(cfg.AegisPIDFile)
	if err != nil {
		if socketErr := sendCommand(fallbackCmd, nil); socketErr != nil {
			log.Fatalf("daemon does not appear to be running: %v", socketErr)
		}
		return
	}

	proc, err := os.FindProcess(pid)
	if err != nil || !isProcessRunning(pid) {
		fmt.Println("Daemon is not running")
		os.Remove(cfg.AegisPIDFile)
		return
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		log.Fatalf("failed to %s daemon (pid %d): %v", verb, pid, err)
	}

	os.Remove(cfg.AegisPIDFile)
	fmt.Printf("Aegis daemon terminated (pid %d)\n", pid)
}

func runDaemonStatus(_ *cobra.Command, _ []string) {
	cfg, err := config.LoadGlobals()
	if err != nil {
		log.Fatal(err)
	}

	pid, err := readPID(cfg.AegisPIDFile)
	if err != nil || !isProcessRunning(pid) {
		fmt.Println("Aegis daemon: stopped")
		return
	}
	fmt.Printf("Aegis daemon: running (pid %d)\n", pid)
}
