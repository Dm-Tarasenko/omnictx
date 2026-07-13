# enabled-toggle-cli

## ADDED Requirements

### Requirement: Persist the enabled state via `omnictx on` / `omnictx off`
The CLI SHALL provide the subcommands `omnictx on` and `omnictx off` that write `enabled: true` / `enabled: false` to the config file (path resolved as `OMNICTX_CONFIG` > `~/.config/omnictx/config.yaml`) and exit with code 0. The write SHALL change only the `enabled:` line, preserving all other keys and comments, and SHALL create the file and its parent directory if absent. On a write error the command SHALL print a warning to stderr and exit with code 1. The session-scoped counterpart is the `OMNICTX_ENABLED` env var, which overrides the persisted value without touching the file.

#### Scenario: Turn off in an existing config
- **WHEN** the config file contains `enabled: true`, a comment line, and `cloud: aws`, and the user runs `omnictx off`
- **THEN** the config file contains `enabled: false`, the comment and `cloud: aws` are unchanged, and the exit code is 0

#### Scenario: Turn on with no config file
- **WHEN** no config file exists and the user runs `omnictx on`
- **THEN** the file and its parent directory are created, the file contains `enabled: true`, and the exit code is 0

#### Scenario: Custom config path via OMNICTX_CONFIG
- **WHEN** `OMNICTX_CONFIG` points to a custom path and the user runs `omnictx off`
- **THEN** the file at the custom path is updated (not the default path)

### Requirement: `on` / `off` are the only enabled-state subcommands
The CLI SHALL NOT recognize `toggle`, `enable`, or `disable` as subcommands. Like any other unrecognized first argument, these words SHALL fall through to render mode, which never writes anything and always exits 0 (the core render invariant). The persistent enabled state is controlled exclusively by `omnictx on` and `omnictx off`.

#### Scenario: `omnictx toggle` does not flip the state
- **WHEN** the config file contains `enabled: true` and the user runs `omnictx toggle`
- **THEN** the config file still contains `enabled: true`, nothing is written, and the exit code is 0

#### Scenario: `omnictx enable` and `omnictx disable` are not aliases
- **WHEN** the config file contains `enabled: false` and the user runs `omnictx enable`, then `omnictx disable`
- **THEN** the config file still contains `enabled: false` after both invocations and both exit with code 0
