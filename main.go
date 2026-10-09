package main

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/jmeiracorbal/claudio/cmd"
	"github.com/jmeiracorbal/claudio/internal/config"
	"github.com/jmeiracorbal/claudio/internal/launcher"
	"github.com/jmeiracorbal/claudio/internal/profile"
)

var version = "dev"

//go:embed claude.toml
var claudeTOML string

func main() {
	if err := profile.LoadRules(claudeTOML); err != nil {
		fmt.Fprintln(os.Stderr, "claudio:", err)
		os.Exit(1)
	}

	// Profile shortcut: claudio <profile-name> [claude-args...]
	// Intercepted before cobra so flags like --continue are forwarded to claude, not parsed here.
	if len(os.Args) > 1 {
		profileName := os.Args[1]
		if !isSubcommand(profileName) && !isFlag(profileName) {
			cfg, err := config.Load()
			if err == nil {
				if p, ok := cfg.Profiles[profileName]; ok {
					configDir := config.ExpandPath(p.ConfigDir)
					claudeArgs := os.Args[2:]
					if len(claudeArgs) > 0 && claudeArgs[0] == "--" {
						claudeArgs = claudeArgs[1:]
					}
					claudeArgs = launcher.WithModel(p.DefaultModel, claudeArgs)
					printLaunchInfo(profileName, configDir)
					if err := launcher.Launch(configDir, claudeArgs); err != nil {
						fmt.Fprintln(os.Stderr, "claudio:", err)
						os.Exit(1)
					}
					return
				}
			}
		}
	}

	cmd.Execute(version)
}

func printLaunchInfo(profileName, configDir string) {
	email := ""
	if e := profile.Email(configDir); e != "" {
		email = " (" + e + ")"
	}
	fmt.Fprintf(os.Stderr, "claudio: launching %q%s\n", profileName, email)
}

func isFlag(s string) bool {
	return len(s) > 0 && s[0] == '-'
}

func isSubcommand(s string) bool {
	subcommands := []string{
		"create", "list", "ls", "current", "manage",
		"run", "exec", "login", "remove", "rm", "delete",
		"rename", "switch", "pin", "unpin", "doctor",
		"model", "status", "help", "completion",
	}
	for _, c := range subcommands {
		if s == c {
			return true
		}
	}
	return false
}
