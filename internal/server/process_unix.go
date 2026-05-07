//go:build !windows

package server

import "syscall"

func backgroundProcessAttributes() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}
