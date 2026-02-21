// Package components: services table, metrics panel, and modal rendering.
package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	v1 "github.com/f9-o/orbit/api/v1"
)

// ─────────────────────────────────────────────────────────────────────────────
// Services Table
// ─────────────────────────────────────────────────────────────────────────────

// RenderServicesTable renders the service list table.
func RenderServicesTable(services []v1.ServiceState, metrics v1.Metrics, selected int, styles interface{}, width, height int) string {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4A5568")).Bold(true).Padding(0, 1)
	rowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E2E8F0")).Padding(0, 1)
	selStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#171A2B")).
		Foreground(lipgloss.Color("#56E0C8")).Bold(true).Padding(0, 1)

	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7B8CDE")).Bold(true).
		Padding(0, 1).
		Render("SERVICES")

	hdr := headerStyle.Render(
		fmt.Sprintf("%-20s %-30s %-10s %-8s %s",
			"NAME", "IMAGE", "HEALTH", "CPU%", "MEM"),
	)

	rows := ""
	for i, svc := range services {
		health := healthBadge(svc.Status)

		cpuStr := "-"
		memStr := "-"
		if m, ok := metrics.Services[svc.Name]; ok {
			cpuStr = fmt.Sprintf("%.1f%%", m.CPUPercent)
			memStr = fmtBytes(m.MemBytes)
		}

		image := svc.Image
		if len(image) > 28 {
			image = "..." + image[len(image)-25:]
		}

		line := fmt.Sprintf("%-20s %-30s %-10s %-8s %s",
			truncate(svc.Name, 18), truncate(image, 28),
			health, cpuStr, memStr,
		)

		if i == selected {
			rows += selStyle.Render("▶ "+line) + "\n"
		} else {
			rows += rowStyle.Render("  "+line) + "\n"
		}
	}

	if len(services) == 0 {
		rows = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4A5568")).
			Padding(2, 2).
			Render("No services running. Run 'orbit up' to start.")
	}

	return lipgloss.NewStyle().Width(width).Height(height).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, hdr, rows))
}

// ─────────────────────────────────────────────────────────────────────────────
// Metrics Panel
// ─────────────────────────────────────────────────────────────────────────────

// RenderMetrics renders the metrics sparkline panel.
func RenderMetrics(metrics v1.Metrics, styles interface{}, width, height int) string {
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7B8CDE")).Bold(true).
		Padding(0, 1).Render("METRICS")

	content := title + "\n\n"

	if len(metrics.Services) == 0 {
		return content + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4A5568")).Padding(1, 2).
			Render("No metrics available. Ensure services are running.")
	}

	for name, m := range metrics.Services {
		bar := cpuBar(m.CPUPercent, 20)
		content += fmt.Sprintf("  %-18s CPU: %s %5.1f%%   MEM: %s/%s\n",
			name, bar, m.CPUPercent, fmtBytes(m.MemBytes), fmtBytes(m.MemLimit))
	}

	return lipgloss.NewStyle().Width(width).Height(height).Render(content)
}

// ─────────────────────────────────────────────────────────────────────────────
// Modal
// ─────────────────────────────────────────────────────────────────────────────

// Modal is a pop-over dialog.
type Modal struct {
	title     string
	body      string
	style     lipgloss.Style
	onConfirm func() tea.Cmd
	input     string
	typ       modalType
}

type modalType int

const (
	modalConfirm modalType = iota
	modalHelp
)

// NewConfirmModal creates a destructive-action confirmation modal.
func NewConfirmModal(title, body string, style lipgloss.Style, onConfirm func() tea.Cmd) *Modal {
	return &Modal{
		title:     title,
		body:      body,
		style:     style,
		onConfirm: onConfirm,
		typ:       modalConfirm,
	}
}

// NewHelpModal creates the keyboard help modal.
func NewHelpModal(style lipgloss.Style) *Modal {
	return &Modal{
		title: "Keyboard Shortcuts",
		body: `
  Tab / Shift+Tab    Cycle panels        l    Logs
  ↑↓  /  j k        Navigate            s    Scale
  ←→  /  h l        Switch tabs         d    Deploy
  Enter              Select              x    Stop
  /                  Search              q    Quit
`,
		style: style,
		typ:   modalHelp,
	}
}

// HandleKey processes a key for the modal. Returns (cmd, done).
func (m *Modal) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "esc", "q":
		return nil, true
	case "enter":
		if m.typ == modalConfirm && m.onConfirm != nil {
			return m.onConfirm(), true
		}
		return nil, true
	default:
		if m.typ == modalConfirm {
			m.input += msg.String()
		}
	}
	return nil, false
}

// Overlay renders the modal centred over the background content.
func (m *Modal) Overlay(bg string, width, height int) string {
	content := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ECC94B")).Bold(true).
		Render("⚠  "+m.title) + "\n\n"
	content += m.body

	if m.typ == modalConfirm {
		content += "\n\n  > " + m.input + "█"
		content += "\n\n  [Enter] Confirm   [Esc] Cancel"
	} else {
		content += "\n\n  [Esc] Close"
	}

	box := m.style.Render(content)
	boxLines := strings.Split(box, "\n")
	boxWidth := 0
	for _, l := range boxLines {
		if len(l) > boxWidth {
			boxWidth = len(l)
		}
	}
	boxHeight := len(boxLines)

	// Simple centre overlay (approximate — production would use overlay library)
	topPad := (height - boxHeight) / 2
	leftPad := (width - boxWidth) / 2
	if topPad < 0 {
		topPad = 0
	}
	if leftPad < 0 {
		leftPad = 0
	}

	_ = bg // In a full implementation, we'd composite over bg
	padding := strings.Repeat("\n", topPad)
	indent := strings.Repeat(" ", leftPad)
	out := padding
	for _, l := range boxLines {
		out += indent + l + "\n"
	}
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

func healthBadge(status v1.ServiceStatus) string {
	switch status {
	case v1.StatusHealthy:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#68D391")).Render("● OK")
	case v1.StatusDegraded:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#ECC94B")).Render("◐ DEG")
	case v1.StatusUnhealthy:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#F56565")).Render("○ ERR")
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#4A5568")).Render("? UNK")
	}
}

func cpuBar(pct float64, width int) string {
	filled := int(pct / 100.0 * float64(width))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	color := lipgloss.Color("#68D391")
	if pct > 80 {
		color = lipgloss.Color("#F56565")
	} else if pct > 50 {
		color = lipgloss.Color("#ECC94B")
	}
	return lipgloss.NewStyle().Foreground(color).Render("[" + bar + "]")
}

func fmtBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1fG", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1fM", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1fK", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%dB", b)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
