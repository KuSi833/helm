package main

import (
	"os/exec"
	"strings"
)

func tmuxSessionSet() map[string]bool {
	out := make(map[string]bool)
	data, err := exec.Command("tmux", "list-sessions", "-F", "#S").Output()
	if err != nil {
		return out
	}
	for _, name := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if name != "" {
			out[name] = true
		}
	}
	return out
}

func tmuxHasSession(name string) bool {
	return exec.Command("tmux", "has-session", "-t", name).Run() == nil
}

func tmuxNewSession(name, dir string) error {
	return exec.Command("tmux", "new-session", "-d", "-s", name, "-c", dir).Run()
}

func tmuxKillSession(name string) error {
	return exec.Command("tmux", "kill-session", "-t", name).Run()
}

func tmuxRenameSession(oldName, newName string) error {
	return exec.Command("tmux", "rename-session", "-t", oldName, newName).Run()
}

func tmuxSplitWithCommand(session, cmd string) error {
	return exec.Command("tmux", "split-window", "-h", "-b", "-t", session, cmd).Run()
}

func tmuxSelectPane(target string) error {
	return exec.Command("tmux", "select-pane", "-t", target).Run()
}
