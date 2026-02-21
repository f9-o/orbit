// Package components: TUI sub-components for Orbit's dashboard.
package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// ─────────────────────────────────────────────────────────────────────────────
// Header component
// ─────────────────────────────────────────────────────────────────────────────

// Header renders the top status bar.
type Header struct {
	node         string
	serviceCount int
	nodeCount    int
}

// NewHeader creates a Header for the named node.
func NewHeader(node string) Header {
	return Header{node: node}
}

func (h *Header) SetServiceCount(n int) { h.serviceCount = n }
func (h *Header) SetNodeCount(n int)    { h.nodeCount = n }

// View renders the header bar. Accepts total terminal width.
func (h *Header) View(width int) string {
	left := fmt.Sprintf(" ◉ ORBIT  %s ", h.node)
	right := fmt.Sprintf(" %d nodes · %d services ",
		h.nodeCount, h.serviceCount)
	gap := width - len(left) - len(right)
	if gap < 0 {
		gap = 0
	}
	return lipgloss.NewStyle().
		Background(lipgloss.Color("#7B8CDE")).
		Foreground(lipgloss.Color("#0D0F18")).
		Bold(true).
		Width(width).
		Render(left + spaces(gap) + right)
}

// ─────────────────────────────────────────────────────────────────────────────
// Sidebar component
// ─────────────────────────────────────────────────────────────────────────────

// Sidebar renders the node navigator.
type Sidebar struct {
	selected int
	items    []nodeEntry
}

type nodeEntry struct {
	Name   string
	Status string
}

// NewSidebar creates an empty Sidebar.
func NewSidebar() Sidebar { return Sidebar{} }

// SetNodes updates the node list from name strings.
func (s *Sidebar) SetNodes(names []string) {
	s.items = make([]nodeEntry, len(names))
	for i, n := range names {
		s.items[i] = nodeEntry{Name: n}
	}
}

// View renders the sidebar.
func (s *Sidebar) View(width, height int) string {
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7B8CDE")).Bold(true).
		Render("NODES")

	content := title + "\n"

	if len(s.items) == 0 {
		content += lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4A5568")).
			Render("  (no nodes)")
	}

	for i, item := range s.items {
		icon := "○ "
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("#E2E8F0")).PaddingLeft(2)
		if i == s.selected {
			icon = "▶ "
			style = style.Foreground(lipgloss.Color("#56E0C8")).Bold(true)
		}
		content += style.Render(icon+item.Name) + "\n"
	}

	return lipgloss.NewStyle().
		Background(lipgloss.Color("#171A2B")).
		Width(width).Height(height).
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		BorderForeground(lipgloss.Color("#4A5568")).
		Padding(1, 1).
		Render(content)
}

// ─────────────────────────────────────────────────────────────────────────────
// Footer component
// ─────────────────────────────────────────────────────────────────────────────

// Footer renders the bottom hint bar.
type Footer struct {
	err error
}

// NewFooter creates a Footer.
func NewFooter() Footer { return Footer{} }

// SetError sets an error message to display.
func (f *Footer) SetError(err error) { f.err = err }

// View renders the footer.
func (f *Footer) View(width int) string {
	hints := []struct{ key, desc string }{
		{"↑↓", "navigate"}, {"l", "logs"}, {"s", "scale"},
		{"d", "deploy"}, {"x", "stop"}, {"/", "search"}, {"?", "help"}, {"q", "quit"},
	}

	content := ""
	for _, h := range hints {
		content += lipgloss.NewStyle().Foreground(lipgloss.Color("#7B8CDE")).Bold(true).Render(h.key)
		content += lipgloss.NewStyle().Foreground(lipgloss.Color("#4A5568")).Render(" " + h.desc + "  ")
	}

	if f.err != nil {
		content = lipgloss.NewStyle().Foreground(lipgloss.Color("#F56565")).
			Render("Error: " + f.err.Error())
	}

	return lipgloss.NewStyle().
		Background(lipgloss.Color("#171A2B")).
		Width(width).Padding(0, 1).
		Render(content)
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func spaces(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += " "
	}
	return s
}
