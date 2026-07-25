package shellinit

import (
	"os/exec"
	"strings"
	"testing"
)

func TestGenerateBash(t *testing.T) {
	out, err := Generate("bash", "omnictx")
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, out, []string{
		"__OMNICTX_BASH_LOADED",       // idempotency guard
		"__OMNICTX_ORIG_PS1=\"$PS1\"", // captures original prompt once
		"omnictx hook --shell bash",   // single per-prompt exec, correct shell
		"PROMPT_COMMAND",              // registers hook
		// Env-sync plumbing: markers exported alongside the value, unset
		// removes both, values applied via quoted parameters — never eval'd.
		`export AWS_PROFILE="$__omnictx_profile" __OMNICTX_AWS_PROFILE="$__omnictx_profile"`,
		`export AWS_REGION="$__omnictx_region" __OMNICTX_AWS_REGION="$__omnictx_region"`,
		"unset AWS_PROFILE __OMNICTX_AWS_PROFILE",
		"unset AWS_REGION __OMNICTX_AWS_REGION",
	})
	if strings.Contains(out, "precmd_functions") {
		t.Errorf("bash output should not reference zsh precmd_functions")
	}
	mustNotEval(t, out)
}

func TestGenerateZsh(t *testing.T) {
	out, err := Generate("zsh", "omnictx")
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, out, []string{
		"__OMNICTX_ZSH_LOADED",                 // idempotency guard
		"__OMNICTX_ORIG_PROMPT=\"$PROMPT\"",    // captures original prompt once
		"omnictx hook --shell zsh",             // single per-prompt exec, correct shell
		"precmd_functions+=(__omnictx_precmd)", // registers hook
		`export AWS_PROFILE="$__omnictx_profile" __OMNICTX_AWS_PROFILE="$__omnictx_profile"`,
		"unset AWS_PROFILE __OMNICTX_AWS_PROFILE",
	})
	mustNotEval(t, out)
}

// mustNotEval asserts no snippet code line evals binary output (the install
// one-liner in the leading comment is the only allowed mention of eval).
func mustNotEval(t *testing.T, out string) {
	t.Helper()
	for line := range strings.SplitSeq(out, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if strings.Contains(line, "eval") {
			t.Errorf("snippet code must not eval binary output: %q", line)
		}
	}
}

func TestGenerateUsesCmd(t *testing.T) {
	out, err := Generate("bash", "/opt/bin/omnictx")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "/opt/bin/omnictx hook --shell bash") {
		t.Fatalf("expected custom command path in output, got:\n%s", out)
	}
}

func TestGenerateUnsupportedShell(t *testing.T) {
	if _, err := Generate("fish", "omnictx"); err == nil {
		t.Fatal("expected error for unsupported shell")
	}
}

func TestGenerateDefaultsCmd(t *testing.T) {
	out, err := Generate("bash", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "omnictx hook --shell bash") {
		t.Fatalf("empty cmd should default to 'omnictx', got:\n%s", out)
	}
}

// TestBashSnippetIsValidAndIdempotent eval's the generated bash twice in a
// non-interactive shell and asserts that no shell functions beyond the prompt
// hook are defined (the omnion/omnioff/omnitoggle helpers were removed) and
// that the original prompt was captured exactly once (idempotency).
func TestBashSnippetIsValidAndIdempotent(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available")
	}
	snippet, err := Generate("bash", "true") // use 'true' so the hook is harmless
	if err != nil {
		t.Fatal(err)
	}

	script := `PS1='orig> '
` + snippet + `
` + snippet + `
type omnitoggle >/dev/null 2>&1 && echo HAS_OMNITOGGLE_SHOULD_NOT_EXIST
type omnion >/dev/null 2>&1 && echo HAS_OMNION_SHOULD_NOT_EXIST
type omnioff >/dev/null 2>&1 && echo HAS_OMNIOFF_SHOULD_NOT_EXIST
echo "ORIG=${__OMNICTX_ORIG_PS1}"
__omnictx_prompt
echo "PS1=${PS1}"
`
	cmd := exec.Command(bash, "--norc", "--noprofile", "-c", script)
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bash eval failed: %v\n%s", err, outBytes)
	}
	out := string(outBytes)
	for _, want := range []string{"ORIG=orig> "} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in bash output:\n%s", want, out)
		}
	}
	// After double-eval, the captured original must still be 'orig> ', not a
	// doubled/clobbered value.
	if strings.Contains(out, "ORIG=orig> orig>") {
		t.Errorf("original prompt was double-captured (not idempotent):\n%s", out)
	}
	if strings.Contains(out, "HAS_OMNITOGGLE_SHOULD_NOT_EXIST") {
		t.Errorf("omnitoggle should not be defined:\n%s", out)
	}
	if strings.Contains(out, "HAS_OMNION_SHOULD_NOT_EXIST") {
		t.Errorf("omnion should not be defined:\n%s", out)
	}
	if strings.Contains(out, "HAS_OMNIOFF_SHOULD_NOT_EXIST") {
		t.Errorf("omnioff should not be defined:\n%s", out)
	}
}

// TestBashSnippetAppliesDirectives drives the generated bash snippet through
// the three directive encodings with a fake hook binary (a shell function):
// an export directive sets the variable plus its marker and both are exported
// (visible to child processes), `-` unsets both, and an empty directive leaves
// a manual value alone.
func TestBashSnippetAppliesDirectives(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available")
	}
	snippet, err := Generate("bash", "__omnictx_fake")
	if err != nil {
		t.Fatal(err)
	}

	script := `PS1='orig> '
__omnictx_fake() { printf 'prod\neu-central-1\nSEG\n'; }
` + snippet + `
__omnictx_prompt
echo "PS1=${PS1}"
bash -c 'echo "CHILD=${AWS_PROFILE}/${AWS_REGION}/${__OMNICTX_AWS_PROFILE}/${__OMNICTX_AWS_REGION}"'
__omnictx_fake() { printf -- '-\n-\n\n'; }
__omnictx_prompt
echo "AFTER_UNSET=${AWS_PROFILE:-gone}/${AWS_REGION:-gone}/${__OMNICTX_AWS_PROFILE:-gone}/${__OMNICTX_AWS_REGION:-gone}"
echo "PS1_AFTER=${PS1}"
export AWS_PROFILE=manual
__omnictx_fake() { printf '\n\nSEG2\n'; }
__omnictx_prompt
echo "MANUAL=${AWS_PROFILE}"
`
	cmd := exec.Command(bash, "--norc", "--noprofile", "-c", script)
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bash eval failed: %v\n%s", err, outBytes)
	}
	out := string(outBytes)
	for _, want := range []string{
		"PS1=SEG orig> ", // segment applied on the same cycle
		"CHILD=prod/eu-central-1/prod/eu-central-1", // value + marker exported to children
		"AFTER_UNSET=gone/gone/gone/gone",           // unset removes variable and marker
		"PS1_AFTER=orig> ",                          // empty segment restores the original prompt
		"MANUAL=manual",                             // empty directive leaves a manual value alone
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in bash output:\n%s", want, out)
		}
	}
}

// TestZshSnippetAppliesDirectives is the zsh twin of the bash directive test.
func TestZshSnippetAppliesDirectives(t *testing.T) {
	zsh, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("zsh not available")
	}
	snippet, err := Generate("zsh", "__omnictx_fake")
	if err != nil {
		t.Fatal(err)
	}

	script := `PROMPT='orig> '
__omnictx_fake() { printf 'prod\neu-central-1\nSEG\n'; }
` + snippet + `
__omnictx_precmd
echo "PROMPT=${PROMPT}"
zsh -c 'echo "CHILD=${AWS_PROFILE}/${AWS_REGION}/${__OMNICTX_AWS_PROFILE}/${__OMNICTX_AWS_REGION}"'
__omnictx_fake() { printf -- '-\n-\n\n'; }
__omnictx_precmd
echo "AFTER_UNSET=${AWS_PROFILE:-gone}/${AWS_REGION:-gone}/${__OMNICTX_AWS_PROFILE:-gone}/${__OMNICTX_AWS_REGION:-gone}"
`
	cmd := exec.Command(zsh, "-f", "-c", script)
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("zsh eval failed: %v\n%s", err, outBytes)
	}
	out := string(outBytes)
	for _, want := range []string{
		"PROMPT=SEG orig> ",
		"CHILD=prod/eu-central-1/prod/eu-central-1",
		"AFTER_UNSET=gone/gone/gone/gone",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in zsh output:\n%s", want, out)
		}
	}
}

func mustContain(t *testing.T, haystack string, needles []string) {
	t.Helper()
	for _, n := range needles {
		if !strings.Contains(haystack, n) {
			t.Errorf("output missing %q\n---\n%s", n, haystack)
		}
	}
}
