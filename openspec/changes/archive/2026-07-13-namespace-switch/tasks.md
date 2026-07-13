## 1. kube package: namespace write path

- [x] 1.1 Add a DNS-1123 label validator helper (`kube.ValidNamespace`, regex `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`, length ≤ 63) in `internal/kube/write.go`.
- [x] 1.2 Add `namespaceWriteTarget(files []string, ctx string) (string, bool)` in `internal/kube/write.go`: first file whose `contexts[]` has an entry named `ctx`.
- [x] 1.3 Implement the node-position-guided locator (`setNamespaceLines`): parse to `yaml.Node`, find the matching `contexts[]` item and its `context:` mapping, then replace the `namespace:` value line (preserving inline comments) or insert one as the first child. Refuse (`ErrUnlocatable`) on flow-style / empty-map blocks.
- [x] 1.4 Implement `WriteNamespace(lookup LookupEnv, home, namespace string) error`: resolve active context via `Read`; pick target via 1.2; parse-before-write; apply locator (1.3) to set-or-insert `namespace:`; reuse `atomicWrite`. Sentinel errors `ErrNoActiveContext`, `ErrContextNotFound`, `ErrUnlocatable` (all → exit 1).

## 2. CLI wiring

- [x] 2.1 Add `case "ns", "namespace":` to the subcommand dispatch in `cmd/omnictx/main.go` and a `namespaceUsage` constant (`ns` primary, `namespace` alias).
- [x] 2.2 Implement `runNamespace(args, stdout, stderr)`: no arg → print `Read().Namespace` (empty prints nothing), exit 0; one arg → validate name (invalid → stderr + exit 2), then `WriteNamespace` (errors → exit 1), exit 0; ≥2 args → usage error exit 2.
- [x] 2.3 Add the `ns [<name>]` entry (noting the `namespace` alias) to the grouped `--help` usage output, matching the existing `kube` block style.

## 3. Tests (table-driven)

- [x] 3.1 Fixtures (inline consts + `writeTemp`, matching the existing surgery-test pattern): active-context namespace present (`kindConfig`, with inline comment); context with no namespace key (`nsInsertConfig`); context defined in a second `$KUBECONFIG` file; multiple contexts each with a namespace; broken YAML target.
- [x] 3.2 `WriteNamespace` tests: replace-existing keeps every other byte and preserves the inline comment; insert-when-absent aligns to sibling indent and preserves rest; only the active context changes; second-file target updated while the current-context file is byte-identical; permissions preserved.
- [x] 3.3 Error-path tests: no `current-context` → `ErrNoActiveContext`, no write; active context not in any file → `ErrContextNotFound`, no write; broken sole kubeconfig → error, byte-identical.
- [x] 3.4 Name-validation tests (`TestValidNamespace`): valid labels accepted; invalid (`Bad_NS`, uppercase, leading/trailing `-`, spaces, dots, empty, >63 chars) rejected.
- [x] 3.5 CLI tests in `cmd/omnictx/main_test.go`: `ns` (get) prints current / prints nothing when unset; `ns <name>` writes and next `Read` reflects it; `ns a b` → exit 2; `ns Bad_NS` → exit 2; no-active-context → exit 1; usage lists `ns [<name>]` + alias.
- [x] 3.6 Confirm render mode still never writes (existing invariant tests unaffected — full suite green with `-race`).

## 4. Docs and verification

- [x] 4.1 Update AGENTS.md: add `ns [<name>]` (alias `namespace`) to the Commands subcommand list and note the second `internal/kube` write path (`WriteNamespace`).
- [x] 4.2 Update README with the `omnictx ns` usage and section (alias `namespace`).
- [x] 4.3 Run `make test` (`go test ./... -race -count=1`) and `make lint` — both green; render golden unchanged.
- [x] 4.4 Manual check: built the binary, pointed `$KUBECONFIG` at fixtures, verified replace (comment kept), insert (aligned, other contexts untouched), `namespace` alias, invalid name → exit 2, too-many-args → exit 2, no-active-context → exit 1, and that `ns` reflects the change.
