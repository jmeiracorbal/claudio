package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/jmeiracorbal/claudio/internal/profile"
	"github.com/spf13/cobra"
)

var removeForce bool

var removeCmd = &cobra.Command{
	Use:          "remove <name>",
	Aliases:      []string{"rm", "delete"},
	Short:        "Remove a profile and delete its config directory",
	Args:         cobra.ExactArgs(1),
	RunE:         runRemove,
	SilenceUsage: true,
}

func init() {
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "skip confirmation prompt")
}

func runRemove(_ *cobra.Command, args []string) error {
	name := args[0]
	if !removeForce {
		fmt.Printf("Remove profile %q and delete its config directory? [y/N] ", name)
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		if !strings.EqualFold(strings.TrimSpace(line), "y") {
			fmt.Println("Aborted.")
			return nil
		}
	}
	if err := profile.Remove(name); err != nil {
		return err
	}
	fmt.Printf("Profile %q removed.\n", name)
	return nil
}
