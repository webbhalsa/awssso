package awsconfig

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SSOSession struct {
	Name                  string
	StartURL              string
	Region                string
	RegistrationScopes    string
}

type Profile struct {
	Name         string
	SSOSession   string
	AccountID    string
	RoleName     string
	Region       string
	Output       string
}

type Config struct {
	SSOSessions map[string]*SSOSession
	Profiles    map[string]*Profile
}

func ConfigPath() string {
	if v := os.Getenv("AWS_CONFIG_FILE"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aws", "config")
}

func Load() (*Config, error) {
	cfg := &Config{
		SSOSessions: make(map[string]*SSOSession),
		Profiles:    make(map[string]*Profile),
	}

	f, err := os.Open(ConfigPath())
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var currentSection string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = line[1 : len(line)-1]
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch {
		case strings.HasPrefix(currentSection, "sso-session "):
			name := strings.TrimPrefix(currentSection, "sso-session ")
			s := cfg.getOrCreateSession(name)
			switch key {
			case "sso_start_url":
				s.StartURL = value
			case "sso_region":
				s.Region = value
			case "sso_registration_scopes":
				s.RegistrationScopes = value
			}
		case currentSection == "default":
			p := cfg.getOrCreateProfile("default")
			applyProfileKey(p, key, value)
		case strings.HasPrefix(currentSection, "profile "):
			name := strings.TrimPrefix(currentSection, "profile ")
			p := cfg.getOrCreateProfile(name)
			applyProfileKey(p, key, value)
		}
	}
	return cfg, scanner.Err()
}

func (c *Config) getOrCreateSession(name string) *SSOSession {
	if s, ok := c.SSOSessions[name]; ok {
		return s
	}
	s := &SSOSession{Name: name}
	c.SSOSessions[name] = s
	return s
}

func (c *Config) getOrCreateProfile(name string) *Profile {
	if p, ok := c.Profiles[name]; ok {
		return p
	}
	p := &Profile{Name: name}
	c.Profiles[name] = p
	return p
}

func applyProfileKey(p *Profile, key, value string) {
	switch key {
	case "sso_session":
		p.SSOSession = value
	case "sso_account_id":
		p.AccountID = value
	case "sso_role_name":
		p.RoleName = value
	case "region":
		p.Region = value
	case "output":
		p.Output = value
	}
}

// SSOProfiles returns profiles that use SSO (either via sso_session or legacy sso_start_url).
func (c *Config) SSOProfiles() []*Profile {
	var out []*Profile
	for _, p := range c.Profiles {
		if p.SSOSession != "" || p.AccountID != "" {
			out = append(out, p)
		}
	}
	return out
}

func (c *Config) Write() error {
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	var sb strings.Builder

	for _, s := range c.SSOSessions {
		fmt.Fprintf(&sb, "[sso-session %s]\n", s.Name)
		fmt.Fprintf(&sb, "sso_start_url = %s\n", s.StartURL)
		fmt.Fprintf(&sb, "sso_region = %s\n", s.Region)
		if s.RegistrationScopes != "" {
			fmt.Fprintf(&sb, "sso_registration_scopes = %s\n", s.RegistrationScopes)
		}
		sb.WriteString("\n")
	}

	for _, p := range c.Profiles {
		if p.Name == "default" {
			fmt.Fprintf(&sb, "[default]\n")
		} else {
			fmt.Fprintf(&sb, "[profile %s]\n", p.Name)
		}
		if p.SSOSession != "" {
			fmt.Fprintf(&sb, "sso_session = %s\n", p.SSOSession)
			// Write inline for tools that don't understand sso-session references (Terraform, older CLIs).
			if s, ok := c.SSOSessions[p.SSOSession]; ok {
				fmt.Fprintf(&sb, "sso_start_url = %s\n", s.StartURL)
				fmt.Fprintf(&sb, "sso_region = %s\n", s.Region)
			}
		}
		if p.AccountID != "" {
			fmt.Fprintf(&sb, "sso_account_id = %s\n", p.AccountID)
		}
		if p.RoleName != "" {
			fmt.Fprintf(&sb, "sso_role_name = %s\n", p.RoleName)
		}
		if p.Region != "" {
			fmt.Fprintf(&sb, "region = %s\n", p.Region)
		}
		if p.Output != "" {
			fmt.Fprintf(&sb, "output = %s\n", p.Output)
		}
		sb.WriteString("\n")
	}

	return os.WriteFile(path, []byte(sb.String()), 0600)
}

// UpsertSession adds or replaces an SSO session entry.
func (c *Config) UpsertSession(s *SSOSession) {
	c.SSOSessions[s.Name] = s
}

// UpsertProfile adds or replaces a profile entry.
func (c *Config) UpsertProfile(p *Profile) {
	c.Profiles[p.Name] = p
}

// DeleteProfile removes a profile entry.
func (c *Config) DeleteProfile(name string) {
	delete(c.Profiles, name)
}
