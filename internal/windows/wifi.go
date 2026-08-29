package windows

import (
	"context"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

func CurrentSSID(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "netsh", "wlan", "show", "interfaces")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	text := string(out)
	match := regexp.MustCompile(`(?m)^\s*SSID\s*:\s*(.+?)\s*$`).FindStringSubmatch(text)
	if match == nil {
		return "", nil
	}
	ssid := strings.TrimSpace(match[1])
	if strings.EqualFold(ssid, "BSSID") {
		return "", nil
	}
	return ssid, nil
}

func ConnectWiFi(ctx context.Context, ssid string) error {
	cmd := exec.CommandContext(ctx, "netsh", "wlan", "connect", "name="+ssid)
	return cmd.Run()
}

func WaitForWiFi(ctx context.Context, ssid string, retries int, delay time.Duration) (bool, string, error) {
	var last string
	for i := 0; i < retries; i++ {
		current, err := CurrentSSID(ctx)
		if err != nil {
			return false, last, err
		}
		last = current
		if current == ssid {
			return true, current, nil
		}
		if i == 0 {
			_ = ConnectWiFi(ctx, ssid)
		}
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return false, last, ctx.Err()
		}
	}
	return false, last, nil
}
