# Treehouse - Agent Guide

## What is this?

Treehouse is a Go CLI tool that manages a pool of git worktrees for parallel AI coding agent workflows. It maintains reusable, pre-warmed worktrees so agents get isolated environments instantly.

## Project Structure

- `main.go` - entry point, calls `cmd.Execute()`
- `cmd/` - CLI commands (cobra): `get` (incl. `get --lease`), `enter`, `return`, `status`, `prune`, `destroy`
- `internal/config/` - config file loading (`treehouse.toml`)
- `internal/hooks/` - user-configured lifecycle hook command execution
- `internal/localsync/` - copies gitignored local files (e.g. `appsettings*.local.json`) from the source repo into an acquired worktree
- `internal/pool/` - pool manager (acquire, release, list, destroy, prune) + state file
- `internal/git/` - git operations (shells out to `git` binary)
- `internal/process/` - in-use detection and lingering process termination for worktrees
- `internal/shell/` - subshell spawning
- `internal/ui/` - Y/n confirmation prompts

## Building

```sh
go build -o treehouse .
# or
make build
```

## Testing

```sh
go test ./...
# or
make test
```

## Key Design Decisions

- No daemon - all operations are inline CLI commands
- Detached HEAD worktrees reset to whichever of local or origin default branch is further ahead (prefers origin on divergence)
- In-use detection uses process scanning plus short-lived persisted owner reservations for lifecycle operations
- Durable leases are a separate, process-independent reservation: `WorktreeEntry.Leased`/`LeaseID`/`LeaseHolder`/`LeasedAt` persist in the state file with `omitempty`. Every acquisition generates a new immutable 128-bit random `LeaseID`; older state without it loads with an empty ID and remains releasable through the legacy unconditional path. A lease is NOT derived from live processes, so it survives with zero processes inside the worktree and `healState` never clears it. Leased worktrees are skipped by `Acquire` and `prune`, classified `DestroyLeased` by destroy (removable only when the exact path is named with `--include-leased`, NEVER via `--all`), surfaced by `status` as `StatusLeased`, and cleared by `Release` (`return`)
- `destroy` is safe-by-default and mirrors `prune`: dry-run unless `--yes`, narrow explicit targets (`destroy <path>` for one worktree; `destroy <pool> --all` for that pool only - there is NO cross-pool/global destroy, and `--all` with no pool target is an error). The old blunt `--force` flag is REMOVED (this was the v2.0.0 breaking change); each risk class is its own opt-in: `--include-unlanded` (dirty, unmerged, or unverified), `--include-in-use` (running process or owner reservation; processes terminated cleanly first), `--include-leased` (leased, single named path only). A bare `--all --yes` removes only the disposable set (merged, clean, idle, unleased) and skips the rest with the flag that would include each. Bulk skips exit 0; a single-target skip exits non-zero. Entry points: `pool.DestroyWorktree` (single path, `allowLeased=true`) and `pool.DestroyPool` (bulk, `allowLeased=false`). Both share `classifyForDestroy` in `internal/pool/destroy.go`, which reuses prune's classification primitives (`ownerAlive`, `process.FindProcessesInWorktree`, `backingRepositoryMissing`, `git.IsDirty`, `git.IsHeadMergedIntoRef` against the `resolvePruneDefaultRef` ref) so destroy and prune agree on leased/in-use/unlanded/unverified/disposable. Removal keeps the same two-phase reservation as prune (reserve under flock, run `pre_destroy` hooks, remove only worktrees whose `sameDestroyReservation` still holds), so a worktree re-acquired during its hook is never deleted
- `get --lease` (see `getLeaseRunE`) is the non-interactive acquire: it opens no subshell, routes hook output and banners to stderr, and keeps path-only stdout unchanged. `--lease-holder`/`$TREEHOUSE_LEASE_HOLDER` set the recorded holder. `get --lease --json` returns `pool.AcquireLeaseInfo`, and `status --json` exposes the same `lease_id`, holder, and timestamp. Conditional return uses `pool.ReleaseConditional` with `--if-lease-id` and optional `--if-lease-holder`; comparison, caller-side preparation, reset, and final clear share one `WithStateLock`, while return without conditions keeps the legacy path-only behavior. `pool.AcquireLease`/`pool.AcquireLeaseInfo` are the entry points; both they and `Acquire` delegate to the shared `acquire(..., acquireOptions)` core, and `markAcquired` stamps either a lease or an owner reservation
- `get --branch <ref>` overrides the starting ref for the acquired worktree (interactive and `--lease`). `cmd/get.go` threads the flag into `pool.Acquire`/`pool.AcquireLease`/`pool.AcquireLeaseInfo` as a trailing `startRef` argument, carried through `acquireOptions.startRef`. The shared `acquire` core resolves the ref once after the `origin` fetch via `resolveStartRef` (empty means default branch), so a bad ref fails before any worktree is created or reset, and the same resolved ref feeds both the create (`git.AddWorktree`) and reuse (`git.ResetWorktree`) paths. `git.ResolveStartRef` verifies the ref with `rev-parse --verify` and uses it directly (branch, tag, SHA, or remote-tracking ref), falling back to the local-vs-origin `branchRef` expansion only for a bare branch name; the default branch goes through `git.DefaultBranchRef`. The override is per-acquire, not sticky: `Release` (`return`) still resets to the default branch. No `$TREEHOUSE_BRANCH` fallback; worktrees stay detached (no branch creation)
- Dirty checks include untracked files even when repository config hides them from normal `git status` output
- Prune deletes only idle managed worktrees that are clean and whose HEAD is merged into the default branch; dry run is the default
- Prune reports unsafe idle worktrees in grouped, stable categories and keeps raw git diagnostics for verbose output instead of default output
- Prune treats backing-repository-missing linked worktrees as orphans; they are only deletable with explicit `--prune-orphans --yes`, and each candidate warns that content could not be verified
- Prune never treats an unreachable origin as a deletable orphan; those worktrees stay skipped because the repository may still be valid
- Global prune enumerates managed pool directories under the user-level treehouse root and derives each worktree's owning repository from git metadata instead of relying on the current directory
- Global prune loads user-level config and hooks only because it can run without a repository context
- State file tracks pool membership, temporary owner/destroy reservations, and explicit durable leases.
  It still does not infer long-term usage from processes.
- `WriteState` is atomic: it writes to a temp file in the pool directory, fsyncs it, commits it with the platform replacement primitive, and syncs the parent directory where supported.
  A crash mid-write can never leave a truncated or empty state file.
  `ReadState` treats a state file that exists but fails to parse (empty or truncated) as recoverable rather than a hard failure: it prints a loud warning to stderr and rebuilds a `State` by scanning the pool directory for worktree subdirectories still on disk (`recoverCorruptState` in `internal/pool/state.go`).
  Since the real reservation (owner vs. lease vs. idle) is unknowable from disk alone, every recovered entry is marked `Leased` with a `recoveredLeaseHolder` placeholder.
  `Acquire` and `prune` skip recovered entries, and `destroy` only removes one via a single named `--include-leased` target.
  A human clears a recovered entry with `treehouse status` then `treehouse return` (or `destroy --include-leased`) once verified
- Git operations shell out to `git` (go-git has incomplete worktree support)
- Self-healing: stale state entries are auto-removed
- Local file sync (`internal/localsync`) mirrors gitignored local dev config (default glob `appsettings*.local.json`, configurable via `sync_ignored`) into every acquired worktree, on both the fresh-create and reuse/reset paths, after create/reset and before `post_create` hooks. The load-bearing invariant: only files git reports as ignored (`git.ListIgnoredFiles`, i.e. `git ls-files --others --ignored --exclude-standard`) are ever copied - never a broader untracked-but-not-ignored set - so `git status --porcelain` stays clean and `git clean -fd` still preserves the file, meaning the sync can never interfere with dirty detection, `Acquire` reuse eligibility, `return`'s dirty prompt, or prune/destroy classification. Copies overwrite existing destination files on every acquire (copy, not symlink, since config files may be written to at runtime and worktrees must stay independent). Unlike `Hooks`, `SyncIgnored` runs no commands, so `config.Load` honors it from both repo- and user-level `treehouse.toml`, with repo config winning only when it explicitly sets the key (`config.Load` tracks `repoSetSyncIgnored` via the TOML decode metadata so an unset repo key falls through to the user-level value instead of silently zeroing it)

## Windows Compatibility

This project targets Linux, macOS, and Windows. All new code **must** work on Windows. Follow these rules:

- **Paths**: Never hardcode `/` as a path separator. Use `filepath.Join()`, `filepath.Separator`, or `filepath.ToSlash()` as appropriate.
- **Shell**: Do not assume `/bin/sh` or `$SHELL` exist. On Windows, use `%COMSPEC%` (usually `cmd.exe`). See `internal/shell/shell.go` for the pattern.
- **Syscalls**: Unix-only syscalls (e.g., `syscall.Flock`) must be isolated behind build tags (`//go:build !windows` / `//go:build windows`). See `internal/pool/lock_unix.go` and `lock_windows.go` for the pattern.
- **Build tags**: Follow the existing `_unix.go` / `_windows.go` naming convention (see also `internal/updater/sysproc_*.go`).
- **CI**: The CI matrix runs tests on `ubuntu`, `macOS`, and `windows`. Cross-compile locally with `GOOS=windows go build ./...` to catch issues early.
- **Process detection**: `gopsutil` is cross-platform - no special handling needed, but avoid importing platform-specific process APIs directly.

## Config

Place repo-safe settings in repo root `treehouse.toml` or user-level `~/.config/treehouse/config.toml`:

```toml
max_trees = 16

# Optional worktree root.
# Relative roots need a repo context; use an absolute user-level root for global prune.
# root = "$HOME/worktrees"

# Basename globs of gitignored local files to mirror into every acquired
# worktree. Defaults to ["appsettings*.local.json"] when unset; [] disables it.
# sync_ignored = ["appsettings*.local.json", ".env.local"]

# User-level config only:
[hooks]
post_create = ["./scripts/setup-venv.sh"]
pre_destroy = ["./scripts/teardown.sh"]
```

Hooks are ignored in repo-level config for safety. `sync_ignored` is honored in both repo- and user-level config since it only ever copies files git already ignores and never runs commands.

## Maintaining this file

Keep this file for knowledge useful to almost every future agent session in this project.
Do not repeat what the codebase already shows; point to the authoritative file or command instead.
Prefer rewriting or pruning existing entries over appending new ones.
When updating this file, preserve this bar for all agents and keep entries concise.
