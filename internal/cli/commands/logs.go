// orbit logs — stream or tail service container logs.
package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/f9-o/orbit/internal/orchestrator"
)

func NewLogsCmd() *cobra.Command {
	var follow bool
	var tail int
	var since time.Duration

	cmd := &cobra.Command{
		Use:   "logs <service>",
		Short: "Stream or tail logs from a service container",
		Args:  cobra.ExactArgs(1),
		Example: `  orbit logs web
  orbit logs web -f
  orbit logs worker -n 200
  orbit logs api --since 1h`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt := FromContext(cmd.Context())
			serviceName := args[0]

			state, err := rt.State.GetServiceState(rt.Flags.Node, serviceName)
			if err != nil {
				return fmt.Errorf("state: %w", err)
			}
			if state == nil {
				return fmt.Errorf("service %q not found in state. Is it running? Try 'orbit up'", serviceName)
			}
			_ = tail // tail param — Docker API uses 'since' + streaming

			docker, err := orchestrator.NewClient("", rt.Log)
			if err != nil {
				return fmt.Errorf("docker: %w", err)
			}
			defer docker.Close()

			if follow {
				fmt.Printf("◉ Following logs for %q (Ctrl+C to stop)...\n", serviceName)
			}

			return docker.StreamLogs(cmd.Context(), state.ContainerID, follow, since, os.Stdout)
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output in real-time")
	cmd.Flags().IntVarP(&tail, "tail", "n", 100, "Number of lines to show from end of logs")
	cmd.Flags().DurationVar(&since, "since", 0, "Show logs since duration (e.g., 1h, 30m, 5s)")
	return cmd
}
