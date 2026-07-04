package localsync

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupIgnoringRepo(t *testing.T, gitignore string) string {
	t.Helper()
	repoDir := t.TempDir()

	mustGit(t, "", "init", "--initial-branch=main", repoDir)
	mustGit(t, repoDir, "config", "user.email", "test@test.com")
	mustGit(t, repoDir, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(repoDir, ".gitignore"), []byte(gitignore), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, repoDir, "add", ".")
	mustGit(t, repoDir, "commit", "-m", "initial")

	return repoDir
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func TestSyncCopiesNestedMatchToRelativePath(t *testing.T) {
	repoDir := setupIgnoringRepo(t, "*.local.json\n")
	if err := os.MkdirAll(filepath.Join(repoDir, "Api"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "Api", "appsettings.local.json"), []byte(`{"a":1}`), 0o644); err != nil {
		t.Fatal(err)
	}

	worktree := t.TempDir()
	if err := Sync(repoDir, worktree, []string{"appsettings*.local.json"}); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(worktree, "Api", "appsettings.local.json"))
	if err != nil {
		t.Fatalf("expected synced file: %v", err)
	}
	if string(got) != `{"a":1}` {
		t.Fatalf("unexpected synced content: %q", got)
	}
}

func TestSyncHonorsMultipleGlobs(t *testing.T) {
	repoDir := setupIgnoringRepo(t, "*.local.json\n.env.local\n")
	if err := os.WriteFile(filepath.Join(repoDir, "appsettings.local.json"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, ".env.local"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}

	worktree := t.TempDir()
	if err := Sync(repoDir, worktree, []string{"appsettings*.local.json", ".env.local"}); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(worktree, "appsettings.local.json")); err != nil {
		t.Fatalf("expected appsettings.local.json synced: %v", err)
	}
	if _, err := os.Stat(filepath.Join(worktree, ".env.local")); err != nil {
		t.Fatalf("expected .env.local synced: %v", err)
	}
}

func TestSyncEmptyGlobSetIsNoop(t *testing.T) {
	repoDir := setupIgnoringRepo(t, "*.local.json\n")
	if err := os.WriteFile(filepath.Join(repoDir, "appsettings.local.json"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}

	worktree := t.TempDir()
	if err := Sync(repoDir, worktree, nil); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	entries, err := os.ReadDir(worktree)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no files copied, got %v", entries)
	}
}

func TestSyncNoMatchIsNoop(t *testing.T) {
	repoDir := setupIgnoringRepo(t, "*.local.json\n")
	if err := os.WriteFile(filepath.Join(repoDir, "appsettings.local.json"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}

	worktree := t.TempDir()
	if err := Sync(repoDir, worktree, []string{"*.does-not-match"}); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	entries, err := os.ReadDir(worktree)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no files copied, got %v", entries)
	}
}

func TestSyncOverwritesExistingDestinationOnReuse(t *testing.T) {
	repoDir := setupIgnoringRepo(t, "*.local.json\n")
	target := filepath.Join(repoDir, "appsettings.local.json")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	worktree := t.TempDir()
	if err := os.WriteFile(filepath.Join(worktree, "appsettings.local.json"), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(target, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Sync(repoDir, worktree, []string{"appsettings*.local.json"}); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(worktree, "appsettings.local.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("expected overwritten content %q, got %q", "new", got)
	}
}
