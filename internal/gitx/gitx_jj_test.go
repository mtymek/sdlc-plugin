package gitx

import (
	"errors"
	"strings"
	"testing"

	"github.com/rnagrodzki/sdlc-plugin/internal/jjtest"
	"github.com/rnagrodzki/sdlc-plugin/internal/mcpserver"
)

// jjRepoDir makes a folder that DetectVCS reads as a colocated jj repo.
func jjRepoDir(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	mk(t, d, ".git", ".jj")
	return d
}

func fakeJJBookmarks(t *testing.T, byRev map[string]string) {
	t.Helper()
	old := runJJ
	runJJ = func(_ string, args ...string) (string, error) {
		if len(args) > 0 && args[0] == "--version" {
			return "jj 0.46.0", nil
		}
		for i, a := range args {
			if a == "-r" && i+1 < len(args) {
				return byRev[args[i+1]], nil
			}
		}
		return "", nil
	}
	t.Cleanup(func() { runJJ = old })
}

func TestCurrentBranchJJFake(t *testing.T) {
	cases := []struct {
		name string
		revs map[string]string
		want string
	}{
		{"on @", map[string]string{"@": "feat/x\n"}, "feat/x"},
		{"on @-", map[string]string{"@-": "feat/y\n"}, "feat/y"},
		{"none", map[string]string{}, "HEAD"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fakeJJBookmarks(t, c.revs)
			got, err := CurrentBranch(jjRepoDir(t))
			if err != nil || got != c.want {
				t.Fatalf("got %q, %v; want %q", got, err, c.want)
			}
		})
	}
	t.Run("two bookmarks", func(t *testing.T) {
		fakeJJBookmarks(t, map[string]string{"@": "a\nb\n"})
		_, err := CurrentBranch(jjRepoDir(t))
		var de *mcpserver.DomainError
		if !errors.As(err, &de) || de.Msg != "gitx: change @ has 2 bookmarks: a, b" ||
			de.Suggestion != "Keep one bookmark on this change with jj bookmark delete <name> or jj bookmark move <name> --to <change>, then retry." {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("bookmark named HEAD", func(t *testing.T) {
		fakeJJBookmarks(t, map[string]string{"@": "HEAD\n"})
		_, err := CurrentBranch(jjRepoDir(t))
		var de *mcpserver.DomainError
		if !errors.As(err, &de) || de.Msg != `gitx: bookmark "HEAD" is a reserved name` ||
			de.Suggestion != "Rename it with jj bookmark rename HEAD <name>, then retry." {
			t.Fatalf("got %v", err)
		}
	})
}

func TestCurrentBranchJJReal(t *testing.T) {
	jjtest.Require(t)
	d := jjtest.NewColocatedRepo(t)
	// The helper leaves bookmark main on @-.
	if got, err := CurrentBranch(d); err != nil || got != "main" {
		t.Fatalf("parent bookmark: got %q, %v", got, err)
	}
	jjtest.Run(t, d, "bookmark", "create", "feat/x", "-r", "@")
	if got, err := CurrentBranch(d); err != nil || got != "feat/x" {
		t.Fatalf("on @: got %q, %v", got, err)
	}
	jjtest.Run(t, d, "new")
	if got, err := CurrentBranch(d); err != nil || got != "feat/x" {
		t.Fatalf("on @-: got %q, %v", got, err)
	}
	jjtest.Run(t, d, "bookmark", "create", "other", "-r", "@")
	if got, err := CurrentBranch(d); err != nil || got != "other" {
		t.Fatalf("@ wins: got %q, %v", got, err)
	}
}

func TestCurrentBranchJJMergeFake(t *testing.T) {
	t.Run("duplicate name kept once", func(t *testing.T) {
		fakeJJBookmarks(t, map[string]string{"@-": "feat/x\n\nfeat/x\n"})
		got, err := CurrentBranch(jjRepoDir(t))
		if err != nil || got != "feat/x" {
			t.Fatalf("got %q, %v", got, err)
		}
	})
	t.Run("two different names across parents", func(t *testing.T) {
		fakeJJBookmarks(t, map[string]string{"@-": "zeta\n\nalpha\n"})
		_, err := CurrentBranch(jjRepoDir(t))
		var de *mcpserver.DomainError
		if !errors.As(err, &de) || de.Msg != "gitx: change @- has 2 bookmarks: alpha, zeta" {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("names sorted and deduplicated in message", func(t *testing.T) {
		fakeJJBookmarks(t, map[string]string{"@": "b\na\nb\n"})
		_, err := CurrentBranch(jjRepoDir(t))
		var de *mcpserver.DomainError
		if !errors.As(err, &de) || de.Msg != "gitx: change @ has 2 bookmarks: a, b" {
			t.Fatalf("got %v", err)
		}
	})
}

func TestCurrentBranchJJLogFailure(t *testing.T) {
	d := jjRepoDir(t)
	old := runJJ
	runJJ = func(_ string, args ...string) (string, error) {
		if len(args) > 0 && args[0] == "--version" {
			return "jj 0.46.0", nil
		}
		return "", errors.New("boom")
	}
	t.Cleanup(func() { runJJ = old })
	_, err := CurrentBranch(d)
	var ie *mcpserver.InfraError
	if !errors.As(err, &ie) {
		t.Fatalf("got %T %v", err, err)
	}
	if ie.Msg != "gitx: jj log -r @: boom" ||
		ie.Suggestion != "Run jj log -r @ in the repository root to see the error, fix it, then retry." {
		t.Fatalf("got %q / %q", ie.Msg, ie.Suggestion)
	}
}

func TestCurrentBranchJJMergeReal(t *testing.T) {
	jjtest.Require(t)
	d := jjtest.NewColocatedRepo(t)
	jjtest.Run(t, d, "bookmark", "create", "a", "-r", "@-")
	jjtest.Run(t, d, "new", "main", "-m", "other")
	jjtest.Run(t, d, "bookmark", "create", "b", "-r", "@")
	jjtest.Run(t, d, "new", "a", "b")
	_, err := CurrentBranch(d)
	var de *mcpserver.DomainError
	if !errors.As(err, &de) || !strings.HasPrefix(de.Msg, "gitx: change @- has ") || !strings.Contains(de.Msg, "a, b") {
		t.Fatalf("got %v", err)
	}
}

func TestDefaultAndBaseBranchColocatedJJ(t *testing.T) {
	jjtest.Require(t)
	d := jjtest.NewColocatedRepo(t)
	if got, err := DefaultBranch(d); err != nil || got != "main" {
		t.Fatalf("DefaultBranch: got %q, %v", got, err)
	}
	if got, err := BaseBranch(d, ""); err != nil || got != "main" {
		t.Fatalf("BaseBranch empty: got %q, %v", got, err)
	}
	if got, err := BaseBranch(d, " develop "); err != nil || got != "develop" {
		t.Fatalf("BaseBranch configured: got %q, %v", got, err)
	}
}
