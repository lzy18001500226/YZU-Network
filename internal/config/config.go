package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Username         string
	Password         string
	Service          string
	TargetSSID       string
	CheckURL         string
	CheckHost        string
	UserAgent        string
	CheckInterval    time.Duration
	StartDelay       time.Duration
	RetryInterval    time.Duration
	CheckTimeout     time.Duration
	WiFiRetries      int
	WiFiRetryDelay   time.Duration
	RecoveryFailures int
	RecoveryCooldown time.Duration
	RecoveryTimeout  time.Duration
	Debug            bool
	LogFile          string
	DisableWiFiStep  bool
}

func Load(path string) (Config, error) {
	values := map[string]string{}
	if path != "" {
		if err := readEnvFile(path, values); err != nil && !errors.Is(err, os.ErrNotExist) {
			return Config{}, err
		}
	}

	get := func(key, fallback string) string {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
		if v := strings.TrimSpace(values[key]); v != "" {
			return v
		}
		return fallback
	}

	cfg := Config{
		Username:         get("SSO_USERNAME", ""),
		Password:         get("SSO_PASSWORD", ""),
		Service:          get("CAMPUSNET_SERVICE", "学校互联网服务"),
		TargetSSID:       get("TARGET_SSID", "YZU-WLAN"),
		CheckURL:         get("CHECK_URL", "http://111.13.141.31/generate_204"),
		CheckHost:        get("CHECK_HOST", "connect.rom.miui.com"),
		UserAgent:        get("USER_AGENT", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0 Safari/537.36"),
		CheckInterval:    parseDuration(get("CHECK_INTERVAL", "60s"), 60*time.Second),
		StartDelay:       parseDuration(get("START_DELAY", "5s"), 5*time.Second),
		RetryInterval:    parseDuration(get("RETRY_INTERVAL", "30s"), 30*time.Second),
		CheckTimeout:     parseDuration(get("CHECK_TIMEOUT", "5s"), 5*time.Second),
		WiFiRetries:      parseInt(get("WIFI_RETRIES", "5"), 5),
		WiFiRetryDelay:   parseDuration(get("WIFI_RETRY_DELAY", "3s"), 3*time.Second),
		RecoveryFailures: parseInt(get("RECOVERY_FAILURES", "5"), 5),
		RecoveryCooldown: parseDuration(get("RECOVERY_COOLDOWN", "10m"), 10*time.Minute),
		RecoveryTimeout:  parseDuration(get("RECOVERY_TIMEOUT", "45s"), 45*time.Second),
		Debug:            parseBool(get("DEBUG", "false")),
		LogFile:          os.ExpandEnv(get("LOG_FILE", "yzu-wlan.log")),
	}
	cfg.DisableWiFiStep = parseBool(get("DISABLE_WIFI_STEP", "false"))

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.Username == "" {
		return errors.New("SSO_USERNAME is required")
	}
	if c.Password == "" {
		return errors.New("SSO_PASSWORD is required")
	}
	if c.Service == "" {
		return errors.New("CAMPUSNET_SERVICE is required")
	}
	if !c.DisableWiFiStep && c.TargetSSID == "" {
		return errors.New("TARGET_SSID is required unless DISABLE_WIFI_STEP=true")
	}
	if c.CheckURL == "" {
		return errors.New("CHECK_URL is required")
	}
	if c.RecoveryFailures < 1 {
		return errors.New("RECOVERY_FAILURES must be at least 1")
	}
	if c.RecoveryCooldown < 0 {
		return errors.New("RECOVERY_COOLDOWN cannot be negative")
	}
	if c.RecoveryTimeout <= 0 {
		return errors.New("RECOVERY_TIMEOUT must be positive")
	}
	return nil
}

func readEnvFile(path string, values map[string]string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("%s:%d: expected KEY=VALUE", path, lineNo)
		}
		key = strings.TrimPrefix(strings.TrimSpace(key), "\ufeff")
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key != "" {
			values[key] = value
		}
	}
	return scanner.Err()
}

func parseDuration(value string, fallback time.Duration) time.Duration {
	if value == "" {
		return fallback
	}
	if d, err := time.ParseDuration(value); err == nil {
		return d
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}
	return fallback
}

func parseInt(value string, fallback int) int {
	i, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return i
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
