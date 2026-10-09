package limits

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

const (
	usageURL        = "https://api.anthropic.com/api/oauth/usage"
	oauthBeta       = "oauth-2025-04-20"
	keychainBase    = "Claude Code-credentials"
	keychainTimeout = 4 * time.Second
)

// State summarizes whether a profile's usage could be read.
type State string

const (
	StateOK          State = "ok"
	StateExpired     State = "expired"     // token expired or rejected (401); launching the profile refreshes it
	StateNone        State = "none"        // no stored credentials: never logged in
	StateUnavailable State = "unavailable" // network or server failure
)

// Window is one usage limit, e.g. the 5-hour session or the weekly window.
type Window struct {
	Kind     string    `json:"kind"`
	Group    string    `json:"group"`
	Percent  int       `json:"percent"`
	Severity string    `json:"severity,omitempty"`
	ResetsAt time.Time `json:"resetsAt,omitzero"`
	Active   bool      `json:"active"`
}

// Extra is the pay-as-you-go credit used once plan limits are reached.
type Extra struct {
	Enabled      bool    `json:"enabled"`
	MonthlyLimit float64 `json:"monthlyLimit"`
	UsedCredits  float64 `json:"usedCredits"`
	Currency     string  `json:"currency,omitempty"`
}

// Report is the usage of one profile at FetchedAt.
type Report struct {
	State          State     `json:"state"`
	Error          string    `json:"error,omitempty"`
	Plan           string    `json:"plan,omitempty"`
	Tier           string    `json:"tier,omitempty"`
	TokenExpiresAt time.Time `json:"tokenExpiresAt,omitzero"`
	Windows        []Window  `json:"windows,omitempty"`
	Extra          *Extra    `json:"extra,omitempty"`
	FetchedAt      time.Time `json:"fetchedAt"`
}

// Window returns the first window of the given group ("session", "weekly").
func (r Report) Window(group string) *Window {
	for i := range r.Windows {
		if r.Windows[i].Group == group {
			return &r.Windows[i]
		}
	}
	return nil
}

type credentials struct {
	ClaudeAiOauth struct {
		AccessToken      string `json:"accessToken"`
		ExpiresAt        int64  `json:"expiresAt"`
		SubscriptionType string `json:"subscriptionType"`
		RateLimitTier    string `json:"rateLimitTier"`
	} `json:"claudeAiOauth"`
}

// KeychainService is the macOS Keychain service Claude Code stores the
// credentials of a CLAUDE_CONFIG_DIR under: the first 8 hex chars of the
// sha256 of the directory, exactly as passed in the environment.
func KeychainService(configDir string) string {
	sum := sha256.Sum256([]byte(configDir))
	return keychainBase + "-" + hex.EncodeToString(sum[:])[:8]
}

// readCredentials returns the stored OAuth credentials: the credentials file
// where Claude Code uses one (Linux), otherwise the macOS Keychain.
func readCredentials(configDir string) (*credentials, error) {
	data, err := os.ReadFile(filepath.Join(configDir, ".credentials.json"))
	if err != nil && runtime.GOOS == "darwin" {
		ctx, cancel := context.WithTimeout(context.Background(), keychainTimeout)
		defer cancel()
		data, err = exec.CommandContext(ctx, "security", "find-generic-password", "-s", KeychainService(configDir), "-w").Output()
	}
	if err != nil {
		return nil, err
	}
	var c credentials
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parsing credentials: %w", err)
	}
	if c.ClaudeAiOauth.AccessToken == "" {
		return nil, fmt.Errorf("no OAuth token in credentials")
	}
	return &c, nil
}

// Probe reads the profile's credentials and asks Anthropic for its current
// usage. It never refreshes the token: that is Claude Code's job.
func Probe(ctx context.Context, configDir string) Report {
	now := time.Now()
	r := Report{FetchedAt: now}

	c, err := readCredentials(configDir)
	if err != nil {
		r.State = StateNone
		return r
	}
	o := c.ClaudeAiOauth
	r.Plan, r.Tier = o.SubscriptionType, o.RateLimitTier
	if o.ExpiresAt > 0 {
		r.TokenExpiresAt = time.UnixMilli(o.ExpiresAt)
		if r.TokenExpiresAt.Before(now) {
			r.State = StateExpired
			return r
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, usageURL, nil)
	if err != nil {
		r.State, r.Error = StateUnavailable, err.Error()
		return r
	}
	req.Header.Set("Authorization", "Bearer "+o.AccessToken)
	req.Header.Set("anthropic-beta", oauthBeta)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		r.State, r.Error = StateUnavailable, err.Error()
		return r
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		r.State = StateExpired
		return r
	case resp.StatusCode != http.StatusOK:
		r.State, r.Error = StateUnavailable, "HTTP "+resp.Status
		return r
	}

	var body usageResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		r.State, r.Error = StateUnavailable, "parsing usage: "+err.Error()
		return r
	}
	r.State = StateOK
	r.Windows, r.Extra = body.windows(), body.extra()
	return r
}

type usageWindow struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    string  `json:"resets_at"`
}

type usageResponse struct {
	FiveHour *usageWindow `json:"five_hour"`
	SevenDay *usageWindow `json:"seven_day"`
	Limits   []struct {
		Kind     string `json:"kind"`
		Group    string `json:"group"`
		Percent  int    `json:"percent"`
		Severity string `json:"severity"`
		ResetsAt string `json:"resets_at"`
		IsActive bool   `json:"is_active"`
	} `json:"limits"`
	ExtraUsage *struct {
		IsEnabled    bool    `json:"is_enabled"`
		MonthlyLimit float64 `json:"monthly_limit"`
		UsedCredits  float64 `json:"used_credits"`
		Currency     string  `json:"currency"`
	} `json:"extra_usage"`
}

// windows prefers the generic limits list, which carries severity and picks
// up new limit kinds, and falls back to the fixed five_hour/seven_day fields.
func (u usageResponse) windows() []Window {
	var out []Window
	for _, l := range u.Limits {
		out = append(out, Window{
			Kind:     l.Kind,
			Group:    l.Group,
			Percent:  l.Percent,
			Severity: l.Severity,
			ResetsAt: parseTime(l.ResetsAt),
			Active:   l.IsActive,
		})
	}
	if len(out) > 0 {
		return out
	}
	for _, f := range []struct {
		kind, group string
		w           *usageWindow
	}{{"session", "session", u.FiveHour}, {"weekly_all", "weekly", u.SevenDay}} {
		if f.w != nil {
			out = append(out, Window{Kind: f.kind, Group: f.group, Percent: int(f.w.Utilization + 0.5), ResetsAt: parseTime(f.w.ResetsAt)})
		}
	}
	return out
}

func (u usageResponse) extra() *Extra {
	if u.ExtraUsage == nil {
		return nil
	}
	e := u.ExtraUsage
	return &Extra{Enabled: e.IsEnabled, MonthlyLimit: e.MonthlyLimit, UsedCredits: e.UsedCredits, Currency: e.Currency}
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, s)
	return t
}
