# kube-namespace-cli

## Purpose

The `omnictx ns` subcommand (alias `namespace`): switching the active context's
namespace (`omnictx ns <name>`), printing the current namespace (no argument),
and listing the cluster's namespaces (`omnictx ns list`, via kubectl).
Like `kube <context>`, the write touches a file omnictx does not own — the
kubeconfig — and only on an explicit user command, with DNS-1123 validation,
single-line surgery (replace in place, or insert into the active context's
`context:` mapping), and an atomic write that preserves permissions and every
other byte. `ns list` is the only online code path in the binary: it shells out
to `kubectl get namespaces` with a bounded request timeout and never writes.
Render mode stays strictly read-only and offline, and never breaks the prompt.
## Requirements
### Requirement: Switch the active context's namespace via `omnictx ns <name>`
The CLI SHALL provide a subcommand `omnictx ns <name>` (with `namespace` as an
accepted alias for `ns`) that sets the
`namespace:` of the active context (the `contexts[]` entry whose `name` equals
the resolved `current-context`) in the kubeconfig and exits with code 0. The
edit SHALL preserve every other byte of the file, including comments, ordering,
and formatting: when the matching context block already has a `namespace:` key,
its value SHALL be replaced in place (single-line surgery); when absent, a
`namespace: <name>` line SHALL be inserted into that context's `context:`
mapping using the same indentation as its sibling keys. The write SHALL be
atomic (temp file in the same directory, then rename over the original) and
SHALL preserve the original file permission bits.

#### Scenario: Change an existing namespace value
- **WHEN** the kubeconfig sets `current-context: kind-1`, whose context block has `namespace: payments`, and the user runs `omnictx ns staging`
- **THEN** that block reads `namespace: staging`, every other line (including comments and other contexts) is byte-identical, and the exit code is 0

#### Scenario: Insert a namespace when the context has none
- **WHEN** the active context's block has `cluster` and `user` keys but no `namespace` key, and the user runs `omnictx ns payments`
- **THEN** a `namespace: payments` line is inserted into that context's `context:` mapping (aligned with the sibling keys), the rest of the file is unchanged, and the exit code is 0

#### Scenario: Switch is visible to the next render
- **WHEN** `omnictx ns staging` has succeeded and a render invocation follows
- **THEN** the kube segment shows the active context with namespace `staging`

#### Scenario: Only the active context is touched
- **WHEN** several contexts define a `namespace:` and the user runs `omnictx ns staging`
- **THEN** only the active context's `namespace:` changes; the namespace lines of the other contexts are byte-identical

#### Scenario: `namespace` alias behaves identically to `ns`
- **WHEN** the user runs `omnictx namespace staging`
- **THEN** the result is identical to `omnictx ns staging` (the active context's namespace is set to `staging`, exit code 0)

### Requirement: Choose the write target from the KUBECONFIG list
The subcommand SHALL resolve the active context with the existing read logic
(the first file in the `$KUBECONFIG` list that sets a non-empty
`current-context`, else `~/.kube/config`). The namespace write target SHALL be
the first file in the `$KUBECONFIG` list whose `contexts[]` contains an entry
named for the active context (mirroring the read-side namespace resolution,
first match wins). Other files SHALL never be modified. Without `KUBECONFIG`,
both resolution and the target file SHALL be `~/.kube/config`.

#### Scenario: Context defined in a different file than current-context
- **WHEN** `KUBECONFIG=a.yaml:b.yaml`, `a.yaml` sets `current-context: kind-2`, only `b.yaml` defines the `kind-2` context entry, and the user runs `omnictx ns staging`
- **THEN** `b.yaml`'s `kind-2` block is updated and `a.yaml` is byte-identical to before

#### Scenario: Default kubeconfig used when KUBECONFIG unset
- **WHEN** `KUBECONFIG` is unset, `~/.kube/config` sets `current-context: kind-1`, and the user runs `omnictx ns staging`
- **THEN** `~/.kube/config` is the file updated and the exit code is 0

### Requirement: Validate the namespace name and refuse unsafe writes
The subcommand SHALL accept only a namespace name that is a valid DNS-1123 label
(lowercase alphanumeric characters or `-`, starting and ending with an
alphanumeric character, at most 63 characters). For an invalid name it SHALL
print an error to stderr, SHALL NOT modify any file, and SHALL exit with code 2.
If there is no resolvable active context (no `current-context` set, or no file
in the list defines a matching `contexts[]` entry), it SHALL print an error to
stderr, SHALL NOT modify any file, and SHALL exit with code 1. If the
write-target file cannot be read or parsed as kubeconfig YAML, the subcommand
SHALL refuse to write, print an error, and exit with code 1.

#### Scenario: Invalid namespace name
- **WHEN** the user runs `omnictx ns 'Bad_NS'`
- **THEN** stderr explains the name is not a valid namespace, no file is modified, and the exit code is 2

#### Scenario: No current-context set
- **WHEN** the kubeconfig defines contexts but no `current-context`, and the user runs `omnictx ns staging`
- **THEN** stderr explains there is no active context, no file is modified, and the exit code is 1

#### Scenario: Active context not defined in any file
- **WHEN** `current-context: kind-9` is set but no file defines a `kind-9` context entry, and the user runs `omnictx ns staging`
- **THEN** stderr explains the active context was not found, no file is modified, and the exit code is 1

#### Scenario: Broken kubeconfig refuses the write
- **WHEN** the write-target file contains invalid YAML and the user runs `omnictx ns staging`
- **THEN** no file is modified, an error is printed to stderr, and the exit code is 1

### Requirement: Print the current namespace with `omnictx ns`
When invoked with no argument, the subcommand SHALL print the active context's
namespace — resolved with the existing read logic — to stdout followed by a
newline, and exit 0. When there is no active context, no namespace is set, or no
kubeconfig is readable, it SHALL print nothing on stdout and exit 0.

#### Scenario: Current namespace is printed
- **WHEN** the active context has `namespace: payments` and the user runs `omnictx ns`
- **THEN** stdout is `payments` and the exit code is 0

#### Scenario: No namespace set
- **WHEN** the active context has no `namespace` key and the user runs `omnictx ns`
- **THEN** stdout is empty and the exit code is 0

#### Scenario: No context configured
- **WHEN** no kubeconfig file is readable and the user runs `omnictx ns`
- **THEN** stdout is empty and the exit code is 0

### Requirement: Render mode stays read-only and never breaks the prompt
Kubeconfig writes SHALL happen only in the `ns <name>` subcommand on an explicit
user command. Render mode SHALL never write to any kubeconfig file, and
the never-break-the-prompt invariant (any render error → skip segment, exit 0)
SHALL be unaffected by this change.

#### Scenario: Render never writes
- **WHEN** a render invocation runs (with or without `--shell`)
- **THEN** no kubeconfig file is opened for writing

#### Scenario: Too many arguments is a usage error
- **WHEN** the user runs `omnictx ns a b`
- **THEN** a usage message is printed to stderr and the exit code is 2

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

