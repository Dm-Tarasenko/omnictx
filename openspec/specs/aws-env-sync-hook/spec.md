# aws-env-sync-hook

## Purpose

The per-prompt env-sync protocol between the binary and the `init bash|zsh` snippets: a single hook invocation emits three lines (an `AWS_PROFILE` directive, an `AWS_REGION` directive, and the prompt segment), and the shell snippet applies the directives with ownership markers so hook-managed shells follow global switches while manual exports and aws-vault sessions stay untouched. The invocation follows the render-mode discipline: never writes a file, degrades on any error, always exits 0.

## Requirements

### Requirement: The per-prompt invocation emits three lines
The per-prompt hook invocation of the binary SHALL emit exactly three
newline-terminated lines: line 1 = the `AWS_PROFILE` directive, line 2 = the
`AWS_REGION` directive, line 3 = the prompt segment (exactly what render mode
produces today, possibly empty). Directive encoding: empty string = leave the
variable alone; literal `-` = unset it; any other value = export it. The
invocation SHALL follow the render-mode discipline: it never writes any file,
any internal error degrades to empty directives and a best-effort (possibly
empty) segment, and the exit code is always 0. When the enabled toggle
resolves to false (config `enabled: false` or `OMNICTX_ENABLED=false`), line 3
SHALL be empty while lines 1 and 2 keep carrying directives: the mute controls
display, not state — an AWS switch takes effect under the mute exactly like
gcp/azure/kube switches, whose tools read the switched files directly.

#### Scenario: Managed state produces export directives
- **WHEN** the config file contains `aws_profile: digital-dev` and `aws_region: eu-central-1`, and the invocation runs with neither `AWS_PROFILE`, `AWS_VAULT`, nor markers in the env
- **THEN** line 1 is `digital-dev`, line 2 is `eu-central-1`, and line 3 renders the aws segment showing `digital-dev/eu-central-1`

#### Scenario: No managed state produces empty directives
- **WHEN** the config file sets neither `aws_profile` nor `aws_region`
- **THEN** lines 1 and 2 are empty and line 3 is the segment unchanged from today's behavior

#### Scenario: Broken config never breaks the prompt
- **WHEN** the config file is unparsable
- **THEN** lines 1 and 2 are empty, the exit code is 0, and the prompt line is not corrupted

#### Scenario: The mute empties only the segment line
- **WHEN** the config file contains `enabled: false` and `aws_profile: prod`, and the invocation runs with no AWS env vars set
- **THEN** line 1 is `prod`, line 3 is empty, and the exit code is 0

### Requirement: Pin logic respects sessions the hook does not own
The directive for each variable SHALL be computed in Go from the env: if
`AWS_VAULT` is set, both directives are empty (aws-vault owns the session). For
a variable with persisted value V and marker M (`__OMNICTX_AWS_PROFILE` /
`__OMNICTX_AWS_REGION`): current env value empty or equal to M → directive V;
current env value differing from M (a manual `export`, direnv, etc.) → empty
directive; persisted value absent and current env value equal to M → directive
`-` (the hook set it, the global state was cleared); persisted value absent
otherwise → empty directive.

#### Scenario: Manual export pins the session
- **WHEN** the config file contains `aws_profile: prod` but the env has `AWS_PROFILE=stage` and `__OMNICTX_AWS_PROFILE=prod`
- **THEN** line 1 is empty and line 3 renders the segment with `stage`

#### Scenario: Hook-owned value follows a new global switch
- **WHEN** the config file contains `aws_profile: prod` and the env has `AWS_PROFILE=dev` with `__OMNICTX_AWS_PROFILE=dev`
- **THEN** line 1 is `prod` and line 3 renders the segment with `prod`

#### Scenario: Cleared global state unsets only hook-owned values
- **WHEN** the config file sets no `aws_profile` and the env has `AWS_PROFILE=prod` with `__OMNICTX_AWS_PROFILE=prod`
- **THEN** line 1 is `-`

#### Scenario: aws-vault session is untouched
- **WHEN** `AWS_VAULT=prod-admin` is set and the config file contains `aws_profile: dev`
- **THEN** lines 1 and 2 are empty

### Requirement: The segment reflects the directives it ships with
When line 1 or line 2 carries an export directive, line 3 SHALL be rendered as
if those exports were already applied, so the prompt is correct on the same
prompt cycle that applies the switch.

#### Scenario: No one-cycle lag after a switch
- **WHEN** the persisted `aws_profile` changed from `dev` to `prod` and the invocation runs with `AWS_PROFILE=dev`, `__OMNICTX_AWS_PROFILE=dev`
- **THEN** line 1 is `prod` and line 3 shows `prod`, not `dev`

### Requirement: init templates apply directives and export markers
The `init bash` / `init zsh` snippets SHALL consume the three lines from the
single per-prompt invocation and apply each directive verbatim: export the
value and set the matching marker (`__OMNICTX_AWS_PROFILE` /
`__OMNICTX_AWS_REGION`) for an export directive, `unset` both the variable and
its marker for `-`, and do nothing for an empty directive. Markers SHALL be
exported (visible to child processes) so subshells inherit the value+marker
pair. Values SHALL be applied via quoted shell parameters — snippet code SHALL
NOT eval binary output. The snippet SHALL apply directives before computing the
prompt string, and its output SHALL remain idempotent across repeated `eval`.

#### Scenario: Export directive sets variable and marker
- **WHEN** the hook invocation returns `prod` on line 1 in a bash session
- **THEN** after the prompt function runs, `AWS_PROFILE=prod` and `__OMNICTX_AWS_PROFILE=prod` are both exported

#### Scenario: Unset directive removes variable and marker
- **WHEN** the hook invocation returns `-` on line 1 in a session where `AWS_PROFILE` and `__OMNICTX_AWS_PROFILE` are set
- **THEN** after the prompt function runs, neither variable is set

#### Scenario: Init output stays idempotent
- **WHEN** the user evals `omnictx init zsh` twice in one shell
- **THEN** the prompt hook is installed exactly once
