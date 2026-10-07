package cmd

import (
	"fmt"

	"github.com/jmeiracorbal/claudio/internal/config"
	"github.com/jmeiracorbal/claudio/internal/profile"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new profile",
	Long: `Create a new, fully independent Claude Code profile.

The profile starts empty: Claude builds its configuration directory on first
launch, exactly as it does for ~/.claude. Plugins, skills, MCP servers and
settings you install from the profile stay in it. To start from an existing
configuration instead, use claudio copy.

Examples:
  claudio create personal
  claudio create work`,
	Args:         cobra.ExactArgs(1),
	RunE:         runCreate,
	SilenceUsage: true,
}

func runCreate(_ *cobra.Command, args []string) error {
	name := args[0]
	if err := profile.Create(name); err != nil {
		return err
	}

	cfg, _ := config.Load()
	dir, _ := cfg.ProfileConfigDir(name)

	fmt.Printf("Profile %q created.\n", name)
	fmt.Printf("  config dir: %s\n", dir)
	fmt.Printf("\nTo authenticate: claudio login %s\n", name)
	return nil
}
