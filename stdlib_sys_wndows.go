//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

func prepareCmdForBackground(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// 0x01000000 = CREATE_BREAKAWAY_FROM_JOB
		// 0x08000000 = CREATE_NO_WINDOW
		CreationFlags: 0x01000000 | 0x08000000,
	}
}
