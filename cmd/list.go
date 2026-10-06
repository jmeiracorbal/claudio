package cmd

import (
	"fmt"
	"sort"

	"github.com/jmeiracorbal/claudio/internal/config"
	"github.com/jmeiracorbal/claudio/internal/profile"
	"github.com/jmeiracorbal/claudio/internal/resolver"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:          "list",
	Aliases:      []string{"ls"},
	Short:        "List all profiles",
	Args:         cobra.NoArgs,
	RunE:         runList,
	SilenceUsage: true,
}

func runList(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if len(cfg.Profiles) == 0 {
		fmt.Println("No profiles found. Run: claudio create <name>")
		return nil
	}

	active, _ := resolver.Resolve(cfg)

	names := make([]string, 0, len(cfg.Profiles))
	for n := range cfg.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)

	maxLen := 4
	for _, n := range names {
		if len(n) > maxLen {
			maxLen = len(n)
		}
	}

	fmt.Printf("  %-*s  %-8s  %s\n", maxLen, "NAME", "STATUS", "ACTIVE")
	fmt.Printf("  %-*s  %-8s  %s\n", maxLen, "----", "------", "------")
	for _, n := range names {
		p := cfg.Profiles[n]
		dir := config.ExpandPath(p.ConfigDir)
		status := profile.Status(dir)
		marker := ""
		if n == active {
			marker = "*"
		}
		fmt.Printf("  %-*s  %-8s  %s\n", maxLen, n, status, marker)
	}
	return nil
}
