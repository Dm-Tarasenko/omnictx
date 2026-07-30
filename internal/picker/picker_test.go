package picker

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestDecide covers the full 2×2×{empty, non-empty} input space: the picker
// activates only for TTY ∧ fzf-present ∧ empty opt-out, and any non-empty
// OMNICTX_IGNORE_FZF value disables it (the KUBECTX_IGNORE_FZF semantics).
func TestDecide(t *testing.T) {
	tests := []struct {
		name      string
		tty       bool
		fzf       bool
		ignoreFzf string
		want      bool
	}{
		{"tty, fzf, no opt-out", true, true, "", true},
		{"tty, fzf, opt-out=1", true, true, "1", false},
		{"tty, fzf, opt-out=0 still disables", true, true, "0", false},
		{"tty, fzf, opt-out=false still disables", true, true, "false", false},
		{"tty, no fzf, no opt-out", true, false, "", false},
		{"tty, no fzf, opt-out=1", true, false, "1", false},
		{"no tty, fzf, no opt-out", false, true, "", false},
		{"no tty, fzf, opt-out=1", false, true, "1", false},
		{"no tty, no fzf, no opt-out", false, false, "", false},
		{"no tty, no fzf, opt-out=1", false, false, "1", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Decide(tt.tty, tt.fzf, tt.ignoreFzf); got != tt.want {
				t.Errorf("Decide(%v, %v, %q) = %v, want %v", tt.tty, tt.fzf, tt.ignoreFzf, got, tt.want)
			}
		})
	}
}

// TestCurrentFirst pins the default-selection contract of the bare commands:
// the current entity leads the fzf item list (Enter with no query re-selects
// it), everything else keeps its list order, an absent/empty current changes
// nothing, and the input slice is never mutated (the list tables share it).
func TestCurrentFirst(t *testing.T) {
	tests := []struct {
		name    string
		items   []string
		current string
		want    []string
	}{
		{"current already first", []string{"a", "b", "c"}, "a", []string{"a", "b", "c"}},
		{"current in the middle", []string{"a", "b", "c"}, "b", []string{"b", "a", "c"}},
		{"current last", []string{"a", "b", "c"}, "c", []string{"c", "a", "b"}},
		{"current absent", []string{"a", "b", "c"}, "x", []string{"a", "b", "c"}},
		{"empty current", []string{"a", "b", "c"}, "", []string{"a", "b", "c"}},
		{"empty items", nil, "a", nil},
		{"single item", []string{"a"}, "a", []string{"a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := append([]string(nil), tt.items...)
			got := CurrentFirst(tt.items, tt.current)
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("CurrentFirst(%v, %q) = %v, want %v", orig, tt.current, got, tt.want)
			}
			if strings.Join(tt.items, ",") != strings.Join(orig, ",") {
				t.Errorf("CurrentFirst mutated its input: %v, want %v", tt.items, orig)
			}
		})
	}
}

// An empty item list is a no-selection result, not an fzf invocation: Run must
// return before exec, so it succeeds even with no fzf anywhere on PATH.
func TestRunEmptyItems(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	sel, ok, err := Run(nil, "header")
	if sel != "" || ok || err != nil {
		t.Errorf("Run(nil) = (%q, %v, %v), want empty cancel without exec", sel, ok, err)
	}
}

// With fzf absent from PATH, Run reports an exec error (not a cancel): the
// caller distinguishes "user chose nothing" from "fzf could not run".
func TestRunExecFailure(t *testing.T) {
	t.Setenv("PATH", filepath.Join(t.TempDir(), "empty"))
	sel, ok, err := Run([]string{"a", "b"}, "header")
	if sel != "" || ok || err == nil {
		t.Errorf("Run without fzf = (%q, %v, %v), want error", sel, ok, err)
	}
}
