package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/jesperblomquist/awssso/internal/awsconfig"
	"github.com/jesperblomquist/awssso/internal/sso"
	"github.com/jesperblomquist/awssso/internal/tui"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate via AWS SSO, pick a profile, and export credentials",
	RunE:  runLogin,
}

func runLogin(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	cfg, err := awsconfig.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if len(cfg.SSOSessions) == 0 {
		return fmt.Errorf("no SSO sessions found — run `awssso setup` first")
	}

	var session *awsconfig.SSOSession
	if len(cfg.SSOSessions) == 1 {
		for _, s := range cfg.SSOSessions {
			session = s
		}
	} else {
		items := make([]tui.Item, 0, len(cfg.SSOSessions))
		for _, s := range cfg.SSOSessions {
			items = append(items, tui.NewItem(s.Name, s.StartURL, s))
		}
		chosen, err := tui.Pick("Select SSO session", items)
		if err != nil {
			return err
		}
		if chosen == nil {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
		session = chosen.Value.(*awsconfig.SSOSession)
	}

	if cached, ok := sso.LoadCachedToken(session.StartURL); ok {
		fmt.Fprintf(os.Stderr, "Session %q already authenticated (expires %s).\n",
			session.Name, cached.ExpiresAt.Format("15:04 02 Jan"))
	} else {
		fmt.Fprintf(os.Stderr, "Opening browser for %q...\n", session.Name)
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
		fmt.Fprintf(os.Stderr, "Authenticated until %s.\n", tok.ExpiresAt.Format("15:04 02 Jan 2006"))
	}

	return pickProfileAndExport(cfg, session.Name)
}
