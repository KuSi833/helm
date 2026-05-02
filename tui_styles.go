package main

import (
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
)

var (
	colorYellow  = lipgloss.Color("3")
	colorBlue    = lipgloss.Color("4")
	colorMagenta = lipgloss.Color("5")
	colorRed     = lipgloss.Color("1")
	colorGreen   = lipgloss.Color("2")
	colorGray    = lipgloss.Color("8")
	colorCyan    = lipgloss.Color("6")
	colorWhite   = lipgloss.Color("15")
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

var (
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

func glamourKittyStyle() ansi.StyleConfig {
	str := func(s string) *string { return &s }
	uint1 := func(u uint) *uint { return &u }
	bold := true
	italic := true

	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: str("#EEFFFF"),
			},
		},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: str("#EEFFFF")},
			Indent:         uint1(1),
		},
		Paragraph: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: str("#EEFFFF")},
		},
		List: ansi.StyleList{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{Color: str("#EEFFFF")},
			},
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: str("#82AAFF"),
				Bold:  &bold,
			},
		},
		H1: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "# ",
				Color:  str("#FFCB6B"),
				Bold:   &bold,
			},
		},
		H2: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "## ",
				Color:  str("#C792EA"),
				Bold:   &bold,
			},
		},
		H3: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "### ",
				Color:  str("#89DDFF"),
				Bold:   &bold,
			},
		},
		H4: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: str("#82AAFF"), Bold: &bold},
		},
		H5: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: str("#82AAFF"), Bold: &bold},
		},
		H6: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: str("#82AAFF"), Bold: &bold},
		},
		Strong: ansi.StylePrimitive{Color: str("#FFCB6B"), Bold: &bold},
		Emph:   ansi.StylePrimitive{Color: str("#C3E88D"), Italic: &italic},
		HorizontalRule: ansi.StylePrimitive{
			Color:  str("#636261"),
			Format: "---",
		},
		Item:    ansi.StylePrimitive{Color: str("#EEFFFF")},
		Enumeration: ansi.StylePrimitive{Color: str("#EEFFFF")},
		Task: ansi.StyleTask{
			StylePrimitive: ansi.StylePrimitive{Color: str("#EEFFFF")},
			Ticked:         "[x] ",
			Unticked:       "[ ] ",
		},
		Link: ansi.StylePrimitive{
			Color:  str("#82AAFF"),
			Format: " ",
		},
		LinkText: ansi.StylePrimitive{
			Color: str("#82AAFF"),
		},
		Image: ansi.StylePrimitive{Color: str("#82AAFF")},
		ImageText: ansi.StylePrimitive{
			Color:  str("#82AAFF"),
			Format: "{{.text}}",
		},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: str("#C3E88D")},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{Color: str("#EEFFFF")},
				Margin:         uint1(1),
			},
			Chroma: &ansi.Chroma{
				Text:    ansi.StylePrimitive{Color: str("#EEFFFF")},
				Keyword: ansi.StylePrimitive{Color: str("#C792EA")},
				NameFunction: ansi.StylePrimitive{Color: str("#82AAFF")},
				LiteralString: ansi.StylePrimitive{Color: str("#C3E88D")},
				LiteralNumber: ansi.StylePrimitive{Color: str("#F78C6C")},
				Comment: ansi.StylePrimitive{Color: str("#636261")},
			},
		},
		Table: ansi.StyleTable{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{Color: str("#EEFFFF")},
			},
		},
		DefinitionDescription: ansi.StylePrimitive{
			BlockPrefix: "\n* ",
		},
	}
}
