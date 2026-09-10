package cpu

import (
	"bomb-srpg/engine"
	"testing"
)

func TestDecide_ReturnsEmptySlice(t *testing.T) {
	gs := &engine.GameState{}
	if got, want := Decide(gs), 0; len(got) != want {
		t.Errorf("Expected 0 TurnCommands return, got %+v", got)
	}
}
