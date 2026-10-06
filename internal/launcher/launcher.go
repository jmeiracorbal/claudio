package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
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

// Command builds the Claude process used by the full-screen manager. Bubble Tea
// runs it as a child so it can restore the terminal before handing off control.
func Command(configDir string, args []string) (*exec.Cmd, error) {
	claude, err := exec.LookPath("claude")
	if err != nil {
		return nil, fmt.Errorf("claude not found in PATH — is Claude Code installed?")
	}
	cmd := exec.Command(claude, args...)
	cmd.Env = append(os.Environ(), "CLAUDE_CONFIG_DIR="+configDir)
	return cmd, nil
}
