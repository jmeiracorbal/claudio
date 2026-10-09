package cmd

import (
	"fmt"

	"github.com/jmeiracorbal/claudio/internal/config"
	"github.com/jmeiracorbal/claudio/internal/launcher"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec <name> -- <command> [args...]",
	Short: "Run any command with CLAUDE_CONFIG_DIR set to a profile",
	Long: `Run a command with CLAUDE_CONFIG_DIR pointing at the named profile, without
launching Claude. Use it for installers that write into the Claude config
directory (skills, MCP servers, hooks, instructions) so they target the profile
instead of ~/.claude. Only installers that honor CLAUDE_CONFIG_DIR are affected.

Examples:
  claudio exec work -- npx skills add <owner/repo> -g
  claudio exec work -- mnemo setup refresh --agent=claudecode
  claudio exec work -- sh -c 'echo $CLAUDE_CONFIG_DIR'`,
	Args:               cobra.ArbitraryArgs,
	RunE:               runExec,
	SilenceUsage:       true,
	DisableFlagParsing: true,
}

func runExec(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: claudio exec <name> -- <command> [args...]")
	}
	name := args[0]
	cmdArgs := args[1:]

	// strip leading "--" separator
	if len(cmdArgs) > 0 && cmdArgs[0] == "--" {
		cmdArgs = cmdArgs[1:]
	}
	if len(cmdArgs) == 0 {
		return fmt.Errorf("usage: claudio exec <name> -- <command> [args...]")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	p, ok := cfg.Profiles[name]
	if !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	return launcher.Exec(config.ExpandPath(p.ConfigDir), cmdArgs[0], cmdArgs[1:])
}
