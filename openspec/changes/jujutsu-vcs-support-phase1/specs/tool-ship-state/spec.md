## ADDED Requirements

### Requirement: vcs-mode action
The `vcs-mode` action SHALL return `mode`, `jjVersion`, `branch`, `needsBookmark` and `next`, read git and jj only, need no state file, and write nothing.

| Mode | next |
|---|---|
| git | `git mode: continue with step 3 as written.` |
| jj | `jj mode: stop. Ship does not run in jj repositories yet. Run /execute with execute.commitWaves = false, commit with jj commit, push with jj git push -b <bookmark>, then run /pr.` |

#### Scenario: vcs-mode in a jj repository
- **WHEN** `ship_state({action:"vcs-mode"})` runs in a colocated jj repository
- **THEN** `mode` is `jj`
- **THEN** `next` starts with `jj mode: stop.`

#### Scenario: vcs-mode detection error
- **WHEN** detection fails
- **THEN** the action returns the typed error

### Requirement: commit-check jj stop
In jj mode the `commit-check` action SHALL stop with a DomainError after its input checks and before `git add`.

#### Scenario: commit-check in jj mode
- **WHEN** `commit-check` runs in jj mode
- **THEN** no `git add` runs
- **THEN** the ship state file is unchanged
