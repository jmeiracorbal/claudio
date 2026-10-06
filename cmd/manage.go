package cmd

import (
	"os"

	"github.com/jmeiracorbal/claudio/internal/tui"
	"github.com/spf13/cobra"
)

var (
	manageFrame    bool
	manageCols     int
	manageRows     int
	manageDepth    string
	manageTheme    string
	manageScreen   string
	manageSelected int
)

var manageCmd = &cobra.Command{
	Use:          "manage",
	Short:        "Open the interactive Task Hub",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		if manageFrame {
			return tui.RenderFrame(manageCols, manageRows, manageDepth, manageTheme, manageScreen, manageSelected, os.Stdout)
		}
		return tui.Run()
	},
}

func init() {
	manageCmd.Flags().BoolVar(&manageFrame, "frame", false, "render one illustrative static frame and exit")
	manageCmd.Flags().IntVar(&manageCols, "cols", 80, "static frame width")
	manageCmd.Flags().IntVar(&manageRows, "rows", 24, "static frame height")
	manageCmd.Flags().StringVar(&manageDepth, "depth", "truecolor", "frame color depth: truecolor, 256, or 16")
	manageCmd.Flags().StringVar(&manageTheme, "theme", "", "flat theme JSON (default: embedded Catppuccin Mocha)")
	manageCmd.Flags().StringVar(&manageScreen, "screen", "home", "illustrative frame: home, launch, profiles, projects, setup, help")
	manageCmd.Flags().IntVar(&manageSelected, "selected", 0, "illustrative frame selected row")
}
