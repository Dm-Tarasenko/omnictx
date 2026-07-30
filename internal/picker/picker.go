// Package picker implements the interactive fuzzy-pick branch of the bare
// kube/ns subcommands: a pure activation decision plus a thin fzf shell-out.
// fzf is an optional external binary (like kubectl in `ns list`), never a Go
// dependency — when it is absent the bare commands keep their print behavior.
package picker

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

// Decide reports whether the bare command should run the interactive picker:
// stdout must be a terminal, fzf must have been found on PATH, and ignoreFzf —
// the raw value of OMNICTX_IGNORE_FZF — must be empty. ANY non-empty value
// disables the picker, mirroring kubectx's KUBECTX_IGNORE_FZF semantics: "1",
// "true", and even "0" or "false" all opt out. The inputs are gathered at the
// dispatch edge so this stays a pure function.
func Decide(stdoutIsTTY, fzfOnPath bool, ignoreFzf string) bool {
	return stdoutIsTTY && fzfOnPath && ignoreFzf == ""
}

// CurrentFirst returns items with current moved to the front, the rest keeping
// their order. fzf highlights the first stdin line by default, so the bare
// commands pass their item list through this before Run: Enter with no query
// then re-selects the active entity instead of whatever the list order happened
// to put first — a pick without an explicit choice never changes state. An
// absent (or empty) current leaves the order untouched. The input slice is
// never mutated — the same slice backs the read-only list tables.
func CurrentFirst(items []string, current string) []string {
	for i, it := range items {
		if it != current {
			continue
		}
		if i == 0 {
			return items
		}
		out := make([]string, 0, len(items))
		out = append(out, current)
		out = append(out, items[:i]...)
		out = append(out, items[i+1:]...)
		return out
	}
	return items
}

// Run pipes items (one per line) to `fzf --header <header>` and returns the
// user's selection. fzf draws its UI through the inherited stderr; stdout is
// captured and trimmed to the selected line. ok=false with a nil error means
// the user chose nothing — a non-zero fzf exit (Esc, Ctrl-C, no match) is a
// cancel, not an error. A non-nil error means fzf could not be executed at
// all; the caller then falls back to its non-interactive behavior. An empty
// items list never invokes fzf.
func Run(items []string, header string) (selection string, ok bool, err error) {
	if len(items) == 0 {
		return "", false, nil
	}
	cmd := exec.Command("fzf", "--header", header)
	cmd.Stdin = strings.NewReader(strings.Join(items, "\n") + "\n")
	cmd.Stderr = os.Stderr
	var out strings.Builder
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", false, nil
		}
		return "", false, err
	}
	return strings.TrimSpace(out.String()), true, nil
}
