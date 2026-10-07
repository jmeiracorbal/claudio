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
	Short: "Restore a profile back to the local installation (~/.claude)",
	Long: `Restore copies a claudio profile back to the local installation: its config
directory to ~/.claude and its .claude.json to ~/.claude.json, rewriting the
paths that pointed into the profile. It reverses claudio copy --from-local.

By default it uses the origin account (the profile copied with --from-local).
Use --from-profile to restore a specific profile instead.

Existing ~/.claude and ~/.claude.json are backed up as <name>.bak.<timestamp>
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

	stateDst := filepath.Join(home, ".claude.json")
	stamp := time.Now().Format("20060102150405")
	for _, path := range []string{dst, stateDst} {
		if _, err := os.Stat(path); err == nil {
			bak := path + ".bak." + stamp
			if err := os.Rename(path, bak); err != nil {
				return fmt.Errorf("backing up %s: %w", path, err)
			}
			fmt.Printf("Backed up existing %s → %s\n", path, bak)
		}
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("creating ~/.claude: %w", err)
	}
	if err := profile.CopyDir(src, dst); err != nil {
		return fmt.Errorf("copying files: %w", err)
	}
	if err := profile.RewritePaths(dst, src); err != nil {
		return fmt.Errorf("rewriting paths: %w", err)
	}
	// Outside a profile Claude reads ~/.claude.json, not ~/.claude/.claude.json.
	if state := filepath.Join(dst, ".claude.json"); fileExists(state) {
		if err := os.Rename(state, stateDst); err != nil {
			return fmt.Errorf("moving %s: %w", state, err)
		}
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
