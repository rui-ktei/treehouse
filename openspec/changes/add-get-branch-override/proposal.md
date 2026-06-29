## Why

`treehouse get` always starts a worktree at the default branch's resolved HEAD, with no way to choose a different starting point.
Users who need a worktree based on a feature branch, tag, or specific commit currently have to acquire a default-branch worktree and then manually re-checkout inside it, which defeats the pre-warmed, instant-isolation promise of the pool.

## What Changes

- Add an optional `--branch <ref>` flag to `treehouse get` (including `treehouse get --lease`) that sets the starting ref for the acquired worktree.
- When `--branch` is provided, both the create path and the reuse/reset path base the worktree on the given ref instead of the default branch.
- When `--branch` is omitted, behavior is unchanged: the worktree starts at the resolved default branch HEAD.
- Returning a worktree to the pool continues to reset it to the default branch; a custom starting ref applies only for the session it was acquired in.
- No `$TREEHOUSE_BRANCH` environment fallback is added; the override is flag-only.

## Capabilities

### New Capabilities
- `worktree-acquisition`: Acquiring a worktree from the pool via `get`, including which ref the worktree's HEAD starts at and how an optional caller-specified starting ref overrides the default branch.

### Modified Capabilities
<!-- None: no existing specs to modify. -->

## Impact

- `cmd/get.go`: new `--branch` flag wired into both interactive `get` and `get --lease`.
- `internal/pool/pool.go`: `acquire`/`acquireOptions` carry an optional starting ref; `Acquire` and `AcquireLease` accept and forward it.
- `internal/git/git.go`: ref-resolution helpers (`branchRef`, `AddWorktree`, `ResetWorktree`) accept a caller-specified ref, falling back to default-branch resolution.
- No change to `return`/`Release` semantics, state file format, or config schema.
