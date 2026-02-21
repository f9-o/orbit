// Package tui: Lipgloss style constants for the "Orbit Dark" theme.
package tui

import "github.com/charmbracelet/lipgloss"

// Styles holds all theme-aware Lipgloss styles.
type Styles struct {
	// Colors
	Background lipgloss.Color
	Surface    lipgloss.Color
	Primary    lipgloss.Color
	Accent     lipgloss.Color
	Danger     lipgloss.Color
	Warning    lipgloss.Color
	Success    lipgloss.Color
	Muted      lipgloss.Color
	Text       lipgloss.Color

	// Component styles
	Header       lipgloss.Style
	HeaderTitle  lipgloss.Style
	Sidebar      lipgloss.Style
	NodeItem     lipgloss.Style
	NodeSelected lipgloss.Style
	PanelTitle   lipgloss.Style
	TableHeader  lipgloss.Style
	TableRow     lipgloss.Style
	TableRowSel  lipgloss.Style
	LogViewport  lipgloss.Style
	Footer       lipgloss.Style
	FooterKey    lipgloss.Style
	Modal        lipgloss.Style
	ModalTitle   lipgloss.Style
	ModalInput   lipgloss.Style
	StatusOK     lipgloss.Style
	StatusWarn   lipgloss.Style
	StatusErr    lipgloss.Style
	Border       lipgloss.Style
}

// newStyles returns the "Orbit Dark" theme styles.
func newStyles() Styles {
	bg := lipgloss.Color("#0D0F18")
	surface := lipgloss.Color("#171A2B")
	primary := lipgloss.Color("#7B8CDE")
	accent := lipgloss.Color("#56E0C8")
	danger := lipgloss.Color("#F56565")
	warning := lipgloss.Color("#ECC94B")
	success := lipgloss.Color("#68D391")
	muted := lipgloss.Color("#4A5568")
	text := lipgloss.Color("#E2E8F0")

	border := lipgloss.Border{
		Top: "─", Bottom: "─", Left: "│", Right: "│",
		TopLeft: "┌", TopRight: "┐", BottomLeft: "└", BottomRight: "┘",
	}

	return Styles{
		Background: bg, Surface: surface, Primary: primary,
		Accent: accent, Danger: danger, Warning: warning,
		Success: success, Muted: muted, Text: text,

		Header: lipgloss.NewStyle().
			Background(primary).Foreground(bg).
			Bold(true).Padding(0, 1),

		HeaderTitle: lipgloss.NewStyle().
			Foreground(accent).Bold(true),

		Sidebar: lipgloss.NewStyle().
			Background(surface).Foreground(text).
			BorderStyle(border).BorderRight(true).
			BorderForeground(muted).
			Padding(1, 1),

		NodeItem: lipgloss.NewStyle().
			Foreground(text).PaddingLeft(2),

		NodeSelected: lipgloss.NewStyle().
			Foreground(accent).Bold(true).PaddingLeft(1),

		PanelTitle: lipgloss.NewStyle().
			Foreground(primary).Bold(true).
			BorderStyle(lipgloss.NormalBorder()).BorderBottom(true).
			BorderForeground(muted).Padding(0, 1),

		TableHeader: lipgloss.NewStyle().
			Foreground(muted).Bold(true).Padding(0, 1),

		TableRow: lipgloss.NewStyle().
			Foreground(text).Padding(0, 1),

		TableRowSel: lipgloss.NewStyle().
			Background(surface).Foreground(accent).Bold(true).Padding(0, 1),

		LogViewport: lipgloss.NewStyle().
			Background(bg).Foreground(text).
			Padding(0, 1),

		Footer: lipgloss.NewStyle().
			Background(surface).Foreground(muted).
			Padding(0, 1),

		FooterKey: lipgloss.NewStyle().
			Foreground(primary).Bold(true),

		Modal: lipgloss.NewStyle().
			Background(surface).Foreground(text).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primary).
			Padding(1, 2),

		ModalTitle: lipgloss.NewStyle().
			Foreground(warning).Bold(true),

		ModalInput: lipgloss.NewStyle().
			Foreground(text).Background(bg).
			Border(lipgloss.NormalBorder()).BorderForeground(muted),

		StatusOK:   lipgloss.NewStyle().Foreground(success),
		StatusWarn: lipgloss.NewStyle().Foreground(warning),
		StatusErr:  lipgloss.NewStyle().Foreground(danger),

		Border: lipgloss.NewStyle().BorderStyle(border).BorderForeground(muted),
	}
}
