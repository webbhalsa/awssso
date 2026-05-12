package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Print shell integration code (add `eval \"$(awssso init)\"` to your .zshrc/.bashrc)",
	RunE:  runInit,
}

var shellFlag string

func init() {
	initCmd.Flags().StringVar(&shellFlag, "shell", "", "Shell type: bash or zsh (auto-detected if omitted)")
}

func runInit(_ *cobra.Command, _ []string) error {
	bin, err := os.Executable()
	if err != nil {
		bin = "awssso"
	}

	shell := shellFlag
	if shell == "" {
		shell = detectShell()
	}

	switch shell {
	case "fish":
		fmt.Printf(fishInit, bin, bin)
	default:
		fmt.Printf(bashInit, bin, bin)
	}
	return nil
}

func detectShell() string {
	shell := getEnv("SHELL")
	switch {
	case hasSuffix(shell, "fish"):
		return "fish"
	case hasSuffix(shell, "zsh"):
		return "zsh"
	default:
		return "bash"
	}
}

const bashInit = `
# awssso shell integration
awssso() {
  local cmd="$1"
  local out
  if [[ "$cmd" == "login" || "$cmd" == "use" || "$cmd" == "logout" ]]; then
    out=$(%s "$@" </dev/tty 2>/dev/tty) || return $?
    eval "$out"
  else
    %s "$@"
  fi
}
`

const fishInit = `
# awssso shell integration
function awssso
  set cmd $argv[1]
  if test "$cmd" = "login" -o "$cmd" = "use" -o "$cmd" = "logout"
    set out (command %s $argv </dev/tty 2>/dev/tty) || return $status
    eval $out
  else
    %s $argv
  end
end
`
