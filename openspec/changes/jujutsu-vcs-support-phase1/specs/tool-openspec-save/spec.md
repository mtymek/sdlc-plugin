## ADDED Requirements

### Requirement: jj mode stop
In jj mode the tool SHALL stop after the already-saved shortcut and before any branch or commit with a DomainError whose suggestion is `Save by hand: jj bookmark create openspec/<change> -r @, then jj commit openspec/changes/<change> -m "<message>", then jj git push -b openspec/<change>.`

#### Scenario: Save in a jj repository
- **WHEN** `openspec_save` runs in jj mode for an unsaved change
- **THEN** no branch is created
- **THEN** the tool returns the DomainError with the manual commands

#### Scenario: Already saved
- **WHEN** the plan header shows the change is already saved
- **THEN** the tool returns the already-saved result, as in git mode
