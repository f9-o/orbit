// Package cli defines the root Cobra command and global flag/context setup.
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/f9-o/orbit/internal/cli/commands"
	"github.com/f9-o/orbit/internal/core/config"
	"github.com/f9-o/orbit/internal/core/logger"
	"github.com/f9-o/orbit/internal/core/state"
	"github.com/f9-o/orbit/pkg/pprint"
)

// globalFlags holds values bound to persistent global flags.
var globalFlags struct {
	configFile string
	node       string
	debug      bool
	jsonOutput bool
	dryRun     bool
}

// rootCmd is the base command for orbit.
var rootCmd = &cobra.Command{
	Use:           "orbit",
	Short:         "Orbit — Container Orchestration from the Terminal",
	Long:          ``, // overridden by SetHelpTemplate below
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Bare `orbit` — help func already prints banner
		return cmd.Help()
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "version" || cmd.Name() == "completion" {
			return nil
		}
		return initRuntime(cmd)
	},
}

// Execute runs the CLI. Called by main().
func Execute() {
	// Show banner before every help screen
	origHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		pprint.PrintBanner(commands.Version, commands.BuildDate)
		origHelp(cmd, args)
	})

	if err := rootCmd.Execute(); err != nil {
		pprint.Error("%s", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&globalFlags.configFile, "config", "c", "", "Path to orbit.yaml (defaults to auto-discovery)")
	rootCmd.PersistentFlags().StringVarP(&globalFlags.node, "node", "n", "", "Target node name (overrides config)")
	rootCmd.PersistentFlags().BoolVar(&globalFlags.debug, "debug", false, "Enable debug-level logging")
	rootCmd.PersistentFlags().BoolVar(&globalFlags.jsonOutput, "json", false, "Output in machine-readable JSON")
	rootCmd.PersistentFlags().BoolVar(&globalFlags.dryRun, "dry-run", false, "Print planned actions without executing")

	// Register all subcommands
	rootCmd.AddCommand(
		commands.NewInitCmd(),
		commands.NewUpCmd(),
		commands.NewDownCmd(),
		commands.NewDeployCmd(),
		commands.NewLogsCmd(),
		commands.NewNodesCmd(),
		commands.NewScaleCmd(),
		commands.NewSSLCmd(),
		commands.NewMonitorCmd(),
		commands.NewUICmd(),
		commands.NewVersionCmd(),
	)
}

// initRuntime loads config, logger, and state before each command runs.
func initRuntime(cmd *cobra.Command) error {
	// Load config
	cfg, err := config.Load(globalFlags.configFile)
	if err != nil && globalFlags.configFile != "" {
		return fmt.Errorf("config: %w", err)
	}
	if cfg == nil {
		cfg = &config.Config{}
	}

	// Initialise logger
	orbitHome := config.OrbitHome()
	logFile := filepath.Join(orbitHome, "logs", "orbit.log")
	if err := os.MkdirAll(filepath.Dir(logFile), 0750); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	logLevel := "info"
	logFormat := "text"
	if cfg != nil {
		logLevel = cfg.Log.Level
		logFormat = cfg.Log.Format
	}

	log, err := logger.Init(logLevel, logFormat, logFile, orbitHome, globalFlags.debug)
	if err != nil {
		return fmt.Errorf("logger init: %w", err)
	}

	// Open state DB
	dbPath := filepath.Join(orbitHome, "state.db")
	if err := os.MkdirAll(orbitHome, 0750); err != nil {
		return fmt.Errorf("create orbit home: %w", err)
	}
	db, err := state.Open(dbPath)
	if err != nil {
		return fmt.Errorf("state db: %w", err)
	}

	// Store in command context
	cmd.SetContext(commands.NewContext(cmd.Context(), &commands.Runtime{
		Config: cfg,
		Log:    log,
		State:  db,
		Flags: commands.GlobalFlags{
			Node:       globalFlags.node,
			Debug:      globalFlags.debug,
			JSONOutput: globalFlags.jsonOutput,
			DryRun:     globalFlags.dryRun,
		},
	}))

	return nil
}
