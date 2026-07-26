# promote-provider-commands

> Status: parked — decisions captured with the user on 2026-07-26; design.md,
> specs and tasks deliberately deferred ("спеки подождут"). Do not implement
> before add-fzf-picker ships (its `internal/picker` is reused here).

## Why

`cloud` earns its keep only for slot-level meta operations (`on/off/auto/none`, printing the effective provider, pinning). For the most-typed commands — account switching and listing — it is pure ceremony: `omnictx cloud aws prod` says nothing that `omnictx aws prod` doesn't. Promoting the provider words to the top level shortens the daily path and opens the natural home for the provider fzf picker (bare `omnictx aws` → fuzzy pick a profile), mirroring what add-fzf-picker does for bare `kube`/`ns`.

## What Changes

- **Top-level provider commands**: `omnictx aws|azure|gcp <account>`, `... list`, and `omnictx aws region [<region>|auto]` move to the top level with semantics identical to today's `cloud <provider> ...` forms.
- **Bare `omnictx aws|azure|gcp`** follows the add-fzf-picker two-branch pattern: interactive (TTY + fzf + no `OMNICTX_IGNORE_FZF`) → fzf over the provider's local accounts, selection performs the account switch (which already pins the provider); non-interactive → prints the provider's active account. NOTE the deliberate split from today: bare `cloud <provider>` PINS — the top-level bare form never pins by itself, so a piped `omnictx aws` cannot mutate state.
- **`cloud` stays for slot meta only**: bare `cloud` (print effective), `cloud auto|none|on|off` (selection + visibility incl. the mute-lifting `cloud on` semantics), and `cloud <provider>` (explicit pin) — all unchanged.
- **Old forms kept as aliases** (user decision): `cloud <provider> <account>`, `cloud <provider> list`, `cloud aws region ...` keep working and dispatch to the same code; scripts and muscle memory unaffected.
- Reuses `internal/picker` (Decide + Run) from add-fzf-picker; no new dependency; render mode untouched.

## Capabilities

### New Capabilities

_TBD at spec time — likely none; this reshapes command surface over existing behavior._

### Modified Capabilities

- `cloud-selection-cli`: top-level provider words; bare-provider print/picker; pin confined to `cloud <provider>`.
- `aws-profile-switch-cli`: `omnictx aws <profile>` as the primary form, `cloud aws <profile>` as alias.
- `aws-region-cli`: `omnictx aws region ...` as the primary form, alias likewise.

## Impact

- `cmd/omnictx/main.go` dispatch (new top-level words `aws`, `azure`, `gcp` — collision-checked against existing subcommands), `--help` usage text, AGENTS.md, README, main_test.go.
- Specs: delta files for the three capabilities above — deferred.
