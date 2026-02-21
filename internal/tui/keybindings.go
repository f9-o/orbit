// Package tui: keyboard binding configuration.
package tui

// Keymap defines all keyboard shortcuts for the TUI.
type Keymap struct {
	Quit     string
	TabNext  string
	TabPrev  string
	NavUp    string
	NavDown  string
	NavLeft  string
	NavRight string
	Select   string
	Logs     string
	Scale    string
	Deploy   string
	Stop     string
	Nodes    string
	Search   string
	Help     string
}

// defaultKeymap returns the default Orbit TUI key bindings.
func defaultKeymap() Keymap {
	return Keymap{
		Quit:     "q",
		TabNext:  "tab",
		TabPrev:  "shift+tab",
		NavUp:    "up",
		NavDown:  "down",
		NavLeft:  "left",
		NavRight: "right",
		Select:   "enter",
		Logs:     "l",
		Scale:    "s",
		Deploy:   "d",
		Stop:     "x",
		Nodes:    "n",
		Search:   "/",
		Help:     "?",
	}
}

// HelpText returns the keyboard shortcut reference displayed in the help modal.
func HelpText() string {
	return `
  NAVIGATION
  ──────────────────────────────────────
  Tab / Shift+Tab    Cycle panels
  ↑↓  /  j k        Navigate list
  ←→  /  h l        Switch tabs

  ACTIONS
  ──────────────────────────────────────
  Enter              Select / Expand
  l                  Open service logs
  s                  Scale service
  d                  Deploy (rolling)
  x                  Stop service
  n                  Switch node

  SEARCH & MISC
  ──────────────────────────────────────
  /                  Incremental search
  ?                  Toggle this help
  q                  Quit
  Ctrl+C             Force quit
`
}
