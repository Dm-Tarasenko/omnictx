# promote-provider-commands — tasks

## 1. Top-level provider dispatch + bare-form picker

- [x] 1.1 Add account helpers in `cmd/omnictx/main.go`: `providerCurrentAccount(provider, home) string` (the value the `list` table marks CURRENT: aws `aws.CurrentProfile`, gcp `gcp.CurrentConfiguration`, azure the `isDefault` subscription's name) and `providerAccountNames(provider, home) []string` (the `list` rows' names in table order)
- [x] 1.2 Add `runProvider(provider string, args []string, stdout, stderr io.Writer, pick pickFunc) int`: bare → print-or-pick arm; `list` → the same azure-check + `printCloudList` path as `cloud <provider> list`; aws `region ...` → `runAwsRegion`; one other argument → `runCloudSwitch`; more arguments → provider usage, exit 2. Wire `case "aws", "azure", "gcp"` in `main()` with `interactivePicker()`; `runCloud` is NOT modified (old spellings already dispatch to the same functions)
- [x] 1.3 Bare arm: non-interactive → print `providerCurrentAccount` (empty → nothing), exit 0, never writes/pins; interactive → zero accounts guard (fzf never invoked), `picker`-style pick over `providerAccountNames` with the current account in the header, selection → `runCloudSwitch` (switch + pin, silent success, same exit codes), cancel → exit 0 no write, pick error → fall back to the print
- [x] 1.4 Table-test with fixtures + injected pick: bare print per provider (aws env, gcp active_config, azure isDefault; empty → quiet); bare form never writes the config (no pin); selection switches AND pins like the typed form; cancel leaves everything byte-identical; fzf-error falls back to the print; zero accounts never invoke the pick; top-level `list` / `<account>` / `aws region` / too-many-args outputs and exit codes equal the `cloud ...` spellings

## 2. Help and docs

- [x] 2.1 Update `printUsage`: top-level `aws|azure|gcp [<account>|list]` and `aws region [<region>|auto]` as the primary provider forms; `cloud` entry keeps slot meta and notes the `cloud <provider> ...` aliases; extend the usage tests
- [x] 2.2 Update AGENTS.md: subcommand list gains the top-level provider words (two-branch bare form, no-pin split from `cloud <provider>`, alias spellings kept); mandatory test cases gain the provider picker arm
- [x] 2.3 Update README: usage block shows the short forms first (`omnictx aws prod`, `omnictx gcp list`, `omnictx aws region ...`), notes bare `omnictx <provider>` prints/picks the active account and that `cloud <provider> ...` still works

## 3. Gate

- [x] 3.1 Full gate: `make build`, `go vet ./...`, `make test` (race), `make lint` green; manual smoke-test in a terminal: bare `omnictx aws|azure|gcp` pick + Esc, piped bare form prints without pinning, `omnictx aws region`, old `cloud ...` spellings unchanged
