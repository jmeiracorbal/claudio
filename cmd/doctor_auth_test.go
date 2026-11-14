package cmd

import (
	"strings"
	"testing"
)

func TestCheckAnthropicAPIKeySilentWhenUnset(t *testing.T) {
	called := false
	checkAnthropicAPIKey("", func(label, value, msg string) {
		called = true
	})
	if called {
		t.Fatal("unset ANTHROPIC_API_KEY produced a warning")
	}
}

func TestCheckAnthropicAPIKeyWarnsWithoutExposingSecret(t *testing.T) {
	const secret = "sk-ant-secret-value"

	var gotLabel, gotValue, gotMessage string
	checkAnthropicAPIKey(secret, func(label, value, msg string) {
		gotLabel = label
		gotValue = value
		gotMessage = msg
	})

	if gotLabel != anthropicAPIKeyVar {
		t.Fatalf("label = %q, want %q", gotLabel, anthropicAPIKeyVar)
	}
	if gotValue != "set" {
		t.Fatalf("value = %q, want %q", gotValue, "set")
	}
	if gotMessage == "" {
		t.Fatal("warning message is empty")
	}
	if strings.Contains(gotLabel, secret) || strings.Contains(gotValue, secret) || strings.Contains(gotMessage, secret) {
		t.Fatal("warning exposes ANTHROPIC_API_KEY value")
	}
}
