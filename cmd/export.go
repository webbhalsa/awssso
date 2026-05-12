package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/jesperblomquist/awssso/internal/awsconfig"
	"github.com/jesperblomquist/awssso/internal/tui"
)

func pickProfileAndExport(cfg *awsconfig.Config, sessionName string) error {
	profiles := cfg.SSOProfiles()
	if len(profiles) == 0 {
		return fmt.Errorf("no SSO profiles configured — run `awssso setup` first")
	}

	var items []tui.Item
	for _, p := range profiles {
		if sessionName != "" && p.SSOSession != sessionName {
			continue
		}
		session, ok := cfg.SSOSessions[p.SSOSession]
		if !ok {
			continue
		}
		desc := fmt.Sprintf("%s — %s  (%s)", p.AccountID, p.RoleName, session.Name)
		items = append(items, tui.NewItem(p.Name, desc, p))
	}

	if len(items) == 0 {
		return fmt.Errorf("no profiles found for session %q", sessionName)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Title() < items[j].Title() })

	var profile *awsconfig.Profile
	if len(items) == 1 {
		profile = items[0].Value.(*awsconfig.Profile)
	} else {
		chosen, err := tui.Pick("Select profile", items)
		if err != nil {
			return err
		}
		if chosen == nil {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
		profile = chosen.Value.(*awsconfig.Profile)
	}

	return exportProfile(profile)
}

func exportProfile(profile *awsconfig.Profile) error {
	fmt.Printf("export AWS_PROFILE=%s\n", profile.Name)
	fmt.Println("unset AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_SESSION_TOKEN")
	fmt.Fprintf(os.Stderr, "Active profile: %s (%s — %s, %s)\n", profile.Name, profile.AccountID, profile.RoleName, profile.Region)
	return nil
}
