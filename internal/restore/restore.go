package restore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmeiracorbal/claudio/internal/profile"
)

// Result describes the backups retained after a successful restore.
type Result struct {
	ConfigBackup string
	StateBackup  string
}

type operations struct {
	copyDir      func(string, string) error
	rewritePaths func(string, string) error
	rename       func(string, string) error
}

var defaultOperations = operations{
	copyDir:      profile.CopyDir,
	rewritePaths: profile.RewritePaths,
	rename:       os.Rename,
}

// Apply restores src into dst and stateDst.
//
// Existing local data is renamed to timestamped backups first. From the first
// successful backup rename until commit, every returned error triggers rollback
// before it is returned to the caller. No staging or temporary directory is
// created.
func Apply(src, dst, stateDst string, now time.Time) (Result, error) {
	return apply(src, dst, stateDst, now, defaultOperations)
}

type transaction struct {
	src         string
	dst         string
	stateDst    string
	configBak   string
	stateBak    string
	hadConfig   bool
	hadState    bool
	backedConfig bool
	backedState  bool
	installStarted bool
	installedState bool
	committed      bool
	ops            operations
}

func apply(src, dst, stateDst string, now time.Time, ops operations) (result Result, err error) {
	if err := validatePaths(src, dst); err != nil {
		return Result{}, err
	}

	stamp := now.Format("20060102150405.000000000")
	tx := transaction{
		src:       src,
		dst:       dst,
		stateDst:  stateDst,
		configBak: dst + ".bak." + stamp,
		stateBak:  stateDst + ".bak." + stamp,
		ops:        ops,
	}

	if err := tx.preflight(); err != nil {
		return Result{}, err
	}

	defer func() {
		if err == nil || tx.committed {
			return
		}
		if rollbackErr := tx.rollback(); rollbackErr != nil {
			err = errors.Join(err, fmt.Errorf("rollback failed: %w", rollbackErr))
		}
	}()

	if err = tx.backup(); err != nil {
		return Result{}, err
	}

	tx.installStarted = true
	if err = os.MkdirAll(dst, 0o755); err != nil {
		return Result{}, fmt.Errorf("creating %s: %w", dst, err)
	}

	if err = ops.copyDir(src, dst); err != nil {
		return Result{}, fmt.Errorf("copying files: %w", err)
	}

	if err = ops.rewritePaths(dst, src); err != nil {
		return Result{}, fmt.Errorf("rewriting paths: %w", err)
	}

	state := filepath.Join(dst, ".claude.json")
	hasState, stateErr := pathExists(state)
	if stateErr != nil {
		err = fmt.Errorf("checking restored state %s: %w", state, stateErr)
		return Result{}, err
	}
	if hasState {
		if err = ops.rename(state, stateDst); err != nil {
			return Result{}, fmt.Errorf("moving %s to %s: %w", state, stateDst, err)
		}
		tx.installedState = true
	}

	tx.committed = true
	if tx.backedConfig {
		result.ConfigBackup = tx.configBak
	}
	if tx.backedState {
		result.StateBackup = tx.stateBak
	}
	return result, nil
}

func (tx *transaction) preflight() error {
	info, err := os.Stat(tx.src)
	if err != nil {
		return fmt.Errorf("reading restore source %s: %w", tx.src, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("restore source is not a directory: %s", tx.src)
	}

	var exists bool
	if exists, err = pathExists(tx.dst); err != nil {
		return fmt.Errorf("checking %s: %w", tx.dst, err)
	}
	tx.hadConfig = exists

	if exists, err = pathExists(tx.stateDst); err != nil {
		return fmt.Errorf("checking %s: %w", tx.stateDst, err)
	}
	tx.hadState = exists

	if tx.hadConfig {
		if exists, err = pathExists(tx.configBak); err != nil {
			return fmt.Errorf("checking backup %s: %w", tx.configBak, err)
		}
		if exists {
			return fmt.Errorf("backup already exists: %s", tx.configBak)
		}
	}
	if tx.hadState {
		if exists, err = pathExists(tx.stateBak); err != nil {
			return fmt.Errorf("checking backup %s: %w", tx.stateBak, err)
		}
		if exists {
			return fmt.Errorf("backup already exists: %s", tx.stateBak)
		}
	}
	return nil
}

func (tx *transaction) backup() error {
	if tx.hadConfig {
		if err := tx.ops.rename(tx.dst, tx.configBak); err != nil {
			return fmt.Errorf("backing up %s: %w", tx.dst, err)
		}
		tx.backedConfig = true
	}

	if tx.hadState {
		if err := tx.ops.rename(tx.stateDst, tx.stateBak); err != nil {
			return fmt.Errorf("backing up %s: %w", tx.stateDst, err)
		}
		tx.backedState = true
	}
	return nil
}

func (tx *transaction) rollback() error {
	var rollbackErrors []error

	if tx.installStarted {
		if err := os.RemoveAll(tx.dst); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("removing incomplete %s: %w", tx.dst, err))
		}
	}
	if tx.installedState {
		if err := os.RemoveAll(tx.stateDst); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("removing incomplete %s: %w", tx.stateDst, err))
		}
	}

	if tx.backedConfig {
		exists, err := pathExists(tx.dst)
		if err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("checking rollback destination %s: %w", tx.dst, err))
		} else if exists {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("cannot restore backup while destination exists: %s", tx.dst))
		} else if err := tx.ops.rename(tx.configBak, tx.dst); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("restoring %s: %w", tx.dst, err))
		}
	}

	if tx.backedState {
		exists, err := pathExists(tx.stateDst)
		if err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("checking rollback destination %s: %w", tx.stateDst, err))
		} else if exists {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("cannot restore backup while destination exists: %s", tx.stateDst))
		} else if err := tx.ops.rename(tx.stateBak, tx.stateDst); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("restoring %s: %w", tx.stateDst, err))
		}
	}

	return errors.Join(rollbackErrors...)
}

func validatePaths(src, dst string) error {
	srcAbs, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	dstAbs, err := filepath.Abs(dst)
	if err != nil {
		return err
	}

	if containsPath(srcAbs, dstAbs) || containsPath(dstAbs, srcAbs) {
		return fmt.Errorf("restore source and destination overlap: %s and %s", src, dst)
	}
	return nil
}

func containsPath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func pathExists(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
