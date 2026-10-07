package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/jmeiracorbal/claudio/internal/profile"
)

// Launch replaces the current process with claude using the given configDir.
// Uses syscall.Exec so there is no wrapper process: signals, stdin/stdout/stderr
// are all directly connected to claude.
func Launch(configDir string, args []string) error {
	cmd, err := Command(configDir, args)
	if err != nil {
		return err
	}
	return syscall.Exec(cmd.Path, cmd.Args, cmd.Env)
}

// WithModel prepends --model <model> to args when model is non-empty and
// --model is not already present in args.
func WithModel(model string, args []string) []string {
	if model == "" {
		return args
	}
	for _, a := range args {
		if a == "--model" || strings.HasPrefix(a, "--model=") {
			return args
		}
	}
	return append([]string{"--model", model}, args...)
}

// Command builds the Claude process used by the full-screen manager. Bubble Tea
// runs it as a child so it can restore the terminal before handing off control.
func Command(configDir string, args []string) (*exec.Cmd, error) {
	claude, err := exec.LookPath("claude")
	if err != nil {
		return nil, fmt.Errorf("claude not found in PATH — is Claude Code installed?")
	}
	// Keep the local installation's CLAUDE.md out of the profile (claude.toml,
	// [launch]). A --settings passed by the user comes later and wins.
	settings, err := profile.LaunchSettings()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(claude, append([]string{"--settings", settings}, args...)...)
	cmd.Env = append(os.Environ(), "CLAUDE_CONFIG_DIR="+configDir)
	return cmd, nil
}
