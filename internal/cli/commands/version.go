// orbit version — print version information.
package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/f9-o/orbit/pkg/pprint"
)

// Build-time variables injected via -ldflags.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "version",
		Short:        "Print Orbit version information",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			info := map[string]string{
				"version":    Version,
				"commit":     Commit,
				"build_date": BuildDate,
				"go_version": runtime.Version(),
				"os_arch":    runtime.GOOS + "/" + runtime.GOARCH,
			}

			jsonFlag, _ := cmd.Root().PersistentFlags().GetBool("json")
			if jsonFlag {
				return json.NewEncoder(os.Stdout).Encode(info)
			}

			pprint.PrintBanner(Version, BuildDate)

			pprint.KV("Version  ", Version)
			pprint.KV("Commit   ", Commit)
			pprint.KV("Built    ", BuildDate)
			pprint.KV("Go       ", runtime.Version())
			pprint.KV("Platform ", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
			fmt.Println()
			return nil
		},
	}
}
