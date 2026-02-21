// orbit scale — adjust service replica count.
package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/f9-o/orbit/internal/orchestrator"
)

func NewScaleCmd() *cobra.Command {
	var replicas int

	cmd := &cobra.Command{
		Use:   "scale <service>",
		Short: "Scale a service to the specified number of replicas",
		Args:  cobra.ExactArgs(1),
		Example: `  orbit scale web --replicas 3
  orbit scale worker --replicas 0   # stop all replicas`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt := FromContext(cmd.Context())
			serviceName := args[0]

			svcSpec := rt.Config.ServiceByName(serviceName)
			if svcSpec == nil {
				return fmt.Errorf("service %q not found in orbit.yaml", serviceName)
			}

			nodeName := rt.Flags.Node
			if nodeName == "" {
				nodeName = "local"
			}

			docker, err := orchestrator.NewClient("", rt.Log)
			if err != nil {
				return fmt.Errorf("docker: %w", err)
			}
			defer docker.Close()

			scaler := orchestrator.NewScaler(docker, rt.State, rt.Log)

			if rt.Flags.DryRun {
				fmt.Printf("[dry-run] would scale %q to %d replicas on %q\n", serviceName, replicas, nodeName)
				return nil
			}

			fmt.Printf("◉ Scaling %q to %d replica(s)...\n", serviceName, replicas)
			if err := scaler.Scale(cmd.Context(), *svcSpec, nodeName, replicas); err != nil {
				return fmt.Errorf("scale: %w", err)
			}

			fmt.Printf("✓ %q scaled to %d\n", serviceName, replicas)
			return nil
		},
	}

	cmd.Flags().IntVar(&replicas, "replicas", 1, "Target number of replicas")
	_ = cmd.MarkFlagRequired("replicas")
	return cmd
}
