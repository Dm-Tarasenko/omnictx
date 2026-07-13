## 1. Remove the toggle subcommand and aliases

- [x] 1.1 `cmd/omnictx/main.go`: drop the `"toggle"` case and the `enable`/`disable` alias words from the dispatch switch (keep `"on"` and `"off"`); delete `runToggle`; update the `runEnable` doc comment to name `on`/`off` instead of `enable`/`disable`
- [x] 1.2 `cmd/omnictx/main_test.go`: remove/adjust tests that invoke `runToggle` or the `enable`/`disable` aliases; add coverage for the new contract — `omnictx toggle` with an existing config leaves `enabled:` untouched and exits 0 (spec: "`omnictx toggle` does not flip the state")

## 2. Dead-comment and rename cleanup (no behavior change)

- [x] 2.1 `cmd/omnictx/main.go`: rename `runCloudUse` → `runCloudSwitch` (definition, call site, doc comment, any test references)
- [x] 2.2 `cmd/omnictx/main.go`: replace the `// omnioff path: ...` comment in `runRender` with wording that doesn't reference the removed shell functions
- [x] 2.3 `internal/config/config.go`: fix the `Resolve` doc comment — notes are surfaced as stderr warnings by interactive subcommands; there is no `--debug` flag
- [x] 2.4 `internal/shellinit/shellinit_test.go`: fix the `TestBashSnippetIsValidAndIdempotent` comment (it asserts the toggle functions do NOT exist) and delete the unasserted `echo HAS_OMNION` / `echo HAS_OMNIOFF` script lines

## 3. Documentation fixes

- [x] 3.1 `README.md`: remove all four `toggle` mentions (daily-use block line 56, env table line 69, usage list line 191, configuration table line 249)
- [x] 3.2 `README.md`: fix the icon claims — the header example (line 9) and "Output format" (line 211) say "one `☁` for any provider"; the code renders per-provider Nerd Font glyphs (`󰠅`/``/`󱇶`, PRD §5.7)
- [x] 3.3 `PRD.md`: update §4.7 (namespace switching shipped — no longer out of scope), the header `Branch: feat/omnictx` line, §5.1 (`--cloud` flag was never built; selection is `OMNICTX_CLOUD` > config), and §5.6 + §8.3 (`--help` does not list env vars in a Configuration section)
- [x] 3.4 `AGENTS.md`: fix the Definition of Done reference (`section 7.5 of PRD.md` → §8.3) and add `OMNICTX_SHELL` to the config section as the session-scoped counterpart of `--shell`

## 4. Verification

- [x] 4.1 Run `make test` (race) and `make lint`; both green
- [x] 4.2 Grep sweep: `grep -rni "toggle" cmd internal README.md AGENTS.md PRD.md` shows only legitimate kube/cloud display-toggle wording — no `omnictx toggle`, `runToggle`, `enable`/`disable` alias, `runCloudUse`, `omnioff`, or `--debug` leftovers
