package server

import (
	"bomb-srpg/engine"
	"bomb-srpg/room"
	"testing"
	"time"
)

func TestServerStateManager_ConsumeCPUStatus_DefaultSpawn(t *testing.T) {
	cpuUnitID := engine.NewUnitID(2, 0)
	decider := &recordingDecider{
		plan: func(*engine.GameState, int) []engine.TurnCommand {
			return []engine.TurnCommand{engine.NewMoveCommand(cpuUnitID, engine.Coordinate{X: 4, Y: 1})}
		},
		called: make(chan struct{}, 1),
	}
	roomID, tokens, s := newCPUTurnRoom(t, WithDecider(decider.decide))

	if phase, _, _ := consumeCPUPhase(t, s, roomID, tokens[0]); phase != room.TurnPhasePlanning {
		t.Fatalf("phase = %v, want %v", phase, room.TurnPhasePlanning)
	}
	select {
	case <-decider.called:
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for the CPU goroutine to invoke decide")
	}

	if got, want := mustMatchState(t, s, roomID).ActiveTeam, 1; got != want {
		t.Errorf("active team after the turn = %d, want %d", got, want)
	}

	phase, planGameEvents, _ := consumeCPUPhase(t, s, roomID, tokens[0])
	if phase != room.TurnPhaseReady {
		t.Errorf("phase after the turn = %v, want %v", phase, room.TurnPhaseReady)
	}
	if !hasGameEvent(planGameEvents, engine.GameEvtUnitMoved, cpuUnitID) {
		t.Errorf("Expected unit %#x to have moved, got %#v", cpuUnitID, planGameEvents)
	}
}
