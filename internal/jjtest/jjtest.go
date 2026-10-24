// Package jjtest holds test helpers for tests that need a real jj binary.
package jjtest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// RequireEnv, when set to "1", turns a missing jj binary into a test failure
// instead of a skip. CI sets it so a broken jj install cannot hide coverage.
const RequireEnv = "SDLC_REQUIRE_JJ"

// Require skips t when jj is not on PATH, or fails it when RequireEnv is "1".
func Require(t testing.TB) {
	t.Helper()
	require(t, exec.LookPath, os.Getenv)
}

// require is Require with swappable lookups, so tests reach both branches.
func require(t testing.TB, lookPath func(string) (string, error), getenv func(string) string) {
	t.Helper()
	_, err := lookPath("jj")
	if err == nil {
		return
	}
	if getenv(RequireEnv) == "1" {
		t.Fatalf("jj not found on PATH but %s=1: %v", RequireEnv, err)
		return
	}
	t.Skipf("jj not found on PATH: %v", err)
}

// NewColocatedRepo returns the root of a colocated jj repo under t.TempDir.
// It writes a JJ_CONFIG file with a user name and email and sets JJ_CONFIG
// with t.Setenv, so jj never reads the user's own settings.
func NewColocatedRepo(t testing.TB) string {
	t.Helper()
	Require(t)
	dir := t.TempDir()
	cfg := filepath.Join(t.TempDir(), "jj-config.toml")
	content := "[user]\nname = \"Test User\"\nemail = \"test@example.com\"\n"
	if err := os.WriteFile(cfg, []byte(content), 0o600); err != nil {
		t.Fatalf("write jj config: %v", err)
	}
	t.Setenv("JJ_CONFIG", cfg)
	t.Setenv("GIT_AUTHOR_NAME", "Test User")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test User")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	Run(t, dir, "git", "init", "--colocate")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("init\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	Run(t, dir, "describe", "-m", "initial")
	Run(t, dir, "bookmark", "create", "main", "-r", "@")
	Run(t, dir, "new")
	return dir
}

// Run runs jj in dir and fails t on error; returns trimmed stdout.
func Run(t testing.TB, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("jj", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		stderr := ""
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = string(ee.Stderr)
		}
		t.Fatalf("jj %s: %v\n%s", strings.Join(args, " "), err, stderr)
	}
	return strings.TrimSpace(string(out))
}
