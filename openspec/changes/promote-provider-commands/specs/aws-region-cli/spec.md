# aws-region-cli — delta for promote-provider-commands

## MODIFIED Requirements

### Requirement: Persist a region override via `omnictx aws region <region>`
The CLI SHALL provide `omnictx aws region <region>` — with `omnictx cloud aws
region <region>` as an accepted alias dispatching to the same code — that
validates `<region>` offline against the format `^[a-z]{2}(-[a-z]+)+-\d+$`
(existence checks require network and are out of scope) and, on success,
writes `aws_region: <region>` to the omnictx config file via the single-key
write machinery (other keys and comments preserved; file created if absent)
and exits 0. The override is independent of the active profile: switching
profiles SHALL NOT clear it. An invalid format SHALL write nothing, print a
usage error to stderr, and exit 2. Neither form SHALL touch the
`enabled`/`kube` display toggles — visibility is controlled only by the
on/off commands.

#### Scenario: Valid region is persisted
- **WHEN** the user runs `omnictx aws region eu-central-1`
- **THEN** the config file contains `aws_region: eu-central-1` and the exit code is 0

#### Scenario: The cloud spelling behaves identically
- **WHEN** the user runs `omnictx cloud aws region eu-central-1`
- **THEN** the result is identical to `omnictx aws region eu-central-1`

#### Scenario: Garbage region is rejected
- **WHEN** the user runs `omnictx aws region Frankfurt`
- **THEN** nothing is written, stderr shows a usage error, and the exit code is 2

#### Scenario: Region set under the mute does not lift it
- **WHEN** the config file contains `enabled: false` and the user runs `omnictx aws region eu-central-1`
- **THEN** the config file contains `aws_region: eu-central-1` and `enabled: false` is unchanged

#### Scenario: Profile switch keeps the override
- **WHEN** `aws_region: eu-central-1` is persisted and the user runs `omnictx aws digital-dev`
- **THEN** the config file still contains `aws_region: eu-central-1`

### Requirement: `omnictx aws region auto` clears the override
`auto` — in either spelling — SHALL remove the `aws_region:` override from the
config file (matching the project vocabulary where `auto` means "derive it
yourself"), so the effective region falls back to the active profile's
configured region. Clearing when no override exists SHALL succeed (idempotent)
with exit 0.

#### Scenario: Clear an existing override
- **WHEN** the config file contains `aws_region: eu-central-1` and the user runs `omnictx aws region auto`
- **THEN** the config file no longer sets an `aws_region` override and the exit code is 0

#### Scenario: Clearing twice is fine
- **WHEN** no `aws_region` override is set and the user runs `omnictx aws region auto`
- **THEN** the exit code is 0 and the config file is otherwise unchanged

### Requirement: Bare `omnictx aws region` prints the effective region
With no argument — in either spelling — the command SHALL print the effective
region resolved as `AWS_REGION` env > `AWS_DEFAULT_REGION` env > persisted
`aws_region:` override > the active profile's region from `~/.aws/config`,
and exit 0 without writing anything. When no region can be resolved the
command SHALL print nothing and exit 0. The bare region form is never
interactive: it is a read-only print, not a picker surface.

#### Scenario: Print resolves the override
- **WHEN** no AWS region env vars are set, the config file contains `aws_region: eu-central-1`, and the user runs `omnictx aws region`
- **THEN** stdout is `eu-central-1`, nothing is written, and the exit code is 0

#### Scenario: Env wins over the override
- **WHEN** `AWS_REGION=us-east-1` is set and the config file contains `aws_region: eu-central-1`
- **THEN** `omnictx aws region` prints `us-east-1`
