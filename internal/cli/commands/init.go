// orbit init — scaffold a new orbit.yaml in the target directory.
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/f9-o/orbit/internal/core/config"
)

func NewInitCmd() *cobra.Command {
	var targetPath string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Scaffold a new orbit.yaml in the current (or specified) directory",
		Example: `  orbit init
  orbit init --path ./my-project`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if targetPath == "" {
				targetPath = "."
			}
			outFile := filepath.Join(targetPath, "orbit.yaml")
			if _, err := os.Stat(outFile); err == nil {
				return fmt.Errorf("orbit.yaml already exists at %s — delete it first to reinitialise", outFile)
			}

			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("create dir %q: %w", targetPath, err)
			}

			if err := os.WriteFile(outFile, []byte(config.DefaultConfigTemplate), 0644); err != nil {
				return fmt.Errorf("write orbit.yaml: %w", err)
			}

			fmt.Printf("✓ Created %s\n", outFile)
			fmt.Println("  Edit it to define your services, then run: orbit up")
			return nil
		},
	}

	cmd.Flags().StringVar(&targetPath, "path", ".", "Target directory for orbit.yaml")
	return cmd
}
