## 1. CLI dispatch (`cmd/omnictx/main.go`)

- [x] 1.1 Update `cloudUsage` (line ~271-273) to drop the `use` line and show `omnictx cloud <azure|gcp> <account>` instead.
- [x] 1.2 In `runCloud`, change the `len(args) > 3` guard to `len(args) > 2` (three-or-more arguments is now always a usage error, exit 2).
- [x] 1.3 Remove the `len(args) == 3` branch that calls `runCloudUse`.
- [x] 1.4 In the `len(args) == 2` branch, keep the existing `form == "list"` case as-is, and add a new case: when `provider` is `azure`, `aws`, or `gcp` and `form != "list"`, treat `args[1]` as the account and dispatch to the switch/hint logic (still exit 2 usage error for any other first argument, e.g. `auto`/`none`/`on`/`off`/unknown).
- [x] 1.5 Rename `runCloudUse(args []string, home string, _, stderr io.Writer) int` to drop the `use`-verb check: it now takes `(provider, account string, home string, stderr io.Writer) int` (or equivalent), called with `args[0]` and `args[1]` directly — no `verb != "use"` check, no `args[2]`.
- [x] 1.6 Update the doc comment above the renamed function (currently `runCloudUse handles \`omnictx cloud <provider> use <account>\`...`) to describe the new two-argument form.
- [x] 1.7 Update the AWS hint message/branch to reflect `omnictx cloud aws <profile>` (message text itself already only prints `export AWS_PROFILE=<account>`, no wording change needed there — just confirm it's still reached from the new 2-arg dispatch).

## 2. Help text

- [x] 2.1 Update the `cloud <azure|gcp> use <account>` line in `printUsage` (~line 147-152) to `cloud <azure|gcp> <account>`, and adjust the AWS mention (~line 152) if it still reads naturally without `use`.

## 3. Tests

- [x] 3.1 Update/rename existing table-driven tests in `cmd/omnictx` (or wherever `runCloud`/`runCloudUse` are tested) that invoke `cloud <provider> use <account>` to the new `cloud <provider> <account>` form.
- [x] 3.2 Add a test for the three-argument case now being a hard usage error (e.g. `cloud gcp use work` — where `use` is just an unresolvable account name plus a trailing extra arg — exits 2).
- [x] 3.3 Add a test for `cloud auto work` / `cloud none work` (non-provider first argument in 2-arg form) exiting 2.
- [x] 3.4 Add a test that a gcp configuration or azure subscription literally named `use` can still be activated via `cloud gcp use` / `cloud azure use` (two-argument form, `use` as the account).
- [x] 3.5 Confirm alias resolution (`aliases.<provider>.<short>`), success pinning (`pinCloudAfterUse`), unknown/ambiguous account exit codes, and the AWS hint all still pass under the new call shape (rename call sites only, behavior unchanged).
- [x] 3.6 Run `make test` (`go test ./... -race -count=1`) and `make lint`.

## 4. Docs

- [x] 4.1 Update README examples/snippets that show `omnictx cloud <provider> use <account>` to the new syntax.
- [x] 4.2 Update `AGENTS.md`'s description of the `cloud` subcommand (the `cloud <azure|gcp> use <account>` bullet in the Commands section) to match the new syntax.

## 5. Verify

- [x] 5.1 Build the binary (`make build`) and manually run `omnictx cloud gcp <name>` / `omnictx cloud azure <name>` / `omnictx cloud aws <profile>` against local fixtures to confirm end-to-end behavior matches the updated spec.
- [x] 5.2 Run `omnictx --help` and confirm the printed usage no longer mentions `use`.
