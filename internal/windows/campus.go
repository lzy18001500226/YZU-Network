package windows

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
)

type CampusInterface struct {
	Name   string
	IPv4   net.IP
	Reason string
	Rank   int
}

// DetectCampusInterfaces only returns active interfaces matching an explicitly
// configured campus address range or router SSID.
func DetectCampusInterfaces(ctx context.Context, campusCIDRs, campusSSIDs []string) ([]CampusInterface, error) {
	networks := make([]*net.IPNet, 0, len(campusCIDRs))
	for _, cidr := range campusCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("parse campus CIDR %q: %w", cidr, err)
		}
		networks = append(networks, network)
	}

	wifiLinks, err := CurrentWiFiLinks(ctx)
	if err != nil && len(networks) == 0 && len(campusSSIDs) > 0 {
		return nil, fmt.Errorf("read connected Wi-Fi interfaces: %w", err)
	}
	ssidByInterface := make(map[string]string, len(wifiLinks))
	for _, link := range wifiLinks {
		ssidByInterface[strings.ToLower(link.InterfaceAlias)] = link.SSID
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("list network interfaces: %w", err)
	}

	candidates := make([]CampusInterface, 0)
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}

		var firstIPv4 net.IP
		var campusIPv4 net.IP
		var cidrMatch string
		for _, address := range addresses {
			ipNet, ok := address.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil || ipNet.IP.IsLoopback() {
				continue
			}
			ip := ipNet.IP.To4()
			if firstIPv4 == nil {
				firstIPv4 = ip
			}
			for _, network := range networks {
				if network.Contains(ip) {
					campusIPv4 = ip
					cidrMatch = network.String()
					break
				}
			}
			if campusIPv4 != nil {
				break
			}
		}
		if firstIPv4 == nil {
			continue
		}

		candidate := CampusInterface{Name: iface.Name, IPv4: firstIPv4, Rank: 2}
		if campusIPv4 != nil {
			candidate.IPv4 = campusIPv4
			candidate.Rank = 0
			candidate.Reason = "IPv4 " + campusIPv4.String() + " in " + cidrMatch
		} else if ssid, ok := ssidByInterface[strings.ToLower(iface.Name)]; ok && matchesSSID(ssid, campusSSIDs) {
			candidate.Rank = 1
			candidate.Reason = "connected to campus router SSID " + ssid
		} else {
			continue
		}
		candidates = append(candidates, candidate)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].Rank < candidates[j].Rank
	})
	return candidates, nil
}

func matchesSSID(current string, allowed []string) bool {
	for _, ssid := range allowed {
		if strings.EqualFold(strings.TrimSpace(current), strings.TrimSpace(ssid)) {
			return true
		}
	}
	return false
}
