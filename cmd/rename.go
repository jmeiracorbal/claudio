package cmd

import (
	"fmt"

	"github.com/jmeiracorbal/claudio/internal/profile"
	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:          "rename <old> <new>",
	Short:        "Rename a profile",
	Args:         cobra.ExactArgs(2),
	RunE:         runRename,
	SilenceUsage: true,
}

func runRename(_ *cobra.Command, args []string) error {
	if err := profile.Rename(args[0], args[1]); err != nil {
		return err
	}
	fmt.Printf("Profile %q renamed to %q.\n", args[0], args[1])
	return nil
}
