## 1. Git ref resolution

- [x] 1.1 Add a helper in `internal/git/git.go` that resolves a caller-supplied ref: verify it with `git rev-parse --verify <ref>` and use it directly, falling back to `branchRef` expansion only for a bare branch name
- [x] 1.2 Update `AddWorktree` and `ResetWorktree` so they can base the worktree on a caller-supplied ref while preserving today's default-branch behavior when none is given
- [x] 1.3 Add unit tests in `internal/git/git_test.go` covering branch name, remote-tracking ref, tag, commit SHA, and an unresolvable ref

## 2. Pool acquire override

- [x] 2.1 Add a `startRef string` field to `acquireOptions` in `internal/pool/pool.go`
- [x] 2.2 In `acquire`, use `startRef` when set, otherwise resolve via `git.GetDefaultBranch`; pass the chosen ref to both the create and reuse/reset paths
- [x] 2.3 Add a `startRef` parameter to `Acquire` and `AcquireLease` and forward it into `acquireOptions`
- [x] 2.4 Validate the override up front (after fetch, before any worktree mutation) so an invalid ref fails cleanly with no partial setup
- [x] 2.5 Extend `internal/pool/pool_test.go` to cover acquiring a new worktree on a ref, resetting a reused worktree to a ref, and rejecting an invalid ref

## 3. CLI flag

- [x] 3.1 Add a `--branch <ref>` flag to `getCmd` in `cmd/get.go` with help text noting it selects the starting ref and is not sticky across return
- [x] 3.2 Pass the flag value through both the interactive `getRunE` (`pool.Acquire`) and `getLeaseRunE` (`pool.AcquireLease`) paths
- [x] 3.3 Confirm `return`/`Release` is unchanged so a returned worktree still resets to the default branch

## 4. Verification

- [x] 4.1 `go build ./...` and `GOOS=windows go build ./...` both succeed
- [x] 4.2 `go test ./...` passes
- [x] 4.3 Manually verify `treehouse get --branch <ref>`, `treehouse get --lease --branch <ref>`, default acquire (no flag), and invalid-ref rejection
- [x] 4.4 Update `README.md`/`docs` to document the `--branch` flag
