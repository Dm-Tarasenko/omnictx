# Recipes

How-to scenarios for daily use. For the full key/env reference see
[configuration.md](./configuration.md).

## Toggles — hide/show segments without rc edits

```bash
omnictx off        # persist enabled: false — all future shells start quiet
omnictx on         # persist enabled: true  — restore the segment
omnictx cloud off  # hide just the cloud slot (alias for cloud none)
omnictx cloud on   # show it again (alias for cloud auto — re-pin if you had one)
omnictx kube off   # hide just the kube segment (namespace goes with it)
omnictx kube on    # show it again
```

The commands persist to the config file, so the change survives new shells.
For the **current session only**, use env vars — they override the config
until unset, and all boolean ones accept `on`/`off` as well as `true`/`false`:

| Segment | Persistent (command) | Session-only (env) |
|---|---|---|
| everything | `omnictx on/off` | `export OMNICTX_ENABLED=off` |
| cloud slot | `omnictx cloud on/off` | `export OMNICTX_CLOUD=none` / `unset` |
| kube (+namespace) | `omnictx kube on/off` | `export OMNICTX_KUBE=off` |

## Switching the displayed cloud

```bash
omnictx cloud aws    # persist: all future prompts show AWS
omnictx cloud auto   # persist: back to auto-detect
omnictx cloud        # print the effective selection (env > config > default)
```

The value is written to the `cloud:` key of the config file (comments and
other keys are preserved). For a session-only override use
`export OMNICTX_CLOUD=<v>`. An invalid value is rejected with a usage error
(exit 2) — nothing is written.

To see what there is to pin, list a provider's local accounts — offline, from
the same files omnictx already reads (AWS profiles from `~/.aws/config` +
`~/.aws/credentials` names, gcloud configurations, Azure subscriptions):

```
$ omnictx cloud aws list      # or: gcp list / azure list / bare "cloud list"
CURRENT   NAME      REGION
*         default   us-east-1
          prod      eu-west-1
```

## Switching cloud accounts

Azure and GCP keep their active account in local files, so omnictx can switch
them the same careful way it switches kube-contexts (validate first, atomic
write, never touch an unparsable file):

```bash
omnictx cloud gcp work               # writes <gcloud>/active_config
omnictx cloud azure "My Sub"         # flips isDefault in azureProfile.json
omnictx cloud azure 11111111-2222-3333-4444-555555555555 # by id — for name collisions
```

A successful switch also pins that provider as the displayed cloud (persists
`cloud: <provider>`), so the prompt immediately shows what you just switched
to.

Short aliases live in the omnictx config file and are checked first:

```yaml
aliases:
  azure: { prod: "Azure subscription 1" }   # values may be names or ids
  gcp:   { w: work }
```

AWS is the honest exception: the ecosystem has no persistent "current profile"
(it is the session-scoped `AWS_PROFILE`), so `omnictx cloud aws prod` just
prints the correct command — `export AWS_PROFILE=prod` — and exits non-zero.

## Switching the kube-context

```bash
omnictx kube prod-cluster # switch (rewrites current-context in kubeconfig)
omnictx kube              # print the current context
omnictx kube list         # all contexts across $KUBECONFIG files:
```
```
CURRENT   NAME     CLUSTER   AUTHINFO      NAMESPACE
          kind-1   kind-1    kind-1-user   payments
*         kind-2   kind-2    kind-2-user   staging
```

The switch is deliberately careful: the target context must exist (otherwise a
usage error and exit 2, nothing written), only the `current-context:` line
changes — comments and formatting are preserved byte-for-byte — and the write
is atomic (temp file + rename, permissions kept). An unreadable or unparsable
kubeconfig is never touched. With a multi-file `$KUBECONFIG`, the file that
already sets `current-context` is updated (else the first file), matching
kubectl. `list`, `on`, and `off` are reserved words.

## Switching the namespace

```bash
omnictx ns staging # set the active context's namespace (rewrites kubeconfig)
omnictx ns         # print the active context's namespace
omnictx ns list    # list the cluster's namespaces (needs kubectl, see below)
```

`namespace` is an accepted alias for `ns` (`omnictx namespace staging`). This
switches the namespace of the **active** context (the one matching
`current-context`) with the same care as the context switch: the name must be
a valid Kubernetes namespace (a DNS-1123 label — otherwise a usage error and
exit 2, nothing written); only that context's `namespace:` changes — an
existing value is replaced in place (inline comments preserved) or a
`namespace:` line is inserted into its `context:` block, everything else
byte-for-byte; and the write is atomic (temp file + rename, permissions kept).
If there is no active context, the context is not defined in any file, or the
kubeconfig is unreadable, it fails loudly (exit 1, nothing written).

`ns list` is the **only online command** in the binary: the namespace list
lives in the cluster, so it execs
`kubectl get namespaces -o name --request-timeout=10s` and prints a
CURRENT/NAME table marking the active context's namespace (`default` when the
context sets none):

```
$ omnictx ns list
CURRENT   NAME
          kube-system
*         staging
          payments
```

If kubectl is missing from `PATH` or fails (no cluster, dead VPN — the
request times out in seconds), its stderr passes through and omnictx exits 1.
Nothing is ever written by `ns list`. Render mode never reaches this code and
stays strictly offline.

## Shell integration internals

Curious what `eval "$(omnictx init <shell>)"` actually runs? Just print it:
`omnictx init bash` (or `zsh`) — it is a dozen lines of plain shell: a prompt
hook that prepends the segment to your existing prompt, guarded so it is
idempotent and leaves the prompt untouched when the segment is empty.
