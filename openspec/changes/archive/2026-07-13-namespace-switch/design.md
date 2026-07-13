## Context

omnictx already reads the active context's namespace (`internal/kube/kube.go`,
`Read`) and owns exactly one foreign-file write path — `WriteContext`
(`internal/kube/write.go`), which rewrites the top-level `current-context:` line
with single-line surgery + atomic rename. This change adds a sibling write path
that mutates the `namespace:` of the active context's entry.

The key difference from `WriteContext`: `current-context:` is a single top-level
line, trivially located by a column-0 prefix scan. A context's `namespace:`
lives **nested** inside the matching `contexts[]` entry's `context:` mapping, may
be **absent** (requiring insertion, not replacement), and there may be several
`namespace:` lines in the file belonging to different contexts. So the surgery
must be scoped to exactly one block and be indentation-aware.

Constraints (from AGENTS.md core invariant): offline, no client-go, single
external dep `yaml.v3`; foreign-file writes only in explicit subcommands with
strict validation and loud non-zero exits; render mode never writes.

## Goals / Non-Goals

**Goals:**
- `omnictx namespace` prints the active context's namespace (get).
- `omnictx namespace <name>` sets it in the kubeconfig (set or insert), scoped to
  the single active-context block, preserving all other bytes.
- Reuse the existing resolution and atomic-write discipline; keep the diff small.

**Non-Goals:**
- Listing cluster namespaces (impossible offline — see spec "No offline
  namespace listing").
- Creating a new context, or switching context (that is `omnictx kube`).
- Full YAML re-serialization (would reflow comments/formatting — rejected).
- A separate display toggle for namespace (it follows the kube segment).

## Decisions

### D1: Line-oriented, indentation-scoped surgery (not YAML round-trip)
Mirror `WriteContext`: parse the file with `yaml.v3` only to *validate* and to
*locate* the active context, then edit the raw lines.

Algorithm for the set/insert:
1. Resolve active context via existing `Read` logic (first file with
   `current-context`). If empty → exit 1.
2. Pick the write target = first `$KUBECONFIG` file whose `contexts[]` has an
   entry named for the active context. If none → exit 1 ("active context not
   found").
3. Parse-before-write the target; unparsable → exit 1.
4. Walk raw lines to find the `contexts:` sequence, then the list item whose
   `name:` equals the active context. Within that item's `context:` mapping:
   - if a `namespace:` key line exists at the mapping's indent → replace its
     value in place;
   - else → insert `<indent>namespace: <name>` as the first child line of the
     `context:` mapping (indent taken from an existing sibling like `cluster:`).
5. `atomicWrite` (reused verbatim).

**Why over alternatives:** decoding to a struct and re-emitting with `yaml.v3`
loses comments, quoting style, and key order — unacceptable for a file we don't
own. `WriteContext` already set the line-surgery precedent; this extends it.

### D2: Block boundaries by indentation, YAML-block-style assumption
A kubeconfig context item is conventionally:
```
contexts:
- context:
    cluster: kind-1
    user: kind-1-user
    namespace: payments
  name: kind-1
```
The locator identifies the item by its `name:` and the sibling `context:`
mapping. The `context:` mapping's children are the run of lines more indented
than the `context:` key, until a line at equal-or-lesser indent (e.g. `name:` or
the next `- ` item). We scan that run for `namespace:` (replace) or insert at its
top (aligned to the first child's indent). Flow-style (`{context: {...}}`)
kubeconfigs are out of scope — kubectl never writes them; if the locator can't
confidently find the block, it returns an error (exit 1) rather than guessing.

**Why:** robust for every kubeconfig kubectl produces, and fails safe (no write)
on anything exotic.

### D3: Validate the name as a DNS-1123 label before writing
Regex `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`, length ≤ 63. Invalid → exit 2, no
write. This is a pure-stdlib check (no k8s libs) that prevents corrupting the
kubeconfig with a value that could never be a real namespace, matching the
"validate strictly" rule for write subcommands.

**Why:** offline we cannot confirm the namespace exists on the cluster; a syntax
gate is the strongest safe check and stops obvious typos/injection.

### D4: New package API surface
Add to `internal/kube`:
- `WriteNamespace(lookup LookupEnv, home, namespace string) error` — the set path
  (steps 1–5). Returns typed/sentinel errors so the caller maps them to exit
  codes (no-active-context and not-found → 1, unparsable → 1).
- Namespace get reuses `Read(...).Namespace`; no new reader needed.

`cmd/omnictx/main.go`: add `case "namespace"` to the dispatch and a
`runNamespace(args, stdout, stderr, home)` handler that: no arg → print
`Read().Namespace`; one arg → validate name (exit 2 on failure) → `WriteNamespace`
(map errors to exit 1) → exit 0; ≥2 args → usage error exit 2. Add usage text to
the `--help` grouped output and a `namespaceUsage` constant like `kubeUsage`.

### D5: Exit-code mapping (consistent with `kube`)
- get with nothing to show → 0 (like `kube` print).
- invalid name / too many args → 2 (usage class).
- no active context, active context not found, unparsable target → 1 (source/
  state problem, cannot safely proceed). This matches `WriteContext`'s "broken
  source → exit 1" while treating the missing-precondition cases as exit 1 too,
  since — unlike an unknown *context* argument (exit 2 in `kube`) — the user
  passed a valid namespace and the obstacle is the kubeconfig state.

## Risks / Trade-offs

- **Nested-block line surgery is more error-prone than the top-level
  current-context edit** → Mitigate with an indentation-aware locator gated by a
  `yaml.v3` parse (only write when the parse confirms the context exists), and a
  fixture matrix: replace-existing, insert-when-absent, context-in-second-file,
  multiple-contexts-only-active-changes, comment-preservation, tab/2-space/4-space
  indents. Any locator uncertainty → error, no write.
- **Unusual formatting (flow style, `namespace` value with inline comment,
  quoted keys)** → Locator refuses (exit 1) rather than mangle; documented as
  out of scope. `namespace: payments # note` value replacement must keep the
  trailing comment — covered by a fixture.
- **Write target differs from the current-context file** (context defined in a
  different `$KUBECONFIG` file) → target selection explicitly follows the
  read-side namespace resolution (first file that defines the context), tested.
- **Namespace name rules drift** (k8s could allow more) → the DNS-1123 label
  check is deliberately conservative; loosening it later is backward-compatible.

## Migration Plan

Additive, no migration. New subcommand + new function; no changes to existing
render, config, or `kube` behavior. Rollback = revert the change; no persisted
state format changes. Ship with AGENTS.md subcommand-list and README updates.

## Open Questions

- Alias `omnictx ns` for the subcommand? The user asked specifically for
  `namespace`; `ns` is left out for now (can be added later without breaking
  anything). Not blocking.
