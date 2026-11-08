package cmd

import (
	"encoding/json"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

const baseURLVar = "ANTHROPIC_BASE_URL"

// dialTimeout bounds each reachability probe of a custom API endpoint.
const dialTimeout = 2 * time.Second

// settingsBaseURL returns the ANTHROPIC_BASE_URL a profile's settings.json or
// settings.local.json sets in "env", or "" when it sets none. Claude applies
// that env to its own process, so every request goes to that endpoint.
func settingsBaseURL(configDir string) string {
	found := ""
	for _, name := range []string{"settings.json", "settings.local.json"} {
		data, err := os.ReadFile(filepath.Join(configDir, name))
		if err != nil {
			continue
		}
		var s struct {
			Env map[string]string `json:"env"`
		}
		if json.Unmarshal(data, &s) != nil {
			continue
		}
		if v := s.Env[baseURLVar]; v != "" {
			found = v // settings.local.json wins
		}
	}
	return found
}

// endpointReachable reports whether something accepts TCP connections at the
// host and port of rawURL. A URL it cannot parse is reported as an error.
func endpointReachable(rawURL string) (bool, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Hostname() == "" {
		return false, err
	}
	port := u.Port()
	if port == "" {
		port = "80"
		if u.Scheme == "https" {
			port = "443"
		}
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(u.Hostname(), port), dialTimeout)
	if err != nil {
		return false, err
	}
	conn.Close()
	return true, nil
}

// checkBaseURL warns when label's endpoint override points at a host that
// refuses connections: Claude then fails with ECONNREFUSED even though the
// account is logged in. It stays silent when no override is set.
func checkBaseURL(label, source, rawURL string, pass func(label, value string), warn func(label, value, msg string)) {
	if rawURL == "" {
		return
	}
	if ok, _ := endpointReachable(rawURL); ok {
		pass(label, rawURL)
		return
	}
	warn(label, rawURL, "unreachable (set in "+source+") — Claude fails with ECONNREFUSED; start it or remove "+baseURLVar)
}
