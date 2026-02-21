// orbit down — stop and remove running services.
package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/f9-o/orbit/internal/orchestrator"
)

func NewDownCmd() *cobra.Command {
	var removeVolumes bool

	cmd := &cobra.Command{
		Use:   "down [service...]",
		Short: "Stop and remove running services",
		Example: `  orbit down              # stop all services
  orbit down web worker   # stop specific services
  orbit down --volumes    # also remove named volumes`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt := FromContext(cmd.Context())

			docker, err := orchestrator.NewClient("", rt.Log)
			if err != nil {
				return fmt.Errorf("docker: %w", err)
			}
			defer docker.Close()

			nodeName := rt.Flags.Node
			if nodeName == "" {
				nodeName = "local"
			}

			lm := orchestrator.NewLifecycleManager(docker, rt.State, rt.Log)

			if rt.Flags.DryRun {
				what := "all services"
				if len(args) > 0 {
					what = fmt.Sprintf("%v", args)
				}
				fmt.Printf("[dry-run] would stop: %s on node %q\n", what, nodeName)
				return nil
			}

			if err := lm.Down(cmd.Context(), nodeName, args, removeVolumes); err != nil {
				return fmt.Errorf("down: %w", err)
			}

			fmt.Println("✓ Services stopped")
			return nil
		},
	}

	cmd.Flags().BoolVar(&removeVolumes, "volumes", false, "Remove named volumes along with containers")
	return cmd
}
