## Context

`runNamespace` in `cmd/omnictx/main.go` handles `omnictx ns [<name>]`. Any
single argument that passes DNS-1123 validation is written into the active
context's block in the kubeconfig via `kube.WriteNamespace`. `list` is a valid
DNS-1123 label, so `omnictx ns list` — a natural guess given `omnictx kube
list` and `omnictx cloud list` exist — silently switches the namespace to the
literal `list`. The current spec (`kube-namespace-cli`, requirement "No
offline namespace listing") documents this as intended; this change flips that
requirement.

Constraints (AGENTS.md): fully offline — no kubectl, no network, no client-go;
writes to foreign files only in explicit subcommands, validating strictly and
failing loudly with non-zero exit codes.

## Goals / Non-Goals

**Goals:**
- `omnictx ns list` (and `omnictx namespace list`) never writes; it fails with
  a clear stderr message and exit 2.
- The message tells the user why (offline tool) and what to use instead
  (`kubectl get namespaces`).
- Help text and code comments stop claiming `ns list` sets the namespace to
  `list`.

**Non-Goals:**
- No real namespace listing (online via kubectl, or offline from kubeconfig
  contexts) — explicitly rejected to preserve the offline invariant; can be a
  future change.
- No reservation of `on`/`off` — `ns` has no toggle form, so they stay valid
  namespace names.
- No changes to `internal/kube` (validation, read, write surgery stay as-is).

## Decisions

1. **Reserved-word check lives in `runNamespace`, before validation and the
   write.** Mirrors how `runKube` treats its reserved words at the dispatch
   layer; `internal/kube.ValidNamespace` keeps answering the pure "is this a
   DNS-1123 label" question. Alternative — rejecting `list` inside
   `ValidNamespace` — was rejected: it would conflate CLI vocabulary with the
   Kubernetes naming rule and mislead future callers.

2. **Exit code 2, checked first.** `list` is a usage-level rejection (like an
   invalid name), not a source problem, so exit 2 — and the check runs before
   any kubeconfig access, so it wins even when the kubeconfig is broken or
   absent. Deterministic: same input, same outcome, regardless of local state.

3. **Error message names the alternative.** Something like:
   `omnictx: "list" is reserved: omnictx is offline and cannot list cluster
   namespaces (try: kubectl get namespaces)` followed by the usage line. This
   converts the footgun into a teaching moment instead of a bare usage error.

4. **Only `list` is reserved.** `kube` reserves `list|on|off` because it has a
   toggle; `ns` does not. Reserving unused words would forbid valid (if odd)
   namespace names for no benefit.

## Risks / Trade-offs

- [Breaking change for anyone who genuinely has a namespace named `list`] →
  Accepted: vanishingly rare, and the error message makes the rejection
  obvious. `kubectl config set-context --current --namespace=list` remains an
  escape hatch; noted in the spec delta.
- [Vocabulary drift between subcommands: `kube list` works, `ns list` errors]
  → Mitigated by the error message explaining *why* (offline), and by `--help`
  documenting the reserved word.

## Migration Plan

Single-PR change: code + tests + help text land together; the delta spec is
synced to `openspec/specs/` on archive. No data or config migration — the
binary is stateless. Rollback = revert the commit.
