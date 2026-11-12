package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmeiracorbal/claudio/internal/config"
)

func TestValidateName(t *testing.T) {
	t.Parallel()

	valid := []string{
		"work",
		"personal-2",
		"client_a",
		"team.eu",
		"A1",
		strings.Repeat("a", 64),
	}
	for _, name := range valid {
		name := name
		t.Run("valid_"+name, func(t *testing.T) {
			t.Parallel()
			if err := ValidateName(name); err != nil {
				t.Fatalf("ValidateName(%q) returned error: %v", name, err)
			}
		})
	}

	invalid := []string{
		"",
		".",
		"..",
		"../work",
		"work/other",
		"work other",
		"work@home",
		"-work",
		"_work",
		strings.Repeat("a", 65),
	}
	for _, name := range invalid {
		name := name
		t.Run("invalid_"+name, func(t *testing.T) {
			t.Parallel()
			if err := ValidateName(name); err == nil {
				t.Fatalf("ValidateName(%q) unexpectedly succeeded", name)
			}
		})
	}
}

func TestDefaultConfigDirStaysInsideProfilesRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := DefaultConfigDir("work")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".claudio", "profiles", "work", "claude")
	if got != want {
		t.Fatalf("DefaultConfigDir() = %q, want %q", got, want)
	}

	if _, err := DefaultConfigDir("../../escape"); err == nil {
		t.Fatal("DefaultConfigDir accepted a path-traversal profile name")
	}
}

func TestCreateRejectsPathTraversalName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := Create("../escape"); err == nil {
		t.Fatal("Create accepted a path-traversal profile name")
	}

	escapePath := filepath.Join(home, ".claudio", "escape")
	if _, err := os.Stat(escapePath); !os.IsNotExist(err) {
		t.Fatalf("Create left files outside the profiles root: %s", escapePath)
	}
}

func TestRenameRejectsPathTraversalName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := Create("work"); err != nil {
		t.Fatal(err)
	}
	originalDir, err := DefaultConfigDir("work")
	if err != nil {
		t.Fatal(err)
	}

	if err := Rename("work", "../escape"); err == nil {
		t.Fatal("Rename accepted a path-traversal profile name")
	}

	if _, err := os.Stat(originalDir); err != nil {
		t.Fatalf("original profile moved after rejected rename: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := cfg.Profiles["work"]; !exists {
		t.Fatal("original profile registration was lost after rejected rename")
	}
}

func TestRemoveDeletesStandardProfileWrapper(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := Create("work"); err != nil {
		t.Fatal(err)
	}

	configDir, err := DefaultConfigDir("work")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "settings.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	wrapper := filepath.Dir(configDir)
	if err := Remove("work"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(wrapper); !os.IsNotExist(err) {
		t.Fatalf("profile wrapper still exists after Remove: %s", wrapper)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := cfg.Profiles["work"]; exists {
		t.Fatal("removed profile is still registered in config")
	}
}
