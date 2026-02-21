// orbit monitor — display real-time metrics for all services.
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	v1 "github.com/f9-o/orbit/api/v1"
	"github.com/f9-o/orbit/internal/metrics"
	"github.com/f9-o/orbit/internal/orchestrator"
)

func NewMonitorCmd() *cobra.Command {
	var format string
	var interval time.Duration

	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "Stream real-time resource metrics for all running services",
		Example: `  orbit monitor
  orbit monitor --format json
  orbit monitor --interval 5s`,
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

			collector := metrics.NewCollector(docker, nodeName, rt.Log)

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			// Start collector
			go collector.Run(ctx)

			// Handle Ctrl+C
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigs
				cancel()
			}()

			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			fmt.Printf("◉ Monitoring services on %q (Ctrl+C to stop)...\n\n", nodeName)

			for {
				select {
				case <-ctx.Done():
					return nil
				case <-ticker.C:
					m := collector.AllMetrics()

					switch format {
					case "json":
						data, _ := json.Marshal(m)
						fmt.Println(string(data))
					default:
						printMetricsTable(m, nodeName)
					}
				}
			}
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "Output format: table | json | prometheus")
	cmd.Flags().DurationVar(&interval, "interval", 2*time.Second, "Refresh interval")
	return cmd
}

func printMetricsTable(m v1.Metrics, node string) {
	fmt.Printf("\033[H\033[2J") // clear screen
	fmt.Printf("◉ Orbit Monitor — %s — %s\n\n", node, time.Now().Format("15:04:05"))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "SERVICE\tCPU%\tMEM\tNET RX\tNET TX\tPIDs")
	fmt.Fprintln(w, "-------\t----\t---\t------\t------\t----")
	for name, svc := range m.Services {
		mem := fmt.Sprintf("%.1fMB", float64(svc.MemBytes)/1024/1024)
		rx := fmt.Sprintf("%.1fKB", float64(svc.NetRxBytes)/1024)
		tx := fmt.Sprintf("%.1fKB", float64(svc.NetTxBytes)/1024)
		fmt.Fprintf(w, "%s\t%.1f%%\t%s\t%s\t%s\t%d\n",
			name, svc.CPUPercent, mem, rx, tx, svc.PIDs)
	}
	_ = w.Flush()
}
