package sso

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type CachedToken struct {
	StartURL    string    `json:"startUrl"`
	Region      string    `json:"region"`
	AccessToken string    `json:"accessToken"`
	ExpiresAt   time.Time `json:"expiresAt"`
	// SessionName is not written to JSON; used only to determine extra cache filenames.
	SessionName string `json:"-"`
}

func cacheDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aws", "sso", "cache")
}

func cacheKey(startURL string) string {
	h := sha1.Sum([]byte(startURL))
	return hex.EncodeToString(h[:]) + ".json"
}

// LoadCachedToken returns the cached SSO token for the given start URL, if valid.
func LoadCachedToken(startURL string) (*CachedToken, bool) {
	path := filepath.Join(cacheDir(), cacheKey(startURL))
	data, err := os.ReadFile(path)
	if err != nil {
		// fallback: scan all cache files for matching startUrl
		return scanCacheForURL(startURL)
	}
	var t CachedToken
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, false
	}
	if time.Now().After(t.ExpiresAt) {
		return nil, false
	}
	return &t, true
}

func scanCacheForURL(startURL string) (*CachedToken, bool) {
	entries, err := os.ReadDir(cacheDir())
	if err != nil {
		return nil, false
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(cacheDir(), e.Name()))
		if err != nil {
			continue
		}
		var t CachedToken
		if err := json.Unmarshal(data, &t); err != nil {
			continue
		}
		if t.StartURL == startURL && time.Now().Before(t.ExpiresAt) {
			return &t, true
		}
	}
	return nil, false
}

// DeleteCachedToken removes all cached token files for the given start URL.
func DeleteCachedToken(startURL string) error {
	// Remove the SHA1-keyed file.
	_ = os.Remove(filepath.Join(cacheDir(), cacheKey(startURL)))

	// Also scan for any other files written by the AWS CLI for the same URL.
	entries, err := os.ReadDir(cacheDir())
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(cacheDir(), e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var t CachedToken
		if err := json.Unmarshal(data, &t); err != nil {
			continue
		}
		if t.StartURL == startURL {
			_ = os.Remove(path)
		}
	}
	return nil
}

// SaveCachedToken writes the SSO token to the cache.
// It writes two files: one keyed by SHA1 of the start URL (used by the AWS CLI
// and our own lookup), and one keyed by SHA1 of the session name when set (used
// by the AWS SDK for Go and tools like Terraform that use the sso_session format).
func SaveCachedToken(t *CachedToken) error {
	if err := os.MkdirAll(cacheDir(), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(cacheDir(), cacheKey(t.StartURL)), data, 0600); err != nil {
		return err
	}
	if t.SessionName != "" {
		_ = os.WriteFile(filepath.Join(cacheDir(), cacheKey(t.SessionName)), data, 0600)
	}
	return nil
}
