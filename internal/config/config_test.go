package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoad_SyncIgnoredDefaultsWhenAbsent(t *testing.T) {
	repoDir := t.TempDir()
	setUserHome(t, t.TempDir())

	cfg, err := Load(repoDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	want := []string{"appsettings*.local.json"}
	if !reflect.DeepEqual(cfg.SyncIgnored, want) {
		t.Errorf("SyncIgnored: got %v, want %v", cfg.SyncIgnored, want)
	}
}

func TestLoad_SyncIgnoredEmptyListDisables(t *testing.T) {
	repoDir := t.TempDir()
	setUserHome(t, t.TempDir())

	cfgTOML := `sync_ignored = []`
	if err := os.WriteFile(filepath.Join(repoDir, "treehouse.toml"), []byte(cfgTOML), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(repoDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(cfg.SyncIgnored) != 0 {
		t.Errorf("expected SyncIgnored disabled, got %v", cfg.SyncIgnored)
	}
}

func TestLoad_SyncIgnoredHonorsRepoLevelOverride(t *testing.T) {
	repoDir := t.TempDir()
	setUserHome(t, t.TempDir())

	cfgTOML := `sync_ignored = ["appsettings*.local.json", ".env.local"]`
	if err := os.WriteFile(filepath.Join(repoDir, "treehouse.toml"), []byte(cfgTOML), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(repoDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	want := []string{"appsettings*.local.json", ".env.local"}
	if !reflect.DeepEqual(cfg.SyncIgnored, want) {
		t.Errorf("SyncIgnored: got %v, want %v", cfg.SyncIgnored, want)
	}
}

func TestLoad_SyncIgnoredHonorsUserLevelOverrideWithoutRepoConfig(t *testing.T) {
	repoDir := t.TempDir()
	userHome := t.TempDir()
	setUserHome(t, userHome)

	configDir := filepath.Join(userHome, ".config", "treehouse")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgTOML := `sync_ignored = [".env.local"]`
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(cfgTOML), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(repoDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	want := []string{".env.local"}
	if !reflect.DeepEqual(cfg.SyncIgnored, want) {
		t.Errorf("SyncIgnored: got %v, want %v", cfg.SyncIgnored, want)
	}
}
