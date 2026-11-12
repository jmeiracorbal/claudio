package profile

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyDir recursively reconciles src into dst.
//
// Regular files are compared by SHA-256. Identical files are kept; different
// files are replaced in full from the source. The copy never patches a file
// partially and does not use staging or temporary directories.
//
// If an overwrite fails, the error is returned and the destination may contain
// a partial file. A later execution detects the checksum mismatch and replaces
// that file in full again.
//
// Symlinks are preserved as symlinks; their targets are not deep-copied.
func CopyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if d.Type()&os.ModeSymlink != 0 {
			return copySymlink(path, target)
		}

		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			return ensureDir(target, info.Mode().Perm()|0700)
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported file type: %s", path)
		}
		return copyFile(path, target, info.Mode().Perm())
	})
}

func ensureDir(path string, mode os.FileMode) error {
	info, err := os.Lstat(path)
	switch {
	case err == nil && info.IsDir():
		return os.Chmod(path, mode)
	case err == nil:
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("removing conflicting destination %s: %w", path, err)
		}
	case os.IsNotExist(err):
		// Nothing to reconcile.
	case err != nil:
		return err
	}

	if err := os.MkdirAll(path, mode); err != nil {
		return fmt.Errorf("creating destination directory %s: %w", path, err)
	}
	return nil
}

func copySymlink(src, dst string) error {
	want, err := os.Readlink(src)
	if err != nil {
		return err
	}

	info, err := os.Lstat(dst)
	switch {
	case err == nil && info.Mode()&os.ModeSymlink != 0:
		got, err := os.Readlink(dst)
		if err != nil {
			return err
		}
		if got == want {
			return nil
		}
	case err == nil:
		// Destination exists with a different type.
	case os.IsNotExist(err):
		return os.Symlink(want, dst)
	case err != nil:
		return err
	}

	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("removing destination symlink %s: %w", dst, err)
	}
	if err := os.Symlink(want, dst); err != nil {
		return fmt.Errorf("creating destination symlink %s: %w", dst, err)
	}
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	same, err := sameFileContent(src, dst)
	if err != nil {
		return err
	}
	if same {
		return os.Chmod(dst, mode)
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("removing destination file %s: %w", dst, err)
	}

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("copying %s to %s: %w", src, dst, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("closing destination file %s: %w", dst, err)
	}
	return nil
}

func sameFileContent(src, dst string) (bool, error) {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return false, err
	}

	dstInfo, err := os.Lstat(dst)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !dstInfo.Mode().IsRegular() || srcInfo.Size() != dstInfo.Size() {
		return false, nil
	}

	srcSum, err := fileChecksum(src)
	if err != nil {
		return false, err
	}
	dstSum, err := fileChecksum(dst)
	if err != nil {
		return false, err
	}
	return srcSum == dstSum, nil
}

func fileChecksum(path string) ([sha256.Size]byte, error) {
	var zero [sha256.Size]byte

	f, err := os.Open(path)
	if err != nil {
		return zero, err
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return zero, err
	}

	var sum [sha256.Size]byte
	copy(sum[:], hash.Sum(nil))
	return sum, nil
}

func writeFileIfDifferent(path string, data []byte, mode os.FileMode) error {
	info, err := os.Lstat(path)
	if err == nil && info.Mode().IsRegular() && info.Size() == int64(len(data)) {
		got, err := fileChecksum(path)
		if err != nil {
			return err
		}
		want := sha256.Sum256(data)
		if got == want {
			return os.Chmod(path, mode)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("removing destination file %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		return fmt.Errorf("writing destination file %s: %w", path, err)
	}
	return nil
}
