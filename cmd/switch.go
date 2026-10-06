package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/jmeiracorbal/claudio/internal/config"
	"github.com/jmeiracorbal/claudio/internal/launcher"
	"github.com/jmeiracorbal/claudio/internal/resolver"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Interactively select and launch a profile",
	Long: `Select a Claude account interactively.
Uses fzf if available, otherwise shows a numbered menu.`,
	Args:         cobra.NoArgs,
	RunE:         runSwitch,
	SilenceUsage: true,
}

func runSwitch(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if len(cfg.Profiles) == 0 {
		return fmt.Errorf("no profiles found — run: claudio create <name>")
	}

	names := make([]string, 0, len(cfg.Profiles))
	for n := range cfg.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)

	active, _ := resolver.Resolve(cfg)

	var chosen string
	if fzfPath, err := exec.LookPath("fzf"); err == nil {
		chosen, err = selectWithFzf(fzfPath, names, active)
		if err != nil {
			return err
		}
	} else {
		chosen, err = selectWithMenu(names, active)
		if err != nil {
			return err
		}
	}

	if chosen == "" {
		return nil
	}

	configDir, err := cfg.ProfileConfigDir(chosen)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "claudio: launching %q\n", chosen)
	return launcher.Launch(configDir, nil)
}

func selectWithFzf(fzfPath string, names []string, active string) (string, error) {
	input := strings.Join(names, "\n")
	cmd := exec.Command(fzfPath, "--prompt=Claude account: ", "--height=40%", "--reverse")
	if active != "" {
		cmd.Args = append(cmd.Args, "--query="+active)
	}
	cmd.Stdin = strings.NewReader(input)
	cmd.Stderr = os.Stderr
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		// exit code 130 = user pressed ESC/ctrl-c
		return "", nil
	}
	return strings.TrimSpace(out.String()), nil
}

func selectWithMenu(names []string, active string) (string, error) {
	fmt.Println("Select Claude account:")
	fmt.Println()
	for i, n := range names {
		marker := "  "
		if n == active {
			marker = "* "
		}
		fmt.Printf("  %s%d. %s\n", marker, i+1, n)
	}
	fmt.Printf("\nEnter number (1-%d): ", len(names))

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return "", nil
	}
	n, err := strconv.Atoi(line)
	if err != nil || n < 1 || n > len(names) {
		return "", fmt.Errorf("invalid selection: %q", line)
	}
	return names[n-1], nil
}
