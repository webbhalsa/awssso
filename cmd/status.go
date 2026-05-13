package cmd

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/webbhalsa/awssso/internal/awsconfig"
	"github.com/webbhalsa/awssso/internal/sso"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show SSO session and profile status",
	RunE:  runStatus,
}

var (
	okStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	expiredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	activeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
)

func runStatus(_ *cobra.Command, _ []string) error {
	cfg, err := awsconfig.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if len(cfg.SSOSessions) == 0 {
		fmt.Println("No SSO sessions configured. Run `awssso setup` to get started.")
		return nil
	}

	activeProfile := getEnv("AWS_PROFILE")

	for _, session := range cfg.SSOSessions {
		cached, valid := sso.LoadCachedToken(session.StartURL)
		var statusLine string
		if valid {
			remaining := time.Until(cached.ExpiresAt).Round(time.Minute)
			statusLine = okStyle.Render("✓ authenticated") + dimStyle.Render(fmt.Sprintf("  expires in %s", remaining))
		} else {
			statusLine = expiredStyle.Render("✗ not authenticated")
		}
		fmt.Printf("Session: %s  %s\n", session.Name, statusLine)

		for _, p := range cfg.SSOProfiles() {
			if p.SSOSession != session.Name {
				continue
			}
			name := p.Name
			suffix := ""
			if p.Name == activeProfile {
				name = activeStyle.Render(p.Name)
				suffix = activeStyle.Render(" ◀ active")
			}
			fmt.Printf("  %s  %s / %s%s\n",
				dimStyle.Render("profile:"),
				name,
				dimStyle.Render(fmt.Sprintf("%s — %s", p.AccountID, p.RoleName)),
				suffix,
			)
		}
		fmt.Println()
	}
	return nil
}
