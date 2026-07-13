## Context

`cmd/omnictx/main.go` dispatches subcommands from `args[0]`: the cases `"on", "enable"`, `"off", "disable"`, and `"toggle"` (→ `runToggle`) sit next to the documented `init`/`cloud`/`kube`/`ns` ones. `runToggle` re-reads the config file with a hand-rolled line scan for `enabled:` instead of `internal/config`. README documents `toggle` in four places; `enable`/`disable` are documented nowhere. Several comments and doc passages describe features that were removed (`omnion`/`omnioff`, `cloud use`) or never built (`--debug`, `--cloud`, env vars in `--help`), and README's icon claim ("one `☁` for any provider") contradicts the per-provider Nerd Font glyphs in `internal/cloud/cloud.go` that PRD §5.7 mandates.

## Goals / Non-Goals

**Goals:**
- One consistent CLI surface: `on`/`off` only; code, `--help`, README, AGENTS.md, and PRD all agree.
- Delete the duplicated config parser (`runToggle`).
- Remove dead comments/test lines left by earlier removals; fix stale doc claims.
- Make `OMNICTX_SHELL` officially documented (AGENTS.md) instead of half-legal.

**Non-Goals:**
- No new functionality, no changes to `on`/`off` behavior, render mode, or any provider/kube logic.
- No deprecation window or warning message for `toggle`/`enable`/`disable` — the tool is pre-1.0 and the words simply stop being subcommands.
- No PRD rewrite beyond the four stale passages named in the proposal (the PRD stays a historical living spec).

## Decisions

- **Fall-through, not an error, for removed words.** After removing the switch cases, `omnictx toggle` hits `runRender`, which ignores positional arguments and renders normally (exit 0). This matches the existing contract for any unknown word in render mode and the core invariant (render never errors); adding a "unknown subcommand" error path would create a new error surface render mode must not have. Alternative considered: exit 2 with usage — rejected because bare `omnictx <anything>` has always rendered, and `kube`/`cloud`/`ns` already own their own usage errors.
- **`runCloudUse` → `runCloudSwitch`.** Pure rename (function + comment + any test references); "switch" matches the README/AGENTS wording for the account-switch feature. The test cases using an account literally named `use` stay — they now guard that `use` is a valid account name, which is exactly the post-removal contract.
- **Keep `OMNICTX_SHELL`.** It is implemented (`internal/config/config.go`) and documented in README's configuration table; removing it would be the only user-visible regression in an otherwise cleanup-only change. AGENTS.md gets one line in the config section. Alternative (remove it as redundant with `--shell`) rejected: `--shell` wins by precedence anyway, so the env var is harmless and occasionally useful for pipes/debugging.
- **New capability spec `enabled-toggle-cli`.** The global enabled toggle had no spec under `openspec/specs/` (only cloud/kube/ns do). Rather than a delta against nothing, the change adds the capability spec that positively defines `on`/`off` and explicitly excludes `toggle`/`enable`/`disable`, so a future change can't silently re-grow aliases.

## Risks / Trade-offs

- [Users with `omnictx toggle` in a keybind get a silent no-op that still prints the segment] → acceptable: exit 0 with visible prompt output is the render contract; README loses every `toggle` mention so the docs won't advertise it.
- [Renaming `runCloudUse` touches tests without changing behavior] → mechanical rename, `make test` (-race) guards it.
- [Doc edits (PRD/AGENTS/README) drifting from each other again] → each fix is pinned to a concrete quoted claim in tasks.md so the checker can grep for leftovers.

## Migration Plan

Single commit on `tech/leftovers`; no data or config migration (config files written by `toggle` are plain `enabled:` values that `on`/`off` keep honoring). Rollback = revert the commit.

## Open Questions

(none)
