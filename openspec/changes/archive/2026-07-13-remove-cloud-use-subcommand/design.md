## Context

`omnictx cloud <provider> use <account>` is the only three-argument form the `cloud` subcommand accepts today. The proposal drops `use` so the same switch happens as `omnictx cloud <provider> <account>` — a plain two-argument form, sitting next to the existing `omnictx cloud <provider> list`.

## Goals / Non-Goals

**Goals:**
- Shorten the account-switch command by removing the `use` keyword.
- Keep every other behavior of `cloud-selection-cli` (pin values, `list`, aliases, exit codes, warnings, azure/gcp write semantics, AWS hint) byte-for-byte the same — only the argument shape changes.

**Non-Goals:**
- No change to how gcp/azure actually perform the switch (`internal/gcp`, `internal/azure` write paths are untouched).
- No new provider capabilities, no new config keys.
- No attempt to keep `use` working as a deprecated alias — this is a clean breaking rename, not a soft migration.

## Decisions

- **Two-argument dispatch on the second word.** `omnictx cloud <provider> <arg2>`: if `<arg2>` is exactly `list` (case-insensitive), keep today's list-table behavior; otherwise treat `<arg2>` as the account name and run the same switch logic that used to sit behind the `use` keyword. This means `list` remains the only reserved second argument — it can never be a literal account name (unchanged constraint, just no longer shared with `use`).
- **`use` is no longer reserved.** Since the three-argument `<provider> use <account>` form is deleted outright, `use` has no special meaning left to protect. `omnictx cloud gcp use` (two args) is now valid syntax that means "switch to the gcp configuration literally named `use`."
- **Three-argument invocations become a hard usage error.** With `use` gone, no valid command has three arguments anymore. `omnictx cloud <provider> use <account>` (and any other 3-arg input) prints the usage message to stderr and exits 2 — the same failure path as today's "anything else" case, just covering one more shape.
- **AWS keeps its hint, just shorter.** `omnictx cloud aws <profile>` prints the `export AWS_PROFILE=<profile>` hint to stderr and exits 2, writing nothing — identical semantics to today's `cloud aws use <profile>`, only the trigger syntax changes.
- **Alias resolution moves with the argument, not the keyword.** `aliases.<provider>.<short>` resolution already applied to "whatever the third word was"; it now applies to "whatever the second word is, when it isn't `list`." No change to `internal/config` or alias lookup code — only the call site in `main.go` that decides which string is the account argument.
- **Provider restriction unchanged.** The account-switch form is only meaningful for `azure`/`gcp`/`aws` as the first argument. `auto`, `none`, `on`, `off` are pin-only values and never take a second argument; `omnictx cloud auto <anything>` stays a usage error exit 2, same as today.

## Risks / Trade-offs

- [Breaking change for existing scripts/muscle memory using `use`] → Documented as **BREAKING** in the proposal; no compatibility shim, since the project has no stability guarantee across versions yet and a shim would keep `use` reserved forever for no benefit.
- [Account literally named `list`] → Pre-existing limitation carried over unchanged (already true when `use` existed); not introduced by this change.
- [Silent behavior change if a gcp config or azure subscription happens to be named `use`] → Previously unreachable (the 3-arg form required the literal word `use` as argument 2, so an account named `use` could never be targeted by position 3 without a redundant `use use`... actually it could: `cloud gcp use use`). Document the corner case in the spec scenario but do not special-case it — first-match-wins on the literal argument, same as any other account name.

## Migration Plan

CLI-only, no persisted data format changes and no config migration needed.
1. Update `main.go` argument parsing/dispatch and the `--help` usage text.
2. Update `cloud-selection-cli` spec requirements (delta in this change).
3. Update README examples referencing `cloud <provider> use <account>`.
4. No rollback concerns beyond reverting the commit — no on-disk state is versioned by this syntax.

## Open Questions

None — scope confirmed with the user: drop `use` entirely, `cloud <provider>` (no second arg) keeps meaning "pin the displayed provider," `cloud <provider> <account>` performs the switch.
