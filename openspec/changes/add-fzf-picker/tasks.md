# add-fzf-picker — tasks

## 1. internal/picker: pure decision + fzf wrapper

- [x] 1.1 Create `internal/picker` with `Decide(stdoutIsTTY, fzfOnPath bool, ignoreFzf string) bool` — true only for TTY ∧ fzf-present ∧ empty ignore value; doc comment states the OMNICTX_IGNORE_FZF semantics (any non-empty value disables, mirroring KUBECTX_IGNORE_FZF)
- [x] 1.2 Table-test `Decide` exhaustively over all 2×2×{empty, non-empty} input combinations (the aws.Directive pattern)
- [x] 1.3 Add `Run(items []string, header string) (selection string, ok bool, err error)` — execs `fzf --header <header>` with items joined by newlines on stdin, stderr inherited, stdout captured and trimmed; non-zero fzf exit → `ok=false, err=nil` (cancel); exec failure → `err != nil`; empty items → `ok=false` without invoking fzf

## 2. Wire the picker into bare `kube`

- [x] 2.1 In `cmd/omnictx/main.go`, compute the activation decision at the dispatch edge (existing `isTTY(os.Stdout)`, `exec.LookPath("fzf")`, `os.Getenv("OMNICTX_IGNORE_FZF")`) and pass an interactive flag (or injected pick function) into `runKube` — existing tests must keep compiling with the non-interactive value; the `list` arm is NOT touched
- [x] 2.2 In `runKube`'s no-argument arm: interactive → feed bare context names (same dedup/order as `kube list`) to `picker.Run` with the current context in the header; on selection reuse the exact `kube <context>` switch path (found-check + `kube.WriteContext`, silent success, broken target → exit 1); cancel → exit 0, no write; `Run` error → fall back to the current-context print, exit 0; zero contexts → print nothing, fzf never invoked
- [x] 2.3 Table-test the new arm with an injected pick result: selection switches current-context in the fixture kubeconfig; cancel leaves it byte-identical; fzf-error falls back to the print; non-interactive flag keeps today's print byte-identical; selection under persisted `enabled: false` still switches and leaves `enabled`/`kube` keys untouched

## 3. Wire the picker into bare `ns`

- [x] 3.1 Thread the same interactive flag into `runNamespace`; the `list` arm and `runNamespaceList` are NOT touched (still loud exit 1 on kubectl problems)
- [x] 3.2 In `runNamespace`'s no-argument arm: interactive → fetch names via the same kubectl invocation as `ns list` (extract the fetch into a shared helper); kubectl missing/failing → stderr warning + fall back to the current-namespace print, exit 0; success → `picker.Run` with the active namespace (or `default`) in the header; on selection reuse the `ns <name>` write path (`kube.WriteNamespace`, silent success); cancel → exit 0, no write; `Run` error → print, exit 0
- [x] 3.3 Table-test with an injected pick result and injected namespace-fetch: selection rewrites the active context's namespace in the fixture kubeconfig; cancel and fzf-error leave it untouched; kubectl-failure warns and prints current with exit 0; non-interactive flag keeps today's offline print (kubectl never invoked)

## 4. Docs and gate

- [x] 4.1 Update AGENTS.md: bare `kube` / `ns` descriptions gain the interactive branch (activation condition, selection = the existing switch, cancel = no-op, OMNICTX_IGNORE_FZF opt-out, bare-`ns` kubectl fallback); replace the "ns list is the ONLY online path" sentence with the two-call-site wording; note fzf as an optional external binary alongside the kubectl precedent
- [x] 4.2 Update README: short "fzf integration" note — install fzf to get fuzzy picking on bare `kube`/`ns`, `OMNICTX_IGNORE_FZF=1` to opt out, `list` always prints the table
- [ ] 4.3 Full gate: `make build`, `go vet ./...`, `make test` (race), `make lint` green; manually smoke-test in a terminal: pick, Esc-cancel, `OMNICTX_IGNORE_FZF=1`, piped `omnictx kube`, `kube list` still a table
