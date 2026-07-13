## MODIFIED Requirements

### Requirement: Reject invalid values with a usage error
The subcommand SHALL validate its argument strictly. The word `list` is reserved and SHALL never be persisted or resolved as an account name. For any single argument other than `azure|aws|gcp|auto|none|on|off|list` (case-insensitive, surrounding whitespace ignored) it SHALL print a usage message naming the allowed values to stderr, SHALL NOT modify the config file, and SHALL exit with code 2. A two-argument form is valid only as `<azure|aws|gcp> list` (print the provider's account table) or `<azure|aws|gcp> <account>` (switch/hint, per provider); the first argument of a two-argument form MUST be `azure`, `aws`, or `gcp` — `auto`, `none`, `on`, `off`, or any other value as the first argument is a usage error. Any three-or-more-argument invocation SHALL print the usage message and exit 2. This is deliberately stricter than render mode, which silently normalizes unknown values to `auto` to protect the prompt.

#### Scenario: Unknown value
- **WHEN** the user runs `omnictx cloud awz`
- **THEN** stderr contains a usage message listing the allowed values, the config file is not modified, and the exit code is 2

#### Scenario: Case-insensitive input is normalized
- **WHEN** the user runs `omnictx cloud AWS`
- **THEN** the config file contains `cloud: aws` and the exit code is 0

#### Scenario: Invalid two-argument form with a non-provider first argument
- **WHEN** the user runs `omnictx cloud auto work`
- **THEN** stderr contains the usage message, no file is modified, and the exit code is 2

#### Scenario: Three-argument invocation is always invalid
- **WHEN** the user runs `omnictx cloud gcp use work` or `omnictx cloud azure activate work`
- **THEN** stderr contains the usage message, no file is modified, and the exit code is 2

### Requirement: AWS has no persistent switch — print the session hint
The form `omnictx cloud aws <profile>` SHALL NOT write anything. It SHALL print an explanation that AWS has no persistent current-profile concept together with the session command `export AWS_PROFILE=<profile>` to stderr, and exit 2.

#### Scenario: AWS hint
- **WHEN** the user runs `omnictx cloud aws prod`
- **THEN** stderr contains `export AWS_PROFILE=prod`, no file is modified, and the exit code is 2

### Requirement: Account aliases from the omnictx config file
The account argument (the second word of `omnictx cloud <azure|gcp> <account>`) SHALL first be resolved through a new optional `aliases` config key (`aliases.<provider>.<short> → canonical name or id`), defined only in the omnictx config file (no env var). When the argument matches an alias for that provider, the canonical value is used for matching; otherwise the argument is used as-is. Aliases never apply to the `list`, `on`, `off`, or pin forms.

#### Scenario: GCP alias resolves
- **WHEN** the config contains `aliases: {gcp: {w: work}}` and the user runs `omnictx cloud gcp w`
- **THEN** the `work` configuration is activated and the exit code is 0

#### Scenario: Azure alias to an id resolves
- **WHEN** the config contains `aliases: {azure: {dev: "0000-aaaa"}}` and the user runs `omnictx cloud azure dev`
- **THEN** the subscription with id `0000-aaaa` becomes the default

#### Scenario: Full names keep working without aliases
- **WHEN** no `aliases` key is configured and the user runs `omnictx cloud gcp work`
- **THEN** the switch succeeds exactly as before

## REMOVED Requirements

### Requirement: Switch the active GCP configuration with `omnictx cloud gcp use <name>`
**Reason**: The `use` keyword is removed from the cloud-switch syntax; replaced by `omnictx cloud gcp <name>` (see ADDED requirement of the same behavior under the new syntax).
**Migration**: Replace `omnictx cloud gcp use <name>` with `omnictx cloud gcp <name>` in scripts and muscle memory.

### Requirement: Switch the default Azure subscription with `omnictx cloud azure use <name-or-id>`
**Reason**: The `use` keyword is removed from the cloud-switch syntax; replaced by `omnictx cloud azure <name-or-id>` (see ADDED requirement of the same behavior under the new syntax).
**Migration**: Replace `omnictx cloud azure use <name-or-id>` with `omnictx cloud azure <name-or-id>` in scripts and muscle memory.

## ADDED Requirements

### Requirement: Switch the active GCP configuration with `omnictx cloud gcp <name>`
The form `omnictx cloud gcp <name>` SHALL activate the named gcloud configuration by atomically writing the name to `<gcloud>/active_config` (honoring `CLOUDSDK_CONFIG`), after verifying that `<gcloud>/configurations/config_<name>` exists. On success it SHALL also persist `cloud: gcp` to the omnictx config file, so the prompt immediately displays the provider that was just switched to, and exit 0. An unknown name SHALL print the available configuration names to stderr, write nothing, and exit 2. An unwritable target SHALL exit 1 with nothing partially written. `CLOUDSDK_ACTIVE_CONFIG_NAME` continues to override the file per-session; render reflects the switch on the next invocation. `<name>` MUST NOT be the literal word `list` (reserved for the list-table form).

#### Scenario: Activate an existing configuration
- **WHEN** `<gcloud>/configurations/` contains `config_default` and `config_work` with `active_config` = `default`, and the user runs `omnictx cloud gcp work`
- **THEN** `active_config` contains exactly `work`, the exit code is 0, and `omnictx cloud gcp list` marks `work` as current

#### Scenario: Successful switch pins the displayed cloud
- **WHEN** the omnictx config pins `cloud: azure` and the user runs `omnictx cloud gcp work` successfully
- **THEN** the omnictx config contains `cloud: gcp` and the next render shows the GCP segment

#### Scenario: Unknown configuration
- **WHEN** no `config_prod` exists and the user runs `omnictx cloud gcp prod`
- **THEN** stderr names the available configurations, `active_config` is unchanged, and the exit code is 2

#### Scenario: A configuration literally named "use" still works
- **WHEN** `<gcloud>/configurations/config_use` exists, and the user runs `omnictx cloud gcp use`
- **THEN** the configuration named `use` is activated and the exit code is 0

### Requirement: Switch the default Azure subscription with `omnictx cloud azure <name-or-id>`
The form `omnictx cloud azure <account>` SHALL set `isDefault: true` on the subscription in `azureProfile.json` whose `name` or `id` exactly matches the argument, and `isDefault: false` on all others. The rewrite SHALL be a parsed JSON round-trip (all fields preserved; key order/whitespace may change), SHALL preserve a leading UTF-8 BOM when the original file has one, and SHALL be written atomically. On success it SHALL also persist `cloud: azure` to the omnictx config file (the prompt immediately displays the switched-to provider) and exit 0. A failed switch SHALL NOT touch the omnictx config. When the argument matches more than one subscription by name, the command SHALL print the ambiguous entries with their ids, write nothing, and exit 2. An unknown account SHALL print the available subscriptions to stderr, write nothing, and exit 2. A missing or unparsable `azureProfile.json` SHALL exit 1 with nothing written. `<account>` MUST NOT be the literal word `list` (reserved for the list-table form).

#### Scenario: Switch by name
- **WHEN** `azureProfile.json` lists `dev-subscription` (`isDefault: false`) and `prod-subscription` (`isDefault: true`), and the user runs `omnictx cloud azure dev-subscription`
- **THEN** `dev-subscription` has `isDefault: true`, `prod-subscription` has `isDefault: false`, all other fields survive, and the exit code is 0

#### Scenario: Switch by id
- **WHEN** the user runs `omnictx cloud azure 0000-aaaa`
- **THEN** the subscription with that id becomes the default and the exit code is 0

#### Scenario: BOM is preserved
- **WHEN** the original file starts with the UTF-8 BOM and a switch succeeds
- **THEN** the rewritten file still starts with the BOM (and `omnictx` itself can re-read it)

#### Scenario: Duplicate names require the id
- **WHEN** two subscriptions share the name `N/A(tenant level account)` and the user runs `omnictx cloud azure "N/A(tenant level account)"`
- **THEN** stderr lists the matching entries with their ids, the file is unchanged, and the exit code is 2

#### Scenario: Broken profile refuses the write
- **WHEN** `azureProfile.json` is unparsable and the user runs `omnictx cloud azure anything`
- **THEN** the file is unchanged, an error is printed, and the exit code is 1
