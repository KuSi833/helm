package main

import (
	"fmt"
	"strings"
)

func cmdList(args []string) error {
	var statusFilter []Status
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--open":
			statusFilter = []Status{StatusWIP, StatusBlocked}
		case a == "-s" || a == "--status":
			if i+1 >= len(args) {
				return fmt.Errorf("%s requires a value", a)
			}
			i++
			statusFilter = parseStatuses(args[i])
		case strings.HasPrefix(a, "--status="):
			statusFilter = parseStatuses(strings.TrimPrefix(a, "--status="))
		case strings.HasPrefix(a, "-s="):
			statusFilter = parseStatuses(strings.TrimPrefix(a, "-s="))
		default:
			return fmt.Errorf("unknown argument: %s", a)
		}
	}

	wfs, err := ScanWorkflows()
	if err != nil {
		return err
	}
	for _, w := range wfs {
		if statusFilter != nil && !containsStatus(statusFilter, w.Meta.Status) {
			continue
		}
		fmt.Printf("%s  %s  %s\n", w.Name, w.Meta.Status, w.Dir)
	}
	return nil
}

func parseStatuses(s string) []Status {
	parts := strings.Split(s, ",")
	out := make([]Status, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, Status(p))
		}
	}
	return out
}

func containsStatus(list []Status, s Status) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func cmdNew(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: wf new <name...>")
	}
	name := strings.Join(args, " ")
	w, err := CreateWorkflow(name, true)
	if err != nil {
		return err
	}
	obsidian, _ := ObsidianDir()
	fmt.Printf("Created %s\n", w.Name)
	fmt.Printf("  dir   %s\n", w.Dir)
	fmt.Printf("  note  %s\n", w.Dir+"/notes/"+w.Name+".md")
	fmt.Printf("  link  %s -> %s\n", obsidianLinkPath(obsidian, w.Name), w.Dir)
	fmt.Printf("  tmux  %s\n", w.Name)
	return nil
}
