package fsatomic

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWrite(t *testing.T) {
	t.Run("creates a new file with defaultMode", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.yaml")
		if err := Write(path, []byte("cloud: aws\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "cloud: aws\n" {
			t.Errorf("content = %q", got)
		}
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != 0o644 {
			t.Errorf("mode = %v, want 0644", fi.Mode().Perm())
		}
	})

	t.Run("replaces content and preserves existing permissions", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "kubeconfig")
		if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := Write(path, []byte("new"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "new" {
			t.Errorf("content = %q, want %q", got, "new")
		}
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != 0o600 {
			t.Errorf("mode = %v, want the original 0600, not defaultMode", fi.Mode().Perm())
		}
	})

	t.Run("leaves no temp files behind", func(t *testing.T) {
		dir := t.TempDir()
		if err := Write(filepath.Join(dir, "f"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".omnictx-") {
				t.Errorf("temp file left behind: %s", e.Name())
			}
		}
	})

	t.Run("missing parent directory is an error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "no", "such", "dir", "f")
		if err := Write(path, []byte("x"), 0o644); err == nil {
			t.Fatal("want error for missing parent directory")
		}
	})

	t.Run("failed write leaves the original untouched", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "f")
		if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
			t.Fatal(err)
		}
		// A read-only directory makes CreateTemp fail before any change.
		if err := os.Chmod(dir, 0o500); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
		if err := Write(path, []byte("clobbered"), 0o644); err == nil {
			t.Fatal("want error in read-only directory")
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "original" {
			t.Errorf("original file changed: %q", got)
		}
	})
}
