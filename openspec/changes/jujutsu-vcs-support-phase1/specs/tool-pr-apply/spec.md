## MODIFIED Requirements

### Requirement: Push before create or edit
The tool SHALL run `git push -u origin HEAD` when the branch has no upstream or is ahead of it, and SHALL skip the push when the upstream is caught up. In jj mode the tool SHALL never push and SHALL compare `refs/heads/<bookmark>` with `refs/remotes/origin/<bookmark>` before any remote write.

| Condition | Class | Message / Suggestion (short) |
|---|---|---|
| Upstream check fails | `InfraError` | `checking upstream: <error>` |
| Commits-ahead check fails | `InfraError` | `checking commits ahead of upstream: <error>` |
| Push fails | `InfraError` | `git push: <error>` / push manually with `git push -u origin HEAD`, retry `pr_apply` |
| jj: no remote bookmark | `DomainError` | `bookmark <b> is not on origin` / `Push it with jj git push -b <b>, then retry pr_apply.` |
| jj: bookmark differs from origin | `DomainError` | `bookmark <b> differs from origin/<b>` / same suggestion |
| jj: no bookmark | `DomainError` | `no bookmark on @ or @-` / create one with `jj bookmark create <name> -r @-` |
| jj: local ref missing | `InfraError` | `bookmark <b> has no git branch` / run `jj git export` |

#### Scenario: No upstream
- **WHEN** the branch has no upstream
- **THEN** the tool pushes to `origin`

#### Scenario: Upstream behind
- **WHEN** HEAD is 2 commits ahead of the upstream
- **THEN** the tool pushes

#### Scenario: Caught up
- **WHEN** HEAD is 0 commits ahead of the upstream
- **THEN** no push runs

#### Scenario: jj bookmark matches origin
- **WHEN** the repository is in jj mode
- **WHEN** `feat/x` and `origin/feat/x` point to the same commit
- **THEN** no push runs
- **THEN** the tool runs `gh pr view feat/x` and creates the PR with `--head feat/x`
- **THEN** the result returns `headBranch: feat/x`

#### Scenario: jj bookmark not pushed
- **WHEN** the repository is in jj mode
- **WHEN** `origin/feat/x` does not exist
- **THEN** the tool returns a DomainError with suggestion `Push it with jj git push -b feat/x, then retry pr_apply.`
- **THEN** no label, push or PR call runs
