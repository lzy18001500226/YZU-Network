package windows

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const TaskName = "YZU-WLAN-AutoLogin"

func InstallStartupTask(exePath, envPath string) error {
	exePath, err := filepath.Abs(exePath)
	if err != nil {
		return err
	}
	args := fmt.Sprintf(`"%s" -once -e "%s"`, exePath, envPath)
	return runSchTasks("/Create", "/F",
		"/SC", "ONSTART",
		"/TN", TaskName,
		"/TR", args,
		"/DELAY", "0000:30",
		"/RU", currentUser())
}

func InstallRepeatTask(exePath, envPath string) error {
	exePath, err := filepath.Abs(exePath)
	if err != nil {
		return err
	}
	args := fmt.Sprintf(`"%s" -once -e "%s"`, exePath, envPath)
	return runSchTasks("/Create", "/F",
		"/SC", "HOURLY",
		"/TN", TaskName+"-Repeat",
		"/TR", args,
		"/MO", "1",
		"/RU", currentUser())
}

func UninstallTasks() error {
	if err := runSchTasks("/Delete", "/F", "/TN", TaskName); err != nil {
		return err
	}
	return runSchTasks("/Delete", "/F", "/TN", TaskName+"-Repeat")
}

func runSchTasks(args ...string) error {
	cmd := exec.Command("schtasks", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("schtasks %v failed: %w: %s", args, err, string(out))
	}
	return nil
}

func currentUser() string {
	domain := os.Getenv("USERDOMAIN")
	user := os.Getenv("USERNAME")
	if domain != "" && user != "" {
		return domain + `\` + user
	}
	return user
}
