package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		runTUI()
		return
	}
	switch args[0] {
	case "list":
		if err := cmdList(args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "new":
		if err := cmdNew(args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "-h", "--help", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", args[0])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`wf — workflow manager

Usage:
  wf                       launch interactive TUI
  wf list [flags]          list workflows
  wf new <name...>         create a new workflow

List flags:
  -s, --status <list>      comma-separated status filter
      --open               shorthand for --status wip,blocked`)
}

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
