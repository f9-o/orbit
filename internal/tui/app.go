// Package tui defines the Bubble Tea model for Orbit's interactive dashboard.
package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	v1 "github.com/f9-o/orbit/api/v1"
	"github.com/f9-o/orbit/internal/core/config"
	"github.com/f9-o/orbit/internal/core/logger"
	"github.com/f9-o/orbit/internal/core/state"
	"github.com/f9-o/orbit/internal/metrics"
	"github.com/f9-o/orbit/internal/orchestrator"
	"github.com/f9-o/orbit/internal/tui/components"
)

// Config carries dependencies into the TUI app.
type Config struct {
	Node         string
	DockerClient *orchestrator.Client
	State        *state.DB
	Log          *logger.Logger
	OrbitConfig  *config.Config
}

// ActivePanel identifies which main panel has focus.
type ActivePanel int

const (
	PanelServices ActivePanel = iota
	PanelLogs
	PanelMetrics
)

// Model is the root Bubble Tea model (Elm architecture).
type Model struct {
	cfg Config

	// Dimensions
	width  int
	height int

	// Panels
	panel       ActivePanel
	services    []v1.ServiceState
	nodes       []v1.NodeInfo
	logViewport viewport.Model
	logLines    []string
	metrics     v1.Metrics

	// Sub-components
	header  components.Header
	sidebar components.Sidebar
	footer  components.Footer
	modal   *components.Modal

	// Selected service for log/metrics view
	selectedService int

	// Collector
	collector *metrics.Collector

	// Error state
	lastError error

	// Theme
	styles Styles
}

// tickMsg is emitted by the metrics ticker.
type tickMsg time.Time

// logLineMsg carries a new log line from a streaming goroutine.
type logLineMsg string

// metricsMsg carries a fresh Metrics snapshot.
type metricsMsg v1.Metrics

// serviceListMsg carries an updated services list.
type serviceListMsg []v1.ServiceState

// nodeListMsg carries an updated nodes list.
type nodeListMsg []v1.NodeInfo

// errMsg carries an error to display in the status bar.
type errMsg error

// New constructs a new TUI Model.
func New(cfg Config) *Model {
	styles := newStyles()
	lv := viewport.New(0, 0)
	lv.Style = styles.LogViewport

	collector := metrics.NewCollector(cfg.DockerClient, cfg.Node, cfg.Log)

	return &Model{
		cfg:         cfg,
		logViewport: lv,
		styles:      styles,
		header:      components.NewHeader(cfg.Node),
		sidebar:     components.NewSidebar(),
		footer:      components.NewFooter(),
		collector:   collector,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Init
// ─────────────────────────────────────────────────────────────────────────────

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.tickCmd(),
		m.loadServicesCmd(),
		m.loadNodesCmd(),
		m.startCollectorCmd(),
	)
}

// ─────────────────────────────────────────────────────────────────────────────
// Update
// ─────────────────────────────────────────────────────────────────────────────

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.logViewport.Width = m.width - 22 // sidebar width
		m.logViewport.Height = m.height - 10

	case tea.KeyMsg:
		// Modal intercepts key events when open
		if m.modal != nil {
			cmd, done := m.modal.HandleKey(msg)
			if done {
				m.modal = nil
			}
			return m, cmd
		}
		cmds = append(cmds, m.handleKey(msg))

	case tickMsg:
		cmds = append(cmds, m.tickCmd(), m.loadServicesCmd())
		m.metrics = m.collector.AllMetrics()

	case serviceListMsg:
		m.services = msg
		m.header.SetServiceCount(len(msg))

	case nodeListMsg:
		m.nodes = msg
		nodeNames := make([]string, len(msg))
		for i, n := range msg {
			nodeNames[i] = n.Spec.Name
		}
		m.sidebar.SetNodes(nodeNames)
		m.header.SetNodeCount(len(msg))

	case metricsMsg:
		m.metrics = v1.Metrics(msg)

	case logLineMsg:
		m.logLines = append(m.logLines, string(msg))
		if len(m.logLines) > 500 {
			m.logLines = m.logLines[len(m.logLines)-500:]
		}
		m.logViewport.SetContent(joinLines(m.logLines))
		m.logViewport.GotoBottom()

	case errMsg:
		m.lastError = msg
		m.footer.SetError(msg)
	}

	// Propagate to viewport
	var lvCmd tea.Cmd
	m.logViewport, lvCmd = m.logViewport.Update(msg)
	cmds = append(cmds, lvCmd)

	return m, tea.Batch(cmds...)
}

// handleKey processes keyboard input when no modal is open.
func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	kb := defaultKeymap()

	switch msg.String() {
	case kb.Quit:
		return tea.Quit

	case kb.TabNext:
		m.panel = (m.panel + 1) % 3

	case kb.TabPrev:
		m.panel = (m.panel + 2) % 3 // wrap backwards

	case kb.NavDown, "j":
		if m.panel == PanelServices && m.selectedService < len(m.services)-1 {
			m.selectedService++
		}

	case kb.NavUp, "k":
		if m.panel == PanelServices && m.selectedService > 0 {
			m.selectedService--
		}

	case "l":
		m.panel = PanelLogs

	case "?":
		m.modal = components.NewHelpModal(m.styles.Modal)

	case "x":
		if len(m.services) > 0 && m.selectedService < len(m.services) {
			svc := m.services[m.selectedService]
			m.modal = components.NewConfirmModal(
				fmt.Sprintf("Stop %s?", svc.Name),
				fmt.Sprintf("This will stop and remove container %s", svc.ContainerID[:12]),
				m.styles.Modal,
				nil,
			)
		}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// View
// ─────────────────────────────────────────────────────────────────────────────

func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	header := m.header.View(m.width)
	sidebar := m.sidebar.View(20, m.height-4)
	mainPanel := m.renderMain()
	footer := m.footer.View(m.width)

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, mainPanel)

	view := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)

	if m.modal != nil {
		view = m.modal.Overlay(view, m.width, m.height)
	}

	return view
}

func (m *Model) renderMain() string {
	mainWidth := m.width - 22

	switch m.panel {
	case PanelServices:
		return components.RenderServicesTable(m.services, m.metrics, m.selectedService, m.styles, mainWidth, m.height-6)
	case PanelLogs:
		title := m.styles.PanelTitle.Render("LOGS")
		return lipgloss.JoinVertical(lipgloss.Left, title, m.logViewport.View())
	case PanelMetrics:
		return components.RenderMetrics(m.metrics, m.styles, mainWidth, m.height-6)
	}
	return ""
}

// ─────────────────────────────────────────────────────────────────────────────
// Commands (async data fetchers)
// ─────────────────────────────────────────────────────────────────────────────

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) loadServicesCmd() tea.Cmd {
	return func() tea.Msg {
		states, err := m.cfg.State.ListServiceStates(m.cfg.Node)
		if err != nil {
			return errMsg(err)
		}
		return serviceListMsg(states)
	}
}

func (m *Model) loadNodesCmd() tea.Cmd {
	return func() tea.Msg {
		nodes, err := m.cfg.State.ListNodes()
		if err != nil {
			return errMsg(err)
		}
		return nodeListMsg(nodes)
	}
}

func (m *Model) startCollectorCmd() tea.Cmd {
	return func() tea.Msg {
		// Collector is started in a separate goroutine — no msg returned
		return nil
	}
}

// joinLines concatenates log lines with newlines.
func joinLines(lines []string) string {
	out := ""
	for _, l := range lines {
		out += l + "\n"
	}
	return out
}
