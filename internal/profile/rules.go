package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/jmeiracorbal/claudio/internal/config"
)

// Rules mirrors claude.toml at the repository root.
type Rules struct {
	Launch struct {
		ClaudeMdExcludes []string `toml:"claude_md_excludes"`
	} `toml:"launch"`
	Copy struct {
		History          []string `toml:"history"`
		Exclude          []string `toml:"exclude"`
		StateKeys        []string `toml:"state_keys"`
		HistoryStateKeys []string `toml:"history_state_keys"`
		RewritePathsIn   []string `toml:"rewrite_paths_in"`
	} `toml:"copy"`
}

var rules Rules

// LoadRules parses claude.toml. main embeds the file and calls this at start.
func LoadRules(data string) error {
	if _, err := toml.Decode(data, &rules); err != nil {
		return fmt.Errorf("parsing claude.toml: %w", err)
	}
	return nil
}

// LaunchSettings returns the --settings JSON passed to Claude on every launch.
func LaunchSettings() (string, error) {
	excludes := make([]string, 0, len(rules.Launch.ClaudeMdExcludes))
	for _, p := range rules.Launch.ClaudeMdExcludes {
		excludes = append(excludes, config.ExpandPath(p))
	}
	out, err := json.Marshal(map[string][]string{"claudeMdExcludes": excludes})
	return string(out), err
}

// RewritePaths points the absolute paths to fromDir stored in toDir's files
// (rules [copy] rewrite_paths_in) at toDir instead. "~/" and "$HOME/"
// spellings of fromDir are rewritten too.
func RewritePaths(toDir, fromDir string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	from := filepath.Clean(fromDir) + "/"
	to := filepath.Clean(toDir) + "/"
	pairs := []string{from, to}
	if rel, ok := strings.CutPrefix(from, home+"/"); ok {
		for _, h := range []string{"~/", "$HOME/", "${HOME}/"} {
			pairs = append(pairs, h+rel, to)
		}
	}
	replacer := strings.NewReplacer(pairs...)
	for _, name := range rules.Copy.RewritePathsIn {
		path := filepath.Join(toDir, filepath.FromSlash(name))
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		out := replacer.Replace(string(data))
		if out == string(data) {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(out), info.Mode().Perm()); err != nil {
			return err
		}
	}
	return nil
}

// CopyConfig copies the Claude config directory src into dst following
// claude.toml [copy]: excluded entries are skipped, history entries only
// travel with withHistory, and everything else (configuration) is copied.
// .claude.json is handled by CopyState.
func CopyConfig(src, dst string, withHistory bool) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if name == stateFile || matchAny(rules.Copy.Exclude, name) ||
			!withHistory && matchAny(rules.Copy.History, name) {
			continue
		}
		if err := CopyDir(filepath.Join(src, name), filepath.Join(dst, name)); err != nil {
			return fmt.Errorf("copying %s: %w", name, err)
		}
	}
	return nil
}

// CopyState writes to dst/.claude.json the keys of the .claude.json at
// srcFile listed in claude.toml [copy] state_keys (plus history_state_keys
// with withHistory). A missing srcFile is not an error.
func CopyState(srcFile, dst string, withHistory bool) error {
	data, err := os.ReadFile(srcFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var src map[string]any
	if err := json.Unmarshal(data, &src); err != nil {
		return fmt.Errorf("parsing %s: %w", srcFile, err)
	}
	keys := rules.Copy.StateKeys
	if withHistory {
		keys = append(append([]string{}, keys...), rules.Copy.HistoryStateKeys...)
	}
	out := map[string]any{}
	for _, k := range keys {
		if v, ok := src[k]; ok {
			out[k] = v
		}
	}
	encoded, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dst, stateFile), encoded, 0600)
}

// stateFile is Claude's global state file inside a config directory.
const stateFile = ".claude.json"

func matchAny(patterns []string, name string) bool {
	for _, p := range patterns {
		if ok, _ := filepath.Match(p, name); ok {
			return true
		}
	}
	return false
}
