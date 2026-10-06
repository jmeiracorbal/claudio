package cmd

import (
	"fmt"

	"github.com/jmeiracorbal/claudio/internal/config"
	"github.com/spf13/cobra"
)

var modelUnset bool

var modelCmd = &cobra.Command{
	Use:   "model <name> [model]",
	Short: "Get or set the default model for a profile",
	Long: `Get or set the default model for a profile.

With no model argument, prints the current default model.
With a model argument, sets it as the default for that profile.
Use --unset to remove the default model.

Examples:
  claudio model personal
  claudio model personal claude-sonnet-4-6
  claudio model work claude-opus-4-5
  claudio model personal --unset`,
	Args:         cobra.RangeArgs(1, 2),
	RunE:         runModelCmd,
	SilenceUsage: true,
}

func init() {
	modelCmd.Flags().BoolVar(&modelUnset, "unset", false, "remove the default model for this profile")
}

func runModelCmd(_ *cobra.Command, args []string) error {
	name := args[0]
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	p, ok := cfg.Profiles[name]
	if !ok {
		return fmt.Errorf("profile %q not found", name)
	}

	if modelUnset {
		p.DefaultModel = ""
		cfg.Profiles[name] = p
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Printf("Profile %q: default model cleared.\n", name)
		return nil
	}

	if len(args) == 2 {
		model := args[1]
		p.DefaultModel = model
		cfg.Profiles[name] = p
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Printf("Profile %q: default model set to %q.\n", name, model)
		return nil
	}

	if p.DefaultModel == "" {
		fmt.Printf("Profile %q: no default model set.\n", name)
	} else {
		fmt.Printf("Profile %q: default model = %s\n", name, p.DefaultModel)
	}
	return nil
}
