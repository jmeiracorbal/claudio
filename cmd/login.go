package cmd

import (
	"fmt"
	"os"

	"github.com/jmeiracorbal/claudio/internal/config"
	"github.com/jmeiracorbal/claudio/internal/launcher"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login <name>",
	Short: "Open Claude with the named profile to complete authentication",
	Long: `Launch Claude with the given profile's isolated config directory.
If the profile is not yet authenticated, Claude will prompt you to log in.`,
	Args:         cobra.ExactArgs(1),
	RunE:         runLogin,
	SilenceUsage: true,
}

func runLogin(_ *cobra.Command, args []string) error {
	name := args[0]
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	configDir, err := cfg.ProfileConfigDir(name)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "claudio: opening %q — complete authentication in Claude\n", name)
	return launcher.Launch(configDir, nil)
}
