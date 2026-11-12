package profile

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCopyDirOverwritesDifferentFileCompletely(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	srcFile := filepath.Join(src, "settings.json")
	dstFile := filepath.Join(dst, "settings.json")

	if err := os.WriteFile(srcFile, []byte("new"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dstFile, []byte("old-content-that-must-disappear"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := CopyDir(src, dst); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("destination content = %q, want complete overwrite %q", got, "new")
	}
}

func TestCopyDirRepairsPartialFileOnRerun(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	srcFile := filepath.Join(src, "plugin.json")
	dstFile := filepath.Join(dst, "plugin.json")

	want := []byte("complete-payload")
	if err := os.WriteFile(srcFile, want, 0o600); err != nil {
		t.Fatal(err)
	}

	// Simulate a previous copy that stopped after writing only part of the file.
	if err := os.WriteFile(dstFile, []byte("complete"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := CopyDir(src, dst); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("destination content = %q, want %q", got, want)
	}
}

func TestCopyDirOverwritesSameSizeDifferentContent(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	srcFile := filepath.Join(src, "CLAUDE.md")
	dstFile := filepath.Join(dst, "CLAUDE.md")

	if err := os.WriteFile(srcFile, []byte("abcdef"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dstFile, []byte("abcdeg"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := CopyDir(src, dst); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "abcdef" {
		t.Fatalf("destination content = %q, want %q", got, "abcdef")
	}
}

func TestCopyDirKeepsIdenticalFile(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	srcFile := filepath.Join(src, "settings.json")
	dstFile := filepath.Join(dst, "settings.json")

	content := []byte("{\"same\":true}")
	if err := os.WriteFile(srcFile, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dstFile, content, 0o600); err != nil {
		t.Fatal(err)
	}

	oldTime := time.Unix(1_000_000, 0)
	if err := os.Chtimes(dstFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	if err := CopyDir(src, dst); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(dstFile)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(oldTime) {
		t.Fatalf("identical destination file was rewritten: mtime = %s, want %s", info.ModTime(), oldTime)
	}
}

func TestCopyDirReconcilesSymlinkTarget(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	srcLink := filepath.Join(src, "current")
	dstLink := filepath.Join(dst, "current")

	if err := os.Symlink("wanted-target", srcLink); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("old-target", dstLink); err != nil {
		t.Fatal(err)
	}

	if err := CopyDir(src, dst); err != nil {
		t.Fatal(err)
	}

	got, err := os.Readlink(dstLink)
	if err != nil {
		t.Fatal(err)
	}
	if got != "wanted-target" {
		t.Fatalf("destination symlink target = %q, want %q", got, "wanted-target")
	}
}

func TestCopyStateOverwritesDifferentContent(t *testing.T) {
	previousRules := rules
	defer func() { rules = previousRules }()

	rules.Copy.StateKeys = []string{"mcpServers"}
	rules.Copy.HistoryStateKeys = nil

	srcDir := t.TempDir()
	dstDir := t.TempDir()
	srcFile := filepath.Join(srcDir, ".claude.json")
	dstFile := filepath.Join(dstDir, ".claude.json")

	if err := os.WriteFile(srcFile, []byte(`{"mcpServers":{"example":{"command":"new"}},"ignored":"value"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dstFile, []byte(`{"mcpServers":{"example":{"command":"old"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := CopyState(srcFile, dstDir, false); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"mcpServers\": {\n    \"example\": {\n      \"command\": \"new\"\n    }\n  }\n}"
	if string(got) != want {
		t.Fatalf("destination state = %q, want %q", got, want)
	}
}
