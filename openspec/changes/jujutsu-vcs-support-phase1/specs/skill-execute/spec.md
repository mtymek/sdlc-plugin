## ADDED Requirements

### Requirement: jj mode gate
Right after `resolve-config`, the skill SHALL follow its `next`: a `next` that starts with `Stop` or `jj mode: stop` ends the run with no state written.

#### Scenario: jj with wave commits on
- **WHEN** `/execute` runs in jj mode with `execute.commitWaves` true
- **THEN** the skill prints the `next` text
- **THEN** no run state is written

### Requirement: jj mode run
In jj mode the skill SHALL use `vcs.branch` for workspace detection, skip the pre-execution rebase and the base-sync stage with one printed jj hint, follow the jj manual-commit instruction, and add `jj mode: wave changes are uncommitted in @; commit them with jj commit.` to the report.

#### Scenario: Rebase in jj mode
- **WHEN** `--rebase auto` is passed in jj mode
- **THEN** the skill prints `jj mode: rebase skipped — run jj git fetch, then jj rebase -d <base>@origin.`
- **THEN** no git rebase runs
