package gitx

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/rnagrodzki/sdlc-plugin/internal/execx"
	"github.com/rnagrodzki/sdlc-plugin/internal/mcpserver"
)

// VCSMode names the version control system that owns a working folder.
type VCSMode string

const (
	// ModeGit is a plain git repository, or a folder in no repository.
	ModeGit VCSMode = "git"
	// ModeJJ is a jj repository colocated with git (.jj and .git side by side).
	ModeJJ VCSMode = "jj"
)

// MinJJVersion is the oldest jj the plugin accepts: CI pins this release and
// runs the jj tests against it, so older releases are untested.
const MinJJVersion = "0.46.0"

// minJJShort is MinJJVersion without the patch part, for user-facing text.
var minJJShort = strings.TrimSuffix(MinJJVersion, ".0")

const jjOutputCap = 200 // characters of bad jj output shown in an error

// VCSInfo is the result of DetectVCS.
type VCSInfo struct {
	Mode      VCSMode // git or jj
	Root      string  // folder that holds the markers; "" when none found
	JJVersion string  // "0.46.0" in jj mode, "" in git mode
}

// runJJ runs jj in dir; tests replace it.
var runJJ = func(dir string, args ...string) (string, error) {
	return execx.Run("jj", args, execx.Options{Dir: dir})
}

// statMarker checks one marker path; tests replace it to return a permission error.
var statMarker = os.Stat

var jjVersionRe = regexp.MustCompile(`^jj (\d+)\.(\d+)\.(\d+)`)

// markerExists reports whether path exists. A not-found error is false, nil.
func markerExists(path string) (bool, error) {
	_, err := statMarker(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, &mcpserver.InfraError{
		Msg:        fmt.Sprintf("gitx: cannot inspect %s: %v", path, err),
		Suggestion: fmt.Sprintf("Check the folder permissions of %s, then retry.", path),
		Cause:      err,
	}
}

func nonColocated(root string) error {
	return &mcpserver.DomainError{
		Msg:        fmt.Sprintf("gitx: %s is a jj repository that is not colocated with git", root),
		Suggestion: "Run `jj git init --colocate` in the repository root.",
	}
}

// findMarkers walks up from dir and returns the first folder holding .git or
// .jj, with which markers it holds. root is "" when none is found.
func findMarkers(dir string) (root string, hasGit, hasJJ bool, err error) {
	cur, err := filepath.Abs(dir)
	if err != nil {
		return "", false, false, &mcpserver.InfraError{
			Msg:        fmt.Sprintf("gitx: cannot resolve %s: %v", dir, err),
			Suggestion: "Pass an existing folder path, then retry.",
			Cause:      err,
		}
	}
	for {
		hasGit, err = markerExists(filepath.Join(cur, ".git"))
		if err != nil {
			return "", false, false, err
		}
		hasJJ, err = markerExists(filepath.Join(cur, ".jj"))
		if err != nil {
			return "", false, false, err
		}
		if hasGit || hasJJ {
			return cur, hasGit, hasJJ, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", false, false, nil
		}
		cur = parent
	}
}

// NonColocatedError returns the non-colocated DomainError when the nearest
// repository folder above dir holds .jj without .git, else nil.
func NonColocatedError(dir string) error {
	root, hasGit, hasJJ, err := findMarkers(dir)
	if err != nil {
		return err
	}
	if root != "" && hasJJ && !hasGit {
		return nonColocated(root)
	}
	return nil
}

// DetectVCS walks up from dir; the first folder with .git or .jj decides the
// mode. With neither marker it returns git mode with an empty Root.
func DetectVCS(dir string) (VCSInfo, error) {
	root, hasGit, hasJJ, err := findMarkers(dir)
	if err != nil {
		return VCSInfo{}, err
	}
	if root == "" {
		return VCSInfo{Mode: ModeGit}, nil
	}
	if !hasJJ {
		return VCSInfo{Mode: ModeGit, Root: root}, nil
	}
	if !hasGit {
		return VCSInfo{}, nonColocated(root)
	}
	ver, err := checkJJVersion(root)
	if err != nil {
		return VCSInfo{}, err
	}
	return VCSInfo{Mode: ModeJJ, Root: root, JJVersion: ver}, nil
}

func checkJJVersion(root string) (string, error) {
	out, err := runJJ(root, "--version")
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", &mcpserver.InfraError{
				Msg:        fmt.Sprintf("gitx: %s is a jj repository, but jj is not on PATH", root),
				Suggestion: fmt.Sprintf("Install jj %s or newer.", minJJShort),
				Cause:      err,
			}
		}
		return "", &mcpserver.InfraError{
			Msg:        fmt.Sprintf("gitx: jj --version: %v", err),
			Suggestion: "Run jj --version in the repository root, fix the jj install, then retry.",
			Cause:      err,
		}
	}
	out = strings.TrimSpace(out)
	m := jjVersionRe.FindStringSubmatch(out)
	if m == nil {
		shown := out
		if r := []rune(shown); len(r) > jjOutputCap {
			shown = string(r[:jjOutputCap]) + "…"
		}
		return "", &mcpserver.DataError{
			Msg:        fmt.Sprintf("gitx: cannot read the jj version from %q", shown),
			Suggestion: fmt.Sprintf("Run jj --version; reinstall jj %s or newer if it does not print a version.", minJJShort),
		}
	}
	// The Atoi errors are ignored: jjVersionRe matches only digit runs, and
	// MinJJVersion is a constant of three numbers.
	got := [3]int{}
	for i := range got {
		got[i], _ = strconv.Atoi(m[i+1])
	}
	version := fmt.Sprintf("%d.%d.%d", got[0], got[1], got[2])
	minParts := strings.Split(MinJJVersion, ".")
	for i := range got {
		want, _ := strconv.Atoi(minParts[i])
		if got[i] > want {
			break
		}
		if got[i] < want {
			return "", &mcpserver.InfraError{
				Msg:        fmt.Sprintf("gitx: jj %s is older than %s", version, MinJJVersion),
				Suggestion: fmt.Sprintf("Upgrade jj to %s or newer.", minJJShort),
			}
		}
	}
	return version, nil
}
