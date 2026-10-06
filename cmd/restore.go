package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jmeiracorbal/claudio/internal/config"
	"github.com/jmeiracorbal/claudio/internal/profile"
	"github.com/spf13/cobra"
)

var restoreFromProfile string

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore the origin account back to ~/.claude",
	Long: `Restore copies a claudio profile back to ~/.claude, reversing a migration.

By default it uses the profile marked as origin_account=true.
Use --from-profile to restore a specific profile instead.

If ~/.claude already exists it is backed up as ~/.claude.bak.<timestamp>
before being replaced.

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

	if _, err := os.Stat(dst); err == nil {
		bak := dst + ".bak." + time.Now().Format("20060102150405")
		if err := os.Rename(dst, bak); err != nil {
			return fmt.Errorf("backing up %s: %w", dst, err)
		}
		fmt.Printf("Backed up existing ~/.claude → %s\n", bak)
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("creating ~/.claude: %w", err)
	}
	if err := profile.CopyDir(src, dst); err != nil {
		return fmt.Errorf("copying files: %w", err)
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
