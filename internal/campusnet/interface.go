package campusnet

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

func newInterfaceTransport(timeout time.Duration, interfaceName string) (*http.Transport, error) {
	interfaceName = strings.TrimSpace(interfaceName)
	if interfaceName == "" {
		return nil, fmt.Errorf("interface name is required")
	}

	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("find interface %q: %w", interfaceName, err)
	}
	if iface.Flags&net.FlagUp == 0 {
		return nil, fmt.Errorf("interface %q is down", interfaceName)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("list addresses for interface %q: %w", interfaceName, err)
	}
	var localIP net.IP
	for _, addr := range addrs {
		var ip net.IP
		switch value := addr.(type) {
		case *net.IPNet:
			ip = value.IP
		case *net.IPAddr:
			ip = value.IP
		}
		if ipv4 := ip.To4(); ipv4 != nil && !ipv4.IsLoopback() {
			localIP = ipv4
			break
		}
	}
	if localIP == nil {
		return nil, fmt.Errorf("interface %q has no IPv4 address", interfaceName)
	}

	dialer := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: 30 * time.Second,
		LocalAddr: &net.TCPAddr{IP: localIP},
		Control:   interfaceSocketControl(iface.Index),
	}
	return &http.Transport{
		Proxy:       nil,
		DialContext: dialer.DialContext,
	}, nil
}
