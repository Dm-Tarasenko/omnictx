# aws-profile-switch-cli — delta for promote-provider-commands

## MODIFIED Requirements

### Requirement: Switch the active AWS profile via `omnictx aws <profile>`
The CLI SHALL provide `omnictx aws <profile>` — with `omnictx cloud aws
<profile>` as an accepted alias dispatching to the same code — that validates
`<profile>` against the locally configured profiles (the names returned by the
offline profile listing over `~/.aws/config` + `~/.aws/credentials`, honoring
`AWS_CONFIG_FILE`) and, on success, writes `aws_profile: <profile>` to the
omnictx config file (path resolved as `OMNICTX_CONFIG` >
`~/.config/omnictx/config.yaml`) via the same single-key write machinery as
other persisted keys (only the `aws_profile:` line changes; other keys and
comments are preserved; file and parent directory are created if absent), then
persists `cloud: aws` (the same post-switch pin as gcp/azure switches) and
exits 0. The switch changes state, never visibility: it SHALL NOT touch the
`enabled` or `kube` display toggles — under a persisted mute
(`enabled: false`) the state still flips, and the prompt reflects it once
`omnictx on` / `cloud on` lifts the mute. The command SHALL NOT write to any
file under `~/.aws`. On success the command SHALL be silent (no stdout/stderr
output), matching the gcp/azure switches; hook-running shells apply the switch
on their next prompt.

#### Scenario: Successful switch persists profile and pins the cloud
- **WHEN** `~/.aws/config` defines `[profile digital-dev]` and the user runs `omnictx aws digital-dev`
- **THEN** the config file contains `aws_profile: digital-dev` and `cloud: aws`, other keys and comments are unchanged, nothing under `~/.aws` is modified, and the exit code is 0

#### Scenario: The cloud spelling behaves identically
- **WHEN** the user runs `omnictx cloud aws digital-dev` for the same profile
- **THEN** the result is identical to `omnictx aws digital-dev`

#### Scenario: Profile present only in credentials is accepted
- **WHEN** `~/.aws/credentials` defines `[ci-bot]` and `~/.aws/config` does not mention it, and the user runs `omnictx aws ci-bot`
- **THEN** the config file contains `aws_profile: ci-bot` and the exit code is 0

#### Scenario: Unknown profile is rejected
- **WHEN** no local profile named `nope` exists and the user runs `omnictx aws nope`
- **THEN** nothing is written, an error naming the unknown profile goes to stderr, and the exit code is 2

#### Scenario: Switch under the mute flips state but not visibility
- **WHEN** the config file contains `enabled: false` and `kube: true`, and the user runs `omnictx aws digital-dev` for an existing profile
- **THEN** the config file contains `aws_profile: digital-dev` and `cloud: aws`, while `enabled: false` and `kube: true` are unchanged

#### Scenario: Unreadable AWS sources fail loudly
- **WHEN** neither `~/.aws/config` nor `~/.aws/credentials` can be read and the user runs `omnictx aws anything`
- **THEN** nothing is written, an error goes to stderr, and the exit code is 1

### Requirement: Aliases resolve before validation
`omnictx aws <short>` — and the `cloud aws <short>` spelling — SHALL resolve
`<short>` through `aliases.aws.<short>` from the omnictx config before
validating, identically to the gcp/azure switch paths.

#### Scenario: Alias resolves to a canonical profile
- **WHEN** the config contains `aliases: {aws: {dev: digital-dev}}` and `[profile digital-dev]` exists, and the user runs `omnictx aws dev`
- **THEN** the config file contains `aws_profile: digital-dev` and the exit code is 0

### Requirement: `region` and `list` are reserved words
The words `region` and `list` SHALL NOT be accepted as profile arguments in
either spelling: `omnictx aws list` / `omnictx cloud aws list` keep the
listing behavior, and `omnictx aws region ...` / `omnictx cloud aws region
...` dispatch to the region subcommand.

#### Scenario: Reserved word does not switch
- **WHEN** the user runs `omnictx aws list`
- **THEN** the profile table is printed and `aws_profile:` is not written
