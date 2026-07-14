## 1. Reserve `list` in runNamespace

- [x] 1.1 In `cmd/omnictx/main.go` `runNamespace`, reject `list` before validation and any kubeconfig access: print the reserved-word error (offline, cannot list cluster namespaces, `try: kubectl get namespaces`) plus the usage line to stderr, write nothing, return 2
- [x] 1.2 Update the `runNamespace` doc comment and the `ns` section of the `--help` text: drop the "`ns list` sets the namespace to the literal `list`" wording, document `list` as reserved

## 2. Tests

- [x] 2.1 In `cmd/omnictx/main_test.go`, flip the existing `ns list` case: assert exit 2, stderr mentions the reserved word / kubectl hint, and the kubeconfig is byte-identical after the call (note: no old test asserted the write — added `TestRunNamespaceListReserved` instead)
- [x] 2.2 Add cases: `namespace list` alias behaves identically; `ns list` with no readable kubeconfig still exits 2 (reserved check wins over the exit-1 source path); `ns off` still writes namespace `off` and exits 0 (alias covered by shared dispatch `case "ns", "namespace":` — documented in the test comment)

## 3. Verify

- [x] 3.1 Run `make build`, `make test`, `make lint`; confirm all green
- [x] 3.2 Smoke-check by hand against a scratch kubeconfig: `omnictx ns list` errors with exit 2 and leaves the file untouched; `omnictx ns <real-name>` still switches
