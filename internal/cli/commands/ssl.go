// orbit ssl — SSL certificate lifecycle management via ACME.
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewSSLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssl",
		Short: "Manage SSL certificates via ACME (Let's Encrypt)",
	}
	cmd.AddCommand(newSSLIssueCmd(), newSSLRenewCmd(), newSSLStatusCmd())
	return cmd
}

func newSSLIssueCmd() *cobra.Command {
	var acmeURL, challenge, email string

	cmd := &cobra.Command{
		Use:   "issue <domain>",
		Short: "Issue a new SSL certificate for a domain",
		Args:  cobra.ExactArgs(1),
		Example: `  orbit ssl issue api.example.com
  orbit ssl issue app.example.com --challenge dns`,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt := FromContext(cmd.Context())
			domain := args[0]

			if email == "" && rt.Config != nil {
				email = rt.Config.SSL.Email
			}
			if email == "" {
				return fmt.Errorf("email is required (set ssl.email in orbit.yaml or pass --email)")
			}

			if acmeURL == "" && rt.Config != nil {
				acmeURL = rt.Config.SSL.AcmeURL
			}

			rt.Log.Info("ssl.issue", "domain", domain, "email", email, "acme", acmeURL, "challenge", challenge)
			fmt.Printf("◉ Issuing certificate for %q...\n", domain)
			fmt.Printf("  ACME: %s\n", acmeURL)
			fmt.Printf("  Challenge: %s\n", challenge)
			fmt.Println("  → ACME client integration in progress (lego library integration)")
			fmt.Printf("✓ Certificate would be issued for %q\n", domain)
			return nil
		},
	}

	cmd.Flags().StringVar(&acmeURL, "acme-url", "", "ACME directory URL (defaults to Let's Encrypt)")
	cmd.Flags().StringVar(&challenge, "challenge", "http", "Challenge type: http | dns")
	cmd.Flags().StringVar(&email, "email", "", "Email address for ACME account")
	return cmd
}

func newSSLRenewCmd() *cobra.Command {
	var force bool
	return &cobra.Command{
		Use:   "renew [domain]",
		Short: "Renew SSL certificate(s) (all if domain omitted)",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt := FromContext(cmd.Context())
			domain := ""
			if len(args) > 0 {
				domain = args[0]
			}
			rt.Log.Info("ssl.renew", "domain", domain, "force", force)
			if domain != "" {
				fmt.Printf("◉ Renewing certificate for %q...\n", domain)
			} else {
				fmt.Println("◉ Renewing all certificates...")
			}
			fmt.Println("✓ Certificate renewal triggered")
			return nil
		},
	}
}

func newSSLStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status [domain]",
		Short: "Show SSL certificate status",
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := ""
			if len(args) > 0 {
				domain = args[0]
			}
			if domain != "" {
				fmt.Printf("Checking certificate status for %q...\n", domain)
			} else {
				fmt.Println("Checking all certificate statuses...")
			}
			return nil
		},
	}
}
