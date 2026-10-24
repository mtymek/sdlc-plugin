## ADDED Requirements

### Requirement: jj mode write stops
In jj mode the `wave-commit`, `base-sync` and `base-sync-resolve` actions SHALL stop with a DomainError before their first write, after the paths that write nothing.

#### Scenario: Disabled base sync in jj mode
- **WHEN** `execute.baseSync = false`
- **WHEN** `base-sync` runs in jj mode
- **THEN** the status is `disabled`, as in git mode

#### Scenario: Wave commit in jj mode
- **WHEN** `commitWaves` is true
- **WHEN** `wave-commit` runs in jj mode
- **THEN** the tool returns the DomainError and runs no `git add`
- **THEN** the suggestion starts with `Set execute.commitWaves = false in .sdlc-v2/config.toml and re-run;`

#### Scenario: Base sync in jj mode
- **WHEN** base sync is enabled
- **WHEN** `base-sync` runs in jj mode
- **THEN** the tool returns the DomainError before any fetch or merge
- **THEN** the suggestion is `Rebase by hand: jj git fetch, then jj rebase -d <base>@origin. Or set execute.baseSync = false.`

#### Scenario: Base sync resolve in jj mode
- **WHEN** `base-sync-resolve` runs in jj mode after the branch check
- **THEN** the tool returns the DomainError
- **THEN** the suggestion starts with `There is nothing to resolve in jj mode: base-sync makes no merge there.`

### Requirement: jj manual-commit instruction
When `commitWaves` is false in jj mode, `wave-commit` SHALL return the instruction `jj mode: do not commit. Leave wave <N>'s changes in the working-copy change @, then call wave-committed for wave <N> with no sha before wave-start for wave <N+1>.`

#### Scenario: Manual commit text in jj mode
- **WHEN** `commitWaves` is false in jj mode
- **THEN** `next.instruction` is the jj text

### Requirement: resolve-config VCS status
The `resolve-config` action SHALL return `vcs` (`mode`, `jjVersion`, `branch`, `needsBookmark`) and `next`, picking `next` from the first matching condition in the scenarios below, and SHALL add a VCS error with its suggestion to `warnings`.

#### Scenario: Detector error
- **WHEN** detection fails
- **THEN** `vcs.mode` is `unknown`
- **THEN** the error text and its suggestion are in `warnings`
- **THEN** `next` starts with `Stop: the version-control check failed.`

#### Scenario: Bookmark check failed
- **WHEN** the repository is in jj mode
- **WHEN** the branch lookup fails, or `[git] baseBranch` is set but invalid
- **THEN** `next` starts with `jj mode: stop. The bookmark check failed.`

#### Scenario: Git repository
- **WHEN** the repository is in git mode
- **THEN** `next` is `Continue Step 0 with the plan-argument gate.`

#### Scenario: jj with wave commits on
- **WHEN** resolve-config runs in jj mode with `commitWaves` true
- **THEN** `next` starts with `jj mode: stop. Wave commits`

#### Scenario: jj without a feature bookmark
- **WHEN** the repository is in jj mode with `commitWaves` false
- **WHEN** `vcs.needsBookmark` is true
- **THEN** `next` is `jj mode: stop. Create a feature bookmark with jj bookmark create <name> -r @, then re-run /execute.`

#### Scenario: No default branch
- **WHEN** the default-branch lookup finds no default branch
- **THEN** `vcs.needsBookmark` is not set by that lookup

#### Scenario: jj ready to run
- **WHEN** the repository is in jj mode with `commitWaves` false and a feature bookmark `b`
- **THEN** `next` starts with `jj mode: continue on bookmark b.`
