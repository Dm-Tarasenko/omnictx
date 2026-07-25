# aws-switch — design

## Context

Every switchable entity in omnictx has a persistent state file the binary can
rewrite (kubeconfig `current-context`, gcloud `active_config`, Azure
`isDefault`). AWS has none: the active profile is `AWS_PROFILE` in the parent
shell's environment, and a child process cannot set its parent's env. The
ecosystem answers this with session-scoped shell functions that eval `export`
lines (awsp, awsume, granted) or with GUI tools that rewrite `[default]`
credentials (Leapp). omnictx has an asset none of them do: an `init`-installed
hook that already runs inside the parent shell on every prompt, plus a config
file it already machine-writes (`on`/`off`, `pinCloudAfterUse`).

Render is already env-first for AWS (`AWS_PROFILE > AWS_VAULT > "default"`,
`AWS_REGION > AWS_DEFAULT_REGION > profile config`), so the display side needs
no changes — only the switching side is missing.

## Goals / Non-Goals

**Goals:**

- `cloud aws <profile>` switches globally: every terminal running the omnictx
  hook follows on its next prompt — same semantics as gcp/azure switches.
- Region override (`cloud aws region <r>`) with the same global semantics,
  independent of the profile choice.
- Manual `export AWS_PROFILE=...` (or direnv, or aws-vault) in a session pins
  that session: the hook never stomps values it did not set itself.
- All state in omnictx's own config file; `~/.aws/*` stays read-only.
- All decision logic in Go under table-driven tests; shell templates stay dumb.
- Render invariant intact: the per-prompt code path never writes, never fails
  the prompt, exits 0.

**Non-Goals:**

- Session-scoped switching as a feature (`scope: session`, kubeconfig overlays,
  `OMNICTX_SESSION_ID`) — a possible future change; this design must not block
  it, but builds none of it.
- Credential handling (STS, assume-role, SSO login) — omnictx stays an offline
  pointer-switcher; aws CLI v2 + `aws sso login` handle credentials.
- Interactive pickers (fzf) inside the binary — UI sugar belongs in user
  aliases on top of `cloud aws list`; the yaml.v3-only dependency rule stands.
- An `OMNICTX_AWS_PROFILE`-style env override — AWS already has native env
  overrides (`AWS_PROFILE`, `AWS_REGION`), which the pin logic honors instead.

## Decisions

### D1: State lives in config.yaml (`aws_profile:`, `aws_region:`)

Alternatives considered:

- **Write to `~/.aws/*`** — rejected: AWS defines no current-profile
  semantics, so anything we wrote there would be our invention inside a foreign
  file (worst of both worlds). Rewriting `[default]` breaks SSO /
  `credential_process` profiles.
- **Separate state file** (gcloud `active_config` analogy) — rejected: the
  project already persists machine-written pins in config.yaml
  (`cloud:`, `enabled:`, `kube:` via `setConfigKeys`); a second store diverges
  from that for no benefit once the hook reads values through the binary (D2).

`aws_profile` absent = omnictx does not manage `AWS_PROFILE` at all (today's
behavior). `aws_region` absent = region comes from the profile's own config.
Writes go through the existing `setConfigKeys` (atomic, comment-preserving).

### D2: One exec per prompt, three-line output; decisions in Go

The hook already execs the binary once per prompt for the segment. That same
invocation returns three lines:

```
line 1: AWS_PROFILE directive   ("" = leave alone, "-" = unset, else = export)
line 2: AWS_REGION directive    (same encoding)
line 3: prompt segment          (exactly today's render output)
```

Alternatives considered:

- **Second exec** (binary or grep) to fetch the value — rejected: doubles
  per-prompt process spawns for nothing; the info is available in the exec we
  already pay for.
- **Shell parses config.yaml** — rejected: YAML scraping in shell is fragile
  and untestable.
- **`eval "$(omnictx hook ...)"`** (direnv-style) — rejected: eval'ing binary
  output raises the stakes of the render invariant (a quoting bug breaks the
  shell, profile names become injection surface). With the line protocol the
  snippet does `export AWS_PROFILE="$p"` itself; hostile names cannot escape.

The binary computes the directives from env (`AWS_PROFILE`, `AWS_REGION`,
`AWS_VAULT`, `__OMNICTX_AWS_PROFILE`, `__OMNICTX_AWS_REGION`) plus config, so
the whole decision table is Go and unit-testable. Because the binary knows what
line 1/2 will export, it renders line 3 as if the export already happened — the
prompt is correct on the very prompt that applies the switch, no one-cycle lag.

Pin logic per variable (same for profile and region):

| config value | current env value        | directive |
|--------------|--------------------------|-----------|
| any          | `AWS_VAULT` set          | `""` (aws-vault owns the session) |
| set to V     | empty, or equals marker  | `V` (follow global) |
| set to V     | differs from marker      | `""` (session manually pinned) |
| absent       | equals marker            | `-` (we set it, global cleared → unset) |
| absent       | anything else            | `""` |

Markers (`__OMNICTX_AWS_PROFILE`, `__OMNICTX_AWS_REGION`) are **exported** so
subshells/tmux inherit the value+marker pair and keep following the global
state instead of misreading the inherited value as a manual pin.

Failure discipline: this is render mode — any error yields two empty directive
lines and whatever segment can be rendered (possibly empty), exit 0.

### D3: Command surface under `cloud aws`

- `cloud aws <profile>`: resolve `aliases.aws.<short>` (already generic in the
  dispatcher), validate against `aws.Profiles()` (config + credentials names);
  unknown → exit 2, unreadable sources → exit 1 (mirrors gcp/azure); on success
  write `aws_profile:` and reuse `pinCloudAfterUse("aws")`. The
  hint-and-exit-2 branch is deleted. stderr note on success mentions the value
  applies to hook-running shells on their next prompt.
- `cloud aws region <r>`: offline format validation only
  (`^[a-z]{2}(-[a-z]+)+-\d+$` — existence would need network); invalid → exit 2.
- `cloud aws region auto`: removes the override (region falls back to the
  profile's configured region). `auto` matches the project vocabulary
  ("derive it yourself").
- `cloud aws region`: prints the effective region (env > override > profile
  config), read-only.
- `region` joins `list` as a reserved word for the profile argument.

### D4: Template changes stay mechanical

`init bash|zsh` snippets read three lines from the single exec and apply the
directives verbatim (export / unset / skip). No conditionals beyond the
directive decoding, no shell functions beyond the existing prompt function.
Idempotency guard unchanged. The snippet/binary contract changes together in
one release; rc-file `eval "$(omnictx init ...)"` picks the new snippet up on
the next shell start, and a version-skewed pairing degrades to "no env sync"
(old snippet reads line 1 as the whole output = segment; acceptable during the
one-shell-restart window — verify in tests that an old-style single-line
consumer still gets a usable prompt or document the restart requirement in the
release notes).

## Risks / Trade-offs

- [Silent flip of background terminals: tab 2 quietly becomes prod on its next
  prompt] → This is omnictx's existing, documented semantics for kube/gcp/azure
  — AWS becomes consistent, not more dangerous; the prompt segment displaying
  the flip is the product's core mitigation. Manual-pin logic gives blast-
  radius-sensitive users their native escape hatch (`export AWS_PROFILE=...`).
- [User has `export AWS_PROFILE=x` in their rc file] → marker is empty when the
  hook first runs, so every shell looks manually pinned and global switches
  never apply. Accepted: explicit config wins; must be documented prominently
  ("remove the rc export, run `omnictx cloud aws x` once instead").
- [Shells without the omnictx hook (starship users) see no env change] →
  switch's stderr note states the hook requirement; the config value alone
  still feeds nothing — this feature requires `init`.
- [Region typo passes format validation] → offline tool cannot know the real
  region list; the prompt immediately shows the wrong region, which is the
  detection mechanism.
- [Line protocol is positional] → any future third variable appends a line
  before the segment; binary and templates ship together, and shellinit golden
  tests pin the contract.
- [`region` reserved word collides with a real profile named "region"] →
  vanishingly unlikely; consistent with the existing `list` reservation.

## Migration Plan

1. Ship binary + templates in one release; README/AGENTS.md updated in the same
   change (AWS "excluded provider" wording replaced).
2. Users: restart shells (or re-eval `init`) once; optionally delete homemade
   sticky files/aliases.
3. Rollback: revert the release — config keys `aws_profile`/`aws_region` are
   ignored by older binaries (unknown keys are dropped by the config reader),
   and no foreign file was ever touched.

## Open Questions

- None blocking. Future session-scope work (kubeconfig overlays, per-window
  state) builds on the same hook + directive protocol; deliberately deferred.
