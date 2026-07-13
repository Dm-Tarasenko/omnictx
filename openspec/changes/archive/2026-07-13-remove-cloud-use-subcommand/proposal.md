## Why

`omnictx cloud <azure|gcp> use <account>` requires typing the redundant word `use` to switch an account. Dropping it — `omnictx cloud <azure|gcp> <account>` — shortens the everyday command without losing any functionality.

## What Changes

- **BREAKING**: Remove the `use` keyword from the cloud-switch command. `omnictx cloud azure <account>` and `omnictx cloud gcp <account>` now switch the account directly; `omnictx cloud azure use <account>` / `omnictx cloud gcp use <account>` are no longer accepted (three-argument form with `use` becomes invalid, exit 2).
- **BREAKING**: `omnictx cloud aws <profile>` replaces `omnictx cloud aws use <profile>` for the AWS session hint (still writes nothing, still prints `export AWS_PROFILE=<profile>` to stderr, still exits 2).
- The word `use` stops being reserved for the third argument slot; the two-argument form `cloud <provider> <account>` now uses the *second* argument as the account (not a subcommand keyword), matched through the existing alias/name/id resolution unchanged.
- `list` stays reserved: `omnictx cloud <provider> list` continues to mean "print the table," so `list` can never be a literal account name.
- All other cloud-selection semantics (bare `cloud <value>` pin, `cloud list`, `on`/`off`, alias resolution, warnings on stderr, exit codes for unknown/ambiguous accounts) are unchanged — only the surface syntax for switching an account changes.
- Update `--help` output and README examples that reference `cloud <provider> use <account>`.

## Capabilities

### New Capabilities
(none)

### Modified Capabilities
- `cloud-selection-cli`: the account-switch syntax drops the `use` keyword (`cloud <provider> <account>` instead of `cloud <provider> use <account>`), for azure, gcp, and the aws hint form; `use` is no longer a reserved word, argument-count validation rules change accordingly.

## Impact

- `cmd/omnictx/main.go` — cloud subcommand argument parsing/dispatch.
- `internal/gcp`, `internal/azure` — invoked the same way, just from the new dispatch path; no change to their write logic.
- `openspec/specs/cloud-selection-cli/spec.md` — requirements naming `use` and the reserved-word/argument-count rules.
- README and `--help` usage text showing `cloud <provider> use <account>` examples.
- Any user scripts or muscle memory relying on `cloud <provider> use <account>` breaks (documented as the breaking change above).
