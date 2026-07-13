## Why

omnictx already displays the active kube-context's namespace and can switch the
current-context, but it cannot change the namespace itself — users still fall
back to `kubectl config set-context --current --namespace=...`. Adding a
first-class `omnictx namespace` switch closes that gap and keeps the whole
context/namespace workflow inside omnictx, offline and without kubectl.

## What Changes

- Add a top-level subcommand `omnictx namespace [<name>]`:
  - **no argument** → print the namespace of the active context (empty prints
    nothing, exit 0);
  - **`<name>`** → set `namespace:` on the active context's entry in the
    kubeconfig (the context matching `current-context`), then exit 0.
- The switch writes only on the explicit command, validating strictly (the name
  must be a valid DNS-1123 label) and failing loudly on non-zero exit:
  - no `current-context` set / active context not found in any kubeconfig → exit 1;
  - invalid namespace name or reserved argument → exit 2 (usage error);
  - unparsable target kubeconfig → exit 1 (refuse to modify).
- No cluster namespace listing: omnictx is offline and cannot enumerate cluster
  namespaces, so there is deliberately no `namespace list`.
- Extend the foreign-file write path in `internal/kube` with a namespace writer
  that performs surgery on the matching context block (set existing
  `namespace:` line, or insert one under `context:` when absent), preserving all
  other bytes, with the same parse-before-write + atomic-rename discipline as
  the existing context writer.
- Update `--help` usage grouping and README to document `namespace`.

## Capabilities

### New Capabilities
- `kube-namespace-cli`: the `omnictx namespace` subcommand — print the active
  context's namespace, or switch it by rewriting the matching context entry in
  the kubeconfig (offline, explicit-write, strict validation).

### Modified Capabilities
<!-- None: kube-context-cli's requirements are unchanged; this is a new sibling subcommand. -->

## Impact

- `cmd/omnictx/main.go`: new `namespace` case in the subcommand dispatch, a
  `runNamespace` handler, `--help` usage text.
- `internal/kube`: new `WriteNamespace` (foreign-file write) plus a helper to
  resolve the active context's namespace/owning file; the second write path in
  the package.
- `testdata`: kubeconfig fixtures for set/insert/no-current-context cases.
- Docs: AGENTS.md subcommand list and README.
- No new dependencies; no network access; render mode unchanged (still never
  writes).
