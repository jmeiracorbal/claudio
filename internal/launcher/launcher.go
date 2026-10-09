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
	cmd.Env = Env(configDir)
	return cmd, nil
}

// Exec replaces the current process with an arbitrary program running against
// the profile, so installers that honor CLAUDE_CONFIG_DIR write into it.
func Exec(configDir, name string, args []string) error {
	path, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("%s not found in PATH", name)
	}
	return syscall.Exec(path, append([]string{name}, args...), Env(configDir))
}

// Env returns the current environment with CLAUDE_CONFIG_DIR set to configDir.
// An inherited CLAUDE_CONFIG_DIR (e.g. when called from inside a profile
// session) is dropped rather than shadowed: with duplicate keys many runtimes
// read the first one.
func Env(configDir string) []string {
	env := make([]string, 0, len(os.Environ())+1)
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "CLAUDE_CONFIG_DIR=") {
			env = append(env, kv)
		}
	}
	return append(env, "CLAUDE_CONFIG_DIR="+configDir)
}
