package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmeiracorbal/claudio/internal/config"
	"github.com/jmeiracorbal/claudio/internal/profile"
	"github.com/spf13/cobra"
)

var createIsolated bool

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new profile",
	Long: `Create a new Claude Code profile.

By default, settings.json and hooks/ are symlinked from your existing ~/.claude
directory so the new profile shares your current configuration.

Use --isolated to create a fully independent profile with its own settings
and hooks, with no connection to ~/.claude.

Examples:
  claudio create personal
  claudio create work
  claudio create client --isolated`,
	Args:         cobra.ExactArgs(1),
	RunE:         runCreate,
	SilenceUsage: true,
}

func init() {
	createCmd.Flags().BoolVar(&createIsolated, "isolated", false, "create a fully independent profile (no symlinks to ~/.claude)")
}

func runCreate(_ *cobra.Command, args []string) error {
	name := args[0]
	if err := profile.Create(name, createIsolated); err != nil {
		return err
	}

	cfg, _ := config.Load()
	dir, _ := cfg.ProfileConfigDir(name)

	fmt.Printf("Profile %q created.\n", name)
	fmt.Printf("  config dir: %s\n", dir)

	if createIsolated {
		fmt.Println("  mode:       isolated (own settings.json and hooks/)")
	} else {
		home, _ := os.UserHomeDir()
		claudeDir := filepath.Join(home, ".claude")
		fmt.Printf("  settings.json → %s/settings.json\n", claudeDir)
		fmt.Printf("  hooks/        → %s/hooks/\n", claudeDir)
	}

	fmt.Printf("\nTo authenticate: claudio login %s\n", name)
	return nil
}
