## 1. Implementation

- [x] 1.1 Replace the reserved-word rejection in `runNamespace` (cmd/omnictx/main.go) with a `list` branch: exec `kubectl get namespaces -o name --request-timeout=10s`, parse `namespace/<name>` lines (skip non-matching), and print a CURRENT/NAME table via `printTable`, marking the active context's namespace (or `default` when unset) from `kube.Read`
- [x] 1.2 Handle failures: kubectl not on PATH → stderr message naming kubectl, exit 1; kubectl non-zero exit → pass through its stderr with an omnictx error line, exit 1; never write any file on the `list` path
- [x] 1.3 Update usage string (`usage: omnictx ns [<name>|list]`), the `--help` subcommand block, and the `runNamespace` doc comment to describe the online listing and the render-stays-offline boundary

## 2. Tests

- [x] 2.1 Replace `TestRunNamespaceListReserved` in cmd/omnictx/main_test.go with table-driven `ns list` tests using a fake `kubectl` executable in a temp dir prepended to PATH: success (marks current namespace), no-namespace-set (marks `default`), kubectl exits non-zero (exit 1, stderr passthrough), kubectl missing (empty PATH dir → exit 1), and no-write assertions on the kubeconfig fixture
- [x] 2.2 Keep/verify the `off is an ordinary namespace name` case and the `namespace` alias equivalence for `list`

## 3. Docs

- [x] 3.1 Update AGENTS.md: reword the offline principle (render mode strictly offline; explicit subcommands may shell out to kubectl — `ns list` is the only such path), update the `ns` subcommand description, keep the no-client-go/no-network-libraries dependency rule
- [x] 3.2 Update the main spec's Purpose paragraph (openspec/specs/kube-namespace-cli/spec.md) when syncing/archiving so it no longer claims there is no namespace listing

## 4. Verification

- [x] 4.1 Run `make build`, `make test`, `make lint`; manually exercise `omnictx ns list` against a fake kubectl on PATH to confirm table output and exit codes
