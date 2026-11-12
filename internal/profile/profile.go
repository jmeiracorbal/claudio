package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/jmeiracorbal/claudio/internal/config"
)

var profileNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// ValidateName keeps profile names portable and prevents path traversal when
// they are embedded in ~/.claudio/profiles/<name>/claude.
func ValidateName(name string) error {
	if !profileNamePattern.MatchString(name) {
		return fmt.Errorf("invalid profile name %q: use 1-64 letters, numbers, dots, underscores or hyphens, starting with a letter or number", name)
	}
	return nil
}

func DefaultConfigDir(name string) (string, error) {
	if err := ValidateName(name); err != nil {
		return "", err
	}
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
	removePath := dir

	// Profiles created by claudio use a wrapper directory
	// ~/.claudio/profiles/<name>/. Remove the complete wrapper so deleting a
	// profile does not leave an empty directory behind. Custom/legacy config
	// paths are limited to their configured directory.
	if expectedDir, err := DefaultConfigDir(name); err == nil &&
		filepath.Clean(dir) == filepath.Clean(expectedDir) {
		removePath = filepath.Dir(expectedDir)
	}

	if err := os.RemoveAll(removePath); err != nil {
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

// Email returns the account email Claude Code recorded in the profile's
// .claude.json, or "" before the first login.
func Email(configDir string) string {
	var cj struct {
		OauthAccount struct {
			EmailAddress string `json:"emailAddress"`
		} `json:"oauthAccount"`
	}
	data, err := os.ReadFile(filepath.Join(configDir, ".claude.json"))
	if err != nil || json.Unmarshal(data, &cj) != nil {
		return ""
	}
	return cj.OauthAccount.EmailAddress
}
