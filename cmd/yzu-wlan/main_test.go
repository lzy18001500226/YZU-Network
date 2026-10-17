package main

import (
	"context"
	"net"
	"testing"
	"time"

	"yzu-wlan/internal/config"
	"yzu-wlan/internal/windows"
)

type fakeCampusPathClient struct {
	connectionResults []bool
	checkCalls        int
	portalCalls       int
	loginCalls        int
	loginErr          error
}

func (c *fakeCampusPathClient) TestConnection(string, string) bool {
	if len(c.connectionResults) == 0 {
		c.checkCalls++
		return false
	}
	index := c.checkCalls
	c.checkCalls++
	if index >= len(c.connectionResults) {
		index = len(c.connectionResults) - 1
	}
	return c.connectionResults[index]
}

func (c *fakeCampusPathClient) GetPortalURL() (string, error) {
	c.portalCalls++
	return "http://portal.example/", nil
}

func (c *fakeCampusPathClient) OldLoginService(string, string, string, string) error {
	c.loginCalls++
	return c.loginErr
}

func TestShouldRecoverNetwork(t *testing.T) {
	cfg := config.Config{
		RecoveryFailures: 3,
		RecoveryCooldown: 10 * time.Minute,
	}
	now := time.Now()

	if shouldRecoverNetwork(cfg, 2, time.Time{}) {
		t.Fatal("recovery triggered before the failure threshold")
	}
	if !shouldRecoverNetwork(cfg, 3, time.Time{}) {
		t.Fatal("recovery did not trigger at the first threshold")
	}
	if shouldRecoverNetwork(cfg, 4, now.Add(-5*time.Minute)) {
		t.Fatal("recovery ignored the cooldown")
	}
	if !shouldRecoverNetwork(cfg, 4, now.Add(-11*time.Minute)) {
		t.Fatal("recovery did not trigger after the cooldown")
	}
}

func TestRunCycleChecksEachCampusPathAndAvoidsRedundantLogin(t *testing.T) {
	wired := &fakeCampusPathClient{connectionResults: []bool{true}}
	wifi := &fakeCampusPathClient{connectionResults: []bool{false, true}}
	clients := map[string]*fakeCampusPathClient{"Ethernet": wired, "WLAN": wifi}
	paths := []windows.CampusInterface{
		{Name: "Ethernet", IPv4: net.ParseIP("10.135.1.5")},
		{Name: "WLAN", IPv4: net.ParseIP("192.168.1.20")},
	}
	cfg := config.Config{StartDelay: 0, CheckURL: "http://check.example/", CheckHost: "check.example"}
	detect := func(context.Context, []string, []string) ([]windows.CampusInterface, error) {
		return paths, nil
	}
	newClient := func(name string) (campusPathClient, error) { return clients[name], nil }

	if _, err := runCycleWith(context.Background(), cfg, detect, newClient); err != nil {
		t.Fatalf("runCycleWith() error = %v", err)
	}
	if wired.checkCalls != 1 {
		t.Errorf("wired checks = %d, want 1", wired.checkCalls)
	}
	if wifi.checkCalls != 2 {
		t.Errorf("Wi-Fi checks = %d, want 2", wifi.checkCalls)
	}
	if wifi.portalCalls != 0 || wifi.loginCalls != 0 {
		t.Errorf("Wi-Fi portal/login calls = %d/%d, want 0/0", wifi.portalCalls, wifi.loginCalls)
	}
}

func TestRunCycleAuthenticatesOnlyTheFailedCampusPath(t *testing.T) {
	wired := &fakeCampusPathClient{connectionResults: []bool{true}}
	wifi := &fakeCampusPathClient{connectionResults: []bool{false, false, true}}
	clients := map[string]*fakeCampusPathClient{"Ethernet": wired, "WLAN": wifi}
	paths := []windows.CampusInterface{
		{Name: "Ethernet", IPv4: net.ParseIP("10.135.1.5")},
		{Name: "WLAN", IPv4: net.ParseIP("192.168.1.20")},
	}
	cfg := config.Config{
		StartDelay: 0, CheckURL: "http://check.example/", CheckHost: "check.example",
		Username: "student", Password: "secret", Service: "campus",
	}
	detect := func(context.Context, []string, []string) ([]windows.CampusInterface, error) {
		return paths, nil
	}
	newClient := func(name string) (campusPathClient, error) { return clients[name], nil }

	if _, err := runCycleWith(context.Background(), cfg, detect, newClient); err != nil {
		t.Fatalf("runCycleWith() error = %v", err)
	}
	if wired.portalCalls != 0 || wired.loginCalls != 0 {
		t.Errorf("wired portal/login calls = %d/%d, want 0/0", wired.portalCalls, wired.loginCalls)
	}
	if wifi.portalCalls != 1 || wifi.loginCalls != 1 {
		t.Errorf("Wi-Fi portal/login calls = %d/%d, want 1/1", wifi.portalCalls, wifi.loginCalls)
	}
}
