# omnictx

> *This is my prompt tool. There are many like it, but this one is mine.*

A tiny, fast Go binary that shows your active **cloud** (Azure, AWS, or GCP —
exactly one), **kube-context**, and **namespace** in the shell prompt — and
lets you **switch** them without leaving it.

```
󰠅 prod-subscription ⎈ prod-cluster:payments
```

- **Offline and fast.** Everything reads and writes local config files
  directly, without shelling out to cloud CLIs and without network access.
  The single exception is `ns list`, which queries the cluster via `kubectl`.
- **Never breaks your prompt.** Any error (missing file, broken config, not
  logged in) silently skips the affected segment. Rendering never writes
  anything; writes happen only in explicit commands, which do the opposite —
  validate strictly and fail loudly.
- **One active cloud.** `auto` (default) shows the provider whose local config
  is present, by priority azure → aws → gcp; or pin one explicitly.
- **Careful switching.** Validate first, write atomically, change nothing
  beyond what the switch needs, never touch an unparsable file.

## Install

```bash
make install      # builds and copies the binary to ~/.local/bin/omnictx
```

Make sure `~/.local/bin` is on your `PATH`, then add one line to your shell rc
(idempotent — safe to `eval` more than once):

```bash
eval "$(omnictx init bash)"   # bash — ~/.bashrc
eval "$(omnictx init zsh)"    # zsh  — ~/.zshrc
```

## Usage

```bash
omnictx                       # print the segment (standalone / debugging)
omnictx on|off                # master toggle (persists to the config file)

omnictx aws                   # show the active AWS profile (fzf pick in a terminal)
omnictx aws prod              # switch the AWS profile in every hook-running shell
omnictx aws list              # offline table of local AWS profiles (also: gcp, azure)
omnictx gcp work              # activate a gcloud configuration
omnictx azure prod            # switch the default Azure subscription (name/id/alias)
omnictx aws region eu-central-1   # persist a region override ("auto" clears it)

omnictx cloud                 # show the effective active-cloud selection
omnictx cloud aws             # pin AWS as the displayed cloud (azure|aws|gcp|auto|none)
omnictx cloud on|off          # show/hide just the cloud slot
omnictx cloud aws prod        # the "cloud <provider> ..." spellings still work

omnictx kube                  # show the current kube-context
omnictx kube list             # table of contexts across $KUBECONFIG files
omnictx kube prod-cluster     # switch the current kube-context
omnictx kube on|off           # show/hide the kube segment (namespace follows)

omnictx ns                    # show the active context's namespace
omnictx ns staging            # switch the active context's namespace (alias: namespace)
omnictx ns list               # table of cluster namespaces (needs kubectl)
```

**fzf integration.** With [fzf](https://github.com/junegunn/fzf) installed,
bare `omnictx kube`, `omnictx ns`, and the bare provider commands
(`omnictx aws|azure|gcp`) in a terminal open a fuzzy picker instead of
printing (the prompt already shows the current values): picking an entry
performs the regular switch, Esc cancels without touching anything. Bare `ns`
gets the namespace list from the cluster via kubectl and falls back to the
plain print when kubectl is unavailable. Set `OMNICTX_IGNORE_FZF=1` to opt
out; pipes and scripts always get the plain print, and `kube list` /
`ns list` always print the table.

Switching a cloud account or kube-context/namespace edits the corresponding
local file (kubeconfig, gcloud `active_config`, `azureProfile.json`, and for
AWS omnictx's own config — see below): the target must exist, the write is
atomic, and no other setting is changed.

Switches are **global**, not per-terminal: they flip the same state `kubectl`,
`az` and `gcloud` read — and, for AWS, every hook-running shell — so the
change applies everywhere at once. To keep one terminal on a different
account, export the tool's own env var yourself in that terminal (e.g.
`export AWS_PROFILE=other`) — omnictx respects it and never overwrites a
manual export.

AWS has no persistent "current profile" of its own, so `omnictx aws prod`
persists the choice in omnictx's **own** config file (`~/.aws` is never
written) and every shell running the `init` hook exports `AWS_PROFILE=prod` on
its next prompt — switch once, every terminal follows. Sessions the hook does
not own are left alone: a manual `export AWS_PROFILE=...`, direnv, or an
aws-vault session pins that shell until you unset it. `cloud aws region <r>`
works the same way for `AWS_REGION`, on top of the profile's configured region.

If a switch seems to have no effect, check two things:

- Your rc file exports `AWS_PROFILE` itself. Every shell then starts with a
  manual pin, which omnictx respects — so switches never apply. Remove the
  `export AWS_PROFILE=...` line; `omnictx aws <profile>` replaces it.
- You upgraded omnictx, but the terminal was opened before that. A shell
  keeps the hook snippet it evaluated at startup — restart it (or re-run
  `eval "$(omnictx init zsh)"`) to pick up the new one.

## Configuration

Everything is optional; a missing or broken config falls back to defaults and
never breaks the prompt. Precedence: **flag > `OMNICTX_*` env > config file >
default** (`~/.config/omnictx/config.yaml`):

```yaml
enabled: true
cloud: auto                          # azure | aws | gcp | auto | none
kube: true                           # show the kube segment
segments: [cloud, kube, namespace]   # order matters
icons: true                          # Nerd Font glyphs vs ASCII labels
separator: " "
colors:                              # names or raw SGR codes (e.g. "1;34")
  cloud: blue
  kube: cyan
  namespace: dim
aliases:                             # short names for `omnictx cloud <p> <alias>`
  azure: { prod: "Azure subscription 1" }
  gcp:   { w: work }
```

## Docs

- [Configuration reference](docs/configuration.md) — every key, env var, and
  where the data comes from.
- [Recipes](docs/recipes.md) — toggles, switching clouds, accounts,
  kube-contexts, and namespaces.

For development conventions see [`AGENTS.md`](./AGENTS.md).
