package main

import (
	"testing"
	"time"

	"yzu-wlan/internal/config"
)

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
