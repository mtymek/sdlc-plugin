## ADDED Requirements

### Requirement: jj push status
In jj mode the tool SHALL set `needsPush` from the bookmark check and SHALL report each problem as a warning with a matching `next`.

| jj state | needsPush | Warning |
|---|---|---|
| match | false | none |
| no remote or differs | true | `jj mode: run jj git push -b <b> before pr_apply` |
| no bookmark | true | `jj mode: no bookmark on @ or @-; create one with jj bookmark create` |
| detector, branch or ref error | true | `jj push check: <err>` |

#### Scenario: jj bookmark not pushed
- **WHEN** the repository is in jj mode
- **WHEN** `origin/feat/x` does not exist
- **THEN** `needsPush` is true
- **THEN** the warning is `jj mode: run jj git push -b feat/x before pr_apply`

#### Scenario: jj bookmark matches
- **WHEN** the repository is in jj mode
- **WHEN** `feat/x` equals `origin/feat/x`
- **THEN** `needsPush` is false
