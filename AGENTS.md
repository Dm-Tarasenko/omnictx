# AGENTS.md — omnictx

## What this is
`omnictx` is a Go CLI that prints a prompt segment with the active **cloud**
(Azure / AWS / GCP — exactly one) and the current **kube-context** + namespace,
and can also **switch** them (kube-context, gcloud configuration, Azure
subscription). Render mode — the code that runs on every shell prompt — works
on local config files directly, without kubectl/az/aws/gcloud and without
network access. Explicit interactive subcommands may shell out to `kubectl`;
the only online paths are `ns list` and the interactive bare-`ns` picker (the
switches stay file-based). Bare `kube`/`ns` upgrade to an fzf fuzzy pick when
interactive — fzf, like kubectl, is an optional external binary, never a Go
dependency.

## Core invariant
RENDER MODE NEVER breaks the prompt line and never writes anything: any error →
skip the segment and exit 0. No panics in production (top-level recover in main).
Writes to foreign files (kubeconfig, active_config, azureProfile.json) happen
ONLY in explicit interactive subcommands, which do the opposite: validate
strictly, warn on stderr, and fail loudly with non-zero exit codes.

## Stack and dependencies
- Go (current stable version, pinned in go.mod).
- Single external runtime dependency: gopkg.in/yaml.v3. Everything else is
  stdlib. Test-only dependencies are allowed sparingly: google/go-cmp
  (cmp.Diff instead of reflect.DeepEqual).
- Do not add client-go or network libraries.

## Commands
- Build:    make build   (go build -o bin/omnictx ./cmd/omnictx)
- Test:     make test    (go test ./... -race -count=1)
- Lint:     make lint    (golangci-lint run)
- Install:  make install (copy binary to ~/.local/bin)
- Update golden: go test ./internal/render -update

## Structure
- cmd/omnictx/main.go — flags/env, glue, top-level recover; dispatches subcommands.
  `--help`/`-h` prints a custom grouped usage (description, usage, subcommands,
  flags, `--flag` double-dash display) and exits 0.
  Only flag exposed in render mode: `--shell <bash|zsh|none>` (supplied by `init`,
  not persisted in config). All other settings via env vars or config file.
  Subcommands: `init <bash|zsh>`, `on` / `off` (persist enabled state to config;
  `on` means "show everything" — it also turns hidden parts back on, rewriting
  `kube: false` → `true` and `cloud: none` → `auto` while a concrete provider
  pin stays untouched; `off` only sets the mute),
  `cloud [azure|aws|gcp|auto|none|on|off]` (persist active cloud to config; `on`/`off`
  alias `auto`/`none` — a pin is not remembered across off/on; the literal `on`
  issued while the global mute is persisted (`enabled: false`) lifts the mute
  exposing ONLY the cloud slot: it writes `enabled: true` AND `kube: false`;
  plain `auto` and `off` never touch `enabled`; no argument prints
  the effective value; invalid value → usage error, exit 2),
  `cloud [azure|aws|gcp] list` (offline read-only table of local accounts: AWS
  profiles from config+credentials names, gcloud configurations, Azure
  subscriptions; bare `cloud list` = active provider; `list` reserved),
  `cloud <azure|aws|gcp> <account>` (switch active account: gcp writes
  <gcloud>/active_config, azure flips isDefault in azureProfile.json via JSON
  round-trip with BOM preserved, aws validates against config+credentials names
  and persists `aws_profile:` to omnictx's OWN config — never writes `~/.aws`;
  success is silent like the other switches (hook-running shells apply it on
  their next prompt); name/id or `aliases.<provider>.<short>` from omnictx config;
  unknown/ambiguous → exit 2, broken source → exit 1; `list` and `region` are
  reserved words for the aws profile argument. Switches change state, never
  visibility: no switch touches the `enabled`/`kube` display toggles — under a
  persisted mute the state still flips, shown once `on`/`cloud on`/`kube on`
  lifts the mute),
  `cloud aws region [<region>|auto]` (persist an `aws_region:` override —
  offline shape validation `^[a-z]{2}(-[a-z]+)+-\d+$`, invalid → exit 2; `auto`
  removes the key idempotently; bare form prints the effective region:
  AWS_REGION > AWS_DEFAULT_REGION > override > profile config; neither form
  touches the display toggles),
  `aws|azure|gcp [<account>|list]` + `aws region [<region>|auto]` (top-level
  provider words: every argument form dispatches to the exact same code as its
  `cloud <provider> ...` spelling, which stays an accepted alias — no
  deprecation; bare `omnictx <provider>` is the two-branch picker form:
  non-interactive it prints the account the `list` table marks CURRENT
  (nothing configured locally → prints nothing) and NEVER writes — unlike bare
  `cloud <provider>`, which pins, so a piped `omnictx aws` cannot mutate
  state; interactive (same Decide inputs) it feeds the provider's account
  names — the `list` rows — to fzf with the current account in the header, and
  a selection performs the `<provider> <account>` switch incl. the post-switch
  pin; cancel → no write exit 0, fzf exec failure → the print, zero accounts →
  fzf never invoked),
  `hook --shell <bash|zsh>` (the per-prompt invocation emitted by `init`
  snippets; render-mode discipline — never writes, always exit 0. Output is
  exactly three newline-terminated lines: AWS_PROFILE directive, AWS_REGION
  directive, prompt segment. Directive encoding: empty = leave alone, `-` =
  unset, else = export. The pin table lives in Go (aws.Directive): AWS_VAULT
  set → both empty; env value differing from the `__OMNICTX_AWS_PROFILE` /
  `__OMNICTX_AWS_REGION` marker = manual pin, never stomped; hook-owned or
  empty env follows the persisted value, including unset when it was cleared.
  Line 3 is rendered as if the directives were already applied — no one-cycle
  lag. The enabled mute empties ONLY line 3: directives keep flowing, because
  the mute controls display, not state — matching gcp/azure/kube, whose
  switches take effect under the mute too),
  `kube [<context>|list|on|off]` (switch current-context in kubeconfig / print
  current / list all / toggle the kube segment via config key `kube:`; the bare
  form becomes an fzf fuzzy pick when stdout is a TTY, `fzf` is on PATH, and
  `OMNICTX_IGNORE_FZF` is empty (any non-empty value opts out, mirroring
  KUBECTX_IGNORE_FZF): the context names — same dedup as `list`, current context
  moved first so Enter with no query keeps it — go to
  fzf with the current context in the header, a selection performs exactly the
  `kube <context>` switch, cancel writes nothing and exits 0, an fzf exec
  failure degrades to the print; `list` stays the read-only table everywhere;
  `on`
  issued while the global mute is persisted (`enabled: false`) lifts the mute
  exposing ONLY the kube segment: it writes `enabled: true` AND `cloud: none` —
  `off` never touches `enabled`; reserved
  words list|on|off; unknown context → exit 2, unparsable target → exit 1),
  `ns [<name>|list]` (alias `namespace`; switch the namespace of the active
  kube-context in the kubeconfig / print current; name validated as a DNS-1123
  label, invalid → exit 2; no active context / context not defined / broken
  source → exit 1; the bare form runs the same fzf picker as bare `kube` over
  the kubectl-sourced namespace list (the same invocation as `ns list`), with
  the active namespace — `default` when unset — in the header; selection
  performs exactly the `ns <name>` switch, cancel writes nothing; here kubectl
  missing/failing warns on stderr and degrades to the current-namespace print,
  exit 0 — bare `ns` keeps its always-exit-0 contract; `list` execs `kubectl
  get namespaces -o name --request-timeout=10s` — kubectl invocation is
  confined to exactly two call sites, `ns list` and the interactive bare `ns`
  — and prints
  a CURRENT/NAME table marking the active context's namespace (`default` when
  unset); kubectl missing or failing in `list` → stderr passthrough, exit 1,
  no write).
- internal/cloud — Provider interface + active-cloud Select (azure|aws|gcp|auto|none).
- internal/azure — Azure provider: active subscription from azureProfile.json (UTF-8 BOM).
- internal/aws — AWS provider: profile (+region) from ~/.aws/config (offline; no STS).
  Also the switch-side helpers: ValidateProfile (unknown → typed error for exit 2,
  unreadable sources → plain error for exit 1), ValidRegion (shape check),
  EffectiveProfile/EffectiveRegion (env > persisted override > profile config),
  and Directive (the hook's pin table — pure function, exhaustively table-tested).
- internal/gcp — GCP provider: active-config project from ~/.config/gcloud (offline).
- internal/ini — tiny stdlib INI reader shared by aws/gcp (no new dependency).
- internal/fsatomic — the single atomic-write primitive (same-dir temp file +
  fsync + rename, permission bits preserved) behind every write path: the
  azure/gcp/kube switches and omnictx's own config.
- internal/kube — current-context + namespace from kubeconfig ($KUBECONFIG-aware).
  Also the TWO write paths to a foreign file: `kube <context>` rewrites the
  current-context line (parse-before-write, single-line surgery, atomic rename;
  target = first $KUBECONFIG file with current-context, else first), and
  `ns <name>` (WriteNamespace) rewrites the namespace of the active
  context's block (parse-before-write, node-position-guided surgery — replace
  the namespace value preserving inline comments, or insert one as the first
  child of the `context:` mapping; target = first $KUBECONFIG file defining the
  active context; atomic rename). Both writes happen only on explicit user
  command — render mode never writes anything.
- internal/picker — the bare-command fuzzy picker: pure activation decision
  `Decide(stdoutIsTTY, fzfOnPath, ignoreFzf)` (true only for TTY ∧ fzf-on-PATH
  ∧ empty OMNICTX_IGNORE_FZF; exhaustively table-tested like aws.Directive) and
  the thin `Run` fzf shell-out (items on stdin one per line, `--header`, stdout
  captured/trimmed, stderr inherited; non-zero exit = cancel, exec failure =
  the caller degrades to its non-interactive print; empty items never invoke
  fzf). Every bare-command arm passes its items through `CurrentFirst` before
  Run: the active entity moves to the head of the list (rest keeps the
  list-table order, input never mutated) because fzf highlights the first
  stdin line by default — Enter with no query then re-selects the current
  entity instead of whatever the sorted list put first, so a pick without an
  explicit choice never changes state.
  The decision inputs are gathered at the dispatch edge in main.go and
  injected into runKube/runNamespace as a pick function (nil =
  non-interactive), keeping those functions deterministic over parameters.
- internal/render — format, ANSI colors, bash (\[ \]) / zsh (%{ %}) escaping; the
  cloud slot is provider-driven (label from the active provider, color colors["cloud"]
  with optional per-provider colors[key] override).
- internal/config — merge flags + env + YAML config file → struct
  (precedence: flag > env > config > default). Config: ~/.config/omnictx/config.yaml.
  Boolean env vars (OMNICTX_ENABLED / OMNICTX_ICONS / OMNICTX_KUBE) accept on/off
  on top of ParseBool forms. `kube: true|false` (default true) gates the kube
  segment on top of the segments list; OMNICTX_KUBE is the session override.
  OMNICTX_SHELL is the session-scoped counterpart of `--shell` (the flag,
  supplied by `init`, wins by precedence); `shell` is deliberately NOT a config
  file key. `aws_profile:` / `aws_region:` are the machine-written AWS pins
  (config-file only — deliberately no OMNICTX_* env override: AWS's native
  AWS_PROFILE / AWS_REGION are the session override, honored by the hook's pin
  logic instead of the merge).
- internal/shellinit — `init bash|zsh` code generation (go:embed templates).
  Output must be idempotent. No shell functions defined (omnion/omnioff removed).
  The prompt function consumes the three-line `hook` output and applies the
  directives verbatim (export value + exported marker / unset both / skip) via
  quoted parameters — it never evals binary output, and all decisions stay in
  Go. Binary and snippet ship together: after an upgrade users must re-eval
  `init` (rc-file eval makes this automatic on new shells); an rc-file
  `export AWS_PROFILE=...` makes every shell look manually pinned, so global
  switches never reach it — documented, by design.
- testdata — fixtures and golden files.

## Conventions
- Business logic lives in internal/*, tested against fixtures in testdata/.
- Errors reading/parsing sources OR config are NOT propagated as fatal — the segment
  is simply skipped / defaults are used.
- Each data source, config merge, render, and shellinit output is covered by
  table-driven tests.
- When OMNICTX_ENABLED=false, print empty and exit 0.
- Mandatory test cases: UTF-8 BOM in azureProfile.json; $KUBECONFIG merge
  (current-context from the first file); color escaping for bash and zsh;
  config precedence; idempotent init output; AWS profile/region precedence and
  GCP active-config/project precedence; INI parsing (sections/comments/broken →
  empty); cloud Select (explicit pin / auto-by-priority / none); the full
  aws.Directive pin table (vault / manual pin / hook-owned / cleared); hook
  three-line contract incl. disabled and broken-config degradation; snippet
  directive application (export+marker, unset both, manual value untouched);
  the picker Decide table (TTY × fzf × opt-out), the CurrentFirst reorder
  (current first/middle/last/absent, no input mutation — and each interactive
  arm feeds fzf the current entity first) and the interactive bare
  kube/ns arms with an injected pick/fetch (selection reuses the switch path,
  cancel and fzf-error write nothing, kubectl failure warns and degrades with
  exit 0, the non-interactive value keeps the print byte-identical); the
  top-level provider forms (bare print never pins, selection switches AND
  pins like the typed switch, argument forms identical to the `cloud ...`
  spellings, zero accounts never invoke the pick).

## Design decisions (resolved during implementation)
- The `namespace` segment is visually coupled to `kube` and rendered as
  `context:namespace` (icons) / `context/namespace` (ASCII). It has no standalone
  representation: if kube is disabled/unavailable, namespace is not shown.
- Exactly ONE cloud is shown. `cloud: azure|aws|gcp|auto|none` selects it
  (precedence `OMNICTX_CLOUD` > config > default `auto`); `auto` picks the single
  present cloud by priority azure→aws→gcp. Kubernetes is an independent segment,
  unaffected by the cloud selection.
- The `segments` list uses a single `cloud` slot; `azure`/`az`/`aws`/`gcp` are
  accepted aliases for `cloud` (kube aliases k/k8s; namespace alias ns). Unknown
  and duplicate entries are dropped while preserving order.
- A `default` namespace is shown as-is (no special suppression) when the segment is
  enabled and the value is non-empty.
- `init` snippets call the binary via its bare name `omnictx` (must be on PATH),
  matching starship/zoxide/direnv conventions.

## CI
GitHub Actions pinned to node24 majors (checkout@v6, setup-go@v6,
golangci-lint-action@v9 with `version: v2.12`, upload-artifact@v7); no Node-20
deprecation warnings. Job shape: go vet → golangci-lint → go test -race → build
matrix linux/amd64,arm64.

## Definition of Done
Build, `go vet`, `go test ./... -race` and golangci-lint green; edge cases
covered by tests; the prompt never breaks; config + init/toggles work; CI
green; README documents the `eval "$(omnictx init bash)"` install path.
