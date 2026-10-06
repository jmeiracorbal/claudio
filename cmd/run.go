package cmd

import (
	"fmt"

	"github.com/jmeiracorbal/claudio/internal/config"
	"github.com/jmeiracorbal/claudio/internal/launcher"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <name> [-- claude-args...]",
	Short: "Launch Claude with a specific profile without changing the default",
	Long: `Launch Claude with the named profile without changing the global default.
All arguments after -- are forwarded to claude.

Examples:
  claudio run work
  claudio run personal -- --continue
  claudio run work -- -p "review this repo"`,
	Args:               cobra.ArbitraryArgs,
	RunE:               runRun,
	SilenceUsage:       true,
	DisableFlagParsing: true,
}

func runRun(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: claudio run <name> [-- claude-args...]")
	}
	name := args[0]
	claudeArgs := args[1:]

	// strip leading "--" separator
	if len(claudeArgs) > 0 && claudeArgs[0] == "--" {
		claudeArgs = claudeArgs[1:]
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	configDir, err := cfg.ProfileConfigDir(name)
	if err != nil {
		return err
	}
	return launcher.Launch(configDir, claudeArgs)
}
