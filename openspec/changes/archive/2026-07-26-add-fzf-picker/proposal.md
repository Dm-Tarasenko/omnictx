# add-fzf-picker

## Why

Kube context names in the wild are long and hostile to typing (`arn:aws:eks:eu-west-1:123456789:cluster/prod-payments`), and namespace lists run into dozens of entries. Aliases (`aliases.<provider>.<short>`) solve this for cloud accounts, whose names the user controls, but kube contexts and namespaces arrive named by the cluster tooling — an alias per context does not scale. kubectx/kubens proved the answer: the bare command becomes an fzf fuzzy pick when interactive, and switching is one keystroke away. omnictx is even better positioned for this trigger: the bare commands' current job — printing the active context/namespace — is redundant in a terminal, because the active values are already in the prompt (that is the whole point of omnictx). Scripts keep the print behavior untouched: they are never a TTY.

## What Changes

- Bare `omnictx kube` becomes an interactive fuzzy picker when (and only when) stdout is a terminal, `fzf` is on `PATH`, and `OMNICTX_IGNORE_FZF` is not set: the context list is piped to `fzf`, and selecting an entry performs exactly the `kube <context>` switch. In a pipe, without fzf, or with the opt-out set, it prints the current context exactly as today.
- Bare `omnictx ns` gets the same upgrade over the kubectl-sourced namespace list (the picker needs the cluster's namespaces, so the interactive branch shells out to kubectl like `ns list` does); selecting performs exactly the `ns <name>` switch. Non-interactive invocations print the current namespace as today and stay strictly offline. If kubectl is missing or fails in the interactive branch, the command warns on stderr and falls back to printing the current namespace, exit 0 — bare `ns` keeps its always-exit-0 contract.
- `kube list` and `ns list` are UNTOUCHED: always the read-only table, byte-identical to today, in every environment.
- Cancelling the picker (Esc / Ctrl-C) writes nothing and exits 0.
- The activate-or-not decision is a pure, table-tested Go function (the `aws.Directive` pattern); the fzf invocation itself is a thin exec shell-out, exactly like `kubectl` in `ns list` today.
- Render mode and the hook are untouched; no new Go dependency (fzf is an optional external binary, not an import).
- Out of scope here: pickers for cloud accounts. They are deferred to the parked follow-up change `promote-provider-commands` (top-level `omnictx aws|azure|gcp` with the same two-branch picker pattern), which reuses `internal/picker` from this change.

## Capabilities

### New Capabilities

_None — the picker is a conditional interactive mode of the existing bare commands, not a standalone capability._

### Modified Capabilities

- `kube-context-cli`: the "Print the current context with `omnictx kube`" requirement gains the interactive branch — TTY + fzf + no opt-out → fuzzy pick that performs the context switch; otherwise the existing print, unchanged.
- `kube-namespace-cli`: the "Print the current namespace with `omnictx ns`" requirement gains the same interactive branch (kubectl-sourced list, switch on selection, graceful fallback to the print when kubectl is unavailable); the "List cluster namespaces" requirement is amended only in its exclusivity wording — kubectl invocation is now allowed from exactly two places (`ns list` and the interactive bare `ns`), render mode and everything else stay offline.

## Impact

- `cmd/omnictx/main.go`: `runKube` (no-argument arm) and `runNamespace` (no-argument arm) route through the picker decision before printing; the `list` arms are not touched.
- New `internal/picker`: pure decision function (inputs: TTY-ness, fzf lookup result, `OMNICTX_IGNORE_FZF` value → interactive yes/no) + thin fzf exec wrapper.
- Specs: delta files for `kube-context-cli` and `kube-namespace-cli`.
- Docs: AGENTS.md (`kube` / `ns` command descriptions; the "ns list is the ONLY online path" sentence becomes "the only online paths are `ns list` and the interactive bare `ns` picker"); README mention of the optional fzf integration and `OMNICTX_IGNORE_FZF`.
- Invariants preserved: switches change state, never visibility (picker selection under a persisted mute still switches); writes happen only on an explicit user action (the selection); render mode stays offline and never writes; scripts and non-TTY consumers see no behavioral change.
