# wf — Workflow Manager Specification
​
A terminal-based workflow manager that tracks numbered tasks/projects with a two-panel TUI, tmux session management, and Obsidian vault integration.
​
---
​
## Configuration
​
Config file: `$XDG_CONFIG_HOME/wf/config.yaml` (defaults to `~/.config/wf/config.yaml`).
​
```yaml
workflows_dir: ~/workflows           # where workflow dirs live
vault_dir: ~/git/ObsiNotes           # Obsidian vault root
vault_subpath: notes/workflows       # path inside the vault for workflow symlinks (default)
```
​
For each value, lookup order is:
​
1. Config file
2. Env var (`WORKFLOWS_DIR`, `OBSIDIAN_VAULT_DIR`) — fallback for `workflows_dir` / `vault_dir`
3. Built-in default — only `vault_subpath` has one (`notes/workflows`)
​
If `workflows_dir` or `vault_dir` cannot be resolved, the command aborts with
`<VAR> is not set`.
​
External tools assumed available on `$PATH`: `tmux`.
​
---
​
## Workflow Directory Layout
​
Each workflow is a numbered directory inside `WORKFLOWS_DIR`:
​
```
<WORKFLOWS_DIR>/
  042-my-task/
    workflow.yaml          # metadata (status, created date, slack URL)
    notes/
      042-my-task.md       # root note (markdown)
    .claude/
      CLAUDE.md            # instructions for Claude Code when working in this directory
```
​
### Directory Naming
​
Format: `NNN-slug` where:
- `NNN` is a zero-padded 3-digit number (e.g., `001`, `042`).
- `slug` is derived from the workflow name: lowercased, non-alphanumeric characters replaced with `-`, leading/trailing `-` stripped.
- Example: input `"Fix auth bug!"` with next number 42 becomes `042-fix-auth-bug`.
​
### workflow.yaml
​
```yaml
status: wip          # required; one of: wip, todo, later, blocked, completed, dead
created: "2026-05-02"  # set at creation time (YYYY-MM-DD)
slack: ""            # optional Slack URL
```
​
If the file is missing or unparseable, status defaults to `"unknown"`.
​
### Root Note
​
Created at `notes/<NNN-slug>.md` with initial content:
​
```markdown
# <NNN-slug>
```
​
### .claude/CLAUDE.md
​
Auto-generated per workflow with boilerplate instructions telling Claude Code to treat the root note as an index and create separate files for specific topics.
​
### Obsidian Symlink
​
A symlink is created at `<OBSIDIAN_VAULT_DIR>/notes/workflows/<NNN-slug>` pointing to the workflow's `notes/` directory, so Obsidian can browse workflow notes.
​
---
​
## Number Assignment
​
The next number is determined by scanning all directories in `WORKFLOWS_DIR` matching the `^\d{3}-.+$` pattern, finding the maximum number, and adding 1. Starts at 1 if no workflows exist.
​
---
​
## Status Values
​
| Status | Color (ANSI) | Considered "active" | Gets tmux session |
|---|---|---|---|
| `wip` | yellow (3) | yes | yes |
| `todo` | blue (4) | no | yes |
| `later` | magenta (5) | no | no |
| `blocked` | red (1), bold | yes | yes |
| `completed` | green (2) | no | no |
| `dead` | gray (8) | no | no |
| `unknown` | gray (8) | no | no |
​
"Active" means `wip` or `blocked` (the `--open` filter).
​
### Tmux Lifecycle on Status Change
​
When a workflow's status is changed:
- If the new status is `wip`, `todo`, or `blocked` and no tmux session exists: a detached tmux session is created (`tmux new-session -d -s <name> -c <dir>`).
- If the new status is anything else and a tmux session exists: the session is killed (`tmux kill-session -t <name>`).
​
---
​
## CLI Commands
​
### `wf` (no subcommand) — Interactive TUI
​
Launches a full-screen alternate-screen TUI (bubbletea). On exit, if the user selected a tmux session to attach, the process replaces itself (`syscall.Exec`) with `tmux attach-session -t <name>`.
​
### `wf list` — List Workflows to stdout
​
Prints one line per workflow: `<NNN-slug>  <status>  <path>`.
​
Sorted by number descending (highest/newest first).
​
**Flags:**
- `--status <statuses>` / `-s <statuses>`: Comma-separated status filter. Only workflows matching one of the given statuses are shown.
- `--open`: Shorthand for `--status wip,blocked`.
- If neither flag is given, all workflows are shown.
​
### `wf new <name...>` — Create a New Workflow
​
All positional arguments are joined with spaces and slugified.
​
Creates:
1. Workflow directory `<WORKFLOWS_DIR>/<NNN-slug>/`
2. `workflow.yaml` with status `wip` and today's date.
3. `notes/<NNN-slug>.md` with `# <NNN-slug>` content.
4. `.claude/CLAUDE.md` with boilerplate.
5. Obsidian symlink: `<OBSIDIAN_VAULT_DIR>/notes/workflows/<NNN-slug>` -> `<dir>/notes/`.
6. Detached tmux session named `<NNN-slug>` with cwd set to the workflow directory.
7. Splits the tmux session horizontally (left/right). Left pane runs `cl` (Claude Code). Right pane is selected as active.
​
Prints a summary:
```
Created 042-fix-auth-bug
  dir   /path/to/042-fix-auth-bug
  note  /path/to/042-fix-auth-bug/notes/042-fix-auth-bug.md
  link  /obsidian/notes/workflows/042-fix-auth-bug -> /path/to/042-fix-auth-bug
  tmux  042-fix-auth-bug
```
​
---
​
## TUI Specification
​
### Layout
​
Two-panel layout using the full terminal (alt screen):
​
```
╭─ [1] Workflows ──────╮╭─ Info ──────────────────────╮
│ [w]ip  [o]pen  [a]ll  ││  status  wip                │
│                       ││  tmux    active              │
│ 066 static-grants wip ││  slack   none                │
│ 065 hca-ses-msg   wip ││  dir     ~/workflows/066-... │
│ ...                   │╰─────────────────────────────╯
│                       │╭─ [2] Note ──────────────────╮
│                       ││ # 066-static-grants          │
│                       ││ ...rendered markdown...      │
╰───────────────────────╯╰─────────────────────────────╯
  n new  r rename  t tmux  s status  d delete  / search  q quit
```
​
- **Left panel** (40% width): Scrollable workflow list with filter tabs.
- **Right panel** (60% width): Split into Info box (4 lines) and Note preview (remaining height).
- **Footer**: Context-sensitive keybinding hints.
​
### Panel Focus
​
Two focusable panels: `[1] Workflows` (list) and `[2] Note` (preview). Focused panel has a blue border; unfocused has a gray border.
​
### Modes
​
The TUI has 7 modes:
​
| Mode | Trigger | Description |
|---|---|---|
| `modeNormal` | default | Navigate the workflow list |
| `modeSearch` | `/` | Text input filters list by name |
| `modeConfirmDelete` | `d` | Asks `Delete <name>? y/n` |
| `modeStatusToggle` | `s` | Shows status picker in footer |
| `modePreview` | `2`, `enter`, `tab` | Focus moves to note viewport for scrolling |
| `modeNewWorkflow` | `n` | Text input to create a new workflow |
| `modeRename` | `r` | Text input to rename current workflow |
​
---
​
### Normal Mode Keybindings
​
| Key | Action |
|---|---|
| `q` | Quit |
| `j` / `down` | Move cursor down |
| `k` / `up` | Move cursor up |
| `g` | Jump to first item |
| `G` | Jump to last item |
| `a` | Set filter to "all" |
| `w` | Set filter to "wip" (default on launch) |
| `o` | Set filter to "open" (wip + blocked) |
| `]` / `right` | Cycle filter forward: wip -> active -> all -> wip |
| `[` / `left` | Cycle filter backward |
| `/` | Enter search mode |
| `s` | Enter status toggle mode (if a workflow is selected) |
| `d` | Enter delete confirmation mode (if a workflow is selected) |
| `t` | Toggle tmux session for selected workflow (create if absent, kill if present). Refreshes list after. |
| `n` | Enter new workflow mode |
| `r` | Enter rename mode (pre-fills current slug) |
| `R` | Force refresh — rescan workflows from disk |
| `O` | Open selected workflow's note in Obsidian via `obsidian://open` URL scheme |
| `S` | Open selected workflow's Slack URL (if set in workflow.yaml) via `open` command |
| `2` / `enter` | Switch focus to note preview panel |
| `tab` / `shift+tab` | Switch focus to note preview panel |
| `1` | No-op (already in list panel) |
​
### Search Mode Keybindings
​
| Key | Action |
|---|---|
| Any character | Updates search input |
| `enter` | Apply search text as filter, return to normal mode |
| `esc` | Clear search text, return to normal mode |
​
Search is case-insensitive substring match against the display name (`NNN-slug`).
​
### Confirm Delete Mode Keybindings
​
Footer shows: `Delete <NNN-slug>? y/n`
​
| Key | Action |
|---|---|
| `y` | Delete workflow (removes directory, Obsidian symlink, kills tmux session if present). Refreshes list. |
| `n` / `esc` | Cancel, return to normal mode |
​
### Status Toggle Mode Keybindings
​
Footer shows all status options with the current selection highlighted (reversed). Each option has a shortcut key.
​
| Key | Action |
|---|---|
| `w` | Set status to `wip` |
| `t` | Set status to `todo` |
| `l` | Set status to `later` |
| `b` | Set status to `blocked` |
| `c` | Set status to `completed` |
| `d` | Set status to `dead` |
| `j` / `down` / `l` / `right` / `tab` | Move status cursor right |
| `k` / `up` / `h` / `left` / `shift+tab` | Move status cursor left |
| `enter` | Confirm selection at cursor position |
| `esc` | Cancel, return to normal mode |
​
Setting a status writes to `workflow.yaml` (preserving existing fields like `created` and `slack`), then triggers the tmux lifecycle logic (create or kill session based on new status).
​
### Preview Mode Keybindings
​
| Key | Action |
|---|---|
| Standard viewport keys (j/k, up/down, pgup/pgdn, etc.) | Scroll the note viewport |
| `O` | Open in Obsidian |
| `S` | Open in Slack |
| `tab` / `shift+tab` / `esc` / `1` / `backspace` | Return focus to list (normal mode) |
| `q` | Quit |
​
### New Workflow Mode Keybindings
​
Footer shows: `New workflow: <input>  enter create  esc cancel`
​
| Key | Action |
|---|---|
| Any character | Updates name input (max 80 chars) |
| `enter` | Create workflow with entered name (same logic as `wf new`), refresh list, return to normal mode |
| `esc` | Cancel, return to normal mode |
​
### Rename Mode Keybindings
​
Footer shows: `Rename: <input>  enter confirm  esc cancel`
​
| Key | Action |
|---|---|
| Any character | Updates name input (max 80 chars, pre-filled with current slug) |
| `enter` | Execute rename, refresh list, return to normal mode |
| `esc` | Cancel, return to normal mode |
​
---
​
## Rename Logic
​
When a workflow is renamed:
​
1. The new name is slugified (same rules as creation).
2. The number is preserved; only the slug portion changes.
3. The note file inside `notes/` is renamed from `<old>.md` to `<new>.md`.
4. References to the old display name inside `.claude/CLAUDE.md` are replaced with the new name.
5. The workflow directory itself is renamed.
6. The old Obsidian symlink is removed and a new one is created pointing to `<new-dir>/notes/`.
7. If a tmux session exists, it is renamed (`tmux rename-session`).
​
---
​
## Delete Logic
​
When a workflow is deleted:
​
1. If a tmux session exists for it, it is killed.
2. The Obsidian symlink is removed.
3. The entire workflow directory is removed recursively (`os.RemoveAll`).
​
---
​
## List Rendering
​
- Workflows are sorted by number **descending** (newest first).
- Each list row shows: `NNN  <slug>  <colored-status>  <dot>` where the dot is cyan/bold if a tmux session is active, gray otherwise.
- The selected row is rendered with a white-on-gray background.
- The list is virtually scrolled: a scroll offset tracks which portion of the list is visible, and adjusts as the cursor moves.
- Names are truncated with `...` if they exceed available width.
​
### Filter Tabs
​
Displayed at the top of the list panel: `[w]ip  [o]pen  [a]ll`. The active filter is bold; inactive filters are dimmed.
​
### Filter Logic
​
| Filter | Statuses Shown |
|---|---|
| `wip` | Only `wip` |
| `open` (active) | `wip` and `blocked` |
| `all` | Everything |
​
If a search term is active, it is applied as an additional case-insensitive substring filter on the display name.
​
When a filter changes, the cursor resets to 0 and the scroll offset resets to 0.
​
---
​
## Note Preview
​
The right-side note panel renders the workflow's root note (`notes/<NNN-slug>.md`) through glamour (terminal markdown renderer) with a custom "Kitty" color theme.
​
Before rendering:
- YAML frontmatter (content between `---` delimiters at the start) is stripped.
- Wiki-style links (`[[target]]`) are converted to markdown links (`[target]()`) so glamour renders them with link styling.
​
The preview is only re-rendered when the cursor changes to a different workflow (tracked via `prevCursor`).
​
---
​
## Info Panel
​
Displays 4 attributes for the selected workflow:
​
| Field | Value |
|---|---|
| `status` | Colored status string |
| `tmux` | `active` (cyan, bold) or `none` (gray) |
| `slack` | Slack URL (blue) or `none` (gray) |
| `dir` | Workflow path with `$HOME` replaced by `~` |
​
---
​
## Tmux Integration
​
- **Session creation**: `tmux new-session -d -s <NNN-slug> -c <workflow-dir>` (detached, working directory set).
- **Session detection**: `tmux has-session -t <NNN-slug>` (exit code 0 = exists).
- **Session kill**: `tmux kill-session -t <NNN-slug>`.
- **Session rename**: `tmux rename-session -t <old-name> <new-name>`.
- **Attach on exit**: After the TUI exits, if `attachTmux` is set, the Go process replaces itself with `tmux attach-session -t <name>` via `syscall.Exec` (Unix only).
- **Split on create (`wf new` only)**: After creating the session, splits horizontally (`tmux split-window -h -b -t <session> cl`), then selects the right pane (`tmux select-pane -t <session>:0.1`). The `cl` command runs in the left pane.
​
Note: The TUI's `t` keybinding and `CreateWorkflow` (from `n` in TUI) do not perform the split-and-setup; only the `wf new` CLI command does.
​
---
​
## Obsidian Integration
​
- **Symlink**: `<OBSIDIAN_VAULT_DIR>/notes/workflows/<NNN-slug>` -> `<workflow-dir>/notes/`
- **Open**: Uses `open obsidian://open?vault=<vault-name>&file=notes/workflows/<NNN-slug>/<NNN-slug>.md` where vault name is derived as `filepath.Base(obsidianDir)` (i.e., the vault root directory name).
​
---
​
## Glamour Theme (Kitty Style)
​
Custom terminal markdown rendering theme:
​
| Element | Color | Style |
|---|---|---|
| Document / Paragraph | `#EEFFFF` | — |
| H1 | `#FFCB6B` | bold, `# ` prefix |
| H2 | `#C792EA` | bold, `## ` prefix |
| H3 | `#89DDFF` | bold, `### ` prefix |
| Other headings | `#82AAFF` | bold |
| Strong | `#FFCB6B` | bold |
| Emphasis | `#C3E88D` | italic |
| Links | `#82AAFF` | text only (URL suppressed via format `" "`) |
| Code (inline) | `#C3E88D` | — |
| Code block text | `#EEFFFF` | margin 1 |
| Code keywords | `#C792EA` | — |
| Code functions | `#82AAFF` | — |
| Code strings | `#C3E88D` | — |
| Code numbers | `#F78C6C` | — |
| Code comments | `#636261` | — |
| Horizontal rule | `#636261` | — |
​
---
​
## Platform
​
- Unix only: uses `syscall.Exec` for process replacement (the `exec_unix.go` build file).
- macOS assumed for `open` command (Obsidian URL scheme, Slack URL opening).
