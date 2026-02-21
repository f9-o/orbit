// orbit deploy — rolling update a specific service.
package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/f9-o/orbit/internal/health"
	"github.com/f9-o/orbit/internal/orchestrator"
	"github.com/f9-o/orbit/pkg/pprint"
)

func NewDeployCmd() *cobra.Command {
	var tag string
	var timeout time.Duration
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "deploy <service>",
		Short: "Rolling update a running service to a new image tag",
		Args:  cobra.ExactArgs(1),
		Example: `  orbit deploy web
  orbit deploy web --tag v1.2.0
  orbit deploy web --tag latest --timeout 3m
  orbit deploy web --dry-run`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt := FromContext(cmd.Context())
			name := args[0]

			svc := rt.Config.ServiceByName(name)
			if svc == nil {
				pprint.Error("Service %q not found in orbit.yaml", name)
				return fmt.Errorf("service %q not found", name)
			}

			pprint.Header("Rolling Deploy — " + name)
			pprint.KV("Service", name)
			pprint.KV("Image", svc.Image)
			if tag != "" {
				pprint.KV("Tag", tag)
			}
			pprint.KV("Node", func() string {
				if rt.Flags.Node != "" {
					return rt.Flags.Node
				}
				return "local"
			}())
			if dryRun {
				pprint.Warn("DRY RUN — no changes will be made")
			}
			fmt.Println()

			docker, err := orchestrator.NewClient("", rt.Log)
			if err != nil {
				return fmt.Errorf("docker: %w", err)
			}
			defer docker.Close()

			checker := health.NewChecker(rt.Log)
			deployer := orchestrator.NewDeployer(docker, rt.State, checker, rt.Log)

			// Step 1: Pull
			sp1 := pprint.NewSpinner("Pulling new image")
			sp1.Start()

			err = deployer.Deploy(cmd.Context(), *svc, rt.Flags.Node, orchestrator.DeployOptions{
				Tag:     tag,
				Timeout: timeout,
				DryRun:  dryRun,
			})

			if err != nil {
				sp1.Stop(false)
				pprint.Error("Deploy failed: %v", err)
				pprint.Info("Run `orbit logs %s` to inspect the failed container.", name)
				return err
			}
			sp1.Stop(true)

			fmt.Println()
			pprint.Success("Deploy complete — %s is running the new image", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&tag, "tag", "", "Image tag to deploy (default: current tag in orbit.yaml)")
	cmd.Flags().DurationVar(&timeout, "timeout", 2*time.Minute, "Health check timeout before rollback")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Simulate deploy without making changes")
	return cmd
}
