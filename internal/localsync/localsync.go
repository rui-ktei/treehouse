// Package localsync mirrors gitignored local files (such as
// appsettings*.local.json) from a source repository's working tree into an
// acquired worktree, so worktrees can be run locally without manual setup.
package localsync

import (
	"io"
	"os"
	"path/filepath"

	"github.com/kunchenguid/treehouse/internal/git"
)

// Sync copies files that git reports as ignored in sourceRepo and whose
// basename matches one of globs into worktreePath, preserving each file's path
// relative to sourceRepo and creating parent directories as needed. Existing
// destination files are overwritten. An empty glob set or no matching ignored
// file is a no-op.
func Sync(sourceRepo, worktreePath string, globs []string) error {
	if len(globs) == 0 {
		return nil
	}

	ignored, err := git.ListIgnoredFiles(sourceRepo)
	if err != nil {
		return err
	}

	for _, rel := range ignored {
		if !matchesAny(filepath.Base(rel), globs) {
			continue
		}
		if err := copyFile(
			filepath.Join(sourceRepo, filepath.FromSlash(rel)),
			filepath.Join(worktreePath, filepath.FromSlash(rel)),
		); err != nil {
			return err
		}
	}

	return nil
}

func matchesAny(name string, globs []string) bool {
	for _, glob := range globs {
		if ok, err := filepath.Match(glob, name); err == nil && ok {
			return true
		}
	}
	return false
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
