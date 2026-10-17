//go:build windows

package campusnet

import (
	"fmt"
	"math/bits"
	"syscall"
)

const ipUnicastIf = 31

func interfaceSocketControl(interfaceIndex int) func(string, string, syscall.RawConn) error {
	indexNetworkOrder := int(bits.ReverseBytes32(uint32(interfaceIndex)))
	return func(network, address string, raw syscall.RawConn) error {
		if network == "tcp6" {
			return fmt.Errorf("IPv6 connections are not supported for interface-bound campus requests")
		}

		var socketErr error
		if err := raw.Control(func(fd uintptr) {
			socketErr = syscall.SetsockoptInt(
				syscall.Handle(fd), syscall.IPPROTO_IP, ipUnicastIf, indexNetworkOrder,
			)
		}); err != nil {
			return fmt.Errorf("set outgoing interface %d: %w", interfaceIndex, err)
		}
		if socketErr != nil {
			return fmt.Errorf("set outgoing interface %d: %w", interfaceIndex, socketErr)
		}
		return nil
	}
}
