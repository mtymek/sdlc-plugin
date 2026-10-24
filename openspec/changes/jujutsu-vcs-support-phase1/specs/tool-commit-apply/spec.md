## ADDED Requirements

### Requirement: jj mode stop
In jj mode the tool SHALL stop before any git command with a DomainError whose suggestion is `Commit by hand: jj commit <paths> -m "<message>", then jj bookmark set <bookmark> -r @-.`

#### Scenario: Commit in a jj repository
- **WHEN** `commit_apply` runs in a colocated jj repository
- **THEN** the git index and HEAD are unchanged
- **THEN** the tool returns the DomainError with the jj commit suggestion
