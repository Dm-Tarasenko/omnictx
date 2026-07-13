## Why

The CLI surface has drifted from the docs: `omnictx toggle` exists in code and README but not in `--help`, AGENTS.md, or the PRD, and the `enable`/`disable` aliases exist only in code — documented nowhere at all. `runToggle` also carries a second, hand-rolled parser of the config file next to `internal/config`. On top of that, several comments and doc claims describe removed or never-built features (`--debug`, `omnion`/`omnioff`, `cloud use`, a `--cloud` flag, per-provider `☁`). This change removes the leftovers so code, help, and docs describe the same tool.

## What Changes

- **BREAKING**: Remove the `omnictx toggle` subcommand. The persistent enabled state is controlled by `omnictx on` / `omnictx off` only; `omnictx toggle` falls through to render mode like any unknown word (prints the segment, exits 0 — render mode never errors). Its duplicated config parser (`runToggle`) is deleted with it.
- **BREAKING**: Remove the undocumented `enable` / `disable` aliases for `on` / `off`. They were never in README, `--help`, AGENTS.md, or the PRD.
- Remove all `toggle` mentions from README (daily-use block, env table, usage list, configuration table).
- Legalize `OMNICTX_SHELL` in AGENTS.md: it exists in code and the README configuration table, but AGENTS.md never mentions it — add it to the config section as the session-scoped counterpart of `--shell`.
- Dead-comment cleanup (no behavior change):
  - `internal/config/config.go` — drop the reference to a `--debug` flag that does not exist (notes go to stderr via interactive commands' warnings).
  - `cmd/omnictx/main.go` — drop the "omnioff path" comment (the `omnion`/`omnioff` shell functions were removed long ago).
  - `cmd/omnictx/main.go` — rename `runCloudUse` → `runCloudSwitch` (the `cloud use` subcommand was removed by `2026-07-13-remove-cloud-use-subcommand`; the name is a leftover).
  - `internal/shellinit/shellinit_test.go` — fix the comment claiming the test "asserts the toggle functions exist" (it asserts the opposite) and delete the never-asserted `HAS_OMNION` / `HAS_OMNIOFF` echo lines.
- Stale-doc fixes:
  - README: the header example and "Output format" claim "one `☁` for any provider" — the code renders per-provider Nerd Font glyphs (PRD §5.7); fix both.
  - PRD: §4.7 still says namespace switching is out of scope (it shipped via `2026-07-13-namespace-switch`); the header says `Branch: feat/omnictx`; §5.1 mentions a `--cloud` flag that was never built (only `OMNICTX_CLOUD`); §5.6/§8.3 claim `--help` lists env vars in a Configuration section (it does not).
  - AGENTS.md: "See section 7.5 of PRD.md" — the PRD has no §7.5; the Definition of Done is §8.3.

## Capabilities

### New Capabilities
- `enabled-toggle-cli`: the persistent global on/off surface — `omnictx on` / `omnictx off` write `enabled:` to the config file; `toggle`, `enable`, and `disable` are explicitly not part of the CLI surface.

### Modified Capabilities
(none — cloud-selection-cli, kube-context-cli, and kube-namespace-cli are untouched)

## Impact

- `cmd/omnictx/main.go` — subcommand dispatch (drop `toggle`, `enable`, `disable` cases), delete `runToggle`, rename `runCloudUse`, comment fix.
- `cmd/omnictx/main_test.go` — drop/adjust tests covering `toggle` and the aliases, if any.
- `internal/config/config.go`, `internal/shellinit/shellinit_test.go` — comment/dead-line cleanup only.
- `README.md` — toggle mentions ×4, `☁` claim ×2.
- `PRD.md` — §4.7, header branch line, §5.1 `--cloud`, §5.6/§8.3 help-env claim.
- `AGENTS.md` — §7.5 → §8.3 reference, `OMNICTX_SHELL` mention.
- Anyone with `omnictx toggle` / `enable` / `disable` in muscle memory or scripts breaks (documented as the breaking changes above).
