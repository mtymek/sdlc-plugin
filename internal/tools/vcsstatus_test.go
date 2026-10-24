package tools

import (
	"errors"
	"strings"
	"testing"

	"github.com/rnagrodzki/sdlc-plugin/internal/gitx"
	"github.com/rnagrodzki/sdlc-plugin/internal/mcpserver"
)

// stubVCS replaces the detector and the branch seams for one test.
func stubVCS(t *testing.T, info gitx.VCSInfo, detErr error, branch string, brErr error, def, base string) {
	t.Helper()
	stubVCSDetect(t, func(string) (gitx.VCSInfo, error) { return info, detErr })
	oldB, oldD, oldBase := vcsBranch, vcsDefaultBranch, vcsBaseBranch
	vcsBranch = func(string) (string, error) { return branch, brErr }
	vcsDefaultBranch = func(string) (string, error) { return def, nil }
	vcsBaseBranch = func(string, string) (string, error) { return base, nil }
	t.Cleanup(func() { vcsBranch, vcsDefaultBranch, vcsBaseBranch = oldB, oldD, oldBase })
}

var (
	jjInfo  = gitx.VCSInfo{Mode: gitx.ModeJJ, JJVersion: "0.46.0"}
	gitInfo = gitx.VCSInfo{Mode: gitx.ModeGit}
)

func TestVCSStatus_NeedsBookmark(t *testing.T) {
	rows := []struct {
		name   string
		info   gitx.VCSInfo
		branch string
		want   bool
	}{
		{"jj feature bookmark", jjInfo, "feat/x", false},
		{"jj detached", jjInfo, "HEAD", true},
		{"jj default branch", jjInfo, "main", true},
		{"jj base branch", jjInfo, "develop", true},
		{"git detached", gitInfo, "HEAD", false},
		{"git default branch", gitInfo, "main", false},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			stubVCS(t, r.info, nil, r.branch, nil, "main", "develop")
			got, err := vcsStatus("/x", "/x")
			if err != nil {
				t.Fatal(err)
			}
			if got.NeedsBookmark != r.want || got.Branch != r.branch || got.Mode != string(r.info.Mode) || got.JJVersion != r.info.JJVersion {
				t.Fatalf("got %+v, want needsBookmark=%v", got, r.want)
			}
		})
	}
}

func TestVCSStatus_Errors(t *testing.T) {
	det := errors.New("detect failed")
	stubVCS(t, gitx.VCSInfo{}, det, "", nil, "", "")
	got, err := vcsStatus("/x", "/x")
	if !errors.Is(err, det) || got.Mode != "unknown" {
		t.Fatalf("detector error: got %+v, %v", got, err)
	}
	br := errors.New("two bookmarks")
	stubVCS(t, jjInfo, nil, "", br, "main", "main")
	got, err = vcsStatus("/x", "/x")
	if !errors.Is(err, br) || got.Mode != "jj" || got.Branch != "" || got.NeedsBookmark {
		t.Fatalf("branch error: got %+v, %v", got, err)
	}
}

func TestVCSStatus_ResolveConfigNextTable(t *testing.T) {
	br := errors.New("two bookmarks")
	rows := []struct {
		name        string
		info        gitx.VCSInfo
		detErr      error
		branch      string
		brErr       error
		commitWaves bool
		wantMode    string
		wantNext    string
	}{
		{"unknown", gitx.VCSInfo{}, errors.New("boom"), "", nil, true, "unknown", execNextUnknown},
		{"jj branch lookup failed", jjInfo, nil, "", br, false, "jj", execNextBookmarkFail},
		{"git", gitInfo, nil, "main", nil, true, "git", execNextGit},
		{"jj commitWaves", jjInfo, nil, "feat/x", nil, true, "jj", execNextJJCommitWaves},
		{"jj needs bookmark", jjInfo, nil, "HEAD", nil, false, "jj", execNextJJNoBookmark},
		{"jj continue", jjInfo, nil, "feat/x", nil, false, "jj", "jj mode: continue on bookmark feat/x. Use vcs.branch for workspace detection; skip the pre-execution rebase and the base-sync stage."},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			stubVCS(t, r.info, r.detErr, r.branch, r.brErr, "main", "main")
			root := t.TempDir()
			out, err := execActionResolveConfig(root, root, ExecuteStateIn{CommitWaves: map[bool]string{true: "true", false: "false"}[r.commitWaves]})
			if err != nil {
				t.Fatal(err)
			}
			res := out.(ExecuteResolveConfigOut)
			if res.VCS.Mode != r.wantMode || res.Next != r.wantNext {
				t.Fatalf("mode=%q next=%q; want %q / %q", res.VCS.Mode, res.Next, r.wantMode, r.wantNext)
			}
			if (r.detErr != nil || r.brErr != nil) != containsAny(res.Warnings, "boom", "two bookmarks") {
				t.Fatalf("warnings = %v", res.Warnings)
			}
		})
	}
}

func containsAny(ws []string, subs ...string) bool {
	for _, w := range ws {
		for _, s := range subs {
			if strings.Contains(w, s) {
				return true
			}
		}
	}
	return false
}

func TestVCSStatus_ShipVCSModeTable(t *testing.T) {
	stubVCS(t, gitInfo, nil, "main", nil, "main", "main")
	out, err := shipState(t.TempDir(), "/x", ShipStateIn{Action: "vcs-mode"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := out.(VCSModeOut); got.Next != shipNextGit || got.Mode != "git" {
		t.Fatalf("git: %+v", got)
	}
	stubVCS(t, jjInfo, nil, "feat/x", nil, "main", "main")
	out, err = shipState(t.TempDir(), "/x", ShipStateIn{Action: "vcs-mode"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := out.(VCSModeOut); got.Next != shipNextJJ || got.JJVersion != "0.46.0" {
		t.Fatalf("jj: %+v", got)
	}
}

func TestVCSStatus_ParityDetectorError(t *testing.T) {
	want := &mcpserver.InfraError{Msg: "gitx: jj is not on PATH", Suggestion: "Install jj."}
	stubVCS(t, gitx.VCSInfo{}, want, "", nil, "", "")
	root := t.TempDir()
	if _, err := shipState(root, root, ShipStateIn{Action: "vcs-mode"}, nil); !errors.Is(err, want) {
		t.Fatalf("vcs-mode must return the typed error, got %v", err)
	}
	out, err := execActionResolveConfig(root, root, ExecuteStateIn{})
	if err != nil {
		t.Fatal(err)
	}
	res := out.(ExecuteResolveConfigOut)
	if !containsAny(res.Warnings, want.Error()) || res.Next != execNextUnknown || res.VCS.Mode != "unknown" {
		t.Fatalf("resolve-config: %+v", res)
	}
}
