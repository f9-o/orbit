// orbit ui — launch the interactive TUI dashboard.
package commands

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/f9-o/orbit/internal/orchestrator"
	"github.com/f9-o/orbit/internal/tui"
)

func NewUICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Launch the interactive TUI dashboard",
		Example: `  orbit ui
  orbit ui --node prod-01`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt := FromContext(cmd.Context())

			docker, err := orchestrator.NewClient("", rt.Log)
			if err != nil {
				return fmt.Errorf("docker: %w", err)
			}

			nodeName := rt.Flags.Node
			if nodeName == "" {
				nodeName = "local"
			}

			// Build initial app model
			app := tui.New(tui.Config{
				Node:         nodeName,
				DockerClient: docker,
				State:        rt.State,
				Log:          rt.Log,
				OrbitConfig:  rt.Config,
			})

			p := tea.NewProgram(app,
				tea.WithAltScreen(),       // use alternate screen buffer
				tea.WithMouseCellMotion(), // enable mouse support
			)

			if _, err := p.Run(); err != nil {
				return fmt.Errorf("tui: %w", err)
			}
			return nil
		},
	}
	return cmd
}
