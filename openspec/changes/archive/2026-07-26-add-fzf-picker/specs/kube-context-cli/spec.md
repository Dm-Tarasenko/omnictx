# kube-context-cli — delta for add-fzf-picker

## MODIFIED Requirements

### Requirement: Print the current context with `omnictx kube`
When invoked with no argument, the subcommand SHALL have two branches selected
by a pure activation decision. The interactive branch activates only when ALL
of: stdout is a terminal, `fzf` is found on `PATH`, and `OMNICTX_IGNORE_FZF`
is unset or empty. In every other situation the non-interactive branch runs
and SHALL print the current context name — resolved with the existing read
logic (first file in the `$KUBECONFIG` list that sets `current-context`, else
`~/.kube/config`) — to stdout followed by a newline, and exit 0. When no
context is set or no kubeconfig is readable, it SHALL print nothing on stdout
and exit 0.

In the interactive branch the command SHALL pipe the bare context names (one
per line, deduplicated by name, first definition wins, in file-then-definition
order — the same list `kube list` shows) to `fzf`, passing the current context
via `--header`, and on selection perform exactly the switch of `omnictx kube
<context>` (same validation, same write path, same exit codes, silent
success). A non-zero fzf exit (Esc, Ctrl-C, no match) SHALL write nothing and
exit 0. If executing fzf fails despite the activation decision, the command
SHALL fall back to the non-interactive print, exit 0. With no contexts found,
fzf SHALL NOT be invoked: the command prints nothing and exits 0. The picker
SHALL NOT read or write the `enabled`/`kube` display toggles: a selection
under a persisted mute still switches (switches change state, never
visibility).

#### Scenario: Current context is printed
- **WHEN** the kubeconfig sets `current-context: kind-1` and the user runs `omnictx kube` non-interactively (stdout not a terminal, or fzf absent, or `OMNICTX_IGNORE_FZF` set)
- **THEN** stdout is `kind-1` and the exit code is 0

#### Scenario: No context configured
- **WHEN** no kubeconfig file exists and the user runs `omnictx kube`
- **THEN** stdout is empty, fzf is never invoked, and the exit code is 0

#### Scenario: Interactive selection switches the context
- **WHEN** stdout is a terminal, fzf is on `PATH`, `OMNICTX_IGNORE_FZF` is unset, the kubeconfig defines `kind-1` and `kind-2` with `current-context: kind-1`, the user runs `omnictx kube` and selects `kind-2` in fzf
- **THEN** the kubeconfig's `current-context` becomes `kind-2` via the same write path as `omnictx kube kind-2`, stdout carries no extra output on success, and the exit code is 0

#### Scenario: Cancelling the picker writes nothing
- **WHEN** the interactive branch is active and the user exits fzf without selecting (Esc or Ctrl-C)
- **THEN** no file is modified and the exit code is 0

#### Scenario: OMNICTX_IGNORE_FZF disables the picker
- **WHEN** stdout is a terminal and fzf is on `PATH`, but `OMNICTX_IGNORE_FZF` is set to any non-empty value, and the user runs `omnictx kube`
- **THEN** the current context is printed exactly as in the non-interactive branch, fzf is never invoked, and the exit code is 0

#### Scenario: fzf absent keeps the print
- **WHEN** stdout is a terminal but `fzf` is not on `PATH`, and the user runs `omnictx kube`
- **THEN** the current context is printed and the exit code is 0

#### Scenario: fzf exec failure degrades to the print
- **WHEN** the activation decision selected the interactive branch but executing fzf fails
- **THEN** the current context is printed, no file is modified, and the exit code is 0

#### Scenario: Selection under a persisted mute still switches
- **WHEN** the omnictx config persists `enabled: false`, the interactive branch is active, and the user selects a context in fzf
- **THEN** the kubeconfig switch is performed and the `enabled`/`kube` config keys are untouched
