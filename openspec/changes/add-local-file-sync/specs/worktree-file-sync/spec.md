## ADDED Requirements

### Requirement: Mirror ignored local files into acquired worktrees
On every acquire, treehouse SHALL copy files from the source repository working tree that git reports as ignored and whose basename matches the configured glob set into the acquired worktree, preserving each file's path relative to the source repository root and creating any missing parent directories.
The copy SHALL run on both the fresh-create path and the reuse/reset path, after the worktree is created or reset and before `post_create` hooks run.
Existing destination files SHALL be overwritten so a reused worktree reflects the current source files.

#### Scenario: Fresh worktree receives ignored local config
- **GIVEN** a source repository whose `.gitignore` ignores `appsettings.local.json`
- **AND** an `appsettings.local.json` exists in the source working tree under `Api/`
- **WHEN** a new worktree is created by `treehouse get`
- **THEN** `Api/appsettings.local.json` exists in the acquired worktree with the same contents

#### Scenario: Reused worktree is refreshed with current config
- **GIVEN** a pooled worktree that previously received a synced `appsettings.local.json`
- **AND** the source `appsettings.local.json` has since changed
- **WHEN** the worktree is reused by a later `treehouse get`
- **THEN** the worktree's `appsettings.local.json` is overwritten with the current source contents

#### Scenario: Sync runs before post_create hooks
- **GIVEN** a configured `post_create` hook that reads local config
- **WHEN** a worktree is acquired
- **THEN** the ignored local files are present before the hook runs

### Requirement: Only ignored files are synced
Treehouse SHALL copy only files that git reports as ignored in the source repository.
Files that are tracked, or untracked but not ignored, SHALL NOT be copied by this feature.
As a consequence, a synced file SHALL NOT cause the worktree to be reported as dirty, and SHALL NOT change acquire skip behavior, `return` prompting, or `prune` classification.

#### Scenario: Synced file does not make the worktree dirty
- **GIVEN** a worktree that has received a synced ignored `appsettings.local.json`
- **WHEN** treehouse evaluates whether the worktree is dirty
- **THEN** the worktree is reported clean
- **AND** the worktree remains eligible for reuse by `Acquire` and does not trigger the uncommitted-changes prompt on `return`

#### Scenario: Untracked non-ignored file is not synced
- **GIVEN** a source file matching the glob set that git does not ignore
- **WHEN** a worktree is acquired
- **THEN** the file is not copied into the worktree

### Requirement: No-op when nothing matches
When no ignored source file matches the configured glob set, the sync step SHALL make no changes, produce no output, and return no error.

#### Scenario: Repository with no matching files
- **GIVEN** a source repository with no ignored files matching the glob set
- **WHEN** a worktree is acquired
- **THEN** no files are copied
- **AND** no output or error is produced by the sync step

### Requirement: Configurable, default-on glob set
Treehouse SHALL expose a `sync_ignored` configuration key holding a list of basename globs, defaulting to `["appsettings*.local.json"]` when unset.
An absent key SHALL preserve the default set.
An empty list SHALL disable the feature.
The key SHALL be honored in both user-level and repo-level configuration, in contrast to `hooks`, which are honored only at user level.

#### Scenario: Default applies with no configuration
- **GIVEN** no `sync_ignored` key in any configuration file
- **WHEN** a worktree is acquired
- **THEN** ignored files matching `appsettings*.local.json` are synced

#### Scenario: Empty list disables syncing
- **GIVEN** `sync_ignored = []` in configuration
- **WHEN** a worktree is acquired
- **THEN** no files are synced regardless of what exists in the source working tree

#### Scenario: Repo-level configuration is honored
- **GIVEN** a repo-level `treehouse.toml` that sets `sync_ignored`
- **WHEN** a worktree is acquired for that repository
- **THEN** the repo-level glob set is used
