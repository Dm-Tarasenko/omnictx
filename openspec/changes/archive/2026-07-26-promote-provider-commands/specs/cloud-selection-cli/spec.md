# cloud-selection-cli — delta for promote-provider-commands

## ADDED Requirements

### Requirement: Top-level provider commands
The CLI SHALL accept `aws`, `azure`, and `gcp` as top-level subcommand words.
`omnictx <provider> list` SHALL behave identically to `omnictx cloud
<provider> list` (same table, same warnings, same exit codes), `omnictx
<provider> <account>` identically to `omnictx cloud <provider> <account>`
(same alias resolution, validation, switch, post-switch `cloud: <provider>`
pin, silent success, exit codes), and `omnictx aws region [<region>|auto]`
identically to `omnictx cloud aws region [<region>|auto]`. The old
`cloud <provider> ...` spellings SHALL remain accepted aliases dispatching to
the same code paths — no behavioral difference between the two spellings, and
no deprecation. The reserved words are unchanged: `list` for every provider,
plus `region` for aws. Any invocation with more arguments than these forms
accept SHALL print a usage message to stderr and exit 2. The three words
consequently no longer fall through to render mode.

#### Scenario: Top-level switch equals the cloud form
- **WHEN** `<gcloud>/configurations/config_work` exists and the user runs `omnictx gcp work`
- **THEN** the result is identical to `omnictx cloud gcp work`: `active_config` contains `work`, the omnictx config contains `cloud: gcp`, and the exit code is 0

#### Scenario: Top-level list equals the cloud form
- **WHEN** the user runs `omnictx aws list`
- **THEN** the output and exit code are identical to `omnictx cloud aws list`

#### Scenario: Top-level region form dispatches to the region subcommand
- **WHEN** the user runs `omnictx aws region eu-central-1`
- **THEN** the result is identical to `omnictx cloud aws region eu-central-1`: the config file contains `aws_region: eu-central-1` and the exit code is 0

#### Scenario: Old spellings keep working
- **WHEN** the user runs `omnictx cloud azure prod-subscription` for an existing subscription
- **THEN** the switch succeeds exactly as before this change

#### Scenario: Too many arguments
- **WHEN** the user runs `omnictx gcp work extra`
- **THEN** stderr shows a usage message, nothing is written, and the exit code is 2

### Requirement: Bare `omnictx <provider>` prints or picks the active account
When invoked with no further argument, a top-level provider command SHALL have
two branches selected by the same pure activation decision as bare `kube`/`ns`
(interactive only when stdout is a terminal AND `fzf` is on `PATH` AND
`OMNICTX_IGNORE_FZF` is unset or empty). The non-interactive branch SHALL
print the provider's active account — the same value the provider's `list`
table marks CURRENT (aws: `AWS_PROFILE` > `AWS_VAULT` > `default`; gcp:
`CLOUDSDK_ACTIVE_CONFIG_NAME` > `active_config` file > `default`; azure: the
`isDefault` subscription's name) — to stdout followed by a newline and exit 0;
when nothing is resolvable it SHALL print nothing and exit 0. The bare form is
read-only: unlike `omnictx cloud <provider>` (which persists a pin), it SHALL
NOT write anything — a piped `omnictx aws` cannot mutate state.

In the interactive branch the command SHALL pipe the provider's account names
(the same names, in the same order, as the `list` table rows) to `fzf` with
the active account in the header, and on selection perform exactly the
`omnictx <provider> <account>` switch (alias resolution, validation, write,
post-switch pin, silent success, same exit codes). A non-zero fzf exit SHALL
write nothing and exit 0. If executing fzf fails despite the activation
decision, the command SHALL fall back to the non-interactive print, exit 0.
With zero accounts, fzf SHALL NOT be invoked: the command prints nothing and
exits 0. The picker SHALL NOT read or write the `enabled`/`kube` display
toggles.

#### Scenario: Active AWS profile is printed
- **WHEN** `AWS_PROFILE=prod` is set and the user runs `omnictx aws` non-interactively (stdout not a terminal, or fzf absent, or `OMNICTX_IGNORE_FZF` set)
- **THEN** stdout is `prod`, nothing is written, and the exit code is 0

#### Scenario: Active gcloud configuration is printed
- **WHEN** `active_config` contains `work` and the user runs `omnictx gcp` non-interactively
- **THEN** stdout is `work` and the exit code is 0

#### Scenario: Default Azure subscription is printed
- **WHEN** `azureProfile.json` marks `prod-subscription` with `isDefault: true` and the user runs `omnictx azure` non-interactively
- **THEN** stdout is `prod-subscription` and the exit code is 0

#### Scenario: Bare form never pins
- **WHEN** the omnictx config contains `cloud: auto` and the user runs `omnictx aws` non-interactively
- **THEN** the config file still contains `cloud: auto` (contrast: `omnictx cloud aws` would persist `cloud: aws`)

#### Scenario: Interactive selection switches the account
- **WHEN** the interactive branch is active for `omnictx gcp`, configurations `default` and `work` exist with `default` active, and the user selects `work` in fzf
- **THEN** `active_config` contains `work` and the omnictx config contains `cloud: gcp`, exactly as after `omnictx gcp work`, and the exit code is 0

#### Scenario: Cancelling the picker writes nothing
- **WHEN** the interactive branch is active and the user exits fzf without selecting
- **THEN** no file is modified and the exit code is 0

#### Scenario: fzf exec failure degrades to the print
- **WHEN** the activation decision selected the interactive branch but executing fzf fails
- **THEN** the active account is printed, no file is modified, and the exit code is 0

#### Scenario: Zero accounts stay quiet
- **WHEN** the provider's local files define no accounts and the user runs the bare provider command
- **THEN** stdout is empty, fzf is never invoked, and the exit code is 0

## MODIFIED Requirements

### Requirement: Help lists the cloud subcommand
The grouped `--help` output SHALL list the `cloud` subcommand under the
Subcommands section, including the allowed values, alongside the existing
`init` and `on`/`off` entries. It SHALL additionally list the top-level
provider commands (`aws|azure|gcp [<account>|list]` and
`aws region [<region>|auto]`) as the primary provider forms, and note that
the `cloud <provider> ...` spellings remain accepted aliases.

#### Scenario: Help mentions cloud
- **WHEN** the user runs `omnictx --help`
- **THEN** the Subcommands section contains a `cloud` entry showing `azure|aws|gcp|auto|none`

#### Scenario: Help mentions the top-level provider commands
- **WHEN** the user runs `omnictx --help`
- **THEN** the Subcommands section lists the top-level provider forms and the region form
