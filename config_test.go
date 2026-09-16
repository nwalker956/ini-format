package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".inifmtrc")
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfigExplicitPath(t *testing.T) {
	path := writeConfig(t, "sort = true\ndiff = false\n")

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}
	if !cfg.sort {
		t.Error("cfg.sort = false, want true")
	}
	if cfg.write || cfg.diff || cfg.check {
		t.Errorf("unset keys should stay false, got %+v", cfg)
	}
}

func TestLoadConfigMissingExplicitPath(t *testing.T) {
	_, err := loadConfig(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("loadConfig with a missing explicit path returned nil error, want a missing-file error")
	}
}

func TestLoadConfigMissingDefaultPath(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig("")
	if err != nil {
		t.Fatalf("loadConfig returned error for a missing default config: %v", err)
	}
	if cfg != (config{}) {
		t.Errorf("cfg = %+v, want zero value when no config file is present", cfg)
	}
}

func TestLoadConfigDiscoversDefaultPath(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, defaultConfigName), []byte("w = true\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig("")
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}
	if !cfg.write {
		t.Errorf("cfg.write = false, want true from %s", defaultConfigName)
	}
}

func TestLoadConfigDupeCheck(t *testing.T) {
	path := writeConfig(t, "dupe-check = true\n")

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}
	if !cfg.dupeCheck {
		t.Error("cfg.dupeCheck = false, want true")
	}
}

func TestLoadConfigUnknownKey(t *testing.T) {
	path := writeConfig(t, "banana = true\n")

	if _, err := loadConfig(path); err == nil {
		t.Fatal("loadConfig with an unknown key returned nil error, want an error")
	}
}

func TestLoadConfigBadBoolValue(t *testing.T) {
	path := writeConfig(t, "sort = maybe\n")

	_, err := loadConfig(path)
	if err == nil {
		t.Fatal("loadConfig with a non-boolean value returned nil error, want an error")
	}
	if !strings.Contains(err.Error(), "sort") {
		t.Errorf("error %q doesn't mention the offending key", err)
	}
}
