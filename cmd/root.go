package cmd

import (
	"fmt"
	"os"

	"github.com/jmeiracorbal/claudio/internal/config"
	"github.com/jmeiracorbal/claudio/internal/launcher"
	"github.com/jmeiracorbal/claudio/internal/resolver"
	"github.com/spf13/cobra"
)

var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "claudio",
	Short: "Claude Code Account Manager",
	Long: `claudio manages multiple Claude Code accounts, one independent profile each.

Quick start:
  claudio create personal
  claudio create work
  claudio copy --to-profile=personal --from-local

Profile shortcuts (from anywhere):
  claudio personal
  claudio work -- --continue
  claudio switch    (interactive selector)
  claudio manage    (full-screen Task Hub)`,
	Args:         cobra.NoArgs,
	RunE:         runRoot,
	SilenceUsage: true,
}

func Execute(version string) {
	Version = version
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(
		manageCmd,
		createCmd,
		copyCmd,
		restoreCmd,
		listCmd,
		currentCmd,
		runCmd,
		execCmd,
		loginCmd,
		removeCmd,
		renameCmd,
		switchCmd,
		pinCmd,
		doctorCmd,
		modelCmd,
		statusCmd,
	)
}

func runRoot(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if len(cfg.Profiles) == 0 {
		return fmt.Errorf("no profiles found — run: claudio create <name>")
	}

	profileName, reason := resolver.ResolveEffective(cfg)

	if profileName == "" {
		// Multiple profiles, no rule matched — hand off to interactive switch.
		return runSwitch(nil, nil)
	}

	p := cfg.Profiles[profileName]
	configDir := config.ExpandPath(p.ConfigDir)
	launchArgs := launcher.WithModel(p.DefaultModel, nil)
	fmt.Fprintf(os.Stderr, "claudio: launching %q (%s)\n", profileName, reason)
	return launcher.Launch(configDir, launchArgs)
}
