package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jmeiracorbal/claudio/internal/config"
	"github.com/jmeiracorbal/claudio/internal/profile"
	restorepkg "github.com/jmeiracorbal/claudio/internal/restore"
	"github.com/spf13/cobra"
)

var restoreFromProfile string

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore a profile back to the local installation (~/.claude)",
	Long: `Restore copies a claudio profile back to the local installation: its config
directory to ~/.claude and its .claude.json to ~/.claude.json, rewriting the
paths that pointed into the profile. It reverses claudio copy --from-local.

By default it uses the origin account (the profile copied with --from-local).
Use --from-profile to restore a specific profile instead.

Existing ~/.claude and ~/.claude.json are backed up as <name>.bak.<timestamp>
before being replaced. If any restore step fails after a backup is created,
claudio rolls the local installation back before returning the error.

Examples:
  claudio restore
  claudio restore --from-profile=personal`,
	Args:         cobra.NoArgs,
	RunE:         runRestore,
	SilenceUsage: true,
}

func init() {
	restoreCmd.Flags().StringVar(&restoreFromProfile, "from-profile", "", "profile to restore (defaults to the origin account)")
}

func runRestore(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	profileName, err := resolveRestoreProfile(cfg)
	if err != nil {
		return err
	}

	src, err := cfg.ProfileConfigDir(profileName)
	if err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dst := filepath.Join(home, ".claude")
	stateDst := filepath.Join(home, ".claude.json")

	result, err := restorepkg.Apply(src, dst, stateDst, time.Now())
	if err != nil {
		return err
	}

	if result.ConfigBackup != "" {
		fmt.Printf("Backed up existing %s → %s\n", dst, result.ConfigBackup)
	}
	if result.StateBackup != "" {
		fmt.Printf("Backed up existing %s → %s\n", stateDst, result.StateBackup)
	}
	fmt.Printf("Restored profile %q → %s\n", profileName, dst)
	return nil
}

func resolveRestoreProfile(cfg *config.Config) (string, error) {
	if restoreFromProfile != "" {
		if _, ok := cfg.Profiles[restoreFromProfile]; !ok {
			return "", fmt.Errorf("profile %q not found", restoreFromProfile)
		}
		return restoreFromProfile, nil
	}
	name, ok := profile.FindOriginAccount(cfg)
	if !ok {
		return "", fmt.Errorf("no origin account found — use --from-profile=<name>")
	}
	return name, nil
}
