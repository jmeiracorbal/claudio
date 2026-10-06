package main

import (
	"fmt"
	"os"

	"github.com/jmeiracorbal/claudio/cmd"
	"github.com/jmeiracorbal/claudio/internal/config"
	"github.com/jmeiracorbal/claudio/internal/launcher"
)

var version = "dev"

func main() {
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

func isFlag(s string) bool {
	return len(s) > 0 && s[0] == '-'
}

func isSubcommand(s string) bool {
	subcommands := []string{
		"create", "list", "ls", "current", "manage",
		"run", "login", "remove", "rm", "delete",
		"rename", "switch", "pin", "unpin", "doctor",
		"help", "completion",
	}
	for _, c := range subcommands {
		if s == c {
			return true
		}
	}
	return false
}
