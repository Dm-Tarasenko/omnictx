# aws-switch — tasks

## 1. Config keys

- [x] 1.1 Add `aws_profile` / `aws_region` to the config struct and YAML merge in internal/config (no env override — AWS-native env vars fill that role); table-driven precedence tests
- [x] 1.2 Support key removal in the single-key write path (needed by `region auto`: delete the `aws_region:` line, preserve everything else); tests incl. idempotent removal

## 2. AWS profile switch

- [x] 2.1 internal/aws: validation helper on top of `Profiles()` (known name → ok; unknown → typed error for exit 2; unreadable sources → typed error for exit 1); tests with fixtures
- [x] 2.2 cmd/omnictx: replace the `case "aws"` hint branch — alias resolution (already generic), validate, write `aws_profile:`, `pinCloudAfterUse("aws")`, stderr note about hook shells; delete the export-hint; reserve `region`; main_test coverage for all exit codes and the no-`~/.aws`-write guarantee

## 3. AWS region subcommand

- [x] 3.1 internal/aws: region format validation (`^[a-z]{2}(-[a-z]+)+-\d+$`) and effective-region resolution (env > override > profile config) reusing `resolveRegion`; tests
- [x] 3.2 cmd/omnictx: `cloud aws region [<r>|auto]` dispatch — set / clear (idempotent) / bare print; usage error exit 2; main_test coverage

## 4. Hook protocol

- [x] 4.1 Directive computation in Go (new small package or internal/aws func): the five-row pin table over `AWS_PROFILE`, `AWS_REGION`, `AWS_VAULT`, `__OMNICTX_AWS_PROFILE`, `__OMNICTX_AWS_REGION` + config values; exhaustive table-driven tests
- [x] 4.2 cmd/omnictx: hook-mode entrypoint emitting the three-line output; render line 3 with pending exports applied (no one-cycle lag); render-invariant discipline (errors → empty directives, exit 0, no writes); `OMNICTX_ENABLED=false` → three empty lines; tests
- [x] 4.3 internal/shellinit templates (bash + zsh): consume three lines, apply directives (export+marker / unset both / skip), markers exported, no eval, directives applied before prompt string; regenerate golden files; idempotency tests stay green

## 5. Docs and finishing

- [x] 5.1 Update AGENTS.md (AWS no longer the excluded provider; document the hook contract, config keys, reserved word, rc-export pin caveat) and README (switch usage, migration note: re-eval init / restart shells, remove rc `export AWS_PROFILE`)
- [x] 5.2 Full gate: `make build`, `make test` (-race), `make lint`; manual smoke: switch profile in one terminal, observe the flip in a second hook-running terminal, verify manual `export AWS_PROFILE` pins a session and `region auto` unsets only hook-owned values
