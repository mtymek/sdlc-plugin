## ADDED Requirements

### Requirement: jj mode write stops
In jj mode the `wave-commit`, `base-sync` and `base-sync-resolve` actions SHALL stop with a DomainError before their first write, after the paths that write nothing.

| Action | Stop placed after | Suggestion |
|---|---|---|
| wave-commit | idempotent and `commitWaves=false` returns | `Set execute.commitWaves = false in .sdlc-v2/config.toml and re-run; …` |
| base-sync | disabled return | `Rebase by hand: jj git fetch, then jj rebase -d <base>@origin. Or set execute.baseSync = false.` |
| base-sync-resolve | branch check | `There is nothing to resolve in jj mode: base-sync makes no merge there. …` |

#### Scenario: Disabled base sync in jj mode
- **WHEN** `execute.baseSync = false`
- **WHEN** `base-sync` runs in jj mode
- **THEN** the status is `disabled`, as in git mode

#### Scenario: Wave commit in jj mode
- **WHEN** `commitWaves` is true
- **WHEN** `wave-commit` runs in jj mode
- **THEN** the tool returns the DomainError and runs no `git add`

### Requirement: jj manual-commit instruction
When `commitWaves` is false in jj mode, `wave-commit` SHALL return the instruction `jj mode: do not commit. Leave wave <N>'s changes in the working-copy change @, then call wave-committed for wave <N> with no sha before wave-start for wave <N+1>.`

#### Scenario: Manual commit text in jj mode
- **WHEN** `commitWaves` is false in jj mode
- **THEN** `next.instruction` is the jj text

### Requirement: resolve-config VCS status
The `resolve-config` action SHALL return `vcs` (`mode`, `jjVersion`, `branch`, `needsBookmark`) and `next`, and SHALL pick `next` from the first matching row. A VCS error SHALL be added to `warnings` with its suggestion. A failed default-branch lookup (no default branch) SHALL not set `needsBookmark`.

| Condition | next |
|---|---|
| detection failed | `Stop: the version-control check failed. …` |
| jj, branch lookup failed, or `[git] baseBranch` is set but invalid | `jj mode: stop. The bookmark check failed. …` |
| git | `Continue Step 0 with the plan-argument gate.` |
| jj, commitWaves true | `jj mode: stop. Wave commits do not run in jj repositories yet. …` |
| jj, needsBookmark | `jj mode: stop. Create a feature bookmark with jj bookmark create <name> -r @, then re-run /execute.` |
| jj otherwise | `jj mode: continue on bookmark <b>. …` |

#### Scenario: jj with wave commits on
- **WHEN** resolve-config runs in jj mode with `commitWaves` true
- **THEN** `next` starts with `jj mode: stop. Wave commits`

#### Scenario: Detector error
- **WHEN** detection fails
- **THEN** `vcs.mode` is `unknown`
- **THEN** the error text is in `warnings`
