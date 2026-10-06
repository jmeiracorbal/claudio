package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmeiracorbal/claudio/internal/config"
	"github.com/jmeiracorbal/claudio/internal/profile"
	"github.com/spf13/cobra"
)

var migrateExternalPath string

var migrateCmd = &cobra.Command{
	Use:   "migrate <name>",
	Short: "Migrate an existing Claude account into a claudio profile",
	Long: `Migrate copies an existing Claude config directory into a new claudio profile
and marks it as the origin account (origin_account=true).

By default the source is ~/.claude. Use --from-external-path to migrate
a config directory at a non-standard location.

The original directory is never modified or deleted.

Examples:
  claudio migrate personal
  claudio migrate work --from-external-path=/opt/claude-configs/work`,
	Args:         cobra.ExactArgs(1),
	RunE:         runMigrate,
	SilenceUsage: true,
}

func init() {
	migrateCmd.Flags().StringVar(&migrateExternalPath, "from-external-path", "", "source Claude config directory (defaults to ~/.claude)")
}

func runMigrate(_ *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if _, exists := cfg.Profiles[name]; exists {
		return fmt.Errorf("profile %q already exists", name)
	}
	if existing, ok := profile.FindOriginAccount(cfg); ok {
		return fmt.Errorf("origin account already migrated as profile %q — only one origin account is allowed", existing)
	}

	src, err := resolveSourceDir(migrateExternalPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return fmt.Errorf("source directory not found: %s", src)
	}

	dst, err := profile.DefaultConfigDir(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("creating profile directory: %w", err)
	}

	if err := profile.CopyDir(src, dst); err != nil {
		return fmt.Errorf("copying files: %w", err)
	}

	cfg.Profiles[name] = config.Profile{
		ConfigDir:     dst,
		OriginAccount: true,
	}
	if err := cfg.Save(); err != nil {
		return err
	}

	fmt.Printf("Migrated %q → %s\n", src, dst)
	fmt.Printf("Profile %q registered as origin account.\n", name)
	fmt.Printf("\nOriginal directory left intact: %s\n", src)
	fmt.Printf("To launch: claudio %s\n", name)
	return nil
}

func resolveSourceDir(flag string) (string, error) {
	if flag != "" {
		return config.ExpandPath(flag), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude"), nil
}
