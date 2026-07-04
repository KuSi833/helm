package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// rootCmd builds the wf command tree. With no subcommand it launches the TUI;
// the --choosedir flag is read by the fish wrapper on the TUI's 'c' action.
func rootCmd() *cobra.Command {
	var chooseDir string
	root := &cobra.Command{
		Use:   "wf",
		Short: "workflow manager",
		Long: `wf manages my work-streams. With no subcommand it launches the
interactive TUI; subcommands operate on workflows from the shell.`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			runTUI(chooseDir)
			return nil
		},
	}
	root.PersistentFlags().StringVar(&chooseDir, "choosedir", "",
		"write selected workflow's dir to <path> on 'c' (used by fish wrapper)")

	root.AddCommand(listCmd(), newCmd(), genDocsCmd())
	return root
}
