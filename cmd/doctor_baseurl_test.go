package cmd

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsBaseURL(t *testing.T) {
	dir := t.TempDir()
	if got := settingsBaseURL(dir); got != "" {
		t.Fatalf("no settings: got %q", got)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("settings.json", `{"env":{"ANTHROPIC_BASE_URL":"http://a:1"}}`)
	if got := settingsBaseURL(dir); got != "http://a:1" {
		t.Fatalf("settings.json: got %q", got)
	}
	write("settings.local.json", `{"env":{"ANTHROPIC_BASE_URL":"http://b:2"}}`)
	if got := settingsBaseURL(dir); got != "http://b:2" {
		t.Fatalf("local wins: got %q", got)
	}
}

func TestEndpointReachable(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	if ok, _ := endpointReachable("http://" + addr + "/w/claude"); !ok {
		t.Fatal("listening endpoint reported unreachable")
	}
	l.Close()
	if ok, _ := endpointReachable("http://" + addr); ok {
		t.Fatal("closed endpoint reported reachable")
	}
}
