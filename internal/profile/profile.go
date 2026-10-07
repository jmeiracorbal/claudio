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

// Create registers a profile with an empty config directory. Claude builds
// its own layout there on first launch, so the profile is fully independent.
func Create(name string) error {
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
	return cfg.Save()
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
	if err := os.MkdirAll(filepath.Dir(newDir), 0755); err != nil {
		return fmt.Errorf("creating profile directory: %w", err)
	}
	if err := os.Rename(oldDir, newDir); err != nil {
		return fmt.Errorf("renaming directory: %w", err)
	}
	// Drop the old profiles/<name>/ wrapper; fails harmlessly if not empty.
	_ = os.Remove(filepath.Dir(oldDir))
	if err := RewritePaths(newDir, oldDir); err != nil {
		return fmt.Errorf("rewriting paths: %w", err)
	}

	p.ConfigDir = newDir
	cfg.Profiles[newName] = p
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
