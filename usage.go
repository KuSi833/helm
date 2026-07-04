package main

import (
	"fmt"
	"strings"
)

// column at which flag/command help text is aligned.
const helpCol = 27

// pad renders "  <left><spaces>help" with help aligned at helpCol.
func pad(indent int, left, help string) string {
	prefix := strings.Repeat(" ", indent) + left
	if help == "" {
		return prefix
	}
	if len(prefix) >= helpCol {
		prefix += "  "
	} else {
		prefix += strings.Repeat(" ", helpCol-len(prefix))
	}
	return prefix + help
}

// titleCase upper-cases the first letter (ASCII command names only).
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// usageInvocation renders the "wf <name> <args>" left-column string.
func usageInvocation(c command) string {
	parts := []string{"wf"}
	if c.Name != "" {
		parts = append(parts, c.Name)
	}
	if c.Args != "" {
		parts = append(parts, c.Args)
	}
	return strings.Join(parts, " ")
}

// renderUsage builds the full `wf --help` text from the command table.
func renderUsage() string {
	var b strings.Builder
	b.WriteString("wf — workflow manager\n\n")
	b.WriteString("Usage:\n")
	for _, c := range commands {
		b.WriteString(pad(2, usageInvocation(c), c.Summary) + "\n")
	}

	sections := []struct {
		title string
		flags []flagDoc
	}{
		{"Global flags", globalFlags},
	}
	for _, c := range commands {
		if len(c.Flags) > 0 {
			title := titleCase(c.Name) + " flags"
			sections = append(sections, struct {
				title string
				flags []flagDoc
			}{title, c.Flags})
		}
	}
	for _, s := range sections {
		b.WriteString("\n" + s.title + ":\n")
		for _, f := range s.flags {
			b.WriteString(pad(6, f.Names, f.Help) + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func printUsage() {
	fmt.Println(renderUsage())
}
