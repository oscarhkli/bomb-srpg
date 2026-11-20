package server

import (
	"bomb-srpg/engine"
	"bomb-srpg/room"
	"slices"
	"testing"
)

type recordingDecider struct {
	plan          func(gs *engine.GameState, call int) []engine.TurnCommand
	err           error
	calls         int
	observedBombs int
	called        chan struct{}
}

func (d *recordingDecider) decide(gs *engine.GameState) ([]engine.TurnCommand, error) {
	call := d.calls
	if call == 0 {
		d.observedBombs = len(gs.Bombs)
	}
	d.calls++

	select {
	case d.called <- struct{}{}:
	default:
	}

	if d.plan == nil {
		return nil, d.err
	}
	return d.plan(gs, call), d.err
}

func vsCpuGameCfg() engine.GameCfg {
	gameCfg := validGameCfg()
	gameCfg.VSCpu = true
	return gameCfg
}

func hasGameEvent(gameEvents []engine.GameEvent, evtType engine.GameEvtType, unitID engine.UnitID) bool {
	return slices.ContainsFunc(gameEvents, func(evt engine.GameEvent) bool {
		return evt.Type == evtType && evt.UnitID == unitID
	})
}

type heldCPUTurn struct {
	t    *testing.T
	fns  []func()
	next int
}

func holdCPUTurn(t *testing.T) *heldCPUTurn {
	t.Helper()
	return &heldCPUTurn{t: t}
}

func (h *heldCPUTurn) spawn(f func()) { h.fns = append(h.fns, f) }

func (h *heldCPUTurn) count() int { return len(h.fns) }

func (h *heldCPUTurn) runOutsideRoomLock() {
	h.t.Helper()
	if h.next >= len(h.fns) {
		h.t.Fatal("no held CPU turn to run")
	}
	f := h.fns[h.next]
	h.next++
	f()
}

func moveThenBombPlan(t *testing.T, unitID engine.UnitID, dest engine.Coordinate) func(*engine.GameState, int) []engine.TurnCommand {
	t.Helper()
	return func(gs *engine.GameState, _ int) []engine.TurnCommand {
		t.Helper()
		return []engine.TurnCommand{
			engine.NewMoveCommand(unitID, dest),
			engine.NewPlaceBombCommand(unitID, freeNeighbour(t, gs, dest)),
		}
	}
}

func newCPUTurnHandlerRoom(t *testing.T, opts ...Option) (string, [2]string, *ServerStateManager, *Handler) {
	t.Helper()
	roomID, tokens, s := newCPUTurnRoom(t, opts...)
	return roomID, tokens, s, NewHandler(s)
}

func consumeCPUPhase(t *testing.T, s *ServerStateManager, roomID, token string) (room.CPUTurnPhase, []engine.GameEvent, []engine.GameEvent) {
	t.Helper()
	status, err := mustRoom(t, s, roomID).ConsumeCPU(token)
	if err != nil {
		t.Fatalf("ConsumeCPU() error = %v", err)
	}
	return status.Phase, status.Plan, status.Resolve
}

func newCPUTurnRoom(t *testing.T, opts ...Option) (string, [2]string, *ServerStateManager) {
	t.Helper()
	roomID, tokens, s := createTestRoomWithCfg(t, vsCpuGameCfg(), append([]Option{arrangedMatch(cpuTurnPending)}, opts...)...)
	return roomID, tokens, s
}
