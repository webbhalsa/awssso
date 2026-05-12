package cmd

import (
	"fmt"
	"os"

	"github.com/jesperblomquist/awssso/internal/awsconfig"
	"github.com/jesperblomquist/awssso/internal/sso"
	"github.com/jesperblomquist/awssso/internal/tui"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Invalidate the cached SSO token for a session",
	RunE:  runLogout,
}

func runLogout(_ *cobra.Command, _ []string) error {
	cfg, err := awsconfig.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if len(cfg.SSOSessions) == 0 {
		return fmt.Errorf("no SSO sessions configured — run `awssso setup` first")
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
		chosen, err := tui.Pick("Select session to log out", items)
		if err != nil {
			return err
		}
		if chosen == nil {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
		session = chosen.Value.(*awsconfig.SSOSession)
	}

	if err := sso.DeleteCachedToken(session.StartURL); err != nil {
		return fmt.Errorf("delete token: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Logged out of session %q.\n", session.Name)
	fmt.Println("unset AWS_PROFILE AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_SESSION_TOKEN")
	return nil
}
