package limits

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestKeychainService(t *testing.T) {
	// Name Claude Code gave a real profile's Keychain entry.
	got := KeychainService("/Users/dankkomcg/.claudio/profiles/personal/claude")
	if want := "Claude Code-credentials-7bbe3a0b"; got != want {
		t.Fatalf("KeychainService = %q, want %q", got, want)
	}
}

func TestWindowsPrefersLimits(t *testing.T) {
	var u usageResponse
	mustUnmarshal(t, `{
		"five_hour": {"utilization": 4.0, "resets_at": "2026-10-09T16:10:00.142682+00:00"},
		"limits": [
			{"kind": "session", "group": "session", "percent": 4, "severity": "normal", "resets_at": "2026-10-09T16:10:00.142682+00:00"},
			{"kind": "weekly_all", "group": "weekly", "percent": 19, "severity": "warning", "resets_at": "2026-10-10T18:00:00+00:00", "is_active": true}
		]
	}`, &u)

	ws := u.windows()
	if len(ws) != 2 {
		t.Fatalf("got %d windows, want 2", len(ws))
	}
	w := Report{Windows: ws}.Window("weekly")
	if w == nil || w.Percent != 19 || w.Severity != "warning" || !w.Active || w.ResetsAt.IsZero() {
		t.Fatalf("weekly window = %+v", w)
	}
}

func TestWindowsFallsBackToFixedFields(t *testing.T) {
	var u usageResponse
	mustUnmarshal(t, `{
		"five_hour": {"utilization": 4.6, "resets_at": "2026-10-09T16:10:00.142682+00:00"},
		"seven_day": {"utilization": 19.0, "resets_at": "2026-10-10T18:00:00+00:00"}
	}`, &u)

	r := Report{Windows: u.windows()}
	if w := r.Window("session"); w == nil || w.Percent != 5 || w.ResetsAt.IsZero() {
		t.Fatalf("session window = %+v", w)
	}
	if w := r.Window("weekly"); w == nil || w.Percent != 19 {
		t.Fatalf("weekly window = %+v", w)
	}
}

func TestProbeWithoutCredentials(t *testing.T) {
	if r := Probe(context.Background(), t.TempDir()); r.State != StateNone {
		t.Fatalf("State = %q, want %q", r.State, StateNone)
	}
}

func TestProbeExpiredTokenSkipsRequest(t *testing.T) {
	dir := writeCredentials(t, `{"claudeAiOauth": {"accessToken": "x", "expiresAt": 1000, "subscriptionType": "pro"}}`)
	r := Probe(context.Background(), dir)
	if r.State != StateExpired || r.Plan != "pro" {
		t.Fatalf("report = %+v", r)
	}
}

func mustUnmarshal(t *testing.T, s string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(s), v); err != nil {
		t.Fatal(err)
	}
}

func writeCredentials(t *testing.T, s string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".credentials.json"), []byte(s), 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}
