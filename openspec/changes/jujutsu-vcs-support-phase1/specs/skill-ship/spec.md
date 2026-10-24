## ADDED Requirements

### Requirement: jj mode stop
At step 3 the skill SHALL call `ship_state({action:"vcs-mode"})` first; in jj mode it SHALL print `next` and end the run before `ship_prepare`.

#### Scenario: Ship in a jj repository
- **WHEN** `/ship` runs in a colocated jj repository
- **THEN** the skill prints the `vcs-mode` `next` text
- **THEN** `ship_prepare` is not called

#### Scenario: Ship in a git repository
- **WHEN** `vcs-mode` returns git
- **THEN** step 3 runs as before
