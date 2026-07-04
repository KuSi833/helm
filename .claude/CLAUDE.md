# helm / wf

The `wf` workflow-manager CLI. Hand-rolled arg parsing (no Cobra).

## Single source of truth for commands

`commands.go` holds the `commands` table — the one place that defines wf's
command surface. Three things render from it, so they cannot drift:

- **dispatch** — `main.go` → `lookup(args[0]).Run`
- **`--help`** — `usage.go` `renderUsage()`
- **the doc** — `wf gen-docs` → `COMMANDS.md` (wraps `renderUsage()`)

`COMMANDS.md` is **generated — never hand-edit it.**

### Adding / changing a command

1. Edit the `commands` table (and add its `flagDoc` entries for any flags).
2. `go generate ./...` — regenerates `COMMANDS.md`.
3. `git add COMMANDS.md` and commit.

A pre-commit hook (`scripts/pre-commit`, run via `wf gen-docs --check`) blocks
the commit if `COMMANDS.md` is stale. On a fresh clone, install it once:

```sh
ln -sf ../../scripts/pre-commit .git/hooks/pre-commit
```

### Known gap

Flags are documented in the table by hand, not auto-discovered from the
parsing code. If you add a flag inside a `cmd*` function, add its `flagDoc`
to the table too, or the docs won't mention it.
