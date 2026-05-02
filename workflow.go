package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Status string

const (
	StatusWIP       Status = "wip"
	StatusTodo      Status = "todo"
	StatusLater     Status = "later"
	StatusBlocked   Status = "blocked"
	StatusCompleted Status = "completed"
	StatusDead      Status = "dead"
	StatusUnknown   Status = "unknown"
)

var allStatuses = []Status{StatusWIP, StatusTodo, StatusLater, StatusBlocked, StatusCompleted, StatusDead}

func (s Status) Active() bool {
	return s == StatusWIP || s == StatusBlocked
}

func (s Status) NeedsTmux() bool {
	return s == StatusWIP || s == StatusTodo || s == StatusBlocked
}

type Meta struct {
	Status  Status `yaml:"status"`
	Created string `yaml:"created"`
	Slack   string `yaml:"slack"`
}

type Workflow struct {
	Number  int
	Slug    string
	Name    string // NNN-slug
	Dir     string
	Meta    Meta
	HasTmux bool
}

var dirRe = regexp.MustCompile(`^(\d{3})-(.+)$`)

func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	out := b.String()
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	out = strings.Trim(out, "-")
	return out
}

func envOrDie(name string) (string, error) {
	v := os.Getenv(name)
	if v == "" {
		return "", fmt.Errorf("%s is not set", name)
	}
	return v, nil
}

func WorkflowsDir() (string, error)  { return envOrDie("WORKFLOWS_DIR") }
func ObsidianDir() (string, error)   { return envOrDie("OBSIDIAN_VAULT_DIR") }

func obsidianLinkPath(obsidian, name string) string {
	return filepath.Join(obsidian, "Archive", "Workflows", name)
}

func loadMeta(dir string) Meta {
	m := Meta{Status: StatusUnknown}
	data, err := os.ReadFile(filepath.Join(dir, "workflow.yaml"))
	if err != nil {
		return m
	}
	var parsed Meta
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return m
	}
	if parsed.Status == "" {
		parsed.Status = StatusUnknown
	}
	return parsed
}

func writeMeta(dir string, m Meta) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "workflow.yaml"), data, 0644)
}

func ScanWorkflows() ([]Workflow, error) {
	root, err := WorkflowsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Workflow
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		m := dirRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		num, _ := strconv.Atoi(m[1])
		dir := filepath.Join(root, e.Name())
		wf := Workflow{
			Number:  num,
			Slug:    m[2],
			Name:    e.Name(),
			Dir:     dir,
			Meta:    loadMeta(dir),
			HasTmux: tmuxHasSession(e.Name()),
		}
		out = append(out, wf)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Number > out[j].Number })
	return out, nil
}

func nextNumber() (int, error) {
	wfs, err := ScanWorkflows()
	if err != nil {
		return 0, err
	}
	max := 0
	for _, w := range wfs {
		if w.Number > max {
			max = w.Number
		}
	}
	return max + 1, nil
}

const claudeMD = `# CLAUDE.md

This directory is a workflow. Treat ` + "`notes/" + `%[1]s` + ".md`" + ` as the index for this workflow.

When the user asks you to write notes about a specific topic, create a separate file
in ` + "`notes/`" + ` (e.g., ` + "`notes/investigation.md`" + `, ` + "`notes/decisions.md`" + `) and link to it
from the index using wiki-style links: ` + "`[[topic]]`" + `.

Keep the root note as a high-level table of contents, not a dumping ground.
`

// CreateWorkflow makes the directory, files, symlink, and tmux session.
// If split is true, also splits the tmux window for `wf new`.
func CreateWorkflow(name string, split bool) (*Workflow, error) {
	root, err := WorkflowsDir()
	if err != nil {
		return nil, err
	}
	obsidian, err := ObsidianDir()
	if err != nil {
		return nil, err
	}
	slug := slugify(name)
	if slug == "" {
		return nil, fmt.Errorf("name produces empty slug")
	}
	num, err := nextNumber()
	if err != nil {
		return nil, err
	}
	dirName := fmt.Sprintf("%03d-%s", num, slug)
	dir := filepath.Join(root, dirName)

	if err := os.MkdirAll(filepath.Join(dir, "notes"), 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0755); err != nil {
		return nil, err
	}

	meta := Meta{Status: StatusWIP, Created: time.Now().Format("2006-01-02")}
	if err := writeMeta(dir, meta); err != nil {
		return nil, err
	}

	notePath := filepath.Join(dir, "notes", dirName+".md")
	if err := os.WriteFile(notePath, []byte("# "+dirName+"\n"), 0644); err != nil {
		return nil, err
	}

	claudePath := filepath.Join(dir, ".claude", "CLAUDE.md")
	if err := os.WriteFile(claudePath, []byte(fmt.Sprintf(claudeMD, dirName)), 0644); err != nil {
		return nil, err
	}

	linkDir := filepath.Join(obsidian, "Archive", "Workflows")
	if err := os.MkdirAll(linkDir, 0755); err != nil {
		return nil, err
	}
	linkPath := obsidianLinkPath(obsidian, dirName)
	_ = os.Remove(linkPath)
	if err := os.Symlink(filepath.Join(dir, "notes"), linkPath); err != nil {
		return nil, err
	}

	if err := tmuxNewSession(dirName, dir); err != nil {
		return nil, err
	}

	if split {
		_ = exec.Command("tmux", "split-window", "-h", "-b", "-t", dirName, "cl").Run()
		_ = exec.Command("tmux", "select-pane", "-t", dirName+":0.1").Run()
	}

	return &Workflow{
		Number:  num,
		Slug:    slug,
		Name:    dirName,
		Dir:     dir,
		Meta:    meta,
		HasTmux: true,
	}, nil
}

func DeleteWorkflow(w Workflow) error {
	obsidian, err := ObsidianDir()
	if err != nil {
		return err
	}
	if w.HasTmux {
		_ = tmuxKillSession(w.Name)
	}
	_ = os.Remove(obsidianLinkPath(obsidian, w.Name))
	return os.RemoveAll(w.Dir)
}

func RenameWorkflow(w Workflow, newName string) (*Workflow, error) {
	root, err := WorkflowsDir()
	if err != nil {
		return nil, err
	}
	obsidian, err := ObsidianDir()
	if err != nil {
		return nil, err
	}
	newSlug := slugify(newName)
	if newSlug == "" {
		return nil, fmt.Errorf("name produces empty slug")
	}
	if newSlug == w.Slug {
		copy := w
		return &copy, nil
	}
	newDirName := fmt.Sprintf("%03d-%s", w.Number, newSlug)
	newDir := filepath.Join(root, newDirName)

	// Rename note file inside notes/ first.
	oldNote := filepath.Join(w.Dir, "notes", w.Name+".md")
	newNote := filepath.Join(w.Dir, "notes", newDirName+".md")
	if _, err := os.Stat(oldNote); err == nil {
		if err := os.Rename(oldNote, newNote); err != nil {
			return nil, err
		}
	}

	// Update CLAUDE.md references.
	claudePath := filepath.Join(w.Dir, ".claude", "CLAUDE.md")
	if data, err := os.ReadFile(claudePath); err == nil {
		updated := strings.ReplaceAll(string(data), w.Name, newDirName)
		_ = os.WriteFile(claudePath, []byte(updated), 0644)
	}

	// Rename workflow directory.
	if err := os.Rename(w.Dir, newDir); err != nil {
		return nil, err
	}

	// Recreate Obsidian symlink.
	_ = os.Remove(obsidianLinkPath(obsidian, w.Name))
	_ = os.Symlink(filepath.Join(newDir, "notes"), obsidianLinkPath(obsidian, newDirName))

	// Rename tmux session.
	if w.HasTmux {
		_ = exec.Command("tmux", "rename-session", "-t", w.Name, newDirName).Run()
	}

	out := w
	out.Slug = newSlug
	out.Name = newDirName
	out.Dir = newDir
	return &out, nil
}

// SetStatus writes the new status, preserving other fields, and applies tmux lifecycle.
func SetStatus(w *Workflow, s Status) error {
	w.Meta.Status = s
	if err := writeMeta(w.Dir, w.Meta); err != nil {
		return err
	}
	hasTmux := tmuxHasSession(w.Name)
	if s.NeedsTmux() && !hasTmux {
		if err := tmuxNewSession(w.Name, w.Dir); err != nil {
			return err
		}
		w.HasTmux = true
	} else if !s.NeedsTmux() && hasTmux {
		_ = tmuxKillSession(w.Name)
		w.HasTmux = false
	} else {
		w.HasTmux = hasTmux
	}
	return nil
}

// ToggleTmux creates or kills the session for w.
func ToggleTmux(w *Workflow) error {
	if tmuxHasSession(w.Name) {
		if err := tmuxKillSession(w.Name); err != nil {
			return err
		}
		w.HasTmux = false
		return nil
	}
	if err := tmuxNewSession(w.Name, w.Dir); err != nil {
		return err
	}
	w.HasTmux = true
	return nil
}

func tmuxHasSession(name string) bool {
	return exec.Command("tmux", "has-session", "-t", name).Run() == nil
}

func tmuxNewSession(name, dir string) error {
	return exec.Command("tmux", "new-session", "-d", "-s", name, "-c", dir).Run()
}

func tmuxKillSession(name string) error {
	return exec.Command("tmux", "kill-session", "-t", name).Run()
}
