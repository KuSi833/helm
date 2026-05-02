package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

type filterMode int

const (
	filterWIP filterMode = iota
	filterOpen
	filterAll
)

var filterCycle = []filterMode{filterWIP, filterOpen, filterAll}

type tuiMode int

const (
	modeNormal tuiMode = iota
	modeSearch
	modeConfirmDelete
	modeStatusToggle
	modePreview
	modeNewWorkflow
	modeRename
)

type focusPanel int

const (
	focusList focusPanel = iota
	focusNote
)

type model struct {
	all       []Workflow
	visible   []Workflow
	cursor    int
	prevCursor int
	scroll    int

	width  int
	height int

	mode   tuiMode
	focus  focusPanel
	filter filterMode

	search    textinput.Model
	nameInput textinput.Model

	statusCursor int

	viewport     viewport.Model
	renderedNote string

	attachTmux string // session name to attach on exit

	err string
}

func runTUI() {
	if _, err := WorkflowsDir(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := ObsidianDir(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	m := initialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	res, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if mm, ok := res.(model); ok && mm.attachTmux != "" {
		_ = execTmuxAttach(mm.attachTmux)
	}
}

func initialModel() model {
	si := textinput.New()
	si.Prompt = ""
	si.CharLimit = 80

	ni := textinput.New()
	ni.Prompt = ""
	ni.CharLimit = 80

	vp := viewport.New(0, 0)

	m := model{
		mode:      modeNormal,
		focus:     focusList,
		filter:    filterWIP,
		search:    si,
		nameInput: ni,
		viewport:  vp,
		prevCursor: -1,
	}
	m.refresh()
	return m
}

func (m *model) refresh() {
	wfs, err := ScanWorkflows()
	if err != nil {
		m.err = err.Error()
		return
	}
	m.err = ""
	m.all = wfs
	m.applyFilter()
}

func (m *model) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(m.search.Value()))
	var out []Workflow
	for _, w := range m.all {
		switch m.filter {
		case filterWIP:
			if w.Meta.Status != StatusWIP {
				continue
			}
		case filterOpen:
			if !w.Meta.Status.Active() {
				continue
			}
		}
		if q != "" && !strings.Contains(strings.ToLower(w.Name), q) {
			continue
		}
		out = append(out, w)
	}
	m.visible = out
	if m.cursor >= len(out) {
		m.cursor = 0
	}
	if len(out) == 0 {
		m.cursor = 0
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) selected() *Workflow {
	if m.cursor < 0 || m.cursor >= len(m.visible) {
		return nil
	}
	return &m.visible[m.cursor]
}

func (m *model) renderNote() {
	w := m.selected()
	if w == nil {
		m.renderedNote = ""
		m.viewport.SetContent("")
		return
	}
	notePath := filepath.Join(w.Dir, "notes", w.Name+".md")
	data, err := os.ReadFile(notePath)
	if err != nil {
		m.renderedNote = ""
		m.viewport.SetContent("")
		return
	}
	body := stripFrontmatter(string(data))
	body = wikiToMarkdown(body)

	width := m.viewport.Width
	if width <= 0 {
		width = 60
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(glamourKittyStyle()),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		m.renderedNote = body
		m.viewport.SetContent(body)
		return
	}
	out, err := r.Render(body)
	if err != nil {
		out = body
	}
	m.renderedNote = out
	m.viewport.SetContent(out)
}

var frontmatterRe = regexp.MustCompile(`(?s)^---\n.*?\n---\n?`)

func stripFrontmatter(s string) string {
	return frontmatterRe.ReplaceAllString(s, "")
}

var wikiRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

func wikiToMarkdown(s string) string {
	return wikiRe.ReplaceAllString(s, "[$1]()")
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		m.renderNote()
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *model) layout() {
	leftW := m.width * 40 / 100
	rightW := m.width - leftW

	// Footer is 1 line; body fills the rest. Right column = info box + note box.
	// Info box outer height = N info lines + 2 borders. Worst case is 5 lines
	// (name, status, tmux, slack, dir) so viewport stays stable when slack toggles.
	infoOuter := 5 + 2
	footerH := 1
	noteOuter := m.height - footerH - infoOuter
	noteInner := noteOuter - 2 // borders
	if noteInner < 1 {
		noteInner = 1
	}
	m.viewport.Width = rightW - 4
	m.viewport.Height = noteInner
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeSearch:
		return m.handleSearchKey(msg)
	case modeConfirmDelete:
		return m.handleDeleteKey(msg)
	case modeStatusToggle:
		return m.handleStatusKey(msg)
	case modePreview:
		return m.handlePreviewKey(msg)
	case modeNewWorkflow:
		return m.handleNewKey(msg)
	case modeRename:
		return m.handleRenameKey(msg)
	}
	return m.handleNormalKey(msg)
}

func (m model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		if m.cursor < len(m.visible)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "g":
		m.cursor = 0
	case "G":
		if len(m.visible) > 0 {
			m.cursor = len(m.visible) - 1
		}
	case "a":
		m.filter = filterAll
		m.cursor = 0
		m.scroll = 0
		m.applyFilter()
	case "w":
		m.filter = filterWIP
		m.cursor = 0
		m.scroll = 0
		m.applyFilter()
	case "o":
		m.filter = filterOpen
		m.cursor = 0
		m.scroll = 0
		m.applyFilter()
	case "]", "right":
		m.filter = nextFilter(m.filter, +1)
		m.cursor = 0
		m.scroll = 0
		m.applyFilter()
	case "[", "left":
		m.filter = nextFilter(m.filter, -1)
		m.cursor = 0
		m.scroll = 0
		m.applyFilter()
	case "/":
		m.mode = modeSearch
		m.search.Focus()
		return m, textinput.Blink
	case "s":
		if m.selected() != nil {
			m.mode = modeStatusToggle
			m.statusCursor = statusIndex(m.selected().Meta.Status)
		}
	case "d":
		if m.selected() != nil {
			m.mode = modeConfirmDelete
		}
	case "t":
		if w := m.selected(); w != nil {
			if err := ToggleTmux(w); err != nil {
				m.err = err.Error()
			}
			m.refresh()
		}
	case "n":
		m.mode = modeNewWorkflow
		m.nameInput.SetValue("")
		m.nameInput.Focus()
		return m, textinput.Blink
	case "r":
		if w := m.selected(); w != nil {
			m.mode = modeRename
			m.nameInput.SetValue(w.Slug)
			m.nameInput.CursorEnd()
			m.nameInput.Focus()
			return m, textinput.Blink
		}
	case "R":
		m.refresh()
	case "O":
		if w := m.selected(); w != nil {
			openObsidian(*w)
		}
	case "S":
		if w := m.selected(); w != nil && w.Meta.Slack != "" {
			_ = exec.Command("open", w.Meta.Slack).Run()
		}
	case "2", "enter", "tab", "shift+tab":
		m.focus = focusNote
		m.mode = modePreview
	case "1":
		// no-op
	}
	if m.cursor != m.prevCursor {
		m.prevCursor = m.cursor
		m.renderNote()
	}
	return m, nil
}

func nextFilter(f filterMode, dir int) filterMode {
	idx := 0
	for i, x := range filterCycle {
		if x == f {
			idx = i
			break
		}
	}
	idx = (idx + dir + len(filterCycle)) % len(filterCycle)
	return filterCycle[idx]
}

func statusIndex(s Status) int {
	for i, x := range allStatuses {
		if x == s {
			return i
		}
	}
	return 0
}

func (m model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.mode = modeNormal
		m.search.Blur()
		m.cursor = 0
		m.scroll = 0
		m.applyFilter()
		m.renderNote()
		return m, nil
	case "esc":
		m.search.SetValue("")
		m.search.Blur()
		m.mode = modeNormal
		m.cursor = 0
		m.scroll = 0
		m.applyFilter()
		m.renderNote()
		return m, nil
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	m.applyFilter()
	if m.cursor != m.prevCursor {
		m.prevCursor = m.cursor
		m.renderNote()
	}
	return m, cmd
}

func (m model) handleDeleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		if w := m.selected(); w != nil {
			if err := DeleteWorkflow(*w); err != nil {
				m.err = err.Error()
			}
		}
		m.mode = modeNormal
		m.refresh()
		m.renderNote()
	case "n", "esc":
		m.mode = modeNormal
	}
	return m, nil
}

func (m model) handleStatusKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	apply := func(s Status) {
		if w := m.selected(); w != nil {
			if err := SetStatus(w, s); err != nil {
				m.err = err.Error()
			}
		}
		m.mode = modeNormal
		m.refresh()
	}
	switch msg.String() {
	case "w":
		apply(StatusWIP)
	case "t":
		apply(StatusTodo)
	case "l":
		apply(StatusLater)
	case "b":
		apply(StatusBlocked)
	case "c":
		apply(StatusCompleted)
	case "d":
		apply(StatusDead)
	case "j", "down", "right", "tab":
		m.statusCursor = (m.statusCursor + 1) % len(allStatuses)
	case "k", "up", "h", "left", "shift+tab":
		m.statusCursor = (m.statusCursor - 1 + len(allStatuses)) % len(allStatuses)
	case "enter":
		apply(allStatuses[m.statusCursor])
	case "esc":
		m.mode = modeNormal
	}
	return m, nil
}

func (m model) handlePreviewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "shift+tab", "esc", "1", "backspace":
		m.mode = modeNormal
		m.focus = focusList
		return m, nil
	case "q", "ctrl+c":
		return m, tea.Quit
	case "O":
		if w := m.selected(); w != nil {
			openObsidian(*w)
		}
		return m, nil
	case "S":
		if w := m.selected(); w != nil && w.Meta.Slack != "" {
			_ = exec.Command("open", w.Meta.Slack).Run()
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m model) handleNewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(m.nameInput.Value())
		if name != "" {
			if _, err := CreateWorkflow(name, false); err != nil {
				m.err = err.Error()
			}
		}
		m.nameInput.SetValue("")
		m.nameInput.Blur()
		m.mode = modeNormal
		m.refresh()
		m.renderNote()
		return m, nil
	case "esc":
		m.nameInput.SetValue("")
		m.nameInput.Blur()
		m.mode = modeNormal
		return m, nil
	}
	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

func (m model) handleRenameKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(m.nameInput.Value())
		if w := m.selected(); w != nil && name != "" {
			if _, err := RenameWorkflow(*w, name); err != nil {
				m.err = err.Error()
			}
		}
		m.nameInput.SetValue("")
		m.nameInput.Blur()
		m.mode = modeNormal
		m.refresh()
		m.renderNote()
		return m, nil
	case "esc":
		m.nameInput.SetValue("")
		m.nameInput.Blur()
		m.mode = modeNormal
		return m, nil
	}
	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

func openObsidian(w Workflow) {
	obsidian, err := ObsidianDir()
	if err != nil {
		return
	}
	vault := filepath.Base(filepath.Dir(filepath.Dir(obsidian)))
	url := fmt.Sprintf("obsidian://open?vault=%s&file=Archive/Workflows/%s/%s.md", vault, w.Name, w.Name)
	_ = exec.Command("open", url).Run()
}
