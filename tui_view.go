package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
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
	listH := innerH - 2 // tabs + blank
	rows := m.renderListRows(innerW, listH)

	content := tabs + "\n\n" + rows
	box := border.Width(innerW).Height(innerH).Render(content)
	return titledBox(box, "[1] Workflows", innerW, m.focus == focusList)
}

func (m model) renderFilterTabs() string {
	parts := []string{}
	render := func(label string, f filterMode) string {
		if m.filter == f {
			return filterActive.Render(label)
		}
		return filterInactive.Render(label)
	}
	parts = append(parts, render("[w]ip", filterWIP))
	parts = append(parts, render("[o]pen", filterOpen))
	parts = append(parts, render("[a]ll", filterAll))
	return strings.Join(parts, "  ")
}

func (m model) renderListRows(w, h int) string {
	if h < 1 {
		return ""
	}
	if len(m.visible) == 0 {
		return lipgloss.NewStyle().Foreground(colorGray).Render("(no workflows)")
	}

	// Adjust scroll so cursor is visible.
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

	// budget: num(3) + 1sp + slug + 1sp + status + 1sp + dot(1)
	statusWidth := lipgloss.Width(string(wf.Meta.Status))
	dotWidth := 1
	fixed := 3 + 1 + 1 + statusWidth + 1 + dotWidth
	slugW := w - fixed
	if slugW < 5 {
		slugW = 5
	}
	slug := truncate(wf.Slug, slugW)
	slug = padRight(slug, slugW)

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
		nameTxt := lipgloss.NewStyle().Foreground(lipgloss.Color("#C792EA")).Render(w.Name)
		lines = append(lines, "  name    "+nameTxt)

		statusTxt := statusStyle(w.Meta.Status).Render(string(w.Meta.Status))
		lines = append(lines, "  status  "+statusTxt)

		tmuxTxt := lipgloss.NewStyle().Foreground(colorGray).Render("off")
		if w.HasTmux {
			tmuxTxt = lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render("on")
		}
		lines = append(lines, "  tmux    "+tmuxTxt)

		if w.Meta.Slack != "" {
			slackTxt := lipgloss.NewStyle().Foreground(colorBlue).Render(w.Meta.Slack)
			lines = append(lines, "  slack   "+slackTxt)
		}

		dirTxt := lipgloss.NewStyle().Foreground(colorGray).Render(tildeAbbrev(w.Dir))
		lines = append(lines, "  dir     "+dirTxt)
	}
	content := strings.Join(lines, "\n")
	height := len(lines)
	if height < 1 {
		height = 1
	}
	box := borderUnfocused.Width(innerW).Height(height).Render(content)
	return titledBox(box, "Info", innerW, false)
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
	return titledBox(box, "[2] Note", innerW, m.focus == focusNote)
}

// titledBox replaces the top border of an already-rendered box with one that
// embeds the title, keeping all border characters in the same color.
func titledBox(box, title string, _ int, focused bool) string {
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
	// Top border is: ╭ ─ ─ ... ─ ╮
	// We rebuild as:  ╭ ─ <title> ─...─ ╮
	titleText := " " + title + " "
	// Available dashes (excluding the two corners and one leading dash for
	// breathing room, plus the title width).
	leadingDashes := 1
	innerW := totalW - 2 // strip corners
	titleW := lipgloss.Width(titleText)
	if titleW+leadingDashes >= innerW {
		// Title doesn't fit; return original.
		return box
	}
	trailingDashes := innerW - leadingDashes - titleW

	rebuilt := borderStyle.Render("╭"+strings.Repeat("─", leadingDashes)) +
		titleStyle.Render(titleText) +
		borderStyle.Render(strings.Repeat("─", trailingDashes)+"╮")

	lines[0] = rebuilt
	return strings.Join(lines, "\n")
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
		{"n", "new"},
		{"r", "rename"},
		{"t", "tmux"},
		{"s", "status"},
		{"d", "delete"},
		{"/", "search"},
		{"R", "refresh"},
		{"O", "obsidian"},
		{"S", "slack"},
		{"q", "quit"},
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
		label := string(s)
		st := statusStyle(s)
		if i == m.statusCursor {
			st = st.Reverse(true)
		}
		parts = append(parts, st.Render(label))
	}
	parts = append(parts, footerStyle.Render(" enter set  esc cancel"))
	return strings.Join(parts, "  ")
}

func tildeAbbrev(p string) string {
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}

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

var ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*[A-Za-z]")

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}
