package restore

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jmeiracorbal/claudio/internal/profile"
)

func TestApplyRollsBackPartialCopy(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "profile")
	dst := filepath.Join(root, ".claude")
	stateDst := filepath.Join(root, ".claude.json")

	mustWriteFile(t, filepath.Join(src, "new.txt"), "new")
	mustWriteFile(t, filepath.Join(dst, "old.txt"), "old")
	mustWriteFile(t, stateDst, "old-state")

	ops := operations{
		copyDir: func(_, target string) error {
			mustWriteFile(t, filepath.Join(target, "partial.txt"), "partial")
			return errors.New("copy failed")
		},
		rewritePaths: func(string, string) error { return nil },
		rename:       os.Rename,
	}

	_, err := apply(src, dst, stateDst, fixedTime(), ops)
	if err == nil || !strings.Contains(err.Error(), "copy failed") {
		t.Fatalf("apply error = %v, want copy failure", err)
	}

	assertFileContent(t, filepath.Join(dst, "old.txt"), "old")
	assertNotExists(t, filepath.Join(dst, "partial.txt"))
	assertFileContent(t, stateDst, "old-state")
	assertNoBackups(t, dst, stateDst)
}

func TestApplyRollsBackRewriteFailure(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "profile")
	dst := filepath.Join(root, ".claude")
	stateDst := filepath.Join(root, ".claude.json")

	mustWriteFile(t, filepath.Join(src, "new.txt"), "new")
	mustWriteFile(t, filepath.Join(dst, "old.txt"), "old")
	mustWriteFile(t, stateDst, "old-state")

	ops := operations{
		copyDir: profile.CopyDir,
		rewritePaths: func(string, string) error {
			return errors.New("rewrite failed")
		},
		rename: os.Rename,
	}

	_, err := apply(src, dst, stateDst, fixedTime(), ops)
	if err == nil || !strings.Contains(err.Error(), "rewrite failed") {
		t.Fatalf("apply error = %v, want rewrite failure", err)
	}

	assertFileContent(t, filepath.Join(dst, "old.txt"), "old")
	assertNotExists(t, filepath.Join(dst, "new.txt"))
	assertFileContent(t, stateDst, "old-state")
	assertNoBackups(t, dst, stateDst)
}

func TestApplyRollsBackWhenStateBackupFails(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "profile")
	dst := filepath.Join(root, ".claude")
	stateDst := filepath.Join(root, ".claude.json")

	mustWriteFile(t, filepath.Join(src, "new.txt"), "new")
	mustWriteFile(t, filepath.Join(dst, "old.txt"), "old")
	mustWriteFile(t, stateDst, "old-state")

	ops := operations{
		copyDir:      profile.CopyDir,
		rewritePaths: func(string, string) error { return nil },
		rename: func(from, to string) error {
			if from == stateDst && strings.HasPrefix(to, stateDst+".bak.") {
				return errors.New("state backup failed")
			}
			return os.Rename(from, to)
		},
	}

	_, err := apply(src, dst, stateDst, fixedTime(), ops)
	if err == nil || !strings.Contains(err.Error(), "state backup failed") {
		t.Fatalf("apply error = %v, want state backup failure", err)
	}

	assertFileContent(t, filepath.Join(dst, "old.txt"), "old")
	assertFileContent(t, stateDst, "old-state")
	assertNoBackups(t, dst, stateDst)
}

func TestApplyRollsBackWhenInstallingStateFails(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "profile")
	dst := filepath.Join(root, ".claude")
	stateDst := filepath.Join(root, ".claude.json")

	mustWriteFile(t, filepath.Join(src, "new.txt"), "new")
	mustWriteFile(t, filepath.Join(src, ".claude.json"), "new-state")
	mustWriteFile(t, filepath.Join(dst, "old.txt"), "old")
	mustWriteFile(t, stateDst, "old-state")

	ops := operations{
		copyDir:      profile.CopyDir,
		rewritePaths: func(string, string) error { return nil },
		rename: func(from, to string) error {
			if from == filepath.Join(dst, ".claude.json") && to == stateDst {
				return errors.New("state install failed")
			}
			return os.Rename(from, to)
		},
	}

	_, err := apply(src, dst, stateDst, fixedTime(), ops)
	if err == nil || !strings.Contains(err.Error(), "state install failed") {
		t.Fatalf("apply error = %v, want state install failure", err)
	}

	assertFileContent(t, filepath.Join(dst, "old.txt"), "old")
	assertNotExists(t, filepath.Join(dst, "new.txt"))
	assertFileContent(t, stateDst, "old-state")
	assertNoBackups(t, dst, stateDst)
}

func TestApplyCommitsAndRetainsBackups(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "profile")
	dst := filepath.Join(root, ".claude")
	stateDst := filepath.Join(root, ".claude.json")

	mustWriteFile(t, filepath.Join(src, "new.txt"), "new")
	mustWriteFile(t, filepath.Join(src, ".claude.json"), "new-state")
	mustWriteFile(t, filepath.Join(dst, "old.txt"), "old")
	mustWriteFile(t, stateDst, "old-state")

	ops := operations{
		copyDir:      profile.CopyDir,
		rewritePaths: func(string, string) error { return nil },
		rename:       os.Rename,
	}

	result, err := apply(src, dst, stateDst, fixedTime(), ops)
	if err != nil {
		t.Fatal(err)
	}

	assertFileContent(t, filepath.Join(dst, "new.txt"), "new")
	assertNotExists(t, filepath.Join(dst, "old.txt"))
	assertNotExists(t, filepath.Join(dst, ".claude.json"))
	assertFileContent(t, stateDst, "new-state")

	if result.ConfigBackup == "" || result.StateBackup == "" {
		t.Fatalf("missing successful backup paths: %+v", result)
	}
	assertFileContent(t, filepath.Join(result.ConfigBackup, "old.txt"), "old")
	assertFileContent(t, result.StateBackup, "old-state")
}

func TestApplyRejectsOverlappingSourceAndDestination(t *testing.T) {
	root := t.TempDir()
	dst := filepath.Join(root, ".claude")
	src := filepath.Join(dst, "profile")
	stateDst := filepath.Join(root, ".claude.json")

	mustWriteFile(t, filepath.Join(src, "new.txt"), "new")
	mustWriteFile(t, filepath.Join(dst, "old.txt"), "old")

	_, err := apply(src, dst, stateDst, fixedTime(), operations{
		copyDir:      profile.CopyDir,
		rewritePaths: func(string, string) error { return nil },
		rename:       os.Rename,
	})
	if err == nil || !strings.Contains(err.Error(), "overlap") {
		t.Fatalf("apply error = %v, want overlapping-path error", err)
	}

	assertFileContent(t, filepath.Join(dst, "old.txt"), "old")
	assertFileContent(t, filepath.Join(src, "new.txt"), "new")
}

func TestApplyDoesNotTouchInstallationWhenBackupAlreadyExists(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "profile")
	dst := filepath.Join(root, ".claude")
	stateDst := filepath.Join(root, ".claude.json")
	stamp := fixedTime().Format("20060102150405.000000000")
	configBackup := dst + ".bak." + stamp

	mustWriteFile(t, filepath.Join(src, "new.txt"), "new")
	mustWriteFile(t, filepath.Join(dst, "old.txt"), "old")
	mustWriteFile(t, filepath.Join(configBackup, "existing.txt"), "backup")

	_, err := apply(src, dst, stateDst, fixedTime(), operations{
		copyDir:      profile.CopyDir,
		rewritePaths: func(string, string) error { return nil },
		rename:       os.Rename,
	})
	if err == nil || !strings.Contains(err.Error(), "backup already exists") {
		t.Fatalf("apply error = %v, want backup collision", err)
	}

	assertFileContent(t, filepath.Join(dst, "old.txt"), "old")
	assertFileContent(t, filepath.Join(configBackup, "existing.txt"), "backup")
}

func fixedTime() time.Time {
	return time.Date(2026, 11, 12, 19, 0, 0, 123456789, time.UTC)
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("%s content = %q, want %q", path, got, want)
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("%s exists after rollback", path)
	}
}

func assertNoBackups(t *testing.T, dst, stateDst string) {
	t.Helper()
	matches, err := filepath.Glob(dst + ".bak.*")
	if err != nil {
		t.Fatal(err)
	}
	stateMatches, err := filepath.Glob(stateDst + ".bak.*")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 || len(stateMatches) != 0 {
		t.Fatalf("rollback left backup residues: config=%v state=%v", matches, stateMatches)
	}
}
