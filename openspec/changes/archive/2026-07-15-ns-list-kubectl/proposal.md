## Why

`omnictx ns list` currently rejects `list` as a reserved word because omnictx
never contacts the cluster — but users coming from kubens expect the natural
`ns list` to actually show cluster namespaces. The strict "no kubectl ever"
principle is being deliberately relaxed for explicit interactive subcommands:
they may shell out to `kubectl`, while render mode (the per-prompt hot path)
stays strictly offline so the prompt can never hang on a dead VPN or cluster.

## What Changes

- `omnictx ns list` (and the `namespace` alias) shells out to
  `kubectl get namespaces` and prints the cluster's namespaces as a
  `kube list`-style table with the current namespace marked.
- The reserved-word rejection of `list` in `runNamespace` is replaced by the
  listing implementation (`list` stays reserved as a switch target — it is now
  a real subcommand form instead of an error).
- Failure handling stays loud and non-destructive: kubectl missing from PATH
  or exiting non-zero → clear stderr message, exit 1, nothing written.
- Render mode is untouched: no network, no exec, never breaks the prompt.
- AGENTS.md is updated: the offline principle now reads "render mode is
  strictly offline; explicit subcommands may shell out to kubectl"; the Go
  dependency rule (no client-go, no network libraries) is unchanged.
- Help text (`--help`, usage strings) documents `ns [<name>|list]`.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `kube-namespace-cli`: the "No offline namespace listing" requirement is
  replaced by an online listing requirement — `ns list` runs
  `kubectl get namespaces` with a bounded request timeout, renders the result
  as a table marking the current namespace, and fails loudly (exit 1) when
  kubectl is unavailable or errors; `on`/`off` remain ordinary namespace
  names; render mode remains strictly offline and read-only.

## Impact

- `cmd/omnictx/main.go`: `runNamespace` gains the `list` branch (exec kubectl,
  parse names, print table); usage string and `--help` text updated.
- `cmd/omnictx/main_test.go`: `TestRunNamespaceListReserved` replaced by tests
  that stub `kubectl` with a fake executable prepended to PATH (success,
  kubectl-missing, kubectl-failure cases).
- `AGENTS.md`: offline principle reworded; `ns` subcommand description updated.
- Dependencies: none added — `os/exec` is stdlib; still no client-go or
  network libraries in go.mod.
