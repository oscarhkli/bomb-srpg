package server

import (
	"testing"

	"bomb-srpg/engine"
)

func TestWithMatchFactory_UsedByCreateMatchAndRematch(t *testing.T) {
	var built []*engine.Match
	factory := func(gameCfg engine.GameCfg) (*engine.Match, error) {
		match, err := engine.InitGame(gameCfg)
		if err != nil {
			return nil, err
		}
		match.WorkingState.Turn = len(built) + 2
		built = append(built, match)
		return match, nil
	}
	s := NewServerStateManager(WithMatchFactory(factory))
	roomID, err := s.createMatchRoom()
	if err != nil {
		t.Fatalf("createMatchRoom() error = %v", err)
	}

	tokens := mustCreateMatch(t, s, roomID, validGameCfg())
	if got, want := len(built), 1; got != want {
		t.Fatalf("factory calls after createMatch = %d, want %d", got, want)
	}
	if got, want := mustMatchState(t, s, roomID).Turn, 2; got != want {
		t.Errorf("turn after createMatch = %d, want %d from the factory's match", got, want)
	}

	if _, err := s.rematch(roomID, tokens[0]); err != nil {
		t.Fatalf("rematch() error = %v", err)
	}
	if got, want := len(built), 2; got != want {
		t.Fatalf("factory calls after rematch = %d, want %d", got, want)
	}
	if got, want := mustMatchState(t, s, roomID).Turn, 3; got != want {
		t.Errorf("turn after rematch = %d, want %d from the factory's match", got, want)
	}
}
