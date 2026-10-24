package tools

import (
	"errors"
	"fmt"
	"testing"

	"github.com/rnagrodzki/sdlc-plugin/internal/gitx"
	"github.com/rnagrodzki/sdlc-plugin/internal/mcpserver"
)

// stubVCSDetect replaces the detector for one test.
func stubVCSDetect(t *testing.T, f func(dir string) (gitx.VCSInfo, error)) {
	t.Helper()
	old := vcsDetect
	vcsDetect = f
	t.Cleanup(func() { vcsDetect = old })
}

func TestJJWriteStop_GitModeAllows(t *testing.T) {
	stubVCSDetect(t, func(string) (gitx.VCSInfo, error) { return gitx.VCSInfo{Mode: gitx.ModeGit}, nil })
	if err := jjWriteStop("/x", jjOpCommitApply); err != nil {
		t.Fatalf("git mode must not stop: %v", err)
	}
}

func TestJJWriteStop_DetectorErrorReturned(t *testing.T) {
	want := errors.New("detect failed")
	stubVCSDetect(t, func(string) (gitx.VCSInfo, error) { return gitx.VCSInfo{}, want })
	if err := jjWriteStop("/x", jjOpBaseSync); !errors.Is(err, want) {
		t.Fatalf("want detector error, got %v", err)
	}
}

func TestJJWriteStop_JJModeTable(t *testing.T) {
	stubVCSDetect(t, func(string) (gitx.VCSInfo, error) { return gitx.VCSInfo{Mode: gitx.ModeJJ}, nil })
	rows := []struct {
		op         jjWriteOp
		suggestion string
	}{
		{jjOpCommitApply, `Commit by hand: jj commit <paths> -m "<message>", then jj bookmark set <bookmark> -r @-.`},
		{jjOpWaveCommit, `Set execute.commitWaves = false in .sdlc-v2/config.toml and re-run; in jj mode the wave changes stay in @ for you to commit by hand.`},
		{jjOpBaseSync, `Rebase by hand: jj git fetch, then jj rebase -d <base>@origin. Or set execute.baseSync = false.`},
		{jjOpBaseSyncResolve, `There is nothing to resolve in jj mode: base-sync makes no merge there. Do not call base-sync-resolve; continue with the next wave.`},
		{jjOpCommitCheck, `Commit by hand with jj commit <paths> -m "<message>" and jj bookmark set <bookmark> -r @-, then resume ship.`},
		{jjOpOpenspecSave, `Save by hand: jj bookmark create openspec/<change> -r @, then jj commit openspec/changes/<change> -m "<message>", then jj git push -b openspec/<change>.`},
	}
	if len(rows) != len(jjStopSuggestions) {
		t.Fatalf("table has %d rows, stop table has %d", len(rows), len(jjStopSuggestions))
	}
	for _, r := range rows {
		t.Run(string(r.op), func(t *testing.T) {
			err := jjWriteStop("/x", r.op)
			var de *mcpserver.DomainError
			if !errors.As(err, &de) {
				t.Fatalf("want DomainError, got %T %v", err, err)
			}
			wantMsg := fmt.Sprintf("%s: this is a jj repository; the plugin does not write in jj mode yet", r.op)
			if de.Msg != wantMsg {
				t.Errorf("Msg = %q, want %q", de.Msg, wantMsg)
			}
			if de.Suggestion != r.suggestion {
				t.Errorf("Suggestion = %q, want %q", de.Suggestion, r.suggestion)
			}
		})
	}
}
