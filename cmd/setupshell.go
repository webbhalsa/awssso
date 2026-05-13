package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var setupShellCmd = &cobra.Command{
	Use:   "setup-shell",
	Short: "Inject shell integration into your shell rc file",
	RunE:  runSetupShell,
}

func runSetupShell(_ *cobra.Command, _ []string) error {
	shell := detectShell()

	var rc, line string
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %w", err)
	}

	switch shell {
	case "fish":
		rc = filepath.Join(home, ".config", "fish", "config.fish")
		line = "awssso init | source"
	case "zsh":
		rc = filepath.Join(home, ".zshrc")
		line = `eval "$(awssso init)"`
	default:
		rc = filepath.Join(home, ".bashrc")
		line = `eval "$(awssso init)"`
	}

	// Resolve symlink so we write to the real file.
	if resolved, err := filepath.EvalSymlinks(rc); err == nil {
		rc = resolved
	}

	if data, err := os.ReadFile(rc); err == nil && strings.Contains(string(data), "awssso init") {
		fmt.Fprintf(os.Stderr, "Shell integration already present in %s\n", rc)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(rc), 0o755); err != nil {
		return fmt.Errorf("could not create directory: %w", err)
	}

	f, err := os.OpenFile(rc, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("could not open %s: %w", rc, err)
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "\n# awssso shell integration\n%s\n", line)
	if err != nil {
		return fmt.Errorf("could not write to %s: %w", rc, err)
	}

	fmt.Fprintf(os.Stderr, "Added shell integration to %s\n", rc)
	fmt.Fprintf(os.Stderr, "Restart your shell or run: source %s\n", rc)
	return nil
}
