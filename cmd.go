package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	var status string
	var open bool
	c := &cobra.Command{
		Use:   "list",
		Short: "list workflows",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var statusFilter []Status
			switch {
			case open:
				statusFilter = []Status{StatusWIP, StatusBlocked}
			case status != "":
				statusFilter = parseStatuses(status)
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
		},
	}
	c.Flags().StringVarP(&status, "status", "s", "", "comma-separated status filter")
	c.Flags().BoolVar(&open, "open", false, "shorthand for --status wip,blocked")
	return c
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

func newCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new <name...>",
		Short: "create a new workflow",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
		},
	}
}
