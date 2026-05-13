package cmd

import (
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "awssso",
	Short: "AWS SSO helper — setup and login made easy",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

func Execute(version string) {
	var (
		wg            sync.WaitGroup
		latestVersion string
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		latestVersion = fetchLatestVersion(version)
	}()

	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}

	wg.Wait()
	if latestVersion != "" && rootCmd.CalledAs() != "init" {
		fmt.Fprintf(os.Stderr, "\nUpdate available: %s → %s  Run: brew upgrade awssso\n", version, latestVersion)
	}
}

func init() {
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(useCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(setupShellCmd)
}
