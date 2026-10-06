package cmd

import (
	"fmt"
	"os"

	"github.com/jmeiracorbal/claudio/internal/config"
	"github.com/jmeiracorbal/claudio/internal/resolver"
	"github.com/spf13/cobra"
)

const pinFile = ".claudio-account"

var pinCmd = &cobra.Command{
	Use:   "pin <name>",
	Short: "Pin a profile to the current directory (creates .claudio-account)",
	Long: `Pin a profile to the current directory by creating a .claudio-account file.
claudio will use this profile whenever it runs from this directory or any
subdirectory, before checking configured path rules.

To remove the nearest pin, delete .claudio-account or run: claudio pin unpin`,
	Args:         cobra.ExactArgs(1),
	RunE:         runPin,
	SilenceUsage: true,
}

var unpinCmd = &cobra.Command{
	Use:          "unpin",
	Short:        "Remove the nearest project profile pin",
	Args:         cobra.NoArgs,
	RunE:         runUnpin,
	SilenceUsage: true,
}

func init() {
	pinCmd.AddCommand(unpinCmd)
}

func runPin(_ *cobra.Command, args []string) error {
	name := args[0]
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if _, ok := cfg.Profiles[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	if err := os.WriteFile(pinFile, []byte(name+"\n"), 0644); err != nil {
		return err
	}
	fmt.Printf("Pinned: this directory will use profile %q.\n", name)
	fmt.Printf("Tip: add %s to .gitignore if you do not want it committed.\n", pinFile)
	return nil
}

func runUnpin(_ *cobra.Command, _ []string) error {
	path, err := resolver.ProjectOverridePath()
	if err != nil {
		fmt.Println("No .claudio-account file found in this project.")
		return nil
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	fmt.Println("Pin removed.")
	return nil
}
