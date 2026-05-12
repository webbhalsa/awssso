package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/jesperblomquist/awssso/internal/awsconfig"
	"github.com/jesperblomquist/awssso/internal/sso"
	"github.com/jesperblomquist/awssso/internal/tui"
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:   "use",
	Short: "Pick a profile and export credentials (re-authenticates if needed)",
	RunE:  runUse,
}

func runUse(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	cfg, err := awsconfig.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	profiles := cfg.SSOProfiles()
	if len(profiles) == 0 {
		return fmt.Errorf("no SSO profiles configured — run `awssso setup` first")
	}

	// Build picker across all profiles.
	items := make([]tui.Item, 0, len(profiles))
	for _, p := range profiles {
		session, ok := cfg.SSOSessions[p.SSOSession]
		if !ok {
			continue
		}
		desc := fmt.Sprintf("%s — %s  (%s)", p.AccountID, p.RoleName, session.Name)
		items = append(items, tui.NewItem(p.Name, desc, p))
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Title() < items[j].Title() })

	chosen, err := tui.Pick("Select profile", items)
	if err != nil {
		return err
	}
	if chosen == nil {
		fmt.Fprintln(os.Stderr, "Cancelled.")
		return nil
	}
	profile := chosen.Value.(*awsconfig.Profile)
	session := cfg.SSOSessions[profile.SSOSession]

	// Re-authenticate if the session token is missing or expired.
	if _, ok := sso.LoadCachedToken(session.StartURL); !ok {
		fmt.Fprintf(os.Stderr, "Session %q not authenticated. Opening browser...\n", session.Name)
		tok, err := sso.DeviceAuth(ctx, session.StartURL, session.Region, sso.OpenBrowser)
		if err != nil {
			return fmt.Errorf("SSO auth: %w", err)
		}
		if err := sso.SaveCachedToken(&sso.CachedToken{
			StartURL:    session.StartURL,
			SessionName: session.Name,
			Region:      session.Region,
			AccessToken: tok.AccessToken,
			ExpiresAt:   tok.ExpiresAt,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not cache token: %v\n", err)
		}
	}

	return exportProfile(profile)
}
