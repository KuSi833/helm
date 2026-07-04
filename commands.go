package main

// commands.go is the single source of truth for wf's command surface.
// printUsage() and `wf gen-docs` both render from `commands` and `globalFlags`,
// so help text, generated docs, and dispatch can never drift: adding a command
// here is the only edit needed.

// flagDoc documents one flag for help/doc rendering.
type flagDoc struct {
	Names string // e.g. "-s, --status <list>" or "--open"
	Help  string
}

// command is one wf subcommand.
type command struct {
	Name    string           // "" is the no-arg TUI invocation
	Args    string           // arg summary shown after the name, e.g. "<name...>"
	Summary string           // one-line description
	Flags   []flagDoc        // command-specific flags
	Run     func([]string) error
}

// globalFlags apply regardless of subcommand.
var globalFlags = []flagDoc{
	{"--choosedir <path>", "write selected workflow's dir to <path> on 'c' (used by fish wrapper)"},
}

// commands is the full command table. Order is preserved in help/docs.
var commands = []command{
	{
		Name:    "",
		Summary: "launch interactive TUI",
		Run:     nil, // handled specially in main (needs chooseDir)
	},
	{
		Name:    "list",
		Args:    "[flags]",
		Summary: "list workflows",
		Flags: []flagDoc{
			{"-s, --status <list>", "comma-separated status filter"},
			{"--open", "shorthand for --status wip,blocked"},
		},
		Run: cmdList,
	},
	{
		Name:    "new",
		Args:    "<name...>",
		Summary: "create a new workflow",
		Run:     cmdNew,
	},
}

// lookup returns the command with the given name, or nil.
func lookup(name string) *command {
	for i := range commands {
		if commands[i].Name == name {
			return &commands[i]
		}
	}
	return nil
}
