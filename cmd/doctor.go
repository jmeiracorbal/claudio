package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jmeiracorbal/claudio/internal/config"
	"github.com/jmeiracorbal/claudio/internal/resolver"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:          "doctor",
	Short:        "Check the claudio setup and report potential issues",
	Args:         cobra.NoArgs,
	RunE:         runDoctor,
	SilenceUsage: true,
}

func runDoctor(_ *cobra.Command, _ []string) error {
	ok := true

	check := func(label, value, status string) {
		fmt.Printf("  %-22s %-30s %s\n", label, value, status)
	}
	fail := func(label, value, msg string) {
		check(label, value, "FAIL  "+msg)
		ok = false
	}
	warn := func(label, value, msg string) {
		check(label, value, "WARN  "+msg)
	}
	pass := func(label, value string) {
		check(label, value, "OK")
	}

	fmt.Println()

	// claude binary
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		fail("Claude binary", "not found", "install Claude Code")
	} else {
		// try to get version
		out, err := exec.Command(claudePath, "--version").Output()
		version := strings.TrimSpace(string(out))
		if err != nil || version == "" {
			version = claudePath
		}
		pass("Claude binary", version)
	}

	// claudio config dir
	dir, _ := config.Dir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fail("Config dir", dir, "does not exist — run claudio create <name>")
	} else {
		pass("Config dir", dir)
	}

	// config file
	cfg, err := config.Load()
	if err != nil {
		fail("Config file", "parse error", err.Error())
		fmt.Println()
		return nil
	}
	cfgPath, _ := config.Path()
	pass("Config file", cfgPath)

	// profiles
	fmt.Println()
	if len(cfg.Profiles) == 0 {
		warn("Profiles", "none", "run: claudio create <name>")
	} else {
		for name, p := range cfg.Profiles {
			expanded := config.ExpandPath(p.ConfigDir)
			if _, err := os.Stat(expanded); os.IsNotExist(err) {
				fail("Profile "+name, expanded, "directory missing")
			} else {
				checkSymlinks(name, expanded, check)
			}
		}
	}

	// active profile
	fmt.Println()
	active, reason := resolver.ResolveEffective(cfg)
	if active == "" {
		warn("Active profile", "none", "run: claudio <name> or claudio switch")
	} else {
		pass("Active profile", active+" ("+reason+")")
	}

	// project override in the current directory or a parent up to the git root
	cwd, _ := os.Getwd()
	if pinPath, err := resolver.ProjectOverridePath(); err == nil {
		data, _ := os.ReadFile(pinPath)
		pass("Project pin", strings.TrimSpace(string(data))+" ("+filepath.Dir(pinPath)+")")
	}

	// CLAUDE_CONFIG_DIR conflict
	if ext := os.Getenv("CLAUDE_CONFIG_DIR"); ext != "" {
		warn("CLAUDE_CONFIG_DIR", ext, "set externally — claudio will override it")
	}

	// conflicting .claude dirs in tree
	checkConflictingClaudeDir(cwd, warn)

	fmt.Println()
	if ok {
		fmt.Println("  Everything looks good.")
	} else {
		fmt.Println("  Some checks failed. Review the FAIL items above.")
	}
	fmt.Println()
	return nil
}

func checkSymlinks(profileName, dir string, log func(label, value, status string)) {
	for _, entry := range []string{"settings.json", "hooks"} {
		p := filepath.Join(dir, entry)
		info, err := os.Lstat(p)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, _ := os.Readlink(p)
			if _, err := os.Stat(p); os.IsNotExist(err) {
				log("Profile "+profileName+" "+entry, target, "WARN  symlink target missing")
			} else {
				log("Profile "+profileName+" "+entry, "→ "+target, "OK")
			}
		} else {
			log("Profile "+profileName+" "+entry, "own", "OK")
		}
	}
}

func checkConflictingClaudeDir(cwd string, warn func(string, string, string)) {
	home, _ := os.UserHomeDir()
	dir := cwd
	for {
		// ~/.claude is the source of truth — not a conflict
		if dir == home {
			break
		}
		claudeDir := filepath.Join(dir, ".claude")
		if _, err := os.Stat(claudeDir); err == nil {
			// Only warn if settings.json or hooks/ are real files (not symlinks).
			// A .claude with only CLAUDE.md or project commands is fine.
			for _, entry := range []string{"settings.json", "hooks"} {
				p := filepath.Join(claudeDir, entry)
				info, err := os.Lstat(p)
				if err != nil {
					continue
				}
				if info.Mode()&os.ModeSymlink == 0 {
					warn(".claude conflict", claudeDir+"/"+entry, "own "+entry+" may override CLAUDE_CONFIG_DIR (known Claude Code bug)")
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			break
		}
		dir = parent
	}
}
