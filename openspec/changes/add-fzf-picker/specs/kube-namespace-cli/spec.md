# kube-namespace-cli — delta for add-fzf-picker

## MODIFIED Requirements

### Requirement: Print the current namespace with `omnictx ns`
When invoked with no argument, the subcommand SHALL have two branches selected
by the same pure activation decision as bare `kube` (interactive only when
stdout is a terminal AND `fzf` is on `PATH` AND `OMNICTX_IGNORE_FZF` is unset
or empty). In every other situation the non-interactive branch runs and SHALL
print the active context's namespace — resolved with the existing offline read
logic — to stdout followed by a newline, and exit 0; it SHALL NOT invoke
kubectl. When there is no active context, no namespace is set, or no
kubeconfig is readable, it SHALL print nothing on stdout and exit 0.

In the interactive branch the command SHALL obtain the namespace list exactly
as `ns list` does (`kubectl get namespaces -o name --request-timeout=10s`,
skipping lines that do not match `namespace/<name>`), pipe the bare names to
`fzf` with the active namespace (or `default` when unset) in `--header`, and
on selection perform exactly the switch of `omnictx ns <name>` (same
validation and write path — the active context's namespace is rewritten in
the kubeconfig; same exit codes; silent success). A non-zero fzf exit SHALL
write nothing and exit 0; if executing fzf fails despite the activation
decision, the command SHALL fall back to the non-interactive print, exit 0.
Unlike `ns list`, bare `ns` SHALL keep its always-exit-0 contract when the
cluster is unreachable: kubectl missing from `PATH` or exiting non-zero SHALL
warn on stderr, fall back to printing the current namespace, and exit 0. The
picker SHALL NOT read or write the `enabled`/`kube` display toggles.

#### Scenario: Current namespace is printed
- **WHEN** the active context has `namespace: payments` and the user runs `omnictx ns` non-interactively (stdout not a terminal, or fzf absent, or `OMNICTX_IGNORE_FZF` set)
- **THEN** stdout is `payments`, kubectl is never invoked, and the exit code is 0

#### Scenario: No namespace set
- **WHEN** the active context has no `namespace` key and the user runs `omnictx ns` non-interactively
- **THEN** stdout is empty and the exit code is 0

#### Scenario: No context configured
- **WHEN** no kubeconfig file is readable and the user runs `omnictx ns`
- **THEN** stdout is empty and the exit code is 0

#### Scenario: Interactive selection switches the namespace
- **WHEN** stdout is a terminal, fzf is on `PATH`, `OMNICTX_IGNORE_FZF` is unset, kubectl lists `default`, `payments`, and `staging`, and the user runs `omnictx ns` and selects `staging` in fzf
- **THEN** the active context's namespace in the kubeconfig becomes `staging` via the same write path as `omnictx ns staging`, stdout carries no extra output on success, and the exit code is 0

#### Scenario: Cancelling the picker writes nothing
- **WHEN** the interactive branch is active and the user exits fzf without selecting
- **THEN** no file is modified and the exit code is 0

#### Scenario: kubectl unavailable degrades to the print
- **WHEN** the interactive branch is active but kubectl is missing from `PATH` or `kubectl get namespaces` exits non-zero
- **THEN** stderr carries a warning, the current namespace is printed to stdout as in the non-interactive branch, no file is modified, and the exit code is 0

#### Scenario: OMNICTX_IGNORE_FZF disables the picker
- **WHEN** stdout is a terminal and fzf is on `PATH`, but `OMNICTX_IGNORE_FZF` is set to any non-empty value, and the user runs `omnictx ns`
- **THEN** the current namespace is printed, neither kubectl nor fzf is invoked, and the exit code is 0

#### Scenario: fzf exec failure degrades to the print
- **WHEN** the activation decision selected the interactive branch but executing fzf fails
- **THEN** the current namespace is printed, no file is modified, and the exit code is 0

### Requirement: List cluster namespaces via `omnictx ns list`
The subcommand SHALL provide a `list` form: `omnictx ns list` (and the
`namespace` alias) SHALL run `kubectl get namespaces -o name` with a bounded
request timeout (`--request-timeout=10s`) and print the resulting namespace
names to stdout as a table with columns `CURRENT` and `NAME`, marking with `*`
the row that equals the active context's namespace as resolved by the existing
offline read logic (when the active context sets no namespace, the `default`
row SHALL be marked). Output lines from kubectl that do not match the
`namespace/<name>` form SHALL be skipped. On success the exit code SHALL be 0
and no file SHALL be modified. `ns list` SHALL never invoke fzf and never
become interactive: it is the stable read-only inspection surface in every
environment. `list` SHALL remain unavailable as a switch target: `omnictx ns
list` never writes a namespace named `list`. Only `list` is a subcommand word;
`on` and `off` remain ordinary valid namespace names because `ns` has no
toggle form. kubectl invocation is confined to exactly two call sites —
`ns list` and the interactive branch of bare `ns` — and render mode and every
other subcommand SHALL NOT invoke kubectl or perform network access.

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

#### Scenario: `list` stays a table even when the picker conditions hold
- **WHEN** stdout is a terminal, fzf is on `PATH`, and `OMNICTX_IGNORE_FZF` is unset, and the user runs `omnictx ns list`
- **THEN** the CURRENT/NAME table is printed, fzf is never invoked, and no file is modified

#### Scenario: `list` never becomes a namespace name
- **WHEN** the user runs `omnictx ns list`
- **THEN** no kubeconfig file is modified, regardless of whether the kubectl call succeeds
