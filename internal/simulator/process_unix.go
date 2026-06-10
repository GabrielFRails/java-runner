//go:build !windows

package simulator

import "syscall"

func terminateProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}
