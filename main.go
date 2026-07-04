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
	case "-h", "--help", "help":
		printUsage()
		return
	case "gen-docs":
		if err := cmdGenDocs(args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	c := lookup(args[0])
	if c == nil || c.Run == nil {
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", args[0])
		printUsage()
		os.Exit(1)
	}
	if err := c.Run(args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
