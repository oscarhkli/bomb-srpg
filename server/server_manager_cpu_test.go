package server

import (
	"bomb-srpg/engine"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"
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

func holdCPUTurn(t *testing.T, s *ServerStateManager) *heldCPUTurn {
	t.Helper()
	h := &heldCPUTurn{t: t}
	s.spawn = func(f func()) { h.fns = append(h.fns, f) }
	return h
}

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

func setTeam2DueTurn(gs *engine.GameState) {
	gs.ActiveTeam, gs.Turn = 2, 4
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

func newCPUTurnHandlerRoom(t *testing.T) (string, [2]string, *ServerStateManager, *Handler) {
	t.Helper()
	roomID, tokens, s := newCPUTurnRoom(t)
	return roomID, tokens, s, NewHandler(s)
}

func consumeCPUPhase(t *testing.T, s *ServerStateManager, roomID, token string) (engine.CPUTurnPhase, []engine.GameEvent, []engine.GameEvent) {
	t.Helper()
	phase, planGameEvents, resolveTurnGameEvents, err := s.consumeCPUStatus(roomID, token)
	if err != nil {
		t.Fatalf("consumeCPUStatus() error = %v", err)
	}
	return phase, planGameEvents, resolveTurnGameEvents
}

func TestApplyPlan(t *testing.T) {
	moverUnitID := engine.NewUnitID(1, 0)
	bomberUnitID := engine.NewUnitID(1, 1)

	tests := []struct {
		name     string
		plan     func(t *testing.T, m *engine.Match) []engine.TurnCommand
		wantErr  error
		validate func(t *testing.T, m *engine.Match)
	}{
		{
			name: "Empty Plan",
			plan: func(t *testing.T, m *engine.Match) []engine.TurnCommand {
				t.Helper()
				return nil
			},
			validate: func(t *testing.T, m *engine.Match) {
				t.Helper()
				if got := len(m.PlaybackLog); got != 0 {
					t.Errorf("Expected no events logged, got %d", got)
				}
			},
		},
		{
			name: "Full Plan Applied",
			plan: func(t *testing.T, m *engine.Match) []engine.TurnCommand {
				t.Helper()
				return []engine.TurnCommand{
					engine.NewMoveCommand(moverUnitID, engine.Coordinate{X: 4, Y: 7}),
					engine.NewPlaceBombCommand(bomberUnitID, engine.Coordinate{X: 3, Y: 7}),
				}
			},
			validate: func(t *testing.T, m *engine.Match) {
				t.Helper()
				if !hasGameEvent(m.PlaybackLog, engine.GameEvtUnitMoved, moverUnitID) {
					t.Errorf("Expected unit %#x to have moved, got %#v", moverUnitID, m.PlaybackLog)
				}
				if !hasGameEvent(m.PlaybackLog, engine.GameEvtBombPlaced, bomberUnitID) {
					t.Errorf("Expected unit %#x to have placed a bomb, got %#v", bomberUnitID, m.PlaybackLog)
				}
			},
		},
		{
			name: "Stops At First Rejection",
			plan: func(t *testing.T, m *engine.Match) []engine.TurnCommand {
				t.Helper()
				return []engine.TurnCommand{
					engine.NewMoveCommand(moverUnitID, engine.Coordinate{X: 99, Y: 99}),
					engine.NewPlaceBombCommand(bomberUnitID, engine.Coordinate{X: 3, Y: 7}),
				}
			},
			wantErr: engine.ErrOutOfMoveRange,
			validate: func(t *testing.T, m *engine.Match) {
				t.Helper()
				if hasGameEvent(m.PlaybackLog, engine.GameEvtBombPlaced, bomberUnitID) {
					t.Errorf("Expected command after the rejected one to be skipped, got %#v", m.PlaybackLog)
				}
			},
		},
		{
			name: "Unsupported Command",
			plan: func(t *testing.T, m *engine.Match) []engine.TurnCommand {
				t.Helper()
				return []engine.TurnCommand{{UnitID: moverUnitID, Target: engine.Coordinate{X: 1, Y: 1}}}
			},
			wantErr: engine.ErrUnsupportedCommand,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, _, s := createTestRoom(t)
			match := mustRoom(t, s, roomID).Match

			err := applyPlan(match, tt.plan(t, match))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("applyPlan() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, match)
			}
		})
	}
}

func TestServerStateManager_ConsumeCPUStatus_LaunchesCPUTurn(t *testing.T) {
	humanUnitID := engine.NewUnitID(1, 0)
	cpuUnitID := engine.NewUnitID(2, 0)

	legalCPUMove := []engine.TurnCommand{engine.NewMoveCommand(cpuUnitID, engine.Coordinate{X: 4, Y: 1})}
	rejectedPlan := []engine.TurnCommand{engine.NewMoveCommand(humanUnitID, engine.Coordinate{X: 1, Y: 1})}

	cpuTurnPending := func(t *testing.T, room *matchRoom) int {
		t.Helper()
		room.Match.WorkingState.Turn = 2
		room.Match.WorkingState.ActiveTeam = 2
		room.Match.TrueState = room.Match.WorkingState.DeepCopy()
		return 1
	}

	tests := []struct {
		name           string
		vsCpu          bool
		maxTurns       int
		setup          func(t *testing.T, room *matchRoom) int
		plan           func(gs *engine.GameState, call int) []engine.TurnCommand
		wantCPUTurn    bool
		wantCalls      int
		wantActiveTeam int
		validate       func(t *testing.T, planGameEvents []engine.GameEvent, d *recordingDecider)
	}{
		{
			name:  "Plan Applied",
			vsCpu: true,
			setup: cpuTurnPending,
			plan: func(gs *engine.GameState, call int) []engine.TurnCommand {
				return legalCPUMove
			},
			wantCPUTurn:    true,
			wantCalls:      1,
			wantActiveTeam: 1,
			validate: func(t *testing.T, planGameEvents []engine.GameEvent, d *recordingDecider) {
				t.Helper()
				if !hasGameEvent(planGameEvents, engine.GameEvtUnitMoved, cpuUnitID) {
					t.Errorf("Expected unit %#x to have moved, got %#v", cpuUnitID, planGameEvents)
				}
			},
		},
		{
			name:  "Empty Plan",
			vsCpu: true,
			setup: cpuTurnPending,
			plan: func(gs *engine.GameState, call int) []engine.TurnCommand {
				return nil
			},
			wantCPUTurn:    true,
			wantCalls:      1,
			wantActiveTeam: 1,
			validate: func(t *testing.T, planGameEvents []engine.GameEvent, d *recordingDecider) {
				t.Helper()
				if got := len(planGameEvents); got != 0 {
					t.Errorf("Expected no pending events, got %d", got)
				}
			},
		},
		{
			name:  "Replan On Rejection",
			vsCpu: true,
			setup: cpuTurnPending,
			plan: func(gs *engine.GameState, call int) []engine.TurnCommand {
				if call == 0 {
					return rejectedPlan
				}
				return legalCPUMove
			},
			wantCPUTurn:    true,
			wantCalls:      2,
			wantActiveTeam: 1,
			validate: func(t *testing.T, planGameEvents []engine.GameEvent, d *recordingDecider) {
				t.Helper()
				if !hasGameEvent(planGameEvents, engine.GameEvtUnitMoved, cpuUnitID) {
					t.Errorf("Expected replanned move to apply, got %#v", planGameEvents)
				}
			},
		},
		{
			name:  "Replan Cap Exhausted",
			vsCpu: true,
			setup: cpuTurnPending,
			plan: func(gs *engine.GameState, call int) []engine.TurnCommand {
				return rejectedPlan
			},
			wantCPUTurn:    true,
			wantCalls:      5,
			wantActiveTeam: 1,
			validate: func(t *testing.T, planGameEvents []engine.GameEvent, d *recordingDecider) {
				t.Helper()
				if got := len(planGameEvents); got != 0 {
					t.Errorf("Expected no command to apply, got %#v", planGameEvents)
				}
			},
		},
		{
			name:     "Sudden Death Injected Before Planning",
			vsCpu:    true,
			maxTurns: 1,
			setup: func(t *testing.T, room *matchRoom) int {
				t.Helper()
				room.Match.ResolveTurn()
				return 1
			},
			plan: func(gs *engine.GameState, call int) []engine.TurnCommand {
				return nil
			},
			wantCPUTurn:    true,
			wantCalls:      1,
			wantActiveTeam: 1,
			validate: func(t *testing.T, planGameEvents []engine.GameEvent, d *recordingDecider) {
				t.Helper()
				if got, want := d.observedBombs, engine.SuddenDeathBombs; got < want {
					t.Errorf("Expected at least %d hazard bombs on the board when planning, got %d", want, got)
				}
			},
		},
		{
			name:  "Match Ended",
			vsCpu: true,
			setup: func(t *testing.T, room *matchRoom) int {
				t.Helper()
				tokenIdx := cpuTurnPending(t, room)
				room.Match.WinnerTeamID = 1
				return tokenIdx
			},
			wantCPUTurn:    false,
			wantCalls:      0,
			wantActiveTeam: 2,
		},
		{
			name:           "Not VS CPU",
			vsCpu:          false,
			setup:          cpuTurnPending,
			wantCPUTurn:    false,
			wantCalls:      0,
			wantActiveTeam: 2,
		},
		{
			name:           "Human Turn",
			vsCpu:          true,
			wantCPUTurn:    false,
			wantCalls:      0,
			wantActiveTeam: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gameCfg := validGameCfg()
			gameCfg.VSCpu = tt.vsCpu
			if tt.maxTurns != 0 {
				gameCfg.MaxTurns = tt.maxTurns
			}
			roomID, tokens, s := createTestRoomWithCfg(t, gameCfg)
			room := mustRoom(t, s, roomID)

			tokenIdx := 0
			if tt.setup != nil {
				tokenIdx = tt.setup(t, room)
			}

			decider := &recordingDecider{plan: tt.plan}
			s.decide = decider.decide
			held := holdCPUTurn(t, s)

			wantPhase, wantHeld := engine.TurnPhaseIdle, 0
			if tt.wantCPUTurn {
				wantPhase, wantHeld = engine.TurnPhasePlanning, 1
			}
			phase, planGameEvents, resolveTurnGameEvents := consumeCPUPhase(t, s, roomID, tokens[tokenIdx])
			if phase != wantPhase {
				t.Errorf("consumeCPUStatus() phase = %v, want %v", phase, wantPhase)
			}
			if len(planGameEvents) != 0 || len(resolveTurnGameEvents) != 0 {
				t.Errorf("consumeCPUStatus() events = %#v and %#v, want none", planGameEvents, resolveTurnGameEvents)
			}
			if got := held.count(); got != wantHeld {
				t.Errorf("held CPU turns = %d, want %d", got, wantHeld)
			}

			if tt.wantCPUTurn {
				held.runOutsideRoomLock()
				phase, planGameEvents, _ = consumeCPUPhase(t, s, roomID, tokens[tokenIdx])
				if phase != engine.TurnPhaseReady {
					t.Errorf("consumeCPUStatus() after the turn phase = %v, want %v", phase, engine.TurnPhaseReady)
				}
			}

			if got, want := decider.calls, tt.wantCalls; got != want {
				t.Errorf("Expected %d decide calls, got %d", want, got)
			}
			if got, want := room.Match.TrueState.ActiveTeam, tt.wantActiveTeam; got != want {
				t.Errorf("Expected active team %d, got %d", want, got)
			}
			if tt.validate != nil {
				tt.validate(t, planGameEvents, decider)
			}
		})
	}
}

func TestServerStateManager_ConsumeCPUStatus_StaleReadyDropped(t *testing.T) {
	cpuUnitID := engine.NewUnitID(2, 0)
	posA, posB := engine.Coordinate{X: 4, Y: 1}, engine.Coordinate{X: 3, Y: 1}

	roomID, tokens, s := newCPUTurnRoom(t)
	room := mustRoom(t, s, roomID)
	planA := moveThenBombPlan(t, cpuUnitID, posA)
	decider := &recordingDecider{plan: func(gs *engine.GameState, call int) []engine.TurnCommand {
		if call == 0 {
			return planA(gs, call)
		}
		return []engine.TurnCommand{engine.NewMoveCommand(cpuUnitID, posB)}
	}}
	s.decide = decider.decide
	held := holdCPUTurn(t, s)

	consumeCPUPhase(t, s, roomID, tokens[0])
	held.runOutsideRoomLock()

	setTeam2DueTurn(room.Match.TrueState)
	setTeam2DueTurn(room.Match.WorkingState)

	phase, planGameEvents, resolveTurnGameEvents := consumeCPUPhase(t, s, roomID, tokens[0])
	if phase != engine.TurnPhasePlanning {
		t.Fatalf("launching consume phase = %v, want %v", phase, engine.TurnPhasePlanning)
	}
	if len(planGameEvents) != 0 || len(resolveTurnGameEvents) != 0 {
		t.Errorf("launching consume events = %#v and %#v, want none", planGameEvents, resolveTurnGameEvents)
	}
	if got, want := held.count(), 2; got != want {
		t.Fatalf("held CPU turns = %d, want %d", got, want)
	}

	held.runOutsideRoomLock()
	phase, planGameEvents, _ = consumeCPUPhase(t, s, roomID, tokens[0])
	if phase != engine.TurnPhaseReady {
		t.Fatalf("phase after the second turn = %v, want %v", phase, engine.TurnPhaseReady)
	}
	found := false
	for _, evt := range planGameEvents {
		if evt.Type != engine.GameEvtUnitMoved || evt.UnitID != cpuUnitID {
			continue
		}
		found = true
		if evt.To == nil || *evt.To != posB {
			t.Errorf("move event To = %v, want %v", evt.To, posB)
		}
	}
	if !found {
		t.Errorf("Expected plan B's move event, got %#v", planGameEvents)
	}
}

func TestServerStateManager_ConsumeCPUStatus_ClearsResolveEvents(t *testing.T) {
	cpuUnitID := engine.NewUnitID(2, 0)
	roomID, tokens, s := newCPUTurnRoom(t)
	s.decide = (&recordingDecider{plan: moveThenBombPlan(t, cpuUnitID, engine.Coordinate{X: 4, Y: 1})}).decide
	held := holdCPUTurn(t, s)

	consumeCPUPhase(t, s, roomID, tokens[0])
	held.runOutsideRoomLock()

	phase, _, resolveTurnGameEvents := consumeCPUPhase(t, s, roomID, tokens[0])
	if phase != engine.TurnPhaseReady {
		t.Fatalf("phase after the turn = %v, want %v", phase, engine.TurnPhaseReady)
	}
	if len(resolveTurnGameEvents) == 0 {
		t.Fatal("Expected the placed bomb to tick in the resolve events, got none")
	}

	phase, planAgain, resolveAgain := consumeCPUPhase(t, s, roomID, tokens[0])
	if phase != engine.TurnPhaseIdle {
		t.Errorf("second consume phase = %v, want %v", phase, engine.TurnPhaseIdle)
	}
	if len(planAgain) != 0 || len(resolveAgain) != 0 {
		t.Errorf("second consume events = %#v and %#v, want none", planAgain, resolveAgain)
	}
}

func TestServerStateManager_ConsumeCPUStatus_InvalidToken(t *testing.T) {
	roomID, _, s := newCPUTurnRoom(t)
	decider := &recordingDecider{}
	s.decide = decider.decide
	held := holdCPUTurn(t, s)

	if _, _, _, err := s.consumeCPUStatus(roomID, "wrong"); !errors.Is(err, errInvalidToken) {
		t.Fatalf("ConsumeCPUStatus() error = %v, want %v", err, errInvalidToken)
	}

	if got := held.count(); got != 0 {
		t.Errorf("held CPU turns = %d, want 0", got)
	}
	if got := decider.calls; got != 0 {
		t.Errorf("decide calls = %d, want 0", got)
	}
}

func TestServerStateManager_ConsumeCPUStatus_LaunchesOnce(t *testing.T) {
	roomID, tokens, s := newCPUTurnRoom(t)
	decider := &recordingDecider{}
	s.decide = decider.decide
	held := holdCPUTurn(t, s)

	for range 2 {
		if phase, _, _ := consumeCPUPhase(t, s, roomID, tokens[0]); phase != engine.TurnPhasePlanning {
			t.Fatalf("phase = %v, want %v", phase, engine.TurnPhasePlanning)
		}
	}
	if got, want := held.count(), 1; got != want {
		t.Fatalf("held CPU turns = %d, want %d", got, want)
	}

	held.runOutsideRoomLock()
	if got, want := decider.calls, 1; got != want {
		t.Errorf("decide calls = %d, want %d", got, want)
	}
}

func TestServerStateManager_ConsumeCPUStatus_DefaultSpawn(t *testing.T) {
	cpuUnitID := engine.NewUnitID(2, 0)
	roomID, tokens, s := newCPUTurnRoom(t)
	room := mustRoom(t, s, roomID)
	decider := &recordingDecider{
		plan: func(*engine.GameState, int) []engine.TurnCommand {
			return []engine.TurnCommand{engine.NewMoveCommand(cpuUnitID, engine.Coordinate{X: 4, Y: 1})}
		},
		called: make(chan struct{}, 1),
	}
	s.decide = decider.decide

	if phase, _, _ := consumeCPUPhase(t, s, roomID, tokens[0]); phase != engine.TurnPhasePlanning {
		t.Fatalf("phase = %v, want %v", phase, engine.TurnPhasePlanning)
	}
	select {
	case <-decider.called:
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for the CPU goroutine to invoke decide")
	}

	room.mu.RLock()
	activeTeam := room.Match.TrueState.ActiveTeam
	room.mu.RUnlock()
	if got, want := activeTeam, 1; got != want {
		t.Errorf("active team after the turn = %d, want %d", got, want)
	}

	phase, planGameEvents, _ := consumeCPUPhase(t, s, roomID, tokens[0])
	if phase != engine.TurnPhaseReady {
		t.Errorf("phase after the turn = %v, want %v", phase, engine.TurnPhaseReady)
	}
	if !hasGameEvent(planGameEvents, engine.GameEvtUnitMoved, cpuUnitID) {
		t.Errorf("Expected unit %#x to have moved, got %#v", cpuUnitID, planGameEvents)
	}
}

func TestServerStateManager_runCPUTurn_Abandons(t *testing.T) {
	tests := []struct {
		name      string
		swap      func(t *testing.T, room *matchRoom)
		wantErr   error
		wantPhase engine.CPUTurnPhase
	}{
		{
			name: "Match Replaced",
			swap: func(t *testing.T, room *matchRoom) {
				t.Helper()
				replacement, err := engine.InitGame(vsCpuGameCfg())
				if err != nil {
					t.Fatalf("InitGame() error = %v", err)
				}
				room.Match = replacement
			},
			wantPhase: engine.TurnPhaseIdle,
		},
		{
			name: "Match Deleted",
			swap: func(t *testing.T, room *matchRoom) {
				t.Helper()
				room.Match = nil
			},
			wantErr: errMatchNotFound,
		},
		{
			name: "Match Already Ended",
			swap: func(t *testing.T, room *matchRoom) {
				t.Helper()
				room.Match.WinnerTeamID = 1
				room.Match.TrueState.Units[engine.NewUnitID(1, 0)].HasMoved = true
				room.Match.WorkingState.Units[engine.NewUnitID(1, 0)].HasMoved = true
			},
			wantPhase: engine.TurnPhaseReady,
		},
		{
			name: "Not CPU's Turn",
			swap: func(t *testing.T, room *matchRoom) {
				t.Helper()
				room.Match.WorkingState.ActiveTeam = 1
			},
			wantPhase: engine.TurnPhaseReady,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, tokens, s := newCPUTurnRoom(t)
			room := mustRoom(t, s, roomID)
			captured := room.Match
			decider := &recordingDecider{}
			s.decide = decider.decide
			held := holdCPUTurn(t, s)

			if phase, _, _ := consumeCPUPhase(t, s, roomID, tokens[0]); phase != engine.TurnPhasePlanning {
				t.Fatalf("launching consume phase = %v, want %v", phase, engine.TurnPhasePlanning)
			}
			tt.swap(t, room)
			wantTrueState := captured.TrueState.DeepCopy()
			held.runOutsideRoomLock()

			if got := decider.calls; got != 0 {
				t.Errorf("Expected the abandoned turn to skip planning, got %d decide calls", got)
			}
			if got := captured.TrueState; !reflect.DeepEqual(got, wantTrueState) {
				t.Errorf("Expected the abandoned match's committed state to stay unchanged, got %#v, want %#v", got, wantTrueState)
			}

			phase, planGameEvents, resolveTurnGameEvents, err := s.consumeCPUStatus(roomID, tokens[0])
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ConsumeCPUStatus() error = %v, want %v", err, tt.wantErr)
			}
			if got, want := phase, tt.wantPhase; got != want {
				t.Errorf("ConsumeCPUStatus() phase = %v, want %v", got, want)
			}
			if len(planGameEvents) != 0 || len(resolveTurnGameEvents) != 0 {
				t.Errorf("ConsumeCPUStatus() events = %#v and %#v, want none", planGameEvents, resolveTurnGameEvents)
			}
		})
	}
}

// Readers must not touch the match while the CPU goroutine mutates it. Meaningful under -race.
func TestServerStateManager_ReadsDuringCPUTurn(t *testing.T) {
	cpuUnitID := engine.NewUnitID(2, 0)

	roomID, tokens, s := newCPUTurnRoom(t)
	room := mustRoom(t, s, roomID)

	planning, release := make(chan struct{}), make(chan struct{})
	s.decide = func(*engine.GameState) ([]engine.TurnCommand, error) {
		close(planning)
		<-release
		return nil, nil
	}

	if phase, _, _ := consumeCPUPhase(t, s, roomID, tokens[0]); phase != engine.TurnPhasePlanning {
		t.Fatalf("phase = %v, want %v", phase, engine.TurnPhasePlanning)
	}
	select {
	case <-planning:
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for the CPU goroutine to invoke decide")
	}

	stop := make(chan struct{})
	var readers sync.WaitGroup
	for range 4 {
		readers.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
				}
				gs, err := s.getMatchState(roomID)
				if err != nil {
					t.Errorf("GetMatchState() error = %v", err)
					return
				}
				if _, err := json.Marshal(gs); err != nil {
					t.Errorf("Marshal(gameState) error = %v", err)
					return
				}
				if _, err := s.getAllowedTiles(roomID, cpuUnitID, engine.TurnCmdMove); err != nil {
					t.Errorf("GetAllowedTiles() error = %v", err)
					return
				}
				if _, err := s.getMatchConfig(roomID); err != nil {
					t.Errorf("GetMatchConfig() error = %v", err)
					return
				}
			}
		})
	}

	close(release)
	room.mu.Lock()
	activeTeam := room.Match.TrueState.ActiveTeam
	room.mu.Unlock()
	close(stop)
	readers.Wait()

	if got, want := activeTeam, 1; got != want {
		t.Errorf("active team after the CPU turn = %d, want %d", got, want)
	}
	if phase, _, _ := consumeCPUPhase(t, s, roomID, tokens[0]); phase != engine.TurnPhaseReady {
		t.Errorf("Expected phase %v after the CPU turn, got %v", engine.TurnPhaseReady, phase)
	}
}

func TestServerStateManager_runCPUTurn_Panic(t *testing.T) {
	roomID, tokens, s := newCPUTurnRoom(t)
	room := mustRoom(t, s, roomID)
	kingID := engine.NewUnitID(2, 0)
	target := freeNeighbour(t, room.Match.WorkingState, room.Match.WorkingState.Units[kingID].Position)
	if _, err := room.Match.CommandPlaceBomb(kingID, target); err != nil {
		t.Fatalf("CommandPlaceBomb() error = %v", err)
	}
	room.Match.TrueState = room.Match.WorkingState.DeepCopy()
	s.decide = func(*engine.GameState) ([]engine.TurnCommand, error) { panic("decide exploded") }
	held := holdCPUTurn(t, s)

	consumeCPUPhase(t, s, roomID, tokens[0])
	held.runOutsideRoomLock()

	if got, want := room.Match.TrueState.Turn, 3; got != want {
		t.Errorf("Expected the forfeited turn to still advance to %d, got %d", want, got)
	}

	// Proves the room mutex was released rather than left held by the unwinding goroutine.
	turnPhase, _, resolveTurnGameEvents := consumeCPUPhase(t, s, roomID, tokens[0])
	if got, want := turnPhase, engine.TurnPhaseReady; got != want {
		t.Errorf("Expected %v turnPhase return, got %v", want, got)
	}
	if len(resolveTurnGameEvents) == 0 {
		t.Error("Expected the forfeited turn to report the ticking bomb in the resolve events, got none")
	}
}

func TestServerStateManager_ConsumeCPUStatus(t *testing.T) {
	unitID := engine.NewUnitID(2, 0)
	tests := []struct {
		name     string
		setup    func(t *testing.T) (string, *ServerStateManager, string)
		wantErr  error
		validate func(t *testing.T, turnPhase engine.CPUTurnPhase, planGameEvents, resolveTurnGameEvents []engine.GameEvent, s *ServerStateManager, roomID, token string)
	}{
		{
			name: "Success - TurnPhaseReady and clear status",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				t.Helper()
				roomID, tokens, s := newCPUTurnRoom(t)
				s.decide = (&recordingDecider{plan: func(*engine.GameState, int) []engine.TurnCommand {
					return []engine.TurnCommand{engine.NewMoveCommand(unitID, engine.Coordinate{X: 4, Y: 1})}
				}}).decide
				held := holdCPUTurn(t, s)
				consumeCPUPhase(t, s, roomID, tokens[0])
				held.runOutsideRoomLock()
				return roomID, s, tokens[0]
			},
			wantErr: nil,
			validate: func(t *testing.T, turnPhase engine.CPUTurnPhase, planGameEvents, resolveTurnGameEvents []engine.GameEvent, s *ServerStateManager, roomID, token string) {
				t.Helper()

				if got, want := turnPhase, engine.TurnPhaseReady; got != want {
					t.Errorf("Expected %v turnPhase return, got %v", want, got)
				}
				if !hasGameEvent(planGameEvents, engine.GameEvtUnitMoved, unitID) {
					t.Errorf("Expected the move in planGameEvents, got %#v", planGameEvents)
				}

				phase, planAgain, resolveAgain := consumeCPUPhase(t, s, roomID, token)
				if got, want := phase, engine.TurnPhaseIdle; got != want {
					t.Errorf("Expected the second consume to return %v, got %v", want, got)
				}
				if len(planAgain) != 0 || len(resolveAgain) != 0 {
					t.Errorf("Expected the second consume to return no events, got %#v and %#v", planAgain, resolveAgain)
				}
			},
		},
		{
			name: "Success - non-TurnPhaseReady and do nothing",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				t.Helper()
				roomID, tokens, s := newCPUTurnRoom(t)
				holdCPUTurn(t, s)
				return roomID, s, tokens[0]
			},
			wantErr: nil,
			validate: func(t *testing.T, turnPhase engine.CPUTurnPhase, planGameEvents, resolveTurnGameEvents []engine.GameEvent, s *ServerStateManager, roomID, token string) {
				t.Helper()

				if got, want := turnPhase, engine.TurnPhasePlanning; got != want {
					t.Errorf("Expected %v turnPhase return, got %v", want, got)
				}
				if len(planGameEvents) != 0 || len(resolveTurnGameEvents) != 0 {
					t.Errorf("Expected no events while planning, got %#v and %#v", planGameEvents, resolveTurnGameEvents)
				}

				if phase, _, _ := consumeCPUPhase(t, s, roomID, token); phase != engine.TurnPhasePlanning {
					t.Errorf("Expected the second consume to stay at %v, got %v", engine.TurnPhasePlanning, phase)
				}
			},
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				t.Helper()
				s := NewServerStateManager()
				return "NONEXISTENT", s, "dummy-token"
			},
			wantErr: errRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				return roomID, s, "dummy-token"
			},
			wantErr: errMatchNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s, token := tt.setup(t)
			turnPhase, planGameEvents, resolveTurnGameEvents, err := s.consumeCPUStatus(roomID, token)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ConsumeCPUStatus() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, turnPhase, planGameEvents, resolveTurnGameEvents, s, roomID, token)
			} else if len(planGameEvents) != 0 || len(resolveTurnGameEvents) != 0 {
				t.Errorf("Expected gameEvents to be empty, got %#v and %#v", planGameEvents, resolveTurnGameEvents)
			}
		})
	}
}

func newCPUTurnRoom(t *testing.T) (string, [2]string, *ServerStateManager) {
	t.Helper()
	roomID, tokens, s := createTestRoomWithCfg(t, vsCpuGameCfg())
	room := mustRoom(t, s, roomID)
	room.Match.WorkingState.Turn = 2
	room.Match.WorkingState.ActiveTeam = 2
	room.Match.TrueState = room.Match.WorkingState.DeepCopy()
	return roomID, tokens, s
}

func TestServerStateManager_CPUTurnGate(t *testing.T) {
	cpuMove := engine.NewMoveCommand(engine.NewUnitID(2, 0), engine.Coordinate{X: 4, Y: 1})

	calls := []struct {
		name     string
		call     func(s *ServerStateManager, roomID, token string) error
		resolved bool
	}{
		{
			name: "SubmitTurnCommand",
			call: func(s *ServerStateManager, roomID, token string) error {
				_, err := s.submitTurnCommand(roomID, cpuMove, token)
				return err
			},
		},
		{
			name: "ResetTurn",
			call: func(s *ServerStateManager, roomID, token string) error {
				return s.resetTurn(roomID, token)
			},
		},
		{
			name: "ResolveTurn",
			call: func(s *ServerStateManager, roomID, token string) error {
				_, _, err := s.resolveTurn(roomID, token)
				return err
			},
			resolved: true,
		},
	}

	phases := []string{"Planning", "Ready", "Idle"}

	for _, c := range calls {
		for _, phase := range phases {
			if phase == "Ready" && c.name == "SubmitTurnCommand" {
				continue
			}
			t.Run(c.name+"/"+phase, func(t *testing.T) {
				roomID, tokens, s := newCPUTurnRoom(t)
				room := mustRoom(t, s, roomID)

				callToken, wantErr := tokens[1], error(nil)
				turn, team := 2, 2
				switch phase {
				case "Planning":
					holdCPUTurn(t, s)
					consumeCPUPhase(t, s, roomID, tokens[0])
					wantErr = errCPUTurnInProgress
				case "Ready":
					s.decide = (&recordingDecider{}).decide
					held := holdCPUTurn(t, s)
					consumeCPUPhase(t, s, roomID, tokens[0])
					held.runOutsideRoomLock()
					callToken = tokens[0]
					turn, team = 3, 1
				}

				err := c.call(s, roomID, callToken)
				if !errors.Is(err, wantErr) {
					t.Fatalf("error = %v, want %v", err, wantErr)
				}

				if wantErr == nil && c.resolved {
					turn, team = turn+1, 3-team
				}
				if got := room.Match.TrueState.Turn; got != turn {
					t.Errorf("Turn = %d, want %d", got, turn)
				}
				if got := room.Match.TrueState.ActiveTeam; got != team {
					t.Errorf("ActiveTeam = %d, want %d", got, team)
				}
			})
		}
	}

	t.Run("Bad token still gets 401", func(t *testing.T) {
		roomID, tokens, s := newCPUTurnRoom(t)
		holdCPUTurn(t, s)
		consumeCPUPhase(t, s, roomID, tokens[0])
		if _, _, err := s.resolveTurn(roomID, "wrong"); !errors.Is(err, errInvalidToken) {
			t.Errorf("error = %v, want %v", err, errInvalidToken)
		}
	})
}

func TestServerStateManager_CPUTurnGate_NotGated(t *testing.T) {
	roomID, tokens, s := newCPUTurnRoom(t)
	room := mustRoom(t, s, roomID)
	held := holdCPUTurn(t, s)

	phase, _, _ := consumeCPUPhase(t, s, roomID, tokens[0])
	if want := engine.TurnPhasePlanning; phase != want {
		t.Errorf("ConsumeCPUStatus phase = %v, want %v", phase, want)
	}

	if _, err := s.surrender(roomID, 1, tokens[0]); err != nil {
		t.Fatalf("Surrender() error = %v", err)
	}
	if got, want := room.Match.WinnerTeamID, 2; got != want {
		t.Errorf("WinnerTeamID = %d, want %d", got, want)
	}

	held.runOutsideRoomLock()
	phase, planGameEvents, resolveTurnGameEvents := consumeCPUPhase(t, s, roomID, tokens[0])
	if want := engine.TurnPhaseReady; phase != want {
		t.Errorf("ConsumeCPUStatus phase after the abandoned turn = %v, want %v", phase, want)
	}
	if len(planGameEvents) != 0 || len(resolveTurnGameEvents) != 0 {
		t.Errorf("Expected no events, got %#v and %#v", planGameEvents, resolveTurnGameEvents)
	}
}

func TestServerStateManager_runCPUTurn_DecideError(t *testing.T) {
	cpuUnitID := engine.NewUnitID(2, 0)
	decideErr := fmt.Errorf("unit %#x: %w", cpuUnitID, engine.ErrDesynced)

	tests := []struct {
		name     string
		cmds     []engine.TurnCommand
		wantMove bool
	}{
		{
			name:     "Partial Commands Applied",
			cmds:     []engine.TurnCommand{engine.NewMoveCommand(cpuUnitID, engine.Coordinate{X: 4, Y: 1})},
			wantMove: true,
		},
		{
			name: "No Commands",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, tokens, s := newCPUTurnRoom(t)
			room := mustRoom(t, s, roomID)
			h := &captureHandler{}
			room.Logger = slog.New(h)

			decider := &recordingDecider{
				plan: func(*engine.GameState, int) []engine.TurnCommand { return tt.cmds },
				err:  decideErr,
			}
			s.decide = decider.decide
			held := holdCPUTurn(t, s)

			consumeCPUPhase(t, s, roomID, tokens[0])
			held.runOutsideRoomLock()
			phase, planGameEvents, _ := consumeCPUPhase(t, s, roomID, tokens[0])

			if got, want := decider.calls, 1; got != want {
				t.Errorf("decide calls = %d, want %d", got, want)
			}
			if got, want := phase, engine.TurnPhaseReady; got != want {
				t.Errorf("Phase = %v, want %v", got, want)
			}
			if got, want := room.Match.TrueState.ActiveTeam, 1; got != want {
				t.Errorf("ActiveTeam = %d, want %d", got, want)
			}
			if got := hasGameEvent(planGameEvents, engine.GameEvtUnitMoved, cpuUnitID); got != tt.wantMove {
				t.Errorf("move applied = %v, want %v, events %#v", got, tt.wantMove, planGameEvents)
			}

			h.mu.Lock()
			defer h.mu.Unlock()
			if len(h.records) != 1 {
				t.Fatalf("got %d log records, want 1", len(h.records))
			}
			rec := h.records[0]
			if rec.Level != slog.LevelWarn || rec.Message != "CPU decide reported errors" {
				t.Errorf("record = %v %q, want Warn %q", rec.Level, rec.Message, "CPU decide reported errors")
			}
			var logged error
			rec.Attrs(func(a slog.Attr) bool {
				if a.Key == "error" {
					logged, _ = a.Value.Any().(error)
				}
				return true
			})
			if !errors.Is(logged, engine.ErrDesynced) {
				t.Errorf("logged error = %v, want %v", logged, engine.ErrDesynced)
			}
		})
	}
}

func TestServerStateManager_runCPUTurn_EndsMatch(t *testing.T) {
	gameCfg := vsCpuGameCfg()
	gameCfg.MaxTurns = 0
	roomID, tokens, s := createTestRoomWithCfg(t, gameCfg)
	room := mustRoom(t, s, roomID)
	room.Match.WorkingState.Turn = 2
	room.Match.WorkingState.ActiveTeam = 2
	room.Match.TrueState = room.Match.WorkingState.DeepCopy()

	kingID := engine.NewUnitID(2, 0)
	kingPos := room.Match.WorkingState.Units[kingID].Position
	target := freeNeighbour(t, room.Match.WorkingState, kingPos)
	if _, err := room.Match.CommandPlaceBomb(kingID, target); err != nil {
		t.Fatalf("CommandPlaceBomb() error = %v", err)
	}
	for _, bomb := range room.Match.WorkingState.Bombs {
		if bomb.OwnerID == kingID {
			bomb.Countdown = 1
		}
	}
	hazardIDs := func(gs *engine.GameState) []engine.BombID {
		var ids []engine.BombID
		for id, bomb := range gs.Bombs {
			if bomb.OwnerID == engine.SystemUnitID {
				ids = append(ids, id)
			}
		}
		return ids
	}
	hazardsBefore := hazardIDs(room.Match.WorkingState)
	if len(hazardsBefore) == 0 {
		t.Fatal("Expected hazards at creation with MaxTurns 0")
	}

	decider := &recordingDecider{}
	s.decide = decider.decide
	held := holdCPUTurn(t, s)
	if phase, _, _ := consumeCPUPhase(t, s, roomID, tokens[0]); phase != engine.TurnPhasePlanning {
		t.Fatalf("launching consume phase = %v, want %v", phase, engine.TurnPhasePlanning)
	}
	held.runOutsideRoomLock()

	if got, want := room.Match.WinnerTeamID, 1; got != want {
		t.Errorf("Expected winner %d, got %d", want, got)
	}
	if got, want := room.Match.WorkingState.Turn, 2; got != want {
		t.Errorf("Expected turn %d to stay, got %d", want, got)
	}
	if got, want := room.Match.WorkingState.ActiveTeam, 2; got != want {
		t.Errorf("Expected active team %d to stay, got %d", want, got)
	}
	for _, id := range hazardIDs(room.Match.WorkingState) {
		if !slices.Contains(hazardsBefore, id) {
			t.Errorf("Expected no new hazards, found bomb %v", id)
		}
	}

	phase, planGameEvents, resolveTurnGameEvents := consumeCPUPhase(t, s, roomID, tokens[0])
	if got, want := phase, engine.TurnPhaseReady; got != want {
		t.Errorf("Expected first consume %v, got %v", want, got)
	}
	if !slices.ContainsFunc(resolveTurnGameEvents, func(evt engine.GameEvent) bool {
		return evt.Type == engine.GameEvtMatchEnded && evt.WinnerTeamID == 1
	}) {
		t.Errorf("Expected a matchEnded event with winner 1, got %+v", resolveTurnGameEvents)
	}
	for name, gameEvents := range map[string][]engine.GameEvent{
		"plan":    planGameEvents,
		"resolve": resolveTurnGameEvents,
	} {
		if hasGameEvent(gameEvents, engine.GameEvtBombPlaced, engine.SystemUnitID) {
			t.Errorf("Expected no hazard bombPlaced event in the %s events, got %+v", name, gameEvents)
		}
	}
	for consume := 2; consume <= 3; consume++ {
		phase, planAgain, resolveAgain := consumeCPUPhase(t, s, roomID, tokens[0])
		if got, want := phase, engine.TurnPhaseIdle; got != want {
			t.Errorf("Expected consume %d to be %v, got %v", consume, want, got)
		}
		if len(planAgain) != 0 || len(resolveAgain) != 0 {
			t.Errorf("Expected consume %d to return no events, got %+v and %+v", consume, planAgain, resolveAgain)
		}
	}
	if got, want := decider.calls, 1; got != want {
		t.Errorf("Expected decide to run %d time, got %d", want, got)
	}
}

type captureHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *captureHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= slog.LevelWarn
}

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r)
	return nil
}

func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *captureHandler) WithGroup(string) slog.Handler { return h }
