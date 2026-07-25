# aws-switch

## Why

AWS is the only provider omnictx cannot switch: `cloud aws <account>` prints an
`export AWS_PROFILE=...` hint and exits 2, because AWS — unlike gcloud
(`active_config`) and Azure (`azureProfile.json`) — has no persistent
current-profile file; the active profile lives in the parent shell's
environment, which a child process cannot change. Users fall back to hand-rolled
sticky files and rc aliases. omnictx already owns the two missing pieces — a
machine-written config file and a shell hook that runs inside the parent shell
on every prompt — so it can close the gap without touching `~/.aws` and make
AWS behave exactly like its siblings: switch once, every terminal follows.

## What Changes

- `omnictx cloud aws <profile>` becomes a real switch: validates the profile
  against `~/.aws/config` + `~/.aws/credentials`, persists `aws_profile:` to
  the omnictx config file, pins `cloud: aws` (same post-switch pin as
  gcp/azure). The current hint-and-exit-2 behavior is removed.
- New `omnictx cloud aws region <region>|auto` subcommand: persists an
  `aws_region:` override (session-independent of the profile), `auto` clears it
  (region falls back to the profile's configured region), bare
  `cloud aws region` prints the effective region. `region` becomes a reserved
  word (cannot be an AWS profile argument).
- The per-prompt shell hook contract changes from one line to three: line 1 =
  `AWS_PROFILE` to export, line 2 = `AWS_REGION` to export, line 3 = the prompt
  segment. Empty line = leave the variable alone; literal `-` = unset it (only
  if the hook set it). The decision logic (respect `AWS_VAULT`, respect manual
  `export` pins via `__OMNICTX_AWS_PROFILE`/`__OMNICTX_AWS_REGION` markers)
  lives in Go, not in the shell templates.
- `init bash|zsh` templates gain the export/unset plumbing (dumb line
  consumers; all decisions come from the binary).
- `~/.aws/*` stays read-only: the only state written is omnictx's own config
  file. Render mode still never writes anything.

## Capabilities

### New Capabilities

- `aws-profile-switch-cli`: `cloud aws <profile>` — validation, `aws_profile:`
  persistence, alias resolution, exit codes, cloud pin after switch.
- `aws-region-cli`: `cloud aws region <r>|auto` and bare print — `aws_region:`
  persistence, offline format validation, reserved word.
- `aws-env-sync-hook`: the three-line hook output contract and the pin/unset
  decision logic (env markers, `AWS_VAULT` guard), plus the bash/zsh init
  template plumbing that applies it.

### Modified Capabilities

<!-- none: `omnictx cloud aws` (single-arg cloud pin) keeps its spec'd behavior;
     the removed hint path was never covered by a spec -->

## Impact

- `cmd/omnictx/main.go`: `cloud aws` dispatch (replace the hint branch), new
  `region` sub-dispatch, hook-mode output.
- `internal/aws`: profile validation helper on top of `Profiles()`; region
  format check; effective-region resolution already exists.
- `internal/config`: two new keys `aws_profile` / `aws_region` (written via the
  existing `setConfigKeys` machinery; read via the existing merge).
- `internal/shellinit` + templates: three-line consumption, export/unset lines,
  exported marker variables; golden files regenerated. The snippet/binary
  contract changes — users must re-eval `init` after upgrading (rc-file eval
  makes this automatic on new shells).
- `internal/render` / providers: no changes — `AWS_PROFILE`/`AWS_REGION` env
  precedence is already how render resolves the segment.
- Docs: AGENTS.md and README currently describe AWS as the excluded provider —
  both need the new story.
