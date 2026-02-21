// orbit up — start all services defined in orbit.yaml.
package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/f9-o/orbit/internal/orchestrator"
	"github.com/f9-o/orbit/pkg/pprint"
)

func NewUpCmd() *cobra.Command {
	var forceRecreate bool

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Start all services defined in orbit.yaml",
		Example: `  orbit up
  orbit up --force
  orbit up --node prod-01`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt := FromContext(cmd.Context())

			pprint.Header("Starting Services")

			spinner := pprint.NewSpinner("Connecting to Docker")
			spinner.Start()

			docker, err := orchestrator.NewClient("", rt.Log)
			if err != nil {
				spinner.Stop(false)
				return fmt.Errorf("docker: %w", err)
			}
			defer docker.Close()

			if err := docker.Ping(cmd.Context()); err != nil {
				spinner.Stop(false)
				pprint.Error("Docker daemon is not reachable: %v", err)
				pprint.Info("Make sure Docker Desktop is running.")
				return err
			}
			spinner.Stop(true)

			lm := orchestrator.NewLifecycleManager(docker, rt.State, rt.Log)

			total := len(rt.Config.Services)
			for i, svc := range rt.Config.Services {
				pprint.Step(i+1, total, "Starting %s", svc.Name)
			}

			sp := pprint.NewSpinner("Bringing up all services")
			sp.Start()
			err = lm.Up(cmd.Context(), rt.Config.Services, rt.Flags.Node, forceRecreate)
			if err != nil {
				sp.Stop(false)
				pprint.Error("Failed: %v", err)
				return err
			}
			sp.Stop(true)

			_ = total
			fmt.Println()
			pprint.Success("All services started ◉")
			return nil
		},
	}

	cmd.Flags().BoolVar(&forceRecreate, "force", false, "Force-recreate containers even if already running")
	return cmd
}
