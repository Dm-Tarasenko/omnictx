# promote-provider-commands — design

## Context

Today every provider operation lives under `cloud`: `cloud <provider> list`,
`cloud <provider> <account>`, `cloud aws region ...` (main.go runCloud →
printCloudList / runCloudSwitch / runAwsRegion). The words `aws`, `azure`,
`gcp` are NOT subcommands: `omnictx aws` falls through to render mode, which
parses the word as a harmless positional and prints the prompt segment — an
accident of the dispatch, not a contract. add-fzf-picker shipped
`internal/picker` (pure `Decide` + thin `Run` fzf shell-out) and the
two-branch bare-command pattern for `kube`/`ns`; this change promotes the
provider words to the top level and gives the bare forms the same picker.
Decisions below were captured with the user at proposal time.

## Goals / Non-Goals

**Goals:**
- `omnictx <azure|aws|gcp> <account>`, `... list`, and `omnictx aws region
  [<region>|auto]` at the top level, semantics identical to the `cloud ...`
  forms (same functions, same exit codes, same warnings).
- Bare `omnictx <provider>` with the add-fzf-picker two-branch pattern:
  interactive → fzf over the provider's local accounts, selection performs
  the existing account switch; non-interactive → print the active account.
- Old `cloud <provider> ...` forms stay as aliases dispatching to the same
  code — scripts and muscle memory unaffected.
- `cloud` keeps the slot-level meta operations unchanged (bare print,
  `auto|none|on|off`, explicit `cloud <provider>` pin).

**Non-Goals:**
- No new picker machinery — `internal/picker` is reused as-is.
- No deprecation of the `cloud <provider> ...` forms (user decision: keep
  indefinitely, not "for a transition period").
- No change to render mode, the hook, or any switch/write semantics.
- No `omnictx <provider> on|off` visibility forms — visibility stays on
  `cloud` only.

## Decisions

**D1 — Top-level dispatch to one shared `runProvider`.** main() gains
`case "aws", "azure", "gcp"` routing to
`runProvider(provider, rest, stdout, stderr, interactivePicker())`. The
words collide with no existing subcommand (`init`, `on`, `off`, `cloud`,
`kube`, `ns`, `namespace`, `hook`). Consequence, deliberate: these words no
longer fall through to render mode — `omnictx aws` in a pipe now prints the
active profile instead of the prompt segment. The old fall-through printed
segment output for ANY junk argument; nothing could reasonably depend on it
for exactly these three words, and the proposal's whole point is to give
them a real meaning.

**D2 — Argument forms delegate verbatim.** `omnictx <provider> list` calls
the same azure-check-warning + printCloudList path as `cloud <provider>
list`; `omnictx aws region ...` calls runAwsRegion; `omnictx <provider>
<account>` calls runCloudSwitch (alias resolution, validation, switch,
post-switch `cloud: <provider>` pin — all identical, including exit codes).
The reserved words are the same: `list` for every provider, plus `region`
for aws. More than the accepted argument count → provider usage message,
exit 2. runCloud is not modified: the old forms already call these
functions, which is what makes them free aliases.

**D3 — Bare `omnictx <provider>` never pins; the two branches mirror bare
`kube`.** Non-interactive branch: print the provider's active account — the
same value the `list` table marks CURRENT (aws: `AWS_PROFILE` > `AWS_VAULT`
> `default`; gcp: `CLOUDSDK_ACTIVE_CONFIG_NAME` > `active_config` >
`default`; azure: the `isDefault` subscription's name) — newline-terminated,
exit 0; nothing resolvable → print nothing, exit 0. This is a read-only
path: unlike bare `cloud <provider>` (which PINS the slot), a piped
`omnictx aws` cannot mutate state. Interactive branch (same
`picker.Decide` inputs gathered at the dispatch edge): items are the same
account names the `list` table shows, in the same order; header names the
current account; a selection performs exactly the `<provider> <account>`
switch — which pins, because the user explicitly switched, same as the typed
form. Cancel → nothing written, exit 0. fzf exec failure → fall back to the
print, exit 0. Zero accounts → print nothing, fzf never invoked.

**D4 — Picker items are bare account names.** aws: profile names; gcp:
configuration names; azure: subscription NAMES (ids stay out of the list —
names are what humans pick). An azure selection whose name is ambiguous
(duplicate names) flows into the existing azure.Use ambiguity error: exit 2
listing the candidates with ids — same as typing the name, no special
handling. The selection re-enters the switch path through alias resolution
like any typed account; alias keys are not offered as picker items.

**D5 — Reuse, don't extend, `internal/picker`.** `Decide` and `Run` are
provider-agnostic already. The pick function is injected into runProvider
exactly like runKubeWith/runNamespaceWith (nil = non-interactive), so tests
drive the interactive arm with a recorded fake and existing entry points
stay deterministic over parameters.

**D6 — Help shows the top-level forms as primary.** Usage lists
`aws|azure|gcp [<account>|list]` and `aws region [<region>|auto]` as the
provider commands; the `cloud` entry keeps the slot-meta forms and notes
that `cloud <provider> <account>` / `cloud <provider> list` remain accepted
aliases. AGENTS.md and README follow the same primary/alias framing.

## Risks / Trade-offs

- [A piped `omnictx aws` used to emit the rendered segment; now it prints
  the active profile] → The fall-through was accidental, the segment is
  reachable as plain `omnictx`, and the new output is stable and greppable —
  strictly more useful to scripts.
- [Bare `omnictx <provider>` in a terminal now opens fzf instead of
  printing] → Same mitigation set as add-fzf-picker: the prompt already
  shows the active account, Esc is instant, `OMNICTX_IGNORE_FZF=1` or a pipe
  restores the print; scripts are non-TTY by construction.
- [Azure duplicate names make the picker selection fail with exit 2] →
  Identical to typing the same name today; the error lists ids. Rare enough
  (tenant-level stubs) not to warrant id-in-item complexity now.
- [Three more top-level words shrink the free namespace for future
  subcommands] → They are the product's core nouns; if anything deserves
  top-level words, it is these.

## Open Questions

_None — surface shape (promotion + aliases kept + bare-form no-pin split)
was decided with the user at proposal time._
