//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

func prepareCmdForBackground(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Entkoppelt den Prozess von der Terminal-Session (nohup-Effekt)
	}
}
