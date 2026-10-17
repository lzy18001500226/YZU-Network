package windows

import (
	"context"
	"os/exec"
	"regexp"
	"strings"
)

type WiFiLink struct {
	InterfaceAlias string
	SSID           string
}

var (
	wifiInterfacePattern = regexp.MustCompile(`(?mi)^\s*(?:Name|名称)\s*:\s*(.+?)\s*$`)
	wifiSSIDPattern      = regexp.MustCompile(`(?mi)^\s*SSID\s*:\s*(.+?)\s*$`)
)

func CurrentWiFiLinks(ctx context.Context) ([]WiFiLink, error) {
	cmd := exec.CommandContext(ctx, "netsh", "wlan", "show", "interfaces")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	text := string(out)
	interfaceMatches := wifiInterfacePattern.FindAllStringSubmatchIndex(text, -1)
	links := make([]WiFiLink, 0, len(interfaceMatches))
	for i, match := range interfaceMatches {
		end := len(text)
		if i+1 < len(interfaceMatches) {
			end = interfaceMatches[i+1][0]
		}
		block := text[match[1]:end]
		ssidMatch := wifiSSIDPattern.FindStringSubmatch(block)
		if ssidMatch == nil {
			continue
		}
		ssid := strings.TrimSpace(ssidMatch[1])
		if ssid == "" || strings.EqualFold(ssid, "BSSID") {
			continue
		}
		links = append(links, WiFiLink{
			InterfaceAlias: strings.TrimSpace(text[match[2]:match[3]]),
			SSID:           ssid,
		})
	}
	return links, nil
}
