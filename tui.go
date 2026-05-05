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
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
)

// ====================================================================
// Model
// ====================================================================

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
	modeHelp
)

type focusPanel int

const (
	focusList focusPanel = iota
	focusNote
)

type model struct {
	all        []Workflow
	visible    []Workflow
	cursor     int
	prevCursor int
	scroll     int

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

	attachTmux string
	cdTarget   string

	err string
}

func runTUI(chooseDir string) {
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
	mm, ok := res.(model)
	if !ok {
		return
	}
	if chooseDir != "" && mm.cdTarget != "" {
		if werr := os.WriteFile(chooseDir, []byte(mm.cdTarget), 0644); werr != nil {
			fmt.Fprintln(os.Stderr, werr)
		}
	}
	if mm.attachTmux != "" {
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

	m := model{
		mode:       modeNormal,
		focus:      focusList,
		filter:     filterWIP,
		search:     si,
		nameInput:  ni,
		viewport:   viewport.New(0, 0),
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

func (m model) selected() *Workflow {
	if m.cursor < 0 || m.cursor >= len(m.visible) {
		return nil
	}
	return &m.visible[m.cursor]
}

func (m *model) layout() {
	leftW := m.width * 40 / 100
	rightW := m.width - leftW

	infoOuter := 5 + 2 // worst-case 5 info lines + 2 border chars
	footerH := 1
	noteOuter := m.height - footerH - infoOuter
	noteInner := noteOuter - 2
	if noteInner < 1 {
		noteInner = 1
	}
	m.viewport.Width = rightW - 4
	m.viewport.Height = noteInner
}

// ====================================================================
// Note rendering (markdown via glamour)
// ====================================================================

var (
	frontmatterRe = regexp.MustCompile(`(?s)^---\n.*?\n---\n?`)
	wikiRe        = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	ansiRe        = regexp.MustCompile("\x1b\\[[0-9;]*[A-Za-z]")
)

func stripFrontmatter(s string) string { return frontmatterRe.ReplaceAllString(s, "") }
func wikiToMarkdown(s string) string   { return wikiRe.ReplaceAllString(s, "[$1]()") }
func stripANSI(s string) string        { return ansiRe.ReplaceAllString(s, "") }

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
	body := wikiToMarkdown(stripFrontmatter(string(data)))

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

// ====================================================================
// Bubbletea: Init / Update / handle*
// ====================================================================

func (m model) Init() tea.Cmd { return nil }

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
	case modeHelp:
		return m.handleHelpKey(msg)
	}
	return m.handleNormalKey(msg)
}

func (m model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?", "esc", "q":
		m.mode = modeNormal
	}
	return m, nil
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
		m.cursor, m.scroll = 0, 0
		m.applyFilter()
	case "w":
		m.filter = filterWIP
		m.cursor, m.scroll = 0, 0
		m.applyFilter()
	case "o":
		m.filter = filterOpen
		m.cursor, m.scroll = 0, 0
		m.applyFilter()
	case "]", "right":
		m.filter = nextFilter(m.filter, +1)
		m.cursor, m.scroll = 0, 0
		m.applyFilter()
	case "[", "left":
		m.filter = nextFilter(m.filter, -1)
		m.cursor, m.scroll = 0, 0
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
	case "c":
		if w := m.selected(); w != nil {
			m.cdTarget = w.Dir
			return m, tea.Quit
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
	case "?":
		m.mode = modeHelp
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
		m.cursor, m.scroll = 0, 0
		m.applyFilter()
		m.renderNote()
		return m, nil
	case "esc":
		m.search.SetValue("")
		m.search.Blur()
		m.mode = modeNormal
		m.cursor, m.scroll = 0, 0
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

// ====================================================================
// View
// ====================================================================

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	bg := m.renderBackground()
	if m.mode == modeHelp {
		return overlayCenter(bg, m.renderHelpModal(), m.width, m.height)
	}
	return bg
}

func (m model) renderBackground() string {
	leftW := m.width * 40 / 100
	rightW := m.width - leftW

	footer := m.renderFooter()
	footerH := lipgloss.Height(footer)
	bodyH := m.height - footerH

	left := m.renderLeft(leftW, bodyH)
	right := m.renderRight(rightW, bodyH)

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	return lipgloss.JoinVertical(lipgloss.Left, body, footer)
}

func (m model) renderLeft(w, h int) string {
	border := borderUnfocused
	if m.focus == focusList {
		border = borderFocused
	}
	innerW := w - 2
	innerH := h - 2

	tabs := m.renderFilterTabs()
	listH := innerH - 2
	rows := m.renderListRows(innerW, listH)

	box := border.Width(innerW).Height(innerH).Render(tabs + "\n\n" + rows)
	return titledBox(box, "[1] Workflows", m.focus == focusList)
}

func (m model) renderFilterTabs() string {
	render := func(label string, f filterMode) string {
		if m.filter == f {
			return filterActive.Render(label)
		}
		return filterInactive.Render(label)
	}
	return strings.Join([]string{
		render("[w]ip", filterWIP),
		render("[o]pen", filterOpen),
		render("[a]ll", filterAll),
	}, "  ")
}

func (m model) renderListRows(w, h int) string {
	if h < 1 {
		return ""
	}
	if len(m.visible) == 0 {
		return lipgloss.NewStyle().Foreground(colorGray).Render("(no workflows)")
	}

	scroll := m.scroll
	if m.cursor < scroll {
		scroll = m.cursor
	}
	if m.cursor >= scroll+h {
		scroll = m.cursor - h + 1
	}
	if scroll < 0 {
		scroll = 0
	}

	end := scroll + h
	if end > len(m.visible) {
		end = len(m.visible)
	}

	var lines []string
	for i := scroll; i < end; i++ {
		lines = append(lines, m.renderRow(i, w))
	}
	return strings.Join(lines, "\n")
}

func (m model) renderRow(i, w int) string {
	wf := m.visible[i]
	dot := tmuxDotInactive
	if wf.HasTmux {
		dot = tmuxDotActive
	}
	num := fmt.Sprintf("%03d", wf.Number)
	statusTxt := statusStyle(wf.Meta.Status).Render(string(wf.Meta.Status))

	statusWidth := lipgloss.Width(string(wf.Meta.Status))
	fixed := 3 + 1 + 1 + statusWidth + 1 + 1 // num + sep + sep + status + sep + dot
	slugW := w - fixed
	if slugW < 5 {
		slugW = 5
	}
	slug := padRight(truncate(wf.Slug, slugW), slugW)

	row := fmt.Sprintf("%s %s %s %s", num, slug, statusTxt, dot)
	if i == m.cursor {
		row = selectedRowStyle.Render(stripANSI(row))
	}
	return row
}

func (m model) renderRight(w, h int) string {
	innerW := w - 2
	info := m.renderInfo(innerW)
	infoH := lipgloss.Height(info)
	noteH := h - infoH
	note := m.renderNoteBox(innerW, noteH)
	return lipgloss.JoinVertical(lipgloss.Left, info, note)
}

func (m model) renderInfo(innerW int) string {
	w := m.selected()
	var lines []string
	if w != nil {
		// "  label    " = 2 indent + 6 label + 4 gap = 12; box has 2 border chars
		const labelW = 12
		valW := innerW - 2 - labelW
		if valW < 5 {
			valW = 5
		}
		hangingIndent := strings.Repeat(" ", labelW)
		wrap := func(label, value string, style lipgloss.Style) {
			rows := wrapPlain(value, valW)
			for i, row := range rows {
				prefix := hangingIndent
				if i == 0 {
					prefix = label
				}
				lines = append(lines, prefix+style.Render(row))
			}
		}

		wrap("  name    ", w.Name, lipgloss.NewStyle().Foreground(lipgloss.Color("#C792EA")))
		wrap("  status  ", string(w.Meta.Status), statusStyle(w.Meta.Status))

		tmuxStyle := lipgloss.NewStyle().Foreground(colorGray)
		tmuxVal := "off"
		if w.HasTmux {
			tmuxStyle = lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
			tmuxVal = "on"
		}
		wrap("  tmux    ", tmuxVal, tmuxStyle)

		if w.Meta.Slack != "" {
			wrap("  slack   ", w.Meta.Slack, lipgloss.NewStyle().Foreground(colorBlue))
		}
	}
	height := len(lines)
	if height < 1 {
		height = 1
	}
	box := borderUnfocused.Width(innerW).Height(height).Render(strings.Join(lines, "\n"))
	return titledBox(box, "Info", false)
}

// wrapPlain breaks s into chunks of at most n runes (hard wrap, no word-break logic).
func wrapPlain(s string, n int) []string {
	if n <= 0 {
		return []string{s}
	}
	r := []rune(s)
	if len(r) <= n {
		return []string{s}
	}
	var out []string
	for i := 0; i < len(r); i += n {
		end := i + n
		if end > len(r) {
			end = len(r)
		}
		out = append(out, string(r[i:end]))
	}
	return out
}

func (m model) renderNoteBox(innerW, noteH int) string {
	border := borderUnfocused
	if m.focus == focusNote {
		border = borderFocused
	}
	innerH := noteH - 2
	if innerH < 1 {
		innerH = 1
	}
	box := border.Width(innerW).Height(innerH).Render(m.viewport.View())
	return titledBox(box, "[2] Note", m.focus == focusNote)
}

// titledBox replaces the top border of an already-rendered box with one that
// embeds a colored title, keeping border characters in a single color.
func titledBox(box, title string, focused bool) string {
	borderColor := colorGray
	if focused {
		borderColor = colorBlue
	}
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	titleStyle := lipgloss.NewStyle().Foreground(borderColor).Bold(true)

	lines := strings.Split(box, "\n")
	if len(lines) == 0 {
		return box
	}
	first := stripANSI(lines[0])
	totalW := lipgloss.Width(first)
	if totalW < 4 {
		return box
	}
	titleText := " " + title + " "
	leadingDashes := 1
	innerW := totalW - 2
	titleW := lipgloss.Width(titleText)
	if titleW+leadingDashes >= innerW {
		return box
	}
	trailingDashes := innerW - leadingDashes - titleW

	lines[0] = borderStyle.Render("╭"+strings.Repeat("─", leadingDashes)) +
		titleStyle.Render(titleText) +
		borderStyle.Render(strings.Repeat("─", trailingDashes)+"╮")
	return strings.Join(lines, "\n")
}

func (m model) renderHelpModal() string {
	type binding struct{ k, label string }
	type section struct {
		title    string
		bindings []binding
	}
	sections := []section{
		{"Navigation", []binding{
			{"↑↓", "move"}, {"g/G", "top/bottom"}, {"[ ]", "cycle filter"},
			{"a", "all"}, {"w", "wip"}, {"o", "open"},
		}},
		{"Workflow", []binding{
			{"enter", "preview note"}, {"c", "cd into dir"}, {"t", "toggle tmux"},
			{"s", "set status"}, {"n", "new"}, {"r", "rename"}, {"d", "delete"},
		}},
		{"External", []binding{
			{"O", "open in obsidian"}, {"S", "open slack"}, {"/", "search"}, {"R", "refresh"},
		}},
		{"Help", []binding{
			{"?", "toggle this help"}, {"esc", "close"}, {"q", "quit"},
		}},
	}

	titleStyle := lipgloss.NewStyle().Foreground(colorYellow).Bold(true)
	headerStyle := lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
	keyCol := lipgloss.NewStyle().Foreground(colorWhite).Bold(true)
	labelCol := lipgloss.NewStyle().Foreground(colorGray)

	renderSection := func(sec section) string {
		lines := []string{headerStyle.Render(sec.title)}
		for _, b := range sec.bindings {
			lines = append(lines, "  "+keyCol.Render(padRight(b.k, 8))+labelCol.Render(b.label))
		}
		return strings.Join(lines, "\n")
	}

	colStyle := lipgloss.NewStyle().PaddingRight(4)
	left := lipgloss.JoinVertical(lipgloss.Left,
		colStyle.Render(renderSection(sections[0])),
		"",
		colStyle.Render(renderSection(sections[2])),
	)
	right := lipgloss.JoinVertical(lipgloss.Left,
		renderSection(sections[1]),
		"",
		renderSection(sections[3]),
	)
	cols := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	body := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("wf — keybindings"),
		"",
		cols,
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBlue).
		Padding(1, 2).
		Render(body)
}

// overlayCenter composites fg on top of bg, centered within (w, h).
// Both inputs may contain ANSI styling. Cells of bg under fg are replaced.
func overlayCenter(bg, fg string, w, h int) string {
	fgW := lipgloss.Width(fg)
	fgH := lipgloss.Height(fg)
	x := (w - fgW) / 2
	y := (h - fgH) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")

	for i, fline := range fgLines {
		row := y + i
		if row < 0 || row >= len(bgLines) {
			continue
		}
		bgLines[row] = spliceLine(bgLines[row], fline, x)
	}
	return strings.Join(bgLines, "\n")
}

// spliceLine replaces visible columns [x, x+width(fg)) of bg with fg.
// Pads with spaces if bg is shorter than x. Preserves ANSI on either side.
func spliceLine(bg, fg string, x int) string {
	bgW := lipgloss.Width(bg)
	fgW := lipgloss.Width(fg)

	if bgW < x {
		bg = bg + strings.Repeat(" ", x-bgW)
		bgW = x
	}

	left := xansi.Truncate(bg, x, "")
	var right string
	if bgW > x+fgW {
		right = xansi.TruncateLeft(bg, x+fgW, "")
	}
	return left + fg + right
}

func (m model) renderFooter() string {
	switch m.mode {
	case modeSearch:
		return "/ " + m.search.View() + footerStyle.Render("   enter apply  esc cancel")
	case modeConfirmDelete:
		w := m.selected()
		name := ""
		if w != nil {
			name = w.Name
		}
		return fmt.Sprintf("Delete %s? %s/%s", name, keyStyle.Render("y"), keyStyle.Render("n"))
	case modeStatusToggle:
		return m.renderStatusFooter()
	case modeNewWorkflow:
		return "New workflow: " + m.nameInput.View() + footerStyle.Render("   enter create  esc cancel")
	case modeRename:
		return "Rename: " + m.nameInput.View() + footerStyle.Render("   enter confirm  esc cancel")
	case modePreview:
		return footerStyle.Render("note preview  ") +
			keyStyle.Render("j/k") + footerStyle.Render(" scroll  ") +
			keyStyle.Render("O") + footerStyle.Render(" obsidian  ") +
			keyStyle.Render("S") + footerStyle.Render(" slack  ") +
			keyStyle.Render("tab/esc") + footerStyle.Render(" back  ") +
			keyStyle.Render("q") + footerStyle.Render(" quit")
	}
	hints := []struct{ k, label string }{
		{"↑↓", "move"}, {"enter", "preview"}, {"c", "cd"}, {"?", "help"}, {"q", "quit"},
	}
	parts := make([]string, 0, len(hints))
	for _, h := range hints {
		parts = append(parts, keyStyle.Render(h.k)+" "+footerStyle.Render(h.label))
	}
	out := strings.Join(parts, "  ")
	if m.err != "" {
		out += "\n" + lipgloss.NewStyle().Foreground(colorRed).Render(m.err)
	}
	return out
}

func (m model) renderStatusFooter() string {
	parts := []string{footerStyle.Render("status:")}
	for i, s := range allStatuses {
		st := statusStyle(s)
		if i == m.statusCursor {
			st = st.Reverse(true)
		}
		parts = append(parts, st.Render(string(s)))
	}
	parts = append(parts, footerStyle.Render(" enter set  esc cancel"))
	return strings.Join(parts, "  ")
}

// ====================================================================
// Small render helpers
// ====================================================================

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len([]rune(s)) <= n {
		return s
	}
	if n <= 3 {
		return strings.Repeat(".", n)
	}
	r := []rune(s)
	return string(r[:n-3]) + "..."
}

func padRight(s string, n int) string {
	w := lipgloss.Width(s)
	if w >= n {
		return s
	}
	return s + strings.Repeat(" ", n-w)
}

// ====================================================================
// Styles + glamour theme
// ====================================================================

var (
	colorYellow  = lipgloss.Color("3")
	colorBlue    = lipgloss.Color("4")
	colorMagenta = lipgloss.Color("5")
	colorRed     = lipgloss.Color("1")
	colorGreen   = lipgloss.Color("2")
	colorGray    = lipgloss.Color("8")
	colorCyan    = lipgloss.Color("6")
	colorWhite   = lipgloss.Color("15")

	borderFocused   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorBlue)
	borderUnfocused = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorGray)

	selectedRowStyle = lipgloss.NewStyle().Background(colorGray).Foreground(colorWhite)
	tmuxDotActive    = lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render("●")
	tmuxDotInactive  = lipgloss.NewStyle().Foreground(colorGray).Render("●")

	filterActive   = lipgloss.NewStyle().Bold(true)
	filterInactive = lipgloss.NewStyle().Foreground(colorGray)

	footerStyle = lipgloss.NewStyle().Foreground(colorGray)
	keyStyle    = lipgloss.NewStyle().Foreground(colorWhite).Bold(true)
)

func statusStyle(s Status) lipgloss.Style {
	switch s {
	case StatusWIP:
		return lipgloss.NewStyle().Foreground(colorYellow)
	case StatusTodo:
		return lipgloss.NewStyle().Foreground(colorBlue)
	case StatusLater:
		return lipgloss.NewStyle().Foreground(colorMagenta)
	case StatusBlocked:
		return lipgloss.NewStyle().Foreground(colorRed).Bold(true)
	case StatusCompleted:
		return lipgloss.NewStyle().Foreground(colorGreen)
	case StatusDead, StatusUnknown:
		return lipgloss.NewStyle().Foreground(colorGray)
	}
	return lipgloss.NewStyle()
}

func glamourKittyStyle() ansi.StyleConfig {
	str := func(s string) *string { return &s }
	uint1 := func(u uint) *uint { return &u }
	bold := true
	italic := true

	return ansi.StyleConfig{
		Document:   ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: str("#EEFFFF")}},
		BlockQuote: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: str("#EEFFFF")}, Indent: uint1(1)},
		Paragraph:  ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: str("#EEFFFF")}},
		List:       ansi.StyleList{StyleBlock: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: str("#EEFFFF")}}},
		Heading:    ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: str("#82AAFF"), Bold: &bold}},
		H1:         ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "# ", Color: str("#FFCB6B"), Bold: &bold}},
		H2:         ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "## ", Color: str("#C792EA"), Bold: &bold}},
		H3:         ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "### ", Color: str("#89DDFF"), Bold: &bold}},
		H4:         ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: str("#82AAFF"), Bold: &bold}},
		H5:         ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: str("#82AAFF"), Bold: &bold}},
		H6:         ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: str("#82AAFF"), Bold: &bold}},
		Strong:     ansi.StylePrimitive{Color: str("#FFCB6B"), Bold: &bold},
		Emph:       ansi.StylePrimitive{Color: str("#C3E88D"), Italic: &italic},
		HorizontalRule: ansi.StylePrimitive{Color: str("#636261"), Format: "---"},
		Item:        ansi.StylePrimitive{Color: str("#EEFFFF")},
		Enumeration: ansi.StylePrimitive{Color: str("#EEFFFF")},
		Task: ansi.StyleTask{
			StylePrimitive: ansi.StylePrimitive{Color: str("#EEFFFF")},
			Ticked:         "[x] ",
			Unticked:       "[ ] ",
		},
		Link:      ansi.StylePrimitive{Color: str("#82AAFF"), Format: " "},
		LinkText:  ansi.StylePrimitive{Color: str("#82AAFF")},
		Image:     ansi.StylePrimitive{Color: str("#82AAFF")},
		ImageText: ansi.StylePrimitive{Color: str("#82AAFF"), Format: "{{.text}}"},
		Code:      ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: str("#C3E88D")}},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{Color: str("#EEFFFF")},
				Margin:         uint1(1),
			},
			Chroma: &ansi.Chroma{
				Text:          ansi.StylePrimitive{Color: str("#EEFFFF")},
				Keyword:       ansi.StylePrimitive{Color: str("#C792EA")},
				NameFunction:  ansi.StylePrimitive{Color: str("#82AAFF")},
				LiteralString: ansi.StylePrimitive{Color: str("#C3E88D")},
				LiteralNumber: ansi.StylePrimitive{Color: str("#F78C6C")},
				Comment:       ansi.StylePrimitive{Color: str("#636261")},
			},
		},
		Table: ansi.StyleTable{
			StyleBlock: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: str("#EEFFFF")}},
		},
		DefinitionDescription: ansi.StylePrimitive{BlockPrefix: "\n* "},
	}
}

// openObsidian opens the workflow's note in the Obsidian app via URL scheme.
func openObsidian(w Workflow) {
	obsidian, err := ObsidianDir()
	if err != nil {
		return
	}
	vault := filepath.Base(obsidian)
	url := fmt.Sprintf("obsidian://open?vault=%s&file=%s/%s/%s.md", vault, vaultSubpath(), w.Name, w.Name)
	_ = exec.Command("open", url).Run()
}
