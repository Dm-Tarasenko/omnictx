## ADDED Requirements

### Requirement: List cluster namespaces via `omnictx ns list`
The subcommand SHALL provide a `list` form: `omnictx ns list` (and the
`namespace` alias) SHALL run `kubectl get namespaces -o name` with a bounded
request timeout (`--request-timeout=10s`) and print the resulting namespace
names to stdout as a table with columns `CURRENT` and `NAME`, marking with `*`
the row that equals the active context's namespace as resolved by the existing
offline read logic (when the active context sets no namespace, the `default`
row SHALL be marked). Output lines from kubectl that do not match the
`namespace/<name>` form SHALL be skipped. On success the exit code SHALL be 0
and no file SHALL be modified. `list` SHALL remain unavailable as a switch
target: `omnictx ns list` never writes a namespace named `list`. Only `list`
is a subcommand word; `on` and `off` remain ordinary valid namespace names
because `ns` has no toggle form. This is the only online code path in the
binary: render mode and every other subcommand SHALL NOT invoke kubectl or
perform network access.

#### Scenario: Namespaces are listed with the current one marked
- **WHEN** the active context's namespace is `payments`, and `kubectl get namespaces -o name` prints `namespace/default`, `namespace/payments`, and `namespace/staging`
- **THEN** stdout is a CURRENT/NAME table listing `default`, `payments`, and `staging` with `*` on the `payments` row, the exit code is 0, and no file is modified

#### Scenario: No namespace set marks the default row
- **WHEN** the active context has no `namespace` key and `kubectl` reports namespaces including `default`
- **THEN** the `default` row is marked with `*` and the exit code is 0

#### Scenario: `namespace list` alias behaves identically
- **WHEN** the user runs `omnictx namespace list`
- **THEN** the result is identical to `omnictx ns list`

#### Scenario: kubectl is not installed
- **WHEN** `kubectl` cannot be found on `PATH` and the user runs `omnictx ns list`
- **THEN** stderr explains that kubectl is required for `ns list`, no file is modified, and the exit code is 1

#### Scenario: kubectl fails
- **WHEN** `kubectl get namespaces` exits non-zero (for example the cluster is unreachable or auth fails)
- **THEN** kubectl's stderr is passed through with an omnictx error line, no file is modified, and the exit code is 1

#### Scenario: `list` never becomes a namespace name
- **WHEN** the user runs `omnictx ns list`
- **THEN** no kubeconfig file is modified, regardless of whether the kubectl call succeeds

## REMOVED Requirements

### Requirement: No offline namespace listing
**Reason**: The offline-only principle is relaxed for explicit interactive
subcommands: `ns list` now shells out to kubectl and really lists cluster
namespaces instead of rejecting `list` as a reserved word.
**Migration**: `omnictx ns list` now prints the cluster's namespaces (exit 0
on success) instead of erroring with exit 2. Scripts that relied on the
reserved-word rejection must not pass `list`; `on`/`off` are still ordinary
namespace names.
