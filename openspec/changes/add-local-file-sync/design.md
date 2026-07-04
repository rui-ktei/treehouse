## Context

`treehouse get` acquires a worktree through `pool.Acquire`/`pool.AcquireLease`, which both delegate to the shared `acquire(...)` core in `internal/pool/pool.go`.
That core creates a worktree with `git.AddWorktree` (fresh) or resets an existing one with `git.ResetWorktree` (reuse), then runs `post_create` hooks via `hooks.Run`.
`git worktree add` and `git reset --hard`/`git clean -fd` only ever produce committed content, so gitignored local config such as `appsettings*.local.json` is never present in an acquired worktree.

The pool model depends on a binary cleanliness signal.
`git.IsDirty` (`git status --porcelain --untracked-files=all`) decides whether `Acquire` skips a worktree, whether `return` prompts about uncommitted changes, and whether `prune` may delete it.
Any mechanism that adds files to a worktree must not disturb that signal.

## Goals / Non-Goals

**Goals:**
- Make locally-runnable config appear in every acquired worktree with zero configuration.
- Guarantee that synced files never change treehouse's dirty/prune/return behavior.
- Keep the step a true no-op (no output, no error, negligible cost) when nothing matches.
- Allow teams to share the glob set via repo-level config and individuals to override or disable it.

**Non-Goals:**
- Copying files that git does not ignore. Untracked-but-not-ignored files are deliberately out of scope because they would trip dirty detection.
- Symlinking or otherwise sharing one canonical file across worktrees.
- Syncing on `return`/`Release`; that path continues to reset to the default branch only.
- A general file-provisioning or templating system. This mirrors existing ignored files verbatim.

## Decisions

### Sync only files git reports as ignored (the core invariant)
The set of files to consider is exactly the output of `git -C <sourceRepo> ls-files --others --ignored --exclude-standard`, filtered to those whose basename matches a configured glob.
Restricting to ignored files is not a convenience, it is the correctness mechanism:

- `git status --porcelain` does not report ignored files (it would need `--ignored`), so a copied ignored file leaves `IsDirty` returning false. Acquire will not skip the worktree, return will not prompt, prune will not treat it as unlanded.
- `git clean -fd` does not remove ignored files (it would need `-x`), so a synced file survives a reuse reset rather than being deleted and re-created churnwise.
- The same committed `.gitignore` governs both the source clone and the worktree, so a file ignored in the source is ignored in the worktree by construction.

Enumerating through git (rather than walking the filesystem) also means the scan never descends into large ignored trees like `node_modules`, `bin`, or `obj`; git already knows the ignore set, and the glob filter keeps only the intended files.
Alternative considered: copy any file matching the glob regardless of ignore status, then neutralize it via `.git/info/exclude` in the worktree. Rejected as more complex and more surprising than simply scoping to files git already ignores.

### Run inside `acquire`, after create/reset and before hooks
The sync executes at the same point on both acquire paths, immediately after `AddWorktree`/`ResetWorktree` succeed and before `hooks.Run(postCreate, ...)`.

- Both paths already set `runPostCreate = true`, so syncing on both keeps a reused pooled worktree as fresh as a newly created one.
- Copies overwrite unconditionally, so a worktree reused days later still lands on the current local config rather than a stale snapshot.
- Running before hooks lets a `post_create` hook (for example a restore or build step) rely on the local config already being in place.

### Config: default-on, overridable, repo-safe
A new `SyncIgnored []string` field is added to `Config`, defaulted in `DefaultConfig()` to `["appsettings*.local.json"]`.
Globs match against the file basename, so a pattern matches at any directory depth, which is what nested project layouts need.

- Zero-config: the default set ships active, so `treehouse get` mirrors appsettings local files with no user action.
- Override/extend: set `sync_ignored = ["appsettings*.local.json", ".env.local"]` to change the set.
- Disable: `sync_ignored = []` turns the feature off. An absent key preserves the default because the TOML decoder leaves the pre-set `DefaultConfig` value untouched when the key is not present.
- Repo-safe: unlike `hooks`, which are stripped from repo-level config because they execute arbitrary commands, `sync_ignored` is a list of filename globs that trigger only a file copy of the developer's own ignored files. It is therefore honored in repo-level `treehouse.toml` so a team can standardize it, and also in user-level config.

Naming rationale: `sync_ignored` names the invariant honestly (it only ever touches ignored files) rather than implying it can provision arbitrary content.

### Copy, do not symlink
Each matching file is copied into the worktree at the same path relative to the source repo root, creating parent directories as needed.
Copying keeps every worktree independent, so an app that rewrites its config at runtime cannot corrupt a file shared across the pool.
Alternative considered: symlink to the canonical file for a single source of truth. Rejected as the unsafe default given config files are sometimes written to.

### Enumeration and copy live behind a small seam
`git.ListIgnoredFiles(sourceRepo)` returns the ignored file list, and a new `internal/localsync` package owns glob filtering plus the copy loop.
The glob set is threaded into the shared `acquire` core through a new `acquireOptions` field, matching how lease/holder/hook writers are already carried, so `Acquire` and `AcquireLease` forward it without duplicating the pool-scan body.

## Risks / Trade-offs

- [A local file is untracked but not gitignored] then it is intentionally not synced, and the developer's problem persists. Mitigation: document that only ignored files are mirrored and that the fix is to gitignore the file, which is the correct convention for local-only config anyway. This is preferred over silently making worktrees dirty.
- [Sync latency on acquire] a `git ls-files` invocation plus a handful of small file copies is negligible next to the existing `git fetch` and `git worktree add`, and the enumeration is empty-fast when nothing matches.
- [Repo-level config now influences file copying] the influence is bounded to copying files that already exist and are already ignored in the developer's own clone; no path outside the source working tree is read and no command is executed, so the risk profile is far below that of hooks.
- [A stale synced file lingers after the source file is deleted] a subsequent acquire overwrites present files but does not prune removed ones. Accepted for v1 as low-impact; deletion-sync can be revisited if it proves confusing.
- [Windows path handling] enumeration uses `git ls-files` (slash-separated, portable) and copying uses `filepath.Join` plus Go stdlib file I/O, with no hardcoded separators, per the project's Windows-compatibility rules.
