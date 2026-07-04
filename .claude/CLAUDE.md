# helm / wf

The `wf` workflow-manager CLI, built with [Cobra](https://github.com/spf13/cobra).

## Command surface = the Cobra tree

The Cobra command tree in `main.go` (`rootCmd`) + `cmd.go` (`listCmd`/`newCmd`)
is the single source of truth. Cobra generates `--help` from it, and
`wf gen-docs` (in `gendocs.go`) renders `COMMANDS.md` from the same tree via
`cobra/doc`. So help and docs can't drift from dispatch — there's no
hand-written usage string or flag table.

`COMMANDS.md` is **generated — never hand-edit it.**

### Adding / changing a command

1. Add/modify the `*cobra.Command` (a new `fooCmd()` wired into `rootCmd`, or a
   `Flags()` call). Flags are auto-discovered — no separate doc entry needed.
2. `go generate ./...` — regenerates `COMMANDS.md`.
3. `git add COMMANDS.md` and commit.

A pre-commit hook (`scripts/pre-commit`, runs `wf gen-docs --check`) blocks the
commit if `COMMANDS.md` is stale. On a fresh clone, install it once:

```sh
ln -sf ../../scripts/pre-commit .git/hooks/pre-commit
```

### Notes

- No-subcommand `wf` launches the TUI (`rootCmd`'s `RunE`). `--choosedir` is a
  persistent flag read by the fish wrapper.
- `gen-docs` is a hidden command; it's excluded from `--help` and `COMMANDS.md`.
- `DisableAutoGenTag` is set so the generated docs have no timestamp — keeps
  `--check` output stable.
