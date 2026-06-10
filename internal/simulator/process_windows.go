//go:build windows

package simulator

import "syscall"

func terminateProcess(pid int) error {
	process, err := syscall.OpenProcess(syscall.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(process)
	return syscall.TerminateProcess(process, 0)
}
