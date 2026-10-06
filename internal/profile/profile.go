package profile

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmeiracorbal/claudio/internal/config"
)

func DefaultConfigDir(name string) (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "profiles", name, "claude"), nil
}

// Create sets up the profile directory and registers it in config.
// By default, settings.json and hooks/ are symlinked from ~/.claude so the
// new profile shares your existing configuration.
// Pass isolated=true to create a fully independent profile with no symlinks.
func Create(name string, isolated bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if _, exists := cfg.Profiles[name]; exists {
		return fmt.Errorf("profile %q already exists", name)
	}

	profileDir, err := DefaultConfigDir(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		return fmt.Errorf("creating profile directory: %w", err)
	}

	cfg.Profiles[name] = config.Profile{ConfigDir: profileDir}

	if !isolated {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		claudeDir := filepath.Join(home, ".claude")
		if err := linkShared(claudeDir, profileDir); err != nil {
			return err
		}
	}

	return cfg.Save()
}

// linkShared creates symlinks for settings.json and hooks/ from srcDir into dstDir.
func linkShared(srcDir, dstDir string) error {
	for _, entry := range []string{"settings.json", "hooks"} {
		src := filepath.Join(srcDir, entry)
		dst := filepath.Join(dstDir, entry)
		os.Remove(dst)
		if err := os.Symlink(src, dst); err != nil {
			return fmt.Errorf("symlinking %s: %w", entry, err)
		}
	}
	return nil
}

func Remove(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	p, ok := cfg.Profiles[name]
	if !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	dir := config.ExpandPath(p.ConfigDir)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("removing profile directory: %w", err)
	}
	delete(cfg.Profiles, name)
	return cfg.Save()
}

func Rename(old, newName string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	p, ok := cfg.Profiles[old]
	if !ok {
		return fmt.Errorf("profile %q not found", old)
	}
	if _, exists := cfg.Profiles[newName]; exists {
		return fmt.Errorf("profile %q already exists", newName)
	}

	oldDir := config.ExpandPath(p.ConfigDir)
	newDir, err := DefaultConfigDir(newName)
	if err != nil {
		return err
	}
	if err := os.Rename(oldDir, newDir); err != nil {
		return fmt.Errorf("renaming directory: %w", err)
	}

	cfg.Profiles[newName] = config.Profile{ConfigDir: newDir}
	delete(cfg.Profiles, old)

	for i, rule := range cfg.Rules {
		if rule.Profile == old {
			cfg.Rules[i].Profile = newName
		}
	}

	return cfg.Save()
}

// FindOriginAccount returns the name of the profile marked as origin_account=true.
func FindOriginAccount(cfg *config.Config) (string, bool) {
	for name, p := range cfg.Profiles {
		if p.OriginAccount {
			return name, true
		}
	}
	return "", false
}

func Status(configDir string) string {
	entries, err := os.ReadDir(configDir)
	if os.IsNotExist(err) {
		return "missing"
	}
	if err != nil {
		return "error"
	}
	if len(entries) == 0 {
		return "created"
	}
	return "ready"
}
