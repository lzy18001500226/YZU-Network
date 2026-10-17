package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadReadsBOMAndExpandsLogPath(t *testing.T) {
	for _, key := range []string{
		"SSO_USERNAME", "SSO_PASSWORD", "CAMPUSNET_SERVICE", "CAMPUS_CIDRS",
		"CAMPUS_SSIDS", "LOG_FILE",
	} {
		t.Setenv(key, "")
	}
	t.Setenv("APPDATA", t.TempDir())

	envFile := filepath.Join(t.TempDir(), ".env")
	contents := "\ufeffSSO_USERNAME=student\n" +
		"SSO_PASSWORD=password\n" +
		"CAMPUS_CIDRS=10.135.0.0/17, 10.136.0.0/16\n" +
		"CAMPUS_SSIDS=Campus-Router, YZU-WLAN\n" +
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
	if got, want := strings.Join(cfg.CampusCIDRs, ","), "10.135.0.0/17,10.136.0.0/16"; got != want {
		t.Fatalf("campus CIDRs = %q, want %q", got, want)
	}
	if got, want := strings.Join(cfg.CampusSSIDs, ","), "Campus-Router,YZU-WLAN"; got != want {
		t.Fatalf("campus SSIDs = %q, want %q", got, want)
	}
	wantLog := os.ExpandEnv("$APPDATA/YZU-WLAN/test.log")
	if cfg.LogFile != wantLog {
		t.Fatalf("log file = %q, want %q", cfg.LogFile, wantLog)
	}
}

func TestParseListDropsEmptyEntries(t *testing.T) {
	got := parseList(" campus-a, , campus-b ,, ")
	if gotString, want := strings.Join(got, ","), "campus-a,campus-b"; gotString != want {
		t.Fatalf("parseList() = %q, want %q", gotString, want)
	}
}
