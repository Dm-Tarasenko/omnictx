# ns-reserve-list

## Why

`omnictx ns list` today silently sets the active context's namespace to the
literal value `list` — a valid DNS-1123 label, so the write succeeds. Users
coming from `omnictx kube list` (and every other CLI with a `list` verb)
reasonably expect a listing, and instead get their kubeconfig switched to a
namespace that almost certainly does not exist in the cluster. This is a
footgun in a write path to a foreign file.

omnictx is offline by design (no kubectl, no network — see AGENTS.md core
invariant), so it *cannot* enumerate cluster namespaces. The honest minimal
fix is to reserve `list` in the `ns` subcommand and fail loudly instead of
writing.

## What Changes

- **BREAKING** (edge case): `omnictx ns list` no longer sets the namespace to
  `list`. It becomes a reserved word: print an error to stderr explaining that
  omnictx is offline and cannot list cluster namespaces (pointing at
  `kubectl get namespaces`), write nothing, and exit 2.
- Same for the `namespace` alias (`omnictx namespace list`).
- Only `list` is reserved. `on`/`off` stay valid namespace names: `ns` has no
  toggle form (the kube toggle gates the whole segment), so unlike `kube`
  there is nothing for them to collide with.
- `--help` output and code comments updated to reflect the reserved word.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kube-namespace-cli`: the "No offline namespace listing" requirement flips —
  instead of treating `list` as an ordinary namespace name, `ns list` SHALL be
  rejected as a reserved word (stderr error, no write, exit 2). All other
  requirements (write surgery, target resolution, validation, render
  read-only) are unchanged.

## Impact

- `cmd/omnictx/main.go`: `runNamespace` gains a reserved-word check before
  validation/write; `namespaceUsage`, the `--help` text for `ns`, and the
  function comment change.
- `cmd/omnictx` tests: the existing case asserting `ns list` writes `list`
  flips to asserting exit 2 + no write; new case for the `namespace` alias.
- `openspec/specs/kube-namespace-cli/spec.md`: delta spec in this change,
  synced on archive.
- No new dependencies, no changes to internal/kube (validation and write API
  stay as-is), no render-mode impact.
