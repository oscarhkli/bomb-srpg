package server

import (
	"testing"

	"bomb-srpg/engine"
)

func TestWithMatchFactory_UsedByCreateMatchAndRematch(t *testing.T) {
	var built []*engine.Match
	factory := func(gameCfg engine.GameCfg) (*engine.Match, error) {
		match, err := engine.InitGame(gameCfg)
		built = append(built, match)
		return match, err
	}
	s := NewServerStateManager(WithMatchFactory(factory))
	roomID, err := s.createMatchRoom()
	if err != nil {
		t.Fatalf("createMatchRoom() error = %v", err)
	}
	room := mustRoom(t, s, roomID)

	tokens := mustCreateMatch(t, s, roomID, validGameCfg())
	if got, want := len(built), 1; got != want {
		t.Fatalf("factory calls after createMatch = %d, want %d", got, want)
	}
	if room.Match != built[0] {
		t.Errorf("room holds %p after createMatch, want the factory's match %p", room.Match, built[0])
	}

	if _, err := s.rematch(roomID, tokens[0]); err != nil {
		t.Fatalf("rematch() error = %v", err)
	}
	if got, want := len(built), 2; got != want {
		t.Fatalf("factory calls after rematch = %d, want %d", got, want)
	}
	if room.Match != built[1] {
		t.Errorf("room holds %p after rematch, want the factory's match %p", room.Match, built[1])
	}
}
