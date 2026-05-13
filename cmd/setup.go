package cmd

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/webbhalsa/awssso/internal/awsconfig"
	"github.com/webbhalsa/awssso/internal/sso"
	"github.com/webbhalsa/awssso/internal/tui"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Discover available SSO accounts/roles and configure ~/.aws/config",
	RunE:  runSetup,
}

type accountRoleKey struct{ accountID, roleName string }

func runSetup(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Load config early to pre-fill prompts and detect existing profiles.
	cfg, err := awsconfig.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	existingStartURL, existingRegion := sessionDefaults(cfg)

	startURL, err := promptDefault("SSO start URL", existingStartURL)
	if err != nil {
		return err
	}
	startURL = strings.TrimRight(startURL, "/#")
	region, err := promptDefault("SSO region", existingRegion)
	if err != nil {
		return err
	}
	sessionName := sessionNameFromURL(startURL)
	fmt.Fprintf(os.Stderr, "Using session name %q\n", sessionName)
	defaultRegion, err := promptDefault("Default AWS region for profiles", region)
	if err != nil {
		return err
	}

	// Check for a valid cached token first.
	var accessToken string
	if cached, ok := sso.LoadCachedToken(startURL); ok {
		fmt.Println("Using existing SSO session.")
		accessToken = cached.AccessToken
	} else {
		fmt.Println("Opening browser for SSO login...")
		tok, err := sso.DeviceAuth(ctx, startURL, region, sso.OpenBrowser)
		if err != nil {
			return fmt.Errorf("SSO auth failed: %w", err)
		}
		accessToken = tok.AccessToken
		_ = sso.SaveCachedToken(&sso.CachedToken{
			StartURL:    startURL,
				SessionName: sessionName,
			Region:      region,
			AccessToken: tok.AccessToken,
			ExpiresAt:   tok.ExpiresAt,
		})
	}

	fmt.Println("Discovering accounts and roles...")
	roles, err := sso.ListAccountRoles(ctx, region, accessToken)
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}
	if len(roles) == 0 {
		fmt.Println("No accounts or roles found.")
		return nil
	}

	sort.Slice(roles, func(i, j int) bool {
		if roles[i].AccountName != roles[j].AccountName {
			return roles[i].AccountName < roles[j].AccountName
		}
		return roles[i].RoleName < roles[j].RoleName
	})

	// Build a map of existing profiles keyed by account+role for pre-selection.
	existingByKey := make(map[accountRoleKey]string) // → profile name
	for _, p := range cfg.SSOProfiles() {
		if p.AccountID != "" && p.RoleName != "" {
			existingByKey[accountRoleKey{p.AccountID, p.RoleName}] = p.Name
		}
	}

	items := make([]tui.CheckItem, len(roles))
	for i, r := range roles {
		_, exists := existingByKey[accountRoleKey{r.AccountID, r.RoleName}]
		items[i] = tui.CheckItem{
			Label:   fmt.Sprintf("%s (%s) — %s", r.AccountName, r.AccountID, r.RoleName),
			Value:   r,
			Checked: exists,
		}
	}

	selected, err := tui.MultiSelect("Select account/role combinations to add as profiles", items)
	if err != nil {
		return err
	}
	if selected == nil {
		fmt.Fprintln(os.Stderr, "Cancelled.")
		return nil
	}

	// Build set of selected keys to detect removals.
	selectedKeys := make(map[accountRoleKey]bool)
	for _, sel := range selected {
		r := sel.Value.(sso.AccountRole)
		selectedKeys[accountRoleKey{r.AccountID, r.RoleName}] = true
	}

	// Remove profiles that were deselected.
	for key, name := range existingByKey {
		if !selectedKeys[key] {
			cfg.DeleteProfile(name)
			fmt.Printf("  - profile %q\n", name)
		}
	}

	// Optionally customise per-profile regions.
	regionOverrides := make(map[accountRoleKey]string)
	if len(selected) > 0 && confirmBool("Customize region for any profile?") {
		for {
			pickItems := []tui.Item{tui.NewItem("Done", "finish region customization", nil)}
			for _, sel := range selected {
				r := sel.Value.(sso.AccountRole)
				key := accountRoleKey{r.AccountID, r.RoleName}
				reg := defaultRegion
				if override, ok := regionOverrides[key]; ok {
					reg = override
				}
				name := resolvedProfileName(r, existingByKey)
				pickItems = append(pickItems, tui.NewItem(name, "region: "+reg, r))
			}

			chosen, err := tui.Pick("Select profile to set region (or Done)", pickItems)
			if err != nil {
				return err
			}
			if chosen == nil || chosen.Value == nil {
				break
			}

			r := chosen.Value.(sso.AccountRole)
			key := accountRoleKey{r.AccountID, r.RoleName}
			cur := defaultRegion
			if override, ok := regionOverrides[key]; ok {
				cur = override
			}
			newRegion, err := promptDefault("Region", cur)
			if err != nil {
				return err
			}
			regionOverrides[key] = newRegion
		}
	}

	cfg.UpsertSession(&awsconfig.SSOSession{
		Name:               sessionName,
		StartURL:           startURL,
		Region:             region,
		RegistrationScopes: "sso:account:access",
	})

	for _, sel := range selected {
		r := sel.Value.(sso.AccountRole)
		key := accountRoleKey{r.AccountID, r.RoleName}
		profileRegion := defaultRegion
		if override, ok := regionOverrides[key]; ok {
			profileRegion = override
		}
		name := resolvedProfileName(r, existingByKey)
		cfg.UpsertProfile(&awsconfig.Profile{
			Name:       name,
			SSOSession: sessionName,
			AccountID:  r.AccountID,
			RoleName:   r.RoleName,
			Region:     profileRegion,
			Output:     "json",
		})
		fmt.Printf("  + profile %q (region: %s)\n", name, profileRegion)
	}

	if err := cfg.Write(); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	fmt.Printf("\nWrote %s\n", awsconfig.ConfigPath())
	return nil
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func sessionNameFromURL(startURL string) string {
	u, err := url.Parse(startURL)
	if err != nil || u.Host == "" {
		return "default"
	}
	// e.g. "kry.awsapps.com" → "kry", "d-93677155f2.awsapps.com" → "d-93677155f2"
	host := strings.Split(u.Host, ".")[0]
	return nonAlnum.ReplaceAllString(strings.ToLower(host), "-")
}

func profileName(accountName, roleName string) string {
	a := nonAlnum.ReplaceAllString(strings.ToLower(accountName), "-")
	r := nonAlnum.ReplaceAllString(strings.ToLower(roleName), "-")
	return strings.Trim(a+"-"+r, "-")
}

// resolvedProfileName returns the existing profile name for a role if one exists,
// otherwise generates a new one from the account and role names.
func resolvedProfileName(r sso.AccountRole, existingByKey map[accountRoleKey]string) string {
	if name, ok := existingByKey[accountRoleKey{r.AccountID, r.RoleName}]; ok {
		return name
	}
	return profileName(r.AccountName, r.RoleName)
}

func confirmBool(label string) bool {
	answer, _ := promptDefault(label, "n")
	return strings.ToLower(strings.TrimSpace(answer)) == "y" || strings.ToLower(strings.TrimSpace(answer)) == "yes"
}

// sessionDefaults returns the start URL and region from the first sso-session
// block in the config, so setup can suggest them as prompt defaults.
func sessionDefaults(cfg *awsconfig.Config) (startURL, region string) {
	for _, s := range cfg.SSOSessions {
		return s.StartURL, s.Region
	}
	return "", "eu-west-1"
}

var stdinReader = bufio.NewReader(os.Stdin)

func prompt(label string) (string, error) {
	return promptDefault(label, "")
}

func promptDefault(label, defaultVal string) (string, error) {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Print(label + ": ")
	}
	line, err := stdinReader.ReadString('\n')
	if err != nil {
		return "", err
	}
	v := strings.TrimSpace(line)
	if v == "" {
		return defaultVal, nil
	}
	return v, nil
}
