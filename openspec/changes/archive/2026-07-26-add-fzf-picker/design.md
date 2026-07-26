# add-fzf-picker — design

## Context

Bare `omnictx kube` prints the current context; bare `omnictx ns` prints the
active context's namespace — both offline, always exit 0 (`main.go` runKube /
runNamespace, no-argument arms). `kube list` / `ns list` print read-only
tables; `ns list` is today the binary's only online path (kubectl shell-out).
Switching is `kube <context>` / `ns <name>` with the write paths in
internal/kube. kubectx sets the precedent for the upgrade: the bare command,
when interactive, pipes the list through `fzf` and switches to the selection.
omnictx's twist: the bare print is redundant in a terminal (the prompt already
shows both values), so the picker replaces almost nothing interactively while
scripts — never a TTY — keep the print contract byte-for-byte.
Project constraints: no new Go dependency, render mode untouched, decision
logic pure and table-tested (the `aws.Directive` pattern), writes only on an
explicit user action.

## Goals / Non-Goals

**Goals:**
- Bare `kube` / `ns` become fuzzy pickers under TTY + fzf + no opt-out,
  performing the exact existing switch on selection.
- Byte-identical print output in every non-interactive situation; `list`
  tables untouched everywhere.
- Activation decision as a pure function, exhaustively table-tested.
- fzf as an optional external binary (shell-out), like kubectl in `ns list`.

**Non-Goals:**
- No picker for cloud accounts in this change — deferred to the parked
  `promote-provider-commands` change, which reuses `internal/picker`.
- No fzf UI customization surface (no flags/config for fzf options; the user's
  own `FZF_DEFAULT_OPTS` applies naturally).
- No bundling or vendoring of fzf; no fallback interactive UI of our own.

## Decisions

**D1 — Activation predicate is a pure function in a new `internal/picker`.**
`picker.Decide(stdoutIsTTY, fzfOnPath bool, ignoreFzf string) bool` — true
only when stdout is a terminal AND fzf was found on PATH AND
`OMNICTX_IGNORE_FZF` is empty. Inputs are gathered at the edge (main.go: the
existing `isTTY`, `exec.LookPath("fzf")`, `os.Getenv`) and the result is
passed into `runKube`/`runNamespace`, so every existing test keeps calling
them non-interactively without modification. Alternative considered: deciding
inside runKube — rejected because it hard-wires os state into functions that
are currently deterministic over their parameters.
`OMNICTX_IGNORE_FZF` follows kubectx's `KUBECTX_IGNORE_FZF` semantics: any
non-empty value disables.

**D2 — The trigger is the bare command, not `list`.** `kube list` / `ns list`
stay read-only tables in every environment — the inspection surface is
preserved verbatim. The bare command is the natural picker home because its
interactive job is already done by the prompt segment. This mirrors kubectx
exactly (bare `kubectx` → picker) and keeps the read/write boundary clean:
`list` never writes, the bare command writes only on an explicit selection.
Alternative considered (and initially drafted): upgrading `list` into the
picker — rejected because it takes away the human-readable table in a
terminal and muddies `list`'s read-only contract.

**D3 — fzf is fed bare names, one per line; the current entry goes in a
header, not inline.** `picker.Run(items []string, header string)` execs `fzf`
with the item list on stdin and `--header` naming the current context/
namespace. fzf's stdout is captured (the selection), stderr is inherited (fzf
draws its UI via the terminal). Selection output is therefore exactly a valid
name — no marker stripping, no table parsing. Empty item list → no fzf, fall
through to the non-interactive print.

**D4 — Selection reuses the existing switch paths verbatim.** kube: the
selected name goes through the same found-check + `kube.WriteContext` code as
`kube <context>` (same exit codes: broken target → 1). ns: the selected name
goes through `kube.WriteNamespace` like `ns <name>`. Success is silent,
matching every other switch (hook-running shells apply it on the next
prompt). The picker adds no third write path.

**D5 — Any non-zero fzf exit is a cancel: nothing written, exit 0.** Esc,
Ctrl-C, empty match — all mean "the user chose nothing", which is not an
error. If the exec itself fails (fzf vanished between LookPath and exec),
fall back to the non-interactive behavior — print the current value — exit 0.

**D6 — Interactive bare `ns` becomes the second (and last) kubectl path.**
The namespace list lives in the cluster, so the picker must shell out exactly
like `ns list` (`kubectl get namespaces -o name --request-timeout=10s`).
Difference in failure handling: `ns list` fails loudly (exit 1 — the listing
IS the command), but bare `ns` keeps its always-exit-0 contract — kubectl
missing or failing warns on stderr and falls back to printing the current
namespace. Render mode and every other subcommand stay strictly offline; the
AGENTS.md "only online path" sentence is updated to name both call sites.

**D7 — Mute interaction: none.** Switches change state, never visibility
(core invariant); a picker selection under a persisted mute still switches,
exactly like the typed commands. No display toggle is read or written.

## Risks / Trade-offs

- [A TTY user who wants the raw current value from bare `kube`/`ns` now gets
  a picker] → The prompt already displays both values; Esc exits instantly
  with no side effect; the raw value is one `| cat` or `OMNICTX_IGNORE_FZF=1`
  away; scripts are unaffected by construction (non-TTY).
- [Bare `ns` in a terminal now waits on kubectl — up to the 10s request
  timeout on a dead VPN] → Interactive-only, Ctrl-C works, the timeout is
  bounded, the fallback prints the current namespace so the command still
  answers; opt-out disables the whole branch.
- [fzf UX differs across user configs (`FZF_DEFAULT_OPTS`)] → Deliberate: we
  pass only `--header`, inheriting the user's look-and-feel like kubectx.
- [The exec wrapper itself is untestable in unit tests] → Kept to a few
  lines; everything decidable (activation, fallback, cancel handling, exit
  codes) lives in pure functions or in runKube/runNamespace table tests with
  an injected picker result.

## Open Questions

_None — trigger placement (bare command vs `list`) was resolved with the user
in proposal review; cloud pickers are explicitly out of scope._
