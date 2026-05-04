package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	args := os.Args[1:]

	chooseDir := ""
	filtered := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--choosedir":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--choosedir requires a path")
				os.Exit(1)
			}
			i++
			chooseDir = args[i]
		case strings.HasPrefix(a, "--choosedir="):
			chooseDir = strings.TrimPrefix(a, "--choosedir=")
		default:
			filtered = append(filtered, a)
		}
	}
	args = filtered

	if len(args) == 0 {
		runTUI(chooseDir)
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

TUI flags:
      --choosedir <path>   write selected workflow's dir to <path> on 'c' (used by fish wrapper)

List flags:
  -s, --status <list>      comma-separated status filter
      --open               shorthand for --status wip,blocked`)
}
