//go:build !windows

package campusnet

import (
	"errors"
	"syscall"
)

func interfaceSocketControl(interfaceIndex int) func(string, string, syscall.RawConn) error {
	return func(string, string, syscall.RawConn) error {
		return errors.New("interface-bound campus requests are only supported on Windows")
	}
}
