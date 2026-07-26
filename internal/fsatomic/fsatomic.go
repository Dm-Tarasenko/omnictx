// Package fsatomic is the single atomic-write primitive behind every write
// path in omnictx: the foreign-file switches (kubeconfig, active_config,
// azureProfile.json) and omnictx's own config. Same-directory temp file,
// fsync, rename — a failure at any step leaves the original file untouched.
package fsatomic

import (
	"os"
	"path/filepath"
)

// Write replaces path with data via a temp file in the same directory and an
// atomic rename, fsyncing before the swap so a crash cannot leave a truncated
// file. The original permission bits are preserved when the file exists;
// defaultMode applies otherwise.
func Write(path string, data []byte, defaultMode os.FileMode) error {
	mode := defaultMode
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".omnictx-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // no-op once renamed

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
