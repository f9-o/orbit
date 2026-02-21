// Package pprint provides rich terminal output formatting for Orbit CLI.
// Inspired by Python's `rich` library — tables, spinners, progress bars,
// colored panels, and status lines.
package pprint

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ─────────────────────────────────────────────────────────────────────────────
// Colour palette
// ─────────────────────────────────────────────────────────────────────────────

var (
	ColorPrimary = lipgloss.Color("#7B8CDE") // Orbit blue-purple
	ColorAccent  = lipgloss.Color("#56E0C8") // Teal
	ColorSuccess = lipgloss.Color("#48BB78") // Green
	ColorWarning = lipgloss.Color("#F6AD55") // Amber
	ColorError   = lipgloss.Color("#FC8181") // Red
	ColorMuted   = lipgloss.Color("#4A5568") // Grey
	ColorText    = lipgloss.Color("#E2E8F0") // Off-white
	ColorBg      = lipgloss.Color("#0D0F18") // Near-black
)

// ─────────────────────────────────────────────────────────────────────────────
// Styles
// ─────────────────────────────────────────────────────────────────────────────

var (
	StyleSuccess = lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true)
	StyleWarning = lipgloss.NewStyle().Foreground(ColorWarning).Bold(true)
	StyleError   = lipgloss.NewStyle().Foreground(ColorError).Bold(true)
	StyleMuted   = lipgloss.NewStyle().Foreground(ColorMuted)
	StyleAccent  = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	StylePrimary = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	StyleText    = lipgloss.NewStyle().Foreground(ColorText)

	StyleLabel = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			Width(14)

	StyleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 2)

	StylePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted).
			Padding(1, 2)
)

// ─────────────────────────────────────────────────────────────────────────────
// Simple output helpers
// ─────────────────────────────────────────────────────────────────────────────

// Success prints a green ✓ success line.
func Success(format string, args ...any) {
	fmt.Println(StyleSuccess.Render("✓ ") + StyleText.Render(fmt.Sprintf(format, args...)))
}

// Warn prints an amber ⚠ warning line.
func Warn(format string, args ...any) {
	fmt.Println(StyleWarning.Render("⚠ ") + StyleText.Render(fmt.Sprintf(format, args...)))
}

// Error prints a red ✗ error line to stderr.
func Error(format string, args ...any) {
	fmt.Fprintln(os.Stderr, StyleError.Render("✗ ")+StyleText.Render(fmt.Sprintf(format, args...)))
}

// Info prints a dimmed info line.
func Info(format string, args ...any) {
	fmt.Println(StyleMuted.Render("  " + fmt.Sprintf(format, args...)))
}

// Step prints a step with an index indicator.
func Step(n int, total int, format string, args ...any) {
	idx := StylePrimary.Render(fmt.Sprintf("[%d/%d]", n, total))
	fmt.Println(idx + " " + StyleText.Render(fmt.Sprintf(format, args...)))
}

// Header prints a section header.
func Header(title string) {
	bar := strings.Repeat("─", 60)
	fmt.Println()
	fmt.Println(StylePrimary.Render(bar))
	fmt.Println(StylePrimary.Render(" ◉ " + strings.ToUpper(title)))
	fmt.Println(StylePrimary.Render(bar))
}

// KV prints a labelled key-value pair.
func KV(key, value string) {
	fmt.Println(StyleLabel.Render(key) + StyleText.Render(value))
}

// Rule prints a full-width horizontal rule.
func Rule(w int) {
	fmt.Println(StyleMuted.Render(strings.Repeat("─", w)))
}

// ─────────────────────────────────────────────────────────────────────────────
// Panel
// ─────────────────────────────────────────────────────────────────────────────

// Panel renders a rounded-border box with optional title.
func Panel(title, body string) {
	content := body
	if title != "" {
		content = StyleAccent.Render(" "+title+" ") + "\n" + body
	}
	fmt.Println(StylePanel.Render(content))
}

// ─────────────────────────────────────────────────────────────────────────────
// Table
// ─────────────────────────────────────────────────────────────────────────────

// Table renders a simple terminal table with coloured headers.
type Table struct {
	headers []string
	rows    [][]string
	out     io.Writer
}

// NewTable creates a new Table writing to stdout.
func NewTable(headers ...string) *Table {
	return &Table{headers: headers, out: os.Stdout}
}

// AddRow appends a data row to the table.
func (t *Table) AddRow(cells ...string) {
	t.rows = append(t.rows, cells)
}

// Render prints the table.
func (t *Table) Render() {
	// Calculate column widths
	widths := make([]int, len(t.headers))
	for i, h := range t.headers {
		widths[i] = len(h)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Header
	fmt.Fprintln(t.out)
	header := ""
	for i, h := range t.headers {
		header += fmt.Sprintf("%-*s", widths[i]+2, h)
	}
	fmt.Fprintln(t.out, StylePrimary.Render(header))

	sep := ""
	for _, w := range widths {
		sep += strings.Repeat("─", w+2)
	}
	fmt.Fprintln(t.out, StyleMuted.Render(sep))

	// Rows
	for _, row := range t.rows {
		line := ""
		for i, cell := range row {
			w := 0
			if i < len(widths) {
				w = widths[i]
			}
			line += fmt.Sprintf("%-*s", w+2, cell)
		}
		fmt.Fprintln(t.out, StyleText.Render(line))
	}
	fmt.Fprintln(t.out)
}

// ─────────────────────────────────────────────────────────────────────────────
// Spinner
// ─────────────────────────────────────────────────────────────────────────────

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner is a non-blocking terminal spinner.
type Spinner struct {
	label  string
	done   chan struct{}
	mu     sync.Mutex
	active bool
}

// NewSpinner creates a Spinner with the given label.
func NewSpinner(label string) *Spinner {
	return &Spinner{label: label, done: make(chan struct{})}
}

// Start begins the spinner animation in a goroutine.
func (s *Spinner) Start() {
	s.mu.Lock()
	s.active = true
	s.mu.Unlock()

	go func() {
		i := 0
		for {
			select {
			case <-s.done:
				return
			case <-time.After(80 * time.Millisecond):
				s.mu.Lock()
				frame := spinnerFrames[i%len(spinnerFrames)]
				fmt.Printf("\r%s %s ", StylePrimary.Render(frame), StyleText.Render(s.label))
				i++
				s.mu.Unlock()
			}
		}
	}()
}

// Stop halts the spinner and clears the line.
func (s *Spinner) Stop(success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active {
		return
	}
	close(s.done)
	s.active = false

	if success {
		fmt.Printf("\r%s %s\n", StyleSuccess.Render("✓"), StyleText.Render(s.label))
	} else {
		fmt.Printf("\r%s %s\n", StyleError.Render("✗"), StyleText.Render(s.label))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Progress bar
// ─────────────────────────────────────────────────────────────────────────────

// Progress renders a simple inline progress bar.
type Progress struct {
	label string
	total int
	width int
}

// NewProgress creates a Progress bar.
func NewProgress(label string, total, width int) *Progress {
	return &Progress{label: label, total: total, width: width}
}

// Set renders the progress bar at the given current value.
func (p *Progress) Set(current int) {
	if p.total == 0 {
		return
	}
	pct := float64(current) / float64(p.total)
	filled := int(pct * float64(p.width))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.width-filled)
	fmt.Printf("\r%s [%s] %3.0f%%",
		StyleText.Render(p.label),
		StyleAccent.Render(bar),
		pct*100,
	)
	if current >= p.total {
		fmt.Println()
	}
}
