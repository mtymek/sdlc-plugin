package jjtest

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type fakeTB struct {
	testing.TB
	skipped, failed bool
	msg             string
}

func (f *fakeTB) Helper() {}
func (f *fakeTB) Skipf(format string, args ...any) {
	f.skipped = true
	panic(f)
}
func (f *fakeTB) Fatalf(format string, args ...any) {
	f.failed = true
	f.msg = format
	panic(f)
}

func callRequire(env string, found bool) (f *fakeTB) {
	f = &fakeTB{}
	defer func() {
		if r := recover(); r != nil && r != f {
			panic(r)
		}
	}()
	look := func(string) (string, error) {
		if found {
			return "/bin/jj", nil
		}
		return "", errors.New("missing")
	}
	require(f, look, func(string) string { return env })
	return
}

func TestRequire_Branches(t *testing.T) {
	if f := callRequire("", true); f.skipped || f.failed {
		t.Fatalf("found: %+v", f)
	}
	if f := callRequire("", false); !f.skipped || f.failed {
		t.Fatalf("missing without env should skip: %+v", f)
	}
	if f := callRequire("1", false); !f.failed || f.skipped {
		t.Fatalf("missing with env should fail: %+v", f)
	}
}

func TestNewColocatedRepo(t *testing.T) {
	dir := NewColocatedRepo(t)
	if _, err := os.Stat(filepath.Join(dir, ".jj")); err != nil {
		t.Fatalf(".jj missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		t.Fatalf(".git missing: %v", err)
	}
	out := Run(t, dir, "bookmark", "list")
	if !strings.Contains(out, "main") {
		t.Fatalf("main bookmark missing: %q", out)
	}
	gitOut, err := exec.Command("git", "-C", dir, "branch", "--list", "main").Output()
	if err != nil || !strings.Contains(string(gitOut), "main") {
		t.Fatalf("git does not see main: %q %v", gitOut, err)
	}
}
