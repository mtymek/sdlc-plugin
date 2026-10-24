package tools

import (
	"fmt"

	"github.com/rnagrodzki/sdlc-plugin/internal/gitx"
	"github.com/rnagrodzki/sdlc-plugin/internal/mcpserver"
)

// jjWriteOp names a plugin write operation that stops in jj mode.
type jjWriteOp string

const (
	jjOpCommitApply     jjWriteOp = "commit_apply"
	jjOpWaveCommit      jjWriteOp = "execute_state wave-commit"
	jjOpBaseSync        jjWriteOp = "execute_state base-sync"
	jjOpBaseSyncResolve jjWriteOp = "execute_state base-sync-resolve"
	jjOpCommitCheck     jjWriteOp = "ship_state commit-check"
	jjOpOpenspecSave    jjWriteOp = "openspec_save"
)

// vcsDetect is the VCS detector; tests replace it.
var vcsDetect = gitx.DetectVCS

// jjStopSuggestions holds the per-op suggestion. The stop message itself is
// built from the op name. An op with no entry gets jjStopFallback.
var jjStopSuggestions = map[jjWriteOp]string{
	jjOpCommitApply:     `Commit by hand: jj commit <paths> -m "<message>", then jj bookmark set <bookmark> -r @-.`,
	jjOpWaveCommit:      `Set execute.commitWaves = false in .sdlc-v2/config.toml and re-run; in jj mode the wave changes stay in @ for you to commit by hand.`,
	jjOpBaseSync:        `Rebase by hand: jj git fetch, then jj rebase -d <base>@origin. Or set execute.baseSync = false.`,
	jjOpBaseSyncResolve: `There is nothing to resolve in jj mode: base-sync makes no merge there. Do not call base-sync-resolve; continue with the next wave.`,
	jjOpCommitCheck:     `Commit by hand with jj commit <paths> -m "<message>" and jj bookmark set <bookmark> -r @-, then resume ship.`,
	jjOpOpenspecSave:    `Save by hand: jj bookmark create openspec/<change> -r @, then jj commit openspec/changes/<change> -m "<message>", then jj git push -b openspec/<change>.`,
}

// jjStopFallback is the suggestion for an op missing from jjStopSuggestions,
// so the stop never carries an empty suggestion.
const jjStopFallback = "Make this change by hand with jj (jj commit, jj bookmark set, jj git push), or run the step in a git repository."

// jjWriteStop returns nil in git mode; in jj mode a DomainError for op. A
// detector failure is returned as is. Call it before the first write.
func jjWriteStop(dir string, op jjWriteOp) error {
	info, err := vcsDetect(dir)
	if err != nil {
		return err
	}
	if info.Mode != gitx.ModeJJ {
		return nil
	}
	sugg, ok := jjStopSuggestions[op]
	if !ok {
		sugg = jjStopFallback
	}
	return &mcpserver.DomainError{
		Msg:        fmt.Sprintf("%s: this is a jj repository; the plugin does not write in jj mode yet", op),
		Suggestion: sugg,
	}
}
