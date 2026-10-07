package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmeiracorbal/claudio/internal/config"
	"github.com/jmeiracorbal/claudio/internal/profile"
	"github.com/spf13/cobra"
)

var (
	copyToProfile string
	copyFromLocal bool
	copyFrom      string
	copyHistory   bool
)

var copyCmd = &cobra.Command{
	Use:   "copy --to-profile=<name> (--from-local | --from=<profile>) [--with-history]",
	Short: "Copy a Claude configuration into a new profile",
	Long: `Copy creates a new profile from a copy of an existing Claude
configuration: settings, CLAUDE.md, plugins, skills, hooks and MCP servers.
History, conversations and sessions are personal data of the source account
and are copied only with --with-history (for example, when the new profile
will use the same account). The account identity is never copied. After the
copy the profile is independent: nothing is shared or kept in sync.
What is copied is defined in claude.toml ([copy]).

--from-local copies your local installation (~/.claude and ~/.claude.json)
and marks the profile as the origin account, the default for claudio restore.
--from copies another claudio profile.

Paths that point into the source (hook commands, plugin locations) are
rewritten to the new profile. Login tokens live in the system keychain per
config directory and are not copied: run claudio login <name> afterwards.
The source is never modified.

Examples:
  claudio copy --to-profile=personal --from-local --with-history
  claudio copy --to-profile=work-2 --from=work`,
	Args:         cobra.NoArgs,
	RunE:         runCopy,
	SilenceUsage: true,
}

func init() {
	copyCmd.Flags().StringVar(&copyToProfile, "to-profile", "", "name of the new profile")
	copyCmd.Flags().BoolVar(&copyFromLocal, "from-local", false, "copy the local installation (~/.claude)")
	copyCmd.Flags().StringVar(&copyFrom, "from", "", "copy an existing claudio profile")
	copyCmd.Flags().BoolVar(&copyHistory, "with-history", false, "also copy history, conversations and sessions")
	_ = copyCmd.MarkFlagRequired("to-profile")
	copyCmd.MarkFlagsMutuallyExclusive("from-local", "from")
	copyCmd.MarkFlagsOneRequired("from-local", "from")
}

func runCopy(_ *cobra.Command, _ []string) error {
	name := copyToProfile
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if _, exists := cfg.Profiles[name]; exists {
		return fmt.Errorf("profile %q already exists", name)
	}

	src, state, err := copySource(cfg)
	if err != nil {
		return err
	}
	if copyFromLocal {
		if existing, ok := profile.FindOriginAccount(cfg); ok {
			return fmt.Errorf("the local installation is already copied as origin account %q", existing)
		}
	}

	dst, err := profile.DefaultConfigDir(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("creating profile directory: %w", err)
	}
	if err := profile.CopyConfig(src, dst, copyHistory); err != nil {
		return err
	}
	if err := profile.CopyState(state, dst, copyHistory); err != nil {
		return err
	}
	if err := profile.RewritePaths(dst, src); err != nil {
		return fmt.Errorf("rewriting paths: %w", err)
	}

	cfg.Profiles[name] = config.Profile{ConfigDir: dst, OriginAccount: copyFromLocal}
	if err := cfg.Save(); err != nil {
		return err
	}

	fmt.Printf("Copied %s → %s\n", src, dst)
	fmt.Printf("Profile %q created.\n", name)
	fmt.Printf("\nTo authenticate: claudio login %s\n", name)
	return nil
}

// copySource returns the config dir to copy and its .claude.json, which for
// the local installation lives outside it (~/.claude.json).
func copySource(cfg *config.Config) (dir, state string, err error) {
	if copyFrom != "" {
		dir, err = cfg.ProfileConfigDir(copyFrom)
		return dir, filepath.Join(dir, ".claude.json"), err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	dir = filepath.Join(home, ".claude")
	if _, err := os.Stat(dir); err != nil {
		return "", "", fmt.Errorf("local installation not found: %s", dir)
	}
	return dir, filepath.Join(home, ".claude.json"), nil
}
