package main

import (
	"fmt"
	"os"
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
