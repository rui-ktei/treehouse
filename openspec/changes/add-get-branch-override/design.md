## Context

`treehouse get` acquires a worktree from the pool through `pool.Acquire`/`pool.AcquireLease`, which both delegate to the shared `acquire(...)` core in `internal/pool/pool.go`.
That core resolves a single starting ref with `git.GetDefaultBranch(repoRoot)` and uses it for both branches of the acquire path:

- New worktree: `git.AddWorktree(repoRoot, wtPath, branch)` → `git worktree add --detach <path> <branchRef(branch)>`.
- Reused worktree: `git.ResetWorktree(wt.Path, branch)` → `git checkout --detach --force <branchRef(branch)>` + `reset --hard` + `clean -fd`.

`git.branchRef` turns a plain branch name into the most-ahead of the local branch ref and `origin/<branch>` tracking ref (preferring origin on divergence).
There is currently no way for a caller to specify a different starting point. This change introduces a caller-supplied override that threads through that single resolution point.

## Goals / Non-Goals

**Goals:**
- Let a caller of `treehouse get` (interactive and `--lease`) choose the starting ref via `--branch <ref>`.
- Apply the override consistently to both the create and reuse/reset paths so a reused pooled worktree lands on the same ref a freshly created one would.
- Keep the default (no flag) behavior byte-for-byte identical to today.
- Accept any ref a user would reasonably pass: a branch name, a remote-tracking ref, a tag, or a commit SHA.

**Non-Goals:**
- No `$TREEHOUSE_BRANCH` environment fallback.
- No change to `return`/`Release`: a returned worktree still resets to the default branch. The override is per-acquire, not sticky.
- No new branch creation: `--branch` selects an existing ref to start from; it does not create or track a branch (worktrees stay detached, as today).
- No config-file knob for a default starting ref.

## Decisions

### Thread the override as an optional ref string through `acquireOptions`
`acquireOptions` already carries per-acquire intent (lease, holder, hook writers). Add a `startRef string` field (empty means "use the default branch").
`Acquire` and `AcquireLease` gain a `startRef` parameter that they set on the options. Inside `acquire`, resolution becomes:

```
ref := startRef
if ref == "" {
    ref, err = git.GetDefaultBranch(repoRoot)
    ...
}
```

and the existing `AddWorktree`/`ResetWorktree` calls pass `ref`.
Rationale: a single resolution point already fans out to both paths, so one substitution covers create and reuse with no duplication. Alternative considered: a separate `AcquireFromRef` entry point - rejected because it would duplicate the whole pool-scan body just to vary one value.

### Resolve a user ref distinctly from a default branch name
`git.branchRef(repoRoot, branch)` assumes its argument is a plain branch name and expands it to local-vs-`origin/<branch>`. That is correct for the default branch, but a user-supplied `--branch` may already be a tag, a SHA, or `origin/foo`.
Decision: when a caller-supplied ref is present, verify it with `git rev-parse --verify <ref>` and use it directly; only fall back to the `branchRef` local/remote expansion when the override is a bare branch name that is not directly resolvable.
This keeps default-branch behavior (most-ahead of local/origin) intact while letting `--branch` accept the broader ref set named in Goals.
Alternative considered: always pass `--branch` through `branchRef` - rejected because it would mangle tags/SHAs and silently pick the wrong ref.

### Fetch before resolving
`acquire` already fetches `origin` (when present) before resolution, so a `--branch origin/feature` or a recently pushed branch resolves against fresh refs. No change needed here beyond ordering the override resolution after the existing fetch.

## Risks / Trade-offs

- [Invalid or unknown ref passed to `--branch`] → Validate up front with `git rev-parse --verify` and return a clear error before any worktree is created or reset, so a bad ref never leaves a half-set-up worktree.
- [User expects the worktree to stay on their branch after `return`] → Documented non-goal: return resets to default. Surface this in the flag help text so the per-session scope is explicit.
- [Ambiguous ref name that is both a branch and a tag] → Defer to git's own `rev-parse` resolution rather than inventing precedence rules; behavior matches what the user would get from plain git.
- [Reused worktree whose reset to a custom ref fails] → The existing reuse loop already `continue`s past a worktree whose `ResetWorktree` fails; an invalid override is caught earlier by up-front validation, so the loop only sees resolvable refs.
