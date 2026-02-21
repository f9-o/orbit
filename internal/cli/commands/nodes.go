// orbit nodes — manage the remote node registry.
package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	v1 "github.com/f9-o/orbit/api/v1"
	"github.com/f9-o/orbit/internal/remote"
	"github.com/f9-o/orbit/pkg/sshutil"
)

func NewNodesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodes",
		Short: "Manage remote node registry",
		Long:  "Add, remove, list, inspect, and test remote nodes in the Orbit registry.",
	}

	cmd.AddCommand(
		newNodesAddCmd(),
		newNodesRmCmd(),
		newNodesLsCmd(),
		newNodesInfoCmd(),
		newNodesTestCmd(),
		newNodesTrustCmd(),
	)
	return cmd
}

func newNodesAddCmd() *cobra.Command {
	var keyPath string
	var port int

	cmd := &cobra.Command{
		Use:   "add <name> <user@host>",
		Short: "Register a new remote node",
		Args:  cobra.ExactArgs(2),
		Example: `  orbit nodes add prod-01 deploy@192.168.1.10
  orbit nodes add staging ubuntu@staging.example.com --key ~/.ssh/id_ed25519`,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt := FromContext(cmd.Context())
			name := args[0]
			userAtHost := args[1]

			user, host := parseUserAtHost(userAtHost)
			if port == 0 {
				port = 22
			}
			if keyPath == "" {
				homeDir, _ := os.UserHomeDir()
				keyPath = fmt.Sprintf("%s/.ssh/id_ed25519", homeDir)
			}

			registry := remote.NewRegistry(rt.State)

			nodeInfo := v1.NodeInfo{
				Spec: v1.NodeSpec{
					Name: name,
					Host: host,
					User: user,
					Key:  keyPath,
					Port: port,
				},
				Status: v1.NodeOffline,
			}

			if err := registry.Add(nodeInfo); err != nil {
				return err
			}

			fmt.Printf("✓ Node %q registered (%s@%s)\n", name, user, host)
			fmt.Printf("  Run 'orbit nodes trust %s' to record the host key\n", name)
			fmt.Printf("  Run 'orbit nodes test %s' to verify connectivity\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&keyPath, "key", "", "Path to SSH private key")
	cmd.Flags().IntVar(&port, "port", 22, "SSH port")
	return cmd
}

func newNodesRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <name>",
		Short: "Remove a node from the registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt := FromContext(cmd.Context())
			registry := remote.NewRegistry(rt.State)
			if err := registry.Remove(args[0]); err != nil {
				return err
			}
			fmt.Printf("✓ Node %q removed\n", args[0])
			return nil
		},
	}
}

func newNodesLsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List all registered nodes",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt := FromContext(cmd.Context())
			registry := remote.NewRegistry(rt.State)
			nodes, err := registry.List()
			if err != nil {
				return err
			}

			if rt.Flags.JSONOutput {
				return json.NewEncoder(os.Stdout).Encode(nodes)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "NAME\tHOST\tUSER\tSTATUS\tLAST SEEN\tKEY TRUSTED")
			for _, n := range nodes {
				lastSeen := "never"
				if !n.LastSeen.IsZero() {
					lastSeen = fmtDuration(time.Since(n.LastSeen))
				}
				trusted := "✗"
				if n.HostKeyKnown {
					trusted = "✓"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s ago\t%s\n",
					n.Spec.Name, n.Spec.Host, n.Spec.User,
					statusIcon(n.Status)+string(n.Status),
					lastSeen, trusted,
				)
			}
			return w.Flush()
		},
	}
}

func newNodesInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <name>",
		Short: "Show detailed info for a node",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt := FromContext(cmd.Context())
			registry := remote.NewRegistry(rt.State)
			info, err := registry.Get(args[0])
			if err != nil {
				return err
			}
			data, _ := json.MarshalIndent(info, "", "  ")
			fmt.Println(string(data))
			return nil
		},
	}
}

func newNodesTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test <name>",
		Short: "Test SSH connectivity to a node",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt := FromContext(cmd.Context())
			registry := remote.NewRegistry(rt.State)
			info, err := registry.Get(args[0])
			if err != nil {
				return err
			}

			pool := remote.NewPool(rt.Log)
			defer pool.Close()

			fmt.Printf("◉ Testing SSH connection to %s (%s@%s)...\n",
				info.Spec.Name, info.Spec.User, info.Spec.Host)

			out, code, err := pool.Run(cmd.Context(), info, "echo orbit-ok && uname -sr")
			if err != nil {
				return fmt.Errorf("connection test failed: %w", err)
			}
			_ = code
			fmt.Printf("✓ Connection successful\n  Remote: %s\n", out)
			return nil
		},
	}
}

func newNodesTrustCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "trust <name>",
		Short: "Record the host key fingerprint for a node (enables strict verification)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt := FromContext(cmd.Context())
			registry := remote.NewRegistry(rt.State)

			info, err := registry.Get(args[0])
			if err != nil {
				return err
			}

			addr := fmt.Sprintf("%s:%d", info.Spec.Host, info.Spec.Port)
			if info.Spec.Port == 0 {
				addr = fmt.Sprintf("%s:22", info.Spec.Host)
			}

			fmt.Printf("◉ Gathering host key from %s...\n", addr)
			key, err := sshutil.GatherHostKey(addr, 10*time.Second)
			if err != nil {
				return fmt.Errorf("gather host key: %w", err)
			}

			fingerprint := sshutil.FingerprintMD5(key)
			encodedKey := sshutil.EncodeHostKey(info.Spec.Host, key)

			fmt.Printf("  Fingerprint: %s\n", fingerprint)
			fmt.Printf("  Type:        %s\n", key.Type())
			fmt.Print("  Trust this key? [y/N] ")

			var answer string
			fmt.Scanln(&answer)
			if answer != "y" && answer != "Y" {
				fmt.Println("Aborted.")
				return nil
			}

			if err := registry.Trust(args[0], fingerprint, encodedKey); err != nil {
				return err
			}
			fmt.Printf("✓ Host key for %q trusted\n", args[0])
			return nil
		},
	}
}

// parseUserAtHost splits "user@host" into its parts.
func parseUserAtHost(s string) (user, host string) {
	for i, c := range s {
		if c == '@' {
			return s[:i], s[i+1:]
		}
	}
	return "root", s
}

func statusIcon(s v1.NodeStatus) string {
	switch s {
	case v1.NodeOnline:
		return "● "
	case v1.NodeDegraded:
		return "◐ "
	default:
		return "○ "
	}
}

func fmtDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}
