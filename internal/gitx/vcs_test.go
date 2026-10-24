package gitx

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rnagrodzki/sdlc-plugin/internal/jjtest"
	"github.com/rnagrodzki/sdlc-plugin/internal/mcpserver"
)

func fakeJJ(t *testing.T, out string, err error) {
	t.Helper()
	old := runJJ
	runJJ = func(string, ...string) (string, error) { return out, err }
	t.Cleanup(func() { runJJ = old })
}

func mk(t *testing.T, root string, names ...string) {
	t.Helper()
	for _, n := range names {
		if err := os.MkdirAll(filepath.Join(root, n), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDetectVCSMarkers(t *testing.T) {
	t.Run("no markers", func(t *testing.T) {
		// TempDir parents could hold a marker; stub stat to find none.
		old := statMarker
		statMarker = func(string) (fs.FileInfo, error) { return nil, fs.ErrNotExist }
		t.Cleanup(func() { statMarker = old })
		info, err := DetectVCS(t.TempDir())
		if err != nil || info.Mode != ModeGit || info.Root != "" {
			t.Fatalf("got %+v, %v", info, err)
		}
	})
	t.Run("git only never runs jj", func(t *testing.T) {
		old := runJJ
		runJJ = func(string, ...string) (string, error) { t.Fatal("jj ran"); return "", nil }
		t.Cleanup(func() { runJJ = old })
		d := t.TempDir()
		mk(t, d, ".git", "a/b")
		info, err := DetectVCS(filepath.Join(d, "a", "b"))
		if err != nil || info.Mode != ModeGit || info.Root != d || info.JJVersion != "" {
			t.Fatalf("got %+v, %v", info, err)
		}
	})
	t.Run("git file counts", func(t *testing.T) {
		d := t.TempDir()
		if err := os.WriteFile(filepath.Join(d, ".git"), []byte("gitdir: x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		info, err := DetectVCS(d)
		if err != nil || info.Mode != ModeGit || info.Root != d {
			t.Fatalf("got %+v, %v", info, err)
		}
	})
	t.Run("both gives jj", func(t *testing.T) {
		fakeJJ(t, "jj 0.46.0\n", nil)
		d := t.TempDir()
		mk(t, d, ".git", ".jj", "sub")
		info, err := DetectVCS(filepath.Join(d, "sub"))
		if err != nil || info.Mode != ModeJJ || info.Root != d || info.JJVersion != "0.46.0" {
			t.Fatalf("got %+v, %v", info, err)
		}
	})
	t.Run("nested git inside jj", func(t *testing.T) {
		old := runJJ
		runJJ = func(string, ...string) (string, error) { t.Fatal("jj ran"); return "", nil }
		t.Cleanup(func() { runJJ = old })
		d := t.TempDir()
		mk(t, d, ".git", ".jj", "inner/.git")
		inner := filepath.Join(d, "inner")
		info, err := DetectVCS(inner)
		if err != nil || info.Mode != ModeGit || info.Root != inner {
			t.Fatalf("got %+v, %v", info, err)
		}
	})
}

func TestNonColocatedError(t *testing.T) {
	d := t.TempDir()
	mk(t, d, ".jj", "sub")
	_, err := DetectVCS(filepath.Join(d, "sub"))
	var de *mcpserver.DomainError
	if !errors.As(err, &de) {
		t.Fatalf("want DomainError, got %v", err)
	}
	if de.Msg != "gitx: "+d+" is a jj repository that is not colocated with git" ||
		de.Suggestion != "Run `jj git init --colocate` in the repository root." {
		t.Fatalf("bad error %+v", de)
	}
	if e := NonColocatedError(d); !errors.As(e, &de) {
		t.Fatalf("NonColocatedError = %v", e)
	}
	g := t.TempDir()
	mk(t, g, ".git", ".jj")
	if e := NonColocatedError(g); e != nil {
		t.Fatalf("colocated should be nil, got %v", e)
	}
}

func TestDetectVCSJJVersions(t *testing.T) {
	for _, v := range []string{"0.46.0", "0.47.1", "1.0.0"} {
		fakeJJ(t, "jj "+v+"-rc1\n", nil)
		d := t.TempDir()
		mk(t, d, ".git", ".jj")
		info, err := DetectVCS(d)
		if err != nil || info.JJVersion != v {
			t.Fatalf("%s: got %+v, %v", v, info, err)
		}
	}
	long := strings.Repeat("x", 201)
	exact := strings.Repeat("y", 200)
	const dataSug = "Run jj --version; reinstall jj 0.46 or newer if it does not print a version."
	cases := []struct {
		name string
		out  string
		err  error
		typ  string
		msg  string
		sug  string
	}{
		{"old", "jj 0.45.9", nil, "infra", "gitx: jj 0.45.9 is older than 0.46.0", "Upgrade jj to 0.46 or newer."},
		{"garbage", "hello", nil, "data", `gitx: cannot read the jj version from "hello"`, dataSug},
		{"cap200", exact, nil, "data", `gitx: cannot read the jj version from "` + exact + `"`, dataSug},
		{"cap201", long, nil, "data", `gitx: cannot read the jj version from "` + strings.Repeat("x", 200) + `…"`, dataSug},
		{"missing", "", exec.ErrNotFound, "infra", "", "Install jj 0.46 or newer."},
		{"exit", "", errors.New("exit status 2"), "infra", "gitx: jj --version: exit status 2", "Run jj --version in the repository root, fix the jj install, then retry."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fakeJJ(t, c.out, c.err)
			d := t.TempDir()
			mk(t, d, ".git", ".jj")
			_, err := DetectVCS(d)
			if err == nil {
				t.Fatal("want error")
			}
			var sug, msg string
			var ok bool
			switch c.typ {
			case "infra":
				var e *mcpserver.InfraError
				if ok = errors.As(err, &e); ok {
					sug, msg = e.Suggestion, e.Msg
				}
			case "data":
				var e *mcpserver.DataError
				if ok = errors.As(err, &e); ok {
					sug, msg = e.Suggestion, e.Msg
				}
			}
			if !ok {
				t.Fatalf("wrong class: %T", err)
			}
			want := c.msg
			if c.name == "missing" {
				want = "gitx: " + d + " is a jj repository, but jj is not on PATH"
			}
			if msg != want || sug != c.sug {
				t.Fatalf("got %q / %q", msg, sug)
			}
		})
	}
}

func TestDetectVCSStatError(t *testing.T) {
	old := statMarker
	statMarker = func(string) (fs.FileInfo, error) { return nil, fs.ErrPermission }
	t.Cleanup(func() { statMarker = old })
	d := t.TempDir()
	_, err := DetectVCS(d)
	var e *mcpserver.InfraError
	if !errors.As(err, &e) {
		t.Fatalf("want InfraError, got %v", err)
	}
	p := filepath.Join(d, ".git")
	if !strings.Contains(e.Msg, "cannot inspect "+p) || !strings.Contains(e.Suggestion, p) {
		t.Fatalf("bad error %+v", e)
	}
}

func TestDetectVCSRealJJ(t *testing.T) {
	jjtest.Require(t)
	root := jjtest.NewColocatedRepo(t)
	mk(t, root, "deep/er")
	wantRoot, _ := filepath.EvalSymlinks(root)
	for _, dir := range []string{root, filepath.Join(root, "deep", "er")} {
		info, err := DetectVCS(dir)
		if err != nil || info.Mode != ModeJJ || info.JJVersion == "" {
			t.Fatalf("%s: got %+v, %v", dir, info, err)
		}
		if got, _ := filepath.EvalSymlinks(info.Root); got != wantRoot {
			t.Fatalf("root %s != %s", info.Root, root)
		}
	}
}
