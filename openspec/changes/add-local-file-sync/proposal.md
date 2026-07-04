## Why

`treehouse get` creates a worktree with `git worktree add`, which only ever materializes committed files.
Local-only development config such as `appsettings*.local.json` is gitignored, so it lives only in the developer's main clone and never appears in a freshly acquired worktree.
Without those files the app cannot be run locally in the new worktree, which defeats the instant-isolation promise of the pool.
Today the only workaround is a hand-written `post_create` hook, and even that is blocked because hooks are given no pointer back to the source repository.
Developers want this to work out of the box, with zero configuration and no side effects when no such files exist.

## What Changes

- On every acquire (both fresh-create and reuse/reset paths), treehouse mirrors ignored files that match a configured glob set from the source repository's working tree into the newly acquired worktree, preserving each file's relative path.
- The feature is on by default with a built-in glob set of `["appsettings*.local.json"]`; no configuration is required.
- The glob set is overridable and extensible via a new `sync_ignored` config key, and setting it to an empty list disables the feature.
- Unlike `hooks`, `sync_ignored` is allowed in repo-level `treehouse.toml` (it runs no commands, it only copies the developer's own ignored files), so a team can share a sensible default. It is also settable in user-level config.
- Only files that git reports as ignored are ever copied. This is the invariant that keeps synced files from ever tripping treehouse's dirty detection, so acquire, return, and prune behavior is unchanged.
- When no source file matches the glob set, the step is a silent no-op: no copy, no output, no error, no measurable slowdown.
- Copies overwrite on each acquire so a reused pooled worktree always lands on the current local config.
- Syncing runs after the worktree is created or reset and before `post_create` hooks, so a hook can rely on the local config already being present.

## Capabilities

### New Capabilities
- `worktree-file-sync`: Mirroring ignored local files (matching a configured glob set) from the source repository working tree into a worktree at acquire time, including the ignored-only invariant, default glob set, override/disable config, repo-level allowance, and no-op-when-absent behavior.

### Modified Capabilities
<!-- None: no existing specs to modify. -->

## Impact

- `internal/config/config.go`: new `SyncIgnored []string` field with a built-in default; loaded from both repo-level and user-level config (not cleared like `Hooks`), with an absent key preserving the default and an empty list disabling.
- `internal/git/git.go`: new helper to list ignored files in the source working tree via `git ls-files --others --ignored --exclude-standard`.
- `internal/localsync/` (new package): enumerate matching ignored files and copy them into the worktree, preserving relative paths and creating parent directories.
- `internal/pool/pool.go`: `acquire` runs the sync after the create and reuse/reset paths and before `hooks.Run`; the glob set is threaded through `acquireOptions`.
- `cmd/get.go`: pass `cfg.SyncIgnored` into `pool.Acquire` and `pool.AcquireLease`.
- No change to the state file format, to `return`/`Release` semantics, or to dirty/prune classification.
- `README.md`/docs: document `sync_ignored`, the default, disabling, and the ignored-only rule.
