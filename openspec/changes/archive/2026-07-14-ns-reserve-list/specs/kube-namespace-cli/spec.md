## MODIFIED Requirements

### Requirement: No offline namespace listing
Because omnictx never contacts the cluster, the subcommand SHALL NOT provide a
form that enumerates cluster namespaces, and `list` SHALL be a reserved word
rather than an acceptable namespace name: `omnictx ns list` (and the
`namespace` alias) SHALL print an error to stderr explaining that omnictx is
offline and cannot list cluster namespaces (pointing the user at
`kubectl get namespaces`), SHALL NOT modify any file, and SHALL exit with
code 2. Only `list` is reserved; `on` and `off` remain ordinary valid
namespace names because `ns` has no toggle form.

#### Scenario: `list` is reserved, not a namespace name
- **WHEN** the active context is resolvable and the user runs `omnictx ns list`
- **THEN** stderr explains that omnictx cannot list cluster namespaces offline (mentioning `kubectl get namespaces`), no file is modified, and the exit code is 2

#### Scenario: `namespace list` alias is rejected identically
- **WHEN** the user runs `omnictx namespace list`
- **THEN** the result is identical to `omnictx ns list` (stderr error, no write, exit code 2)

#### Scenario: `list` is rejected before any kubeconfig state check
- **WHEN** no kubeconfig file is readable and the user runs `omnictx ns list`
- **THEN** the reserved-word error is printed to stderr and the exit code is 2 (not the exit-1 broken-source path)

#### Scenario: `on` and `off` stay ordinary namespace names
- **WHEN** the active context is resolvable and the user runs `omnictx ns off`
- **THEN** the active context's namespace is set to `off` and the exit code is 0
