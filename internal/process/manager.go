package process

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// KillExistingProcesses kills any existing instances of the application
func KillExistingProcesses() error {
	if runtime.GOOS != "windows" {
		return nil // Only implemented for Windows
	}

	currentPID := os.Getpid()

	// Get all desktop-server processes (including renamed ones like "desktop-server (2).exe")
	cmd := exec.Command("tasklist", "/FO", "CSV", "/NH")
	output, err := cmd.Output()
	if err != nil {
		return nil // Ignore errors
	}

	lines := strings.Split(string(output), "\n")
	killedCount := 0

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse CSV: "process.exe","PID","Session","Mem"
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}

		// Get process name
		processName := strings.Trim(parts[0], "\" ")

		// Match desktop-server*.exe processes (including renamed ones like "desktop-server (2).exe")
		if !strings.HasPrefix(strings.ToLower(processName), "desktop-server") {
			continue
		}

		pidStr := strings.Trim(parts[1], "\" ")
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		// Skip current process
		if pid == currentPID {
			continue
		}

		// Kill the process
		killCmd := exec.Command("taskkill", "/F", "/PID", strconv.Itoa(pid))
		if err := killCmd.Run(); err == nil {
			killedCount++
			fmt.Printf("Killed existing process: %s (PID %d)\n", processName, pid)
		}
	}

	if killedCount > 0 {
		fmt.Printf("Killed %d existing instance(s)\n", killedCount)
	}

	return nil
}
