## ADDED Requirements

### Requirement: Default starting ref

When acquiring a worktree without a caller-specified starting ref, `treehouse get` SHALL start the worktree's detached HEAD at the repository's resolved default branch.

This applies to both `treehouse get` (interactive subshell) and `treehouse get --lease` (non-interactive durable lease), and to both newly created worktrees and reused pooled worktrees.

#### Scenario: Acquire without --branch starts at default branch

- **WHEN** a user runs `treehouse get` with no `--branch` flag
- **THEN** the acquired worktree's HEAD is the resolved default branch HEAD, identical to behavior before this change

#### Scenario: Lease without --branch starts at default branch

- **WHEN** a user runs `treehouse get --lease` with no `--branch` flag
- **THEN** the leased worktree's HEAD is the resolved default branch HEAD

### Requirement: Caller-specified starting ref

`treehouse get` SHALL accept an optional `--branch <ref>` flag that sets the starting ref for the acquired worktree. When provided, the worktree's detached HEAD SHALL be based on `<ref>` instead of the default branch.

The flag SHALL be honored identically for `treehouse get` and `treehouse get --lease`, and SHALL apply whether the pool hands out a newly created worktree or resets and reuses an existing pooled worktree.

`<ref>` MAY be a local branch name, a remote-tracking ref, a tag, or a commit SHA. There SHALL be no environment-variable fallback for this value.

#### Scenario: Acquire a new worktree on a specified branch

- **WHEN** a user runs `treehouse get --branch feature-x` and the pool creates a new worktree
- **THEN** the new worktree's detached HEAD is at the resolved `feature-x` ref

#### Scenario: Reused worktree is reset to the specified branch

- **WHEN** a user runs `treehouse get --branch feature-x` and the pool reuses an available worktree
- **THEN** the reused worktree is reset so its detached HEAD is at the resolved `feature-x` ref, not the default branch

#### Scenario: Lease honors the specified branch

- **WHEN** a user runs `treehouse get --lease --branch feature-x`
- **THEN** the leased worktree's detached HEAD is at the resolved `feature-x` ref and only the worktree path is printed to stdout

#### Scenario: Tag or commit SHA as starting ref

- **WHEN** a user runs `treehouse get --branch <tag-or-sha>` with a ref that is not a branch
- **THEN** the acquired worktree's detached HEAD is at that tag or commit

#### Scenario: Invalid ref is rejected before setup

- **WHEN** a user runs `treehouse get --branch does-not-exist` with a ref that cannot be resolved
- **THEN** the command fails with an error naming the unresolvable ref
- **AND** no worktree is created, reset, or left partially set up

### Requirement: Starting ref override is not sticky across return

A caller-specified starting ref SHALL apply only for the acquisition in which it was given. Returning a worktree to the pool SHALL reset it to the default branch, regardless of the ref it was acquired on.

#### Scenario: Returned worktree resets to default branch

- **WHEN** a worktree acquired with `--branch feature-x` is returned to the pool via `treehouse return` or by exiting the `get` subshell
- **THEN** the worktree is reset to the default branch HEAD and becomes available like any other pooled worktree
