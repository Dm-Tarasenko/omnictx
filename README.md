# omnictx

> *This is my prompt tool. There are many like it, but this one is mine.*

A tiny, fast Go binary that shows your active **cloud** (Azure, AWS, or GCP —
exactly one), **kube-context**, and **namespace** in the shell prompt — and
lets you **switch** them without leaving it.

```
󰠅 prod-subscription ⎈ prod-cluster:payments
```

- **Offline and fast.** Rendering and switching work on local config files
  directly — no `kubectl`/`az`/`aws`/`gcloud`, no network — so the prompt
  segment fits comfortably inside the prompt budget (cold start + render
  < 10 ms). The single exception is `ns list`, which queries the cluster
  via `kubectl`.
- **Never breaks your prompt.** Any error (missing file, broken config, not
  logged in) silently skips the affected segment. Rendering never writes
  anything; writes happen only in explicit commands, which do the opposite —
  validate strictly and fail loudly.
- **One active cloud.** `auto` (default) shows the provider whose local config
  is present, by priority azure → aws → gcp; or pin one explicitly.
- **Careful switching.** Validate first, write atomically, preserve comments
  and formatting byte-for-byte, never touch an unparsable file.

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

omnictx cloud                 # show the effective active-cloud selection
omnictx cloud aws             # pin AWS as the displayed cloud (azure|aws|gcp|auto|none)
omnictx cloud on|off          # show/hide just the cloud slot
omnictx cloud aws list        # offline table of local AWS profiles (also: gcp, azure)
omnictx cloud gcp work        # activate a gcloud configuration
omnictx cloud azure prod      # switch the default Azure subscription (name/id/alias)

omnictx kube                  # show the current kube-context
omnictx kube list             # table of contexts across $KUBECONFIG files
omnictx kube prod-cluster     # switch the current kube-context
omnictx kube on|off           # show/hide the kube segment (namespace follows)

omnictx ns                    # show the active context's namespace
omnictx ns staging            # switch the active context's namespace (alias: namespace)
omnictx ns list               # table of cluster namespaces (the one command that needs kubectl)
```

Switching a cloud account or kube-context/namespace edits the corresponding
local file (kubeconfig, gcloud `active_config`, `azureProfile.json`) the same
careful way: target must exist, single-line surgery, atomic write. AWS is the
honest exception — it has no persistent "current profile", so
`omnictx cloud aws prod` prints `export AWS_PROFILE=prod` for you to run.

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

For development conventions see [`AGENTS.md`](./AGENTS.md); the full product
requirements live in [`PRD.md`](./PRD.md).
