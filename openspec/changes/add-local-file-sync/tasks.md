## 1. Ignored-file enumeration

- [x] 1.1 Add `git.ListIgnoredFiles(sourceRepo string) ([]string, error)` in `internal/git/git.go` running `git ls-files --others --ignored --exclude-standard`, returning slash-separated relative paths
- [x] 1.2 Add unit tests in `internal/git/git_test.go` covering a repo with ignored files, a repo with none, and confirming tracked/untracked-non-ignored files are excluded

## 2. Local sync package

- [x] 2.1 Create `internal/localsync` with a function that takes the source repo root, the worktree path, and the glob set, enumerates ignored files, filters by basename glob, and copies matches into the worktree preserving relative paths and creating parent dirs
- [x] 2.2 Overwrite existing destination files; skip nothing on the basis of the destination already existing (freshness on reuse)
- [x] 2.3 Make an empty glob set and a no-match result a clean no-op with no output and no error
- [x] 2.4 Ensure Windows safety: no hardcoded separators, `filepath.Join` for destinations, Go stdlib copy with parent-dir creation
- [x] 2.5 Add unit tests: nested match copied to correct relative path, multiple globs, empty glob set disables, no-match no-op, overwrite-on-reuse

## 3. Config surface

- [x] 3.1 Add `SyncIgnored []string` with toml key `sync_ignored` to `Config` in `internal/config/config.go`
- [x] 3.2 Default it to `["appsettings*.local.json"]` in `DefaultConfig()`
- [x] 3.3 Load `SyncIgnored` from repo-level config (do not strip it the way `Hooks` is stripped) and from user-level config; confirm an absent key preserves the default and `sync_ignored = []` disables
- [x] 3.4 Add config tests covering repo-level set, user-level set, absent-key-keeps-default, and empty-list-disables

## 4. Wire into acquire

- [x] 4.1 Add a `syncIgnored []string` field to `acquireOptions` in `internal/pool/pool.go`
- [x] 4.2 In `acquire`, after a successful create or reuse/reset and before `hooks.Run`, invoke the local sync with the source repo root, the acquired worktree path, and `opts.syncIgnored`
- [x] 4.3 Thread the glob set through `Acquire` and `AcquireLease` into `acquireOptions`
- [x] 4.4 Pass `cfg.SyncIgnored` from both `getRunE` and `getLeaseRunE` in `cmd/get.go`
- [x] 4.5 Extend `internal/pool/pool_test.go` to verify a fresh acquire and a reuse acquire both land the synced file, and that a synced ignored file does not make the worktree read as dirty (acquire still reuses it, return does not prompt)

## 5. Verification

- [x] 5.1 `go build ./...` and `GOOS=windows go build ./...` both succeed
- [x] 5.2 `go test ./...` passes
- [x] 5.3 Manually verify: create an ignored `appsettings.local.json` in a repo, run `treehouse get`, confirm the file appears in the worktree; run `treehouse status`/`return` and confirm the worktree is still treated as clean
- [x] 5.4 Manually verify the no-op case (repo with no matching files) produces no output and no error
- [x] 5.5 Manually verify `sync_ignored = []` disables and a repo-level `sync_ignored` override is honored
- [x] 5.6 Update `README.md`/docs with the `sync_ignored` key, its default, disabling, repo-level allowance, and the ignored-only rule
