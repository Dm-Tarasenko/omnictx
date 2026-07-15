## Context

`runNamespace` in `cmd/omnictx/main.go` currently treats `list` as a reserved
word: `omnictx ns list` prints "omnictx is offline and cannot list cluster
namespaces" and exits 2 (change `2026-07-14-ns-reserve-list`). The project-wide
principle has been "no kubectl, no network — including the switches"
(AGENTS.md). That principle is now deliberately relaxed by the project owner:
explicit interactive subcommands may shell out to `kubectl`; render mode — the
code that runs on every shell prompt — remains strictly offline so the prompt
can never hang or break.

The namespace list lives only in the cluster; there is no local file to read
it from, so an online call is the only honest implementation.

## Goals / Non-Goals

**Goals:**
- `omnictx ns list` / `omnictx namespace list` prints the cluster's namespaces
  (via `kubectl get namespaces`) as a `kube list`-style table with the current
  namespace marked, exit 0.
- kubectl missing from PATH or failing → clear stderr message, exit 1, no
  writes anywhere.
- The kubectl invocation is time-bounded (`--request-timeout`) so an
  unreachable cluster fails in seconds, not forever.
- AGENTS.md and help text reflect the relaxed principle.

**Non-Goals:**
- No client-go, no direct Kubernetes API calls, no network libraries in
  go.mod — the single-external-dependency rule (yaml.v3) stands.
- No changes to render mode: it stays offline, read-only, exit-0-on-error.
- No other subcommand goes online in this change (`kube list` keeps reading
  kubeconfig locally; cloud listings stay file-based).
- No namespace-switch-by-picker UX; `ns <name>` behavior is unchanged.

## Decisions

1. **Shell out to `kubectl`, not client-go.** `exec.Command("kubectl", ...)`
   keeps go.mod untouched and reuses the user's kubectl auth stack (exec
   plugins, SSO) for free. client-go would drag in a huge dependency tree and
   its own auth handling — rejected.

2. **Invocation: `kubectl get namespaces -o name --request-timeout=10s`.**
   `-o name` gives stable machine-readable `namespace/<name>` lines that we
   strip; `--request-timeout=10s` bounds the hang on a dead VPN/cluster.
   Parsing the human table output was rejected as fragile.

3. **The `list` branch lives in `runNamespace` (cmd layer), before DNS-1123
   validation** — exactly where the reserved-word check sits today, mirroring
   `runKube`'s dispatch of `list`. `internal/kube` stays a pure local-file
   package; putting exec logic there would poison the package every render
   imports. Ordering also preserves the existing property that `list` is
   handled before any kubeconfig-state check.

4. **Output is a CURRENT/NAME table via the existing `printTable`,** matching
   `kube list`'s look. The marker `*` goes on the row equal to the active
   context's namespace from the offline `kube.Read`; when the context has no
   namespace set, `default` is marked (Kubernetes' effective default).
   A bare name dump (kubens-style) was rejected for consistency with
   `kube list`.

5. **Failure = exit 1, never 2.** kubectl missing or exiting non-zero is an
   environment/source problem, not a usage error — same contract as broken
   kubeconfig paths. kubectl's own stderr is passed through so the user sees
   the real cause (auth, connectivity), prefixed with an omnictx line.

6. **Tests stub kubectl with a fake executable** written to a temp dir
   prepended to `PATH` (a tiny `#!/bin/sh` script echoing fixture output or
   exiting non-zero). This keeps tests offline and hermetic; the
   kubectl-missing case sets `PATH` to an empty dir.

## Risks / Trade-offs

- [First online code path in the binary] → strictly confined to the explicit
  `ns list` subcommand; render mode shares no code with it. AGENTS.md spells
  out the boundary so future changes don't creep online paths into render.
- [kubectl output format drift] → `-o name` is a stable kubectl contract;
  lines that don't match `namespace/<x>` are skipped rather than crashing.
- [Cluster slow/unreachable] → `--request-timeout=10s` bounds the wait;
  failure is loud (exit 1) and writes nothing.
- [Users on machines without kubectl] → the error message names kubectl
  explicitly so the remedy is obvious; every other omnictx feature keeps
  working without it.

## Migration Plan

Single PR. The previous reserved-word behavior (exit 2 + "cannot list")
becomes a working command; no config or file-format migration. Rollback =
revert the commit.

## Open Questions

None.
