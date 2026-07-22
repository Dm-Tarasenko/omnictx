# Configuration reference

Env vars, the optional YAML config file, and the built-in defaults are merged
with this precedence:

**flag > env var > config file > built-in default**

The only render-mode flag is `--shell` (supplied automatically by
`omnictx init`). Everything else is controlled via `OMNICTX_*` env vars or the
config file.

## Keys

| Config key / command | Env | Default | Purpose |
|---|---|---|---|
| `segments` | `OMNICTX_SEGMENTS` | `cloud,kube,namespace` | which segments, in what order |
| `cloud` / `omnictx cloud <v>` | `OMNICTX_CLOUD` | `auto` | active cloud: `azure\|aws\|gcp\|auto\|none` |
| `kube` / `omnictx kube on\|off` | `OMNICTX_KUBE` | `true` | show the kube segment (namespace follows) |
| `icons` | `OMNICTX_ICONS` | `true` | icons vs ASCII labels |
| `separator` | `OMNICTX_SEPARATOR` | `" "` | separator between segments |
| `enabled` / `omnictx on\|off` | `OMNICTX_ENABLED` | `true` | master on/off |
| `colors` | — | blue/cyan/dim | per-segment colors (config file only) |
| `aliases` | — | — | short names for `omnictx cloud <p> <alias>` (config file only) |
| `--shell bash\|zsh\|none` (flag) | `OMNICTX_SHELL` | `none` | color escaping mode |
| — | `OMNICTX_CONFIG` | `~/.config/omnictx/config.yaml` | config file path |

Boolean env vars (`OMNICTX_ENABLED`, `OMNICTX_ICONS`, `OMNICTX_KUBE`) accept
`on`/`off` on top of the usual `true`/`false` forms.

`enabled: false` (the master mute) hides everything regardless of the other
keys. The segment-level `on` commands — `omnictx kube on` and `omnictx cloud
on` — also persist `enabled: true`, so turning a segment on after `omnictx off`
makes it actually appear. Their `off` counterparts (and plain `cloud auto`)
never touch `enabled`.

Segment names accept aliases: `azure`/`az`/`aws`/`gcp` → `cloud`, `k`/`k8s` →
`kube`, `ns` → `namespace`. Unknown and duplicate entries are dropped while
preserving order. The concrete cloud provider is chosen by `cloud:`, not by
the segment name.

> `shell` is intentionally **not** a config key — it is supplied per-shell by
> `omnictx init`, not persisted.

## Config file (`~/.config/omnictx/config.yaml`)

All keys are optional; a missing or broken config falls back to defaults and
never breaks the prompt.

```yaml
enabled: true
cloud: auto                          # azure | aws | gcp | auto | none
kube: true                           # show the kube segment (omnictx kube on/off)
segments: [cloud, kube, namespace]   # order matters
icons: true
separator: " "
colors:                              # names or raw SGR codes (e.g. "1;34")
  cloud: blue                        # optional per-provider overrides: azure/aws/gcp
  kube: cyan
  namespace: dim
aliases:                             # short names for `omnictx cloud <p> <alias>`
  azure:
    prod: "Azure subscription 1"     # value = subscription name or id
  gcp:
    w: work                          # value = gcloud configuration name
```

## Output format

- Icons (default): `<glyph> <cloud> ⎈ <context>:<namespace>`, with a
  per-provider Nerd Font glyph (`󰠅 ` Azure, ` ` AWS, `󱇶 ` GCP).
- ASCII (`icons: false` / `OMNICTX_ICONS=false`): `az:`/`aws:`/`gcp:` `<cloud>`
  `k8s:<context>/<namespace>`.

The cloud value is provider-specific: Azure subscription, AWS
`profile[/region]`, or GCP project. The `namespace` is visually coupled to
`kube` (`context:namespace`) and has no standalone representation. If a
segment's data is unavailable, it is skipped entirely — no empty placeholders.

## Colors and shell escaping

ANSI color codes in a prompt must be wrapped in non-printing markers, or the
shell miscalculates line width and breaks line editing. `--shell` controls
this:

| `--shell` | wrapping |
|---|---|
| `bash` | `\[ <ansi> \]` |
| `zsh`  | `%{ <ansi> %}` |
| `none` (default) | raw ANSI (for pipes / standalone) |

`omnictx init` passes the correct `--shell` value automatically.

## Data sources

- **Kubernetes:** `$KUBECONFIG` (colon-separated) or `~/.kube/config`. The
  `current-context` is taken from the first file that sets it; the namespace is
  looked up by matching that context across all files.
- **Azure:** `$AZURE_CONFIG_DIR/azureProfile.json` or
  `~/.azure/azureProfile.json`. The leading UTF-8 BOM is stripped before
  parsing; the subscription with `isDefault: true` is used.
- **AWS:** profile = `AWS_PROFILE` > `AWS_VAULT` > `default`; region =
  `AWS_REGION` > `AWS_DEFAULT_REGION` > the profile's `region` in
  `~/.aws/config` (`AWS_CONFIG_FILE` overrides the path; non-default profiles
  are `[profile NAME]`). Shows `profile[/region]`. Account-id is out of scope
  (needs STS/network).
- **GCP:** active config = `CLOUDSDK_ACTIVE_CONFIG_NAME` >
  `<gcloud>/active_config` (`<gcloud>` = `CLOUDSDK_CONFIG` or
  `~/.config/gcloud`); project = `CLOUDSDK_CORE_PROJECT` >
  `GOOGLE_CLOUD_PROJECT` > `[core] project` in
  `<gcloud>/configurations/config_<name>`. Shows the project.
