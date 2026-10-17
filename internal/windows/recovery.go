package windows

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

const maxStatusLength = 2000

// NetworkStatus returns concise interface state for failure diagnostics.
func NetworkStatus(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "netsh", "interface", "show", "interface").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("netsh interface show interface: %w: %s", err, compactOutput(out))
	}
	return compactOutput(out), nil
}

// RenewDHCP requests a fresh lease without forcibly disconnecting an adapter.
func RenewDHCP(ctx context.Context, interfaceAlias string) error {
	if strings.TrimSpace(interfaceAlias) == "" {
		return fmt.Errorf("interface alias is required")
	}
	out, err := exec.CommandContext(ctx, "ipconfig", "/renew", interfaceAlias).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ipconfig /renew %q: %w: %s", interfaceAlias, err, compactOutput(out))
	}
	return nil
}

func compactOutput(out []byte) string {
	text := strings.Join(strings.Fields(string(out)), " ")
	if len(text) > maxStatusLength {
		return text[:maxStatusLength] + "..."
	}
	return text
}
