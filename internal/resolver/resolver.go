package resolver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmeiracorbal/claudio/internal/config"
)

const overrideFile = ".claudio-account"

// Resolve returns the profile selected by project overrides and directory rules.
func Resolve(cfg *config.Config) (name, reason string) {
	if name, err := projectOverride(); err == nil {
		if _, ok := cfg.Profiles[name]; ok {
			return name, fmt.Sprintf("project override (%s)", overrideFile)
		}
	}

	cwd, err := os.Getwd()
	if err == nil {
		for _, rule := range cfg.Rules {
			if matchPath(rule.Path, cwd) {
				if _, ok := cfg.Profiles[rule.Profile]; ok {
					return rule.Profile, fmt.Sprintf("directory rule (%s)", rule.Path)
				}
			}
		}
	}

	return "", ""
}

// ResolveEffective includes the sole-profile fallback used by no-argument
// launches and status views.
func ResolveEffective(cfg *config.Config) (name, reason string) {
	name, reason = Resolve(cfg)
	if name == "" && len(cfg.Profiles) == 1 {
		for only := range cfg.Profiles {
			return only, "only profile"
		}
	}
	return name, reason
}

// projectOverride looks for .claudio-account in the current dir up to the git root.
func projectOverride() (string, error) {
	path, err := ProjectOverridePath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(string(data))
	if name == "" {
		return "", fmt.Errorf("empty project override")
	}
	return name, nil
}

// ProjectOverridePath returns the nearest .claudio-account file, stopping at
// the Git root to match profile resolution.
func ProjectOverridePath() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		path := filepath.Join(dir, overrideFile)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		// stop at git root
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("not found")
}

// matchPath checks whether path falls under the given glob pattern.
// Supports ~ expansion and ** for recursive matching.
func matchPath(pattern, path string) bool {
	expanded := config.ExpandPath(pattern)

	if strings.HasSuffix(expanded, "/**") {
		prefix := strings.TrimSuffix(expanded, "/**")
		return path == prefix || strings.HasPrefix(path, prefix+string(filepath.Separator))
	}

	if strings.HasSuffix(expanded, "/*") {
		prefix := strings.TrimSuffix(expanded, "/*")
		rel := strings.TrimPrefix(path, prefix+string(filepath.Separator))
		return strings.HasPrefix(path, prefix+string(filepath.Separator)) &&
			!strings.Contains(rel, string(filepath.Separator))
	}

	matched, _ := filepath.Match(expanded, path)
	return matched || path == expanded
}
