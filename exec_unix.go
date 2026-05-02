//go:build unix

package main

import (
	"os"
	"os/exec"
	"syscall"
)

func execTmuxAttach(name string) error {
	bin, err := exec.LookPath("tmux")
	if err != nil {
		return err
	}
	return syscall.Exec(bin, []string{"tmux", "attach-session", "-t", name}, os.Environ())
}
