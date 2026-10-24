## ADDED Requirements

### Requirement: VCS mode detection
The plugin SHALL decide git or jj mode by walking up from the working folder: the first folder holding `.git` or `.jj` decides, both markers mean jj mode, and no marker means git mode.

#### Scenario: Plain git repository
- **WHEN** the first marker folder holds only `.git`
- **THEN** the mode is git
- **THEN** jj is never run

#### Scenario: Colocated jj repository
- **WHEN** the first marker folder holds `.git` and `.jj`
- **WHEN** `jj --version` prints `jj 0.46.0`
- **THEN** the mode is jj with version `0.46.0`

#### Scenario: Linked worktree
- **WHEN** `.git` is a file
- **THEN** it counts the same as a `.git` folder

### Requirement: Colocated repositories only
The plugin SHALL refuse a repository whose first marker folder holds `.jj` and no `.git` with a DomainError whose suggestion is "Run `jj git init --colocate` in the repository root."

#### Scenario: jj-only repository
- **WHEN** a tool runs in a folder with `.jj` and no `.git`
- **THEN** the tool returns a DomainError
- **THEN** the suggestion is "Run `jj git init --colocate` in the repository root."

### Requirement: Minimum jj version
The plugin SHALL require jj 0.46.0 or newer in jj mode, with the minimum held in one constant.

| Condition | Class | Suggestion |
|---|---|---|
| jj not on PATH | InfraError | `Install jj 0.46 or newer.` |
| version below 0.46.0 | InfraError | `Upgrade jj to 0.46 or newer.` |
| output is not a version | DataError | shows the output, cut to 200 characters |

#### Scenario: Old jj
- **WHEN** `jj --version` prints `jj 0.45.9`
- **THEN** the tool returns an InfraError with suggestion `Upgrade jj to 0.46 or newer.`

#### Scenario: Missing jj
- **WHEN** jj is not on PATH in a colocated repository
- **THEN** the tool returns an InfraError with suggestion `Install jj 0.46 or newer.`

#### Scenario: Unreadable version
- **WHEN** `jj --version` prints `hello`
- **THEN** the tool returns a DataError that shows `hello`

### Requirement: Bookmark as current branch
In jj mode the current branch SHALL be the one local bookmark on `@`, else the one local bookmark on `@-`, else `HEAD`.

#### Scenario: Bookmark on the working change
- **WHEN** `@` has bookmark `feat/x`
- **THEN** the current branch is `feat/x`

#### Scenario: Bookmark on the parent
- **WHEN** `@` has no bookmark
- **WHEN** `@-` has bookmark `feat/x`
- **THEN** the current branch is `feat/x`

#### Scenario: No bookmark
- **WHEN** neither `@` nor `@-` has a bookmark
- **THEN** the current branch is `HEAD`

#### Scenario: Two bookmarks
- **WHEN** the change read has bookmarks `a` and `b`
- **THEN** the tool returns a DomainError that lists `a, b`
- **THEN** the suggestion tells the user to keep one bookmark with `jj bookmark`

#### Scenario: Bookmark named HEAD
- **WHEN** the change read has a bookmark named `HEAD`
- **THEN** the tool returns a DomainError for a reserved name

### Requirement: Default branch in jj mode
In jj mode the default branch SHALL follow the git-mode rule: `[git] baseBranch`, then `origin/HEAD`, then `main`, then `master`.

#### Scenario: Configured base branch
- **WHEN** `[git] baseBranch = "develop"` in a colocated jj repository
- **THEN** the base branch is `develop`

### Requirement: Session-start VCS line
The session-start context SHALL include one VCS line in a repository.

| Mode | Line |
|---|---|
| git | `VCS: git` |
| jj | `VCS: jj <version> (colocated)` |
| error | `VCS: error — <message>. <suggestion>` |

#### Scenario: Detection error at session start
- **WHEN** the session starts in a jj-only repository
- **THEN** the context holds a `VCS: error —` line with the colocate suggestion
- **THEN** the hook still exits 0

### Requirement: jj write stop
In jj mode every plugin tool that writes a commit, branch, push or merge SHALL stop before its first write with a DomainError whose suggestion names the jj command to run by hand.

#### Scenario: Write in jj mode
- **WHEN** a write tool runs in a colocated jj repository
- **THEN** git index, HEAD and branches are unchanged
- **THEN** the tool returns a DomainError with a jj command in its suggestion
