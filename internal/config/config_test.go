package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsBOMAndExpandsLogPath(t *testing.T) {
	for _, key := range []string{
		"SSO_USERNAME", "SSO_PASSWORD", "CAMPUSNET_SERVICE", "TARGET_SSID",
		"DISABLE_WIFI_STEP", "LOG_FILE",
	} {
		t.Setenv(key, "")
	}
	t.Setenv("APPDATA", t.TempDir())

	envFile := filepath.Join(t.TempDir(), ".env")
	contents := "\ufeffSSO_USERNAME=student\n" +
		"SSO_PASSWORD=password\n" +
		"DISABLE_WIFI_STEP=true\n" +
		"LOG_FILE=$APPDATA/YZU-WLAN/test.log\n"
	if err := os.WriteFile(envFile, []byte(contents), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	cfg, err := Load(envFile)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Username != "student" {
		t.Fatalf("username = %q, want student", cfg.Username)
	}
	if cfg.Service != "学校互联网服务" {
		t.Fatalf("service = %q, want default service", cfg.Service)
	}
	wantLog := os.ExpandEnv("$APPDATA/YZU-WLAN/test.log")
	if cfg.LogFile != wantLog {
		t.Fatalf("log file = %q, want %q", cfg.LogFile, wantLog)
	}
}
