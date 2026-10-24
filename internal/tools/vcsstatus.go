package tools

import (
	"fmt"
	"strings"

	"github.com/rnagrodzki/sdlc-plugin/internal/config"
	"github.com/rnagrodzki/sdlc-plugin/internal/gitx"
)

// VCSStatusOut is the version-control state the execute and ship skills read
// before they start. It is built by vcsStatus for both tools.
type VCSStatusOut struct {
	Mode          string `json:"mode" jsonschema_description:"git, jj, or unknown when detection failed (see warnings)."`
	JJVersion     string `json:"jjVersion,omitempty" jsonschema_description:"jj version in jj mode, e.g. 0.46.0."`
	Branch        string `json:"branch" jsonschema_description:"Current branch or jj bookmark; HEAD when there is none; empty after a branch-lookup error (see warnings)."`
	NeedsBookmark bool   `json:"needsBookmark" jsonschema_description:"jj mode only: true when the run must stop until the user creates a feature bookmark."`
}

// VCSModeOut is the output of ship_state vcs-mode.
type VCSModeOut struct {
	VCSStatusOut
	Next string `json:"next" jsonschema_description:"What the ship skill does next in this mode."`
}

const (
	vcsModeUnknown = "unknown"

	execNextUnknown       = "Stop: the version-control check failed. Print the warning, fix it, then re-run /execute."
	execNextBookmarkFail  = "jj mode: stop. The bookmark check failed. Print the warning, follow its suggestion, then re-run /execute."
	execNextGit           = "Continue Step 0 with the plan-argument gate."
	execNextJJCommitWaves = "jj mode: stop. Wave commits do not run in jj repositories yet. Set execute.commitWaves = false in .sdlc-v2/config.toml, then re-run /execute."
	execNextJJNoBookmark  = "jj mode: stop. Create a feature bookmark with jj bookmark create <name> -r @, then re-run /execute."
	execNextJJContinue    = "jj mode: continue on bookmark %s. Use vcs.branch for workspace detection; skip the pre-execution rebase and the base-sync stage."

	shipNextGit = "git mode: continue with step 3 as written."
	shipNextJJ  = "jj mode: stop. Ship does not run in jj repositories yet. Run /execute with execute.commitWaves = false, commit with jj commit, push with jj git push -b <bookmark>, then run /pr."
)

// Branch seams; tests replace them.
var (
	vcsBranch        = gitx.CurrentBranch
	vcsDefaultBranch = gitx.DefaultBranch
	vcsBaseBranch    = gitx.BaseBranch
)

// vcsStatus reads the VCS mode and branch for dir. root anchors the config
// read of the base branch. A detector error returns Mode "unknown" and the
// error. A branch lookup error returns the mode, an empty branch and the
// error. needsBookmark is true only in jj mode, when the branch is "HEAD",
// the default branch or the base branch.
//
// A default-branch lookup error is not returned: gitx.DefaultBranch hides
// git failures and errors only when the repository has no default branch,
// so the bookmark cannot be on it. A base-branch error with [git] baseBranch
// set means the configured value is invalid; it is returned with the mode
// and branch, and the caller stops.
func vcsStatus(dir, root string) (VCSStatusOut, error) {
	info, err := vcsDetect(dir)
	if err != nil {
		return VCSStatusOut{Mode: vcsModeUnknown}, err
	}
	out := VCSStatusOut{Mode: string(info.Mode), JJVersion: info.JJVersion}
	branch, err := vcsBranch(dir)
	if err != nil {
		return out, err
	}
	out.Branch = branch
	if info.Mode != gitx.ModeJJ {
		return out, nil
	}
	if branch == "HEAD" {
		out.NeedsBookmark = true
		return out, nil
	}
	if def, derr := vcsDefaultBranch(dir); derr == nil && branch == def {
		out.NeedsBookmark = true
	}
	configured := config.GitBaseBranch(root)
	base, berr := vcsBaseBranch(dir, configured)
	if berr != nil && strings.TrimSpace(configured) != "" {
		return out, berr
	}
	if berr == nil && branch == base {
		out.NeedsBookmark = true
	}
	return out, nil
}

// execResolveNext picks the resolve-config next text: the first matching row
// of the table, top to bottom. branchErr is the vcsStatus error, if any.
func execResolveNext(v VCSStatusOut, branchErr error, commitWaves bool) string {
	switch {
	case v.Mode == vcsModeUnknown:
		return execNextUnknown
	case v.Mode == string(gitx.ModeJJ) && branchErr != nil:
		return execNextBookmarkFail
	case v.Mode == string(gitx.ModeGit):
		return execNextGit
	case commitWaves:
		return execNextJJCommitWaves
	case v.NeedsBookmark:
		return execNextJJNoBookmark
	default:
		return fmt.Sprintf(execNextJJContinue, v.Branch)
	}
}

// shipVCSMode is the read-only ship_state vcs-mode action. A detector or
// branch error is returned as the typed error.
func shipVCSMode(root, workDir string) (any, error) {
	st, err := vcsStatus(workDir, root)
	if err != nil {
		return nil, err
	}
	next := shipNextGit
	if st.Mode == string(gitx.ModeJJ) {
		next = shipNextJJ
	}
	return VCSModeOut{VCSStatusOut: st, Next: next}, nil
}
