//go:build !windows

package main

import "os/exec"

func makeMKVInitJob()                {}
func makeMKVAttachJob(cmd *exec.Cmd) {}
