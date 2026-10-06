package cmd

import (
	"fmt"

	"github.com/jmeiracorbal/claudio/internal/config"
	"github.com/jmeiracorbal/claudio/internal/resolver"
	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:          "current",
	Short:        "Show the active profile for the current directory",
	Args:         cobra.NoArgs,
	RunE:         runCurrent,
	SilenceUsage: true,
}

func runCurrent(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	name, reason := resolver.ResolveEffective(cfg)
	if name == "" {
		fmt.Println("No active profile — run: claudio switch")
		return nil
	}
	fmt.Printf("%s  (%s)\n", name, reason)
	return nil
}
