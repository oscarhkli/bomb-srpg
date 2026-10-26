package server

import (
	"bomb-srpg/engine"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"
)

// recordingDecider is an injectable cpu.Decide stand-in that counts invocations and
// samples CPU.Phase from inside the CPU goroutine, where the room write lock is held.
type recordingDecider struct {
	match         *engine.Match
	plan          func(gs *engine.GameState, call int) []engine.TurnCommand
	err           error
	calls         int
	observedPhase engine.CPUTurnPhase
	observedBombs int
	called        chan struct{}
}

func (d *recordingDecider) decide(gs *engine.GameState) ([]engine.TurnCommand, error) {
	call := d.calls
	if call == 0 {
		d.observedPhase = d.match.CPU.Phase
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

func TestServerStateManager_StartTurn_CPUTurn(t *testing.T) {
	humanUnitID := engine.NewUnitID(1, 0)
	cpuUnitID := engine.NewUnitID(2, 0)

	legalCPUMove := []engine.TurnCommand{engine.NewMoveCommand(cpuUnitID, engine.Coordinate{X: 4, Y: 1})}
	rejectedPlan := []engine.TurnCommand{engine.NewMoveCommand(humanUnitID, engine.Coordinate{X: 1, Y: 1})}

	cpuTurnPending := func(t *testing.T, room *MatchRoom) int {
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
		setup          func(t *testing.T, room *MatchRoom) int
		plan           func(gs *engine.GameState, call int) []engine.TurnCommand
		wantCPUTurn    bool
		wantCalls      int
		wantActiveTeam int
		validate       func(t *testing.T, room *MatchRoom, d *recordingDecider)
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
			validate: func(t *testing.T, room *MatchRoom, d *recordingDecider) {
				t.Helper()
				if got, want := d.observedPhase, engine.TurnPhasePlanning; got != want {
					t.Errorf("Expected phase %v while planning, got %v", want, got)
				}
				if !hasGameEvent(room.Match.CPU.PlanGameEvents, engine.GameEvtUnitMoved, cpuUnitID) {
					t.Errorf("Expected unit %#x to have moved, got %#v", cpuUnitID, room.Match.CPU.PlanGameEvents)
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
			validate: func(t *testing.T, room *MatchRoom, d *recordingDecider) {
				t.Helper()
				if got := len(room.Match.CPU.PlanGameEvents); got != 0 {
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
			validate: func(t *testing.T, room *MatchRoom, d *recordingDecider) {
				t.Helper()
				if !hasGameEvent(room.Match.CPU.PlanGameEvents, engine.GameEvtUnitMoved, cpuUnitID) {
					t.Errorf("Expected replanned move to apply, got %#v", room.Match.CPU.PlanGameEvents)
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
			wantCalls:      maxCPUReplanAttempts,
			wantActiveTeam: 1,
			validate: func(t *testing.T, room *MatchRoom, d *recordingDecider) {
				t.Helper()
				if got := len(room.Match.CPU.PlanGameEvents); got != 0 {
					t.Errorf("Expected no command to apply, got %#v", room.Match.CPU.PlanGameEvents)
				}
			},
		},
		{
			name:     "Sudden Death Injected Before Planning",
			vsCpu:    true,
			maxTurns: 1,
			setup:    cpuTurnPending,
			plan: func(gs *engine.GameState, call int) []engine.TurnCommand {
				return nil
			},
			wantCPUTurn:    true,
			wantCalls:      1,
			wantActiveTeam: 1,
			validate: func(t *testing.T, room *MatchRoom, d *recordingDecider) {
				t.Helper()
				if got, want := d.observedBombs, engine.SuddenDeathBombs; got < want {
					t.Errorf("Expected at least %d hazard bombs on the board when planning, got %d", want, got)
				}
			},
		},
		{
			name:  "Unconsumed Ready Phase Relaunches",
			vsCpu: true,
			setup: func(t *testing.T, room *MatchRoom) int {
				t.Helper()
				tokenIdx := cpuTurnPending(t, room)
				room.Match.CPU.Phase = engine.TurnPhaseReady
				room.Match.CPU.ResolveTurnGameEvents = []engine.GameEvent{engine.NewMatchEndedEvent(1)}
				room.Match.CPU.PlanGameEvents = []engine.GameEvent{engine.NewUnitMovedEvent(cpuUnitID, engine.Coordinate{X: 0, Y: 0}, engine.Coordinate{X: 1, Y: 0})}
				return tokenIdx
			},
			plan: func(gs *engine.GameState, call int) []engine.TurnCommand {
				return legalCPUMove
			},
			wantCPUTurn:    true,
			wantCalls:      1,
			wantActiveTeam: 1,
			validate: func(t *testing.T, room *MatchRoom, d *recordingDecider) {
				t.Helper()
				if hasGameEvent(room.Match.CPU.ResolveTurnGameEvents, engine.GameEvtMatchEnded, 0) {
					t.Errorf("Expected the stale event dropped, got %#v", room.Match.CPU.ResolveTurnGameEvents)
				}
				if !hasGameEvent(room.Match.CPU.PlanGameEvents, engine.GameEvtUnitMoved, cpuUnitID) {
					t.Errorf("Expected unit %#x to have moved, got %#v", cpuUnitID, room.Match.CPU.PlanGameEvents)
				}
			},
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

			decider := &recordingDecider{match: room.Match, plan: tt.plan, called: make(chan struct{}, 1)}
			s.decide = decider.decide

			if _, _, err := s.StartTurn(roomID, tokens[tokenIdx]); err != nil {
				t.Fatalf("StartTurn() error = %v", err)
			}

			if tt.wantCPUTurn {
				select {
				case <-decider.called:
				case <-time.After(5 * time.Second):
					t.Fatal("Timed out waiting for the CPU goroutine to invoke decide")
				}
			}

			// Blocks until runCPUTurn releases the write lock, ordering every assertion below after it.
			room.mu.RLock()
			defer room.mu.RUnlock()

			if got, want := decider.calls, tt.wantCalls; got != want {
				t.Errorf("Expected %d decide calls, got %d", want, got)
			}

			wantPhase := engine.TurnPhaseIdle
			if tt.wantCPUTurn {
				wantPhase = engine.TurnPhaseReady
			}
			if got := room.Match.CPU.Phase; got != wantPhase {
				t.Errorf("Expected CPU phase %v, got %v", wantPhase, got)
			}
			if got, want := room.Match.TrueState.ActiveTeam, tt.wantActiveTeam; got != want {
				t.Errorf("Expected active team %d, got %d", want, got)
			}
			if tt.validate != nil {
				tt.validate(t, room, decider)
			}
		})
	}
}

func TestServerStateManager_runCPUTurn_Abandons(t *testing.T) {
	tests := []struct {
		name      string
		swap      func(t *testing.T, room *MatchRoom)
		wantPhase engine.CPUTurnPhase
	}{
		{
			name: "Match Replaced",
			swap: func(t *testing.T, room *MatchRoom) {
				t.Helper()
				replacement, err := engine.InitGame(vsCpuGameCfg())
				if err != nil {
					t.Fatalf("InitGame() error = %v", err)
				}
				room.Match = replacement
			},
			wantPhase: engine.TurnPhasePlanning,
		},
		{
			name: "Match Deleted",
			swap: func(t *testing.T, room *MatchRoom) {
				t.Helper()
				room.Match = nil
			},
			wantPhase: engine.TurnPhasePlanning,
		},
		{
			name: "Match Already Ended",
			swap: func(t *testing.T, room *MatchRoom) {
				t.Helper()
				room.Match.WinnerTeamID = 1
			},
			wantPhase: engine.TurnPhaseReady,
		},
		{
			name: "Not CPU's Turn",
			swap: func(t *testing.T, room *MatchRoom) {
				t.Helper()
			},
			wantPhase: engine.TurnPhaseReady,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, _, s := createTestRoomWithCfg(t, vsCpuGameCfg())
			room := mustRoom(t, s, roomID)

			captured := room.Match
			captured.CPU.Phase = engine.TurnPhasePlanning
			decider := &recordingDecider{match: captured, called: make(chan struct{}, 1)}
			s.decide = decider.decide

			tt.swap(t, room)
			s.runCPUTurn(room, captured)

			if got := decider.calls; got != 0 {
				t.Errorf("Expected the abandoned turn to skip planning, got %d decide calls", got)
			}
			if got, want := captured.CPU.Phase, tt.wantPhase; got != want {
				t.Errorf("Expected phase %v on the abandoned match, got %v", want, got)
			}
			if got, want := captured.TrueState.Turn, 1; got != want {
				t.Errorf("Expected turn %d on the abandoned match, got %d", want, got)
			}
			if room.Match != nil && room.Match != captured && room.Match.CPU.Phase != engine.TurnPhaseIdle {
				t.Errorf("Expected the replacement match untouched, got phase %v", room.Match.CPU.Phase)
			}
		})
	}
}

// Readers must not touch the match while the CPU goroutine mutates it. Meaningful under -race.
func TestServerStateManager_ReadsDuringCPUTurn(t *testing.T) {
	cpuUnitID := engine.NewUnitID(2, 0)

	roomID, _, s := createTestRoomWithCfg(t, vsCpuGameCfg())
	room := mustRoom(t, s, roomID)
	room.Match.WorkingState.ActiveTeam = 2
	room.Match.CPU.Phase = engine.TurnPhasePlanning
	match := room.Match

	planning, release := make(chan struct{}), make(chan struct{})
	s.decide = func(*engine.GameState) ([]engine.TurnCommand, error) {
		close(planning)
		<-release
		return nil, nil
	}

	cpuDone := make(chan struct{})
	go func() {
		defer close(cpuDone)
		s.runCPUTurn(room, match)
	}()

	<-planning

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
				gs, err := s.GetMatchState(roomID)
				if err != nil {
					t.Errorf("GetMatchState() error = %v", err)
					return
				}
				if _, err := json.Marshal(gs); err != nil {
					t.Errorf("Marshal(gameState) error = %v", err)
					return
				}
				if _, err := s.GetAllowedTiles(roomID, cpuUnitID, engine.TurnCmdMove); err != nil {
					t.Errorf("GetAllowedTiles() error = %v", err)
					return
				}
				if _, err := s.GetMatchConfig(roomID); err != nil {
					t.Errorf("GetMatchConfig() error = %v", err)
					return
				}
			}
		})
	}

	close(release)
	<-cpuDone
	close(stop)
	readers.Wait()

	if got, want := match.CPU.Phase, engine.TurnPhaseReady; got != want {
		t.Errorf("Expected phase %v after the CPU turn, got %v", want, got)
	}
}

func TestServerStateManager_runCPUTurn_Panic(t *testing.T) {
	roomID, tokens, s := createTestRoomWithCfg(t, vsCpuGameCfg())
	room := mustRoom(t, s, roomID)
	room.Match.WorkingState.ActiveTeam = 2
	room.Match.CPU.Phase = engine.TurnPhasePlanning
	match := room.Match

	s.decide = func(*engine.GameState) ([]engine.TurnCommand, error) { panic("decide exploded") }

	s.runCPUTurn(room, match)

	if got, want := match.CPU.Phase, engine.TurnPhaseReady; got != want {
		t.Errorf("Expected phase %v after the panic, got %v", want, got)
	}
	if got, want := match.TrueState.Turn, 2; got != want {
		t.Errorf("Expected the forfeited turn to still advance to %d, got %d", want, got)
	}

	// Proves the room mutex was released rather than left held by the unwinding goroutine.
	turnPhase, _, _, err := s.ConsumeCPUStatus(roomID, tokens[0])
	if err != nil {
		t.Fatalf("ConsumeCPUStatus() error = %v", err)
	}
	if got, want := turnPhase, engine.TurnPhaseReady; got != want {
		t.Errorf("Expected %v turnPhase return, got %v", want, got)
	}
}

func TestServerStateManager_ConsumeCPUStatus(t *testing.T) {
	unitID := engine.NewUnitID(2, 0)
	tests := []struct {
		name     string
		setup    func(t *testing.T) (string, *ServerStateManager, string)
		wantErr  error
		validate func(t *testing.T, turnPhase engine.CPUTurnPhase, planGameEvents, resolveTurnGameEvents []engine.GameEvent, s *ServerStateManager, roomID string)
	}{
		{
			name: "Success - TurnPhaseReady and clear status",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				t.Helper()
				roomID, tokens, s := createTestRoom(t)
				room := mustRoom(t, s, roomID)
				room.Match.CPU.Phase = engine.TurnPhaseReady
				room.Match.CPU.PlanGameEvents = []engine.GameEvent{
					engine.NewUnitMovedEvent(unitID, engine.Coordinate{X: 1, Y: 2}, engine.Coordinate{X: 2, Y: 2}),
				}
				room.Match.CPU.ResolveTurnGameEvents = []engine.GameEvent{
					engine.NewUnitDiedEvent(unitID, engine.Coordinate{X: 2, Y: 2}),
				}
				return roomID, s, tokens[0]
			},
			wantErr: nil,
			validate: func(t *testing.T, turnPhase engine.CPUTurnPhase, planGameEvents, resolveTurnGameEvents []engine.GameEvent, s *ServerStateManager, roomID string) {
				t.Helper()
				room := mustRoom(t, s, roomID)

				if got, want := turnPhase, engine.TurnPhaseReady; got != want {
					t.Errorf("Expected %v turnPhase return, got %v", want, got)
				}
				if !hasGameEvent(planGameEvents, engine.GameEvtUnitMoved, unitID) {
					t.Errorf("Expected the move in planGameEvents, got %#v", planGameEvents)
				}
				if !hasGameEvent(resolveTurnGameEvents, engine.GameEvtUnitDied, unitID) {
					t.Errorf("Expected the death in resolveTurnGameEvents, got %#v", resolveTurnGameEvents)
				}

				if got, want := room.Match.CPU.Phase, engine.TurnPhaseIdle; got != want {
					t.Errorf("Expected CPU.Phase.TurnPhase reset to %v, got %v", want, got)
				}
				if got, want := len(room.Match.CPU.PlanGameEvents), 0; got != want {
					t.Errorf("Expected CPU.PlanGameEvents cleared, got %#v", room.Match.CPU.PlanGameEvents)
				}
				if got, want := len(room.Match.CPU.ResolveTurnGameEvents), 0; got != want {
					t.Errorf("Expected CPU.ResolveTurnGameEvents cleared, got %#v", room.Match.CPU.ResolveTurnGameEvents)
				}
			},
		},
		{
			name: "Success - non-TurnPhaseReady and do nothing",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				t.Helper()
				roomID, tokens, s := createTestRoom(t)
				room := mustRoom(t, s, roomID)
				room.Match.CPU.Phase = engine.TurnPhasePlanning
				room.Match.CPU.ResolveTurnGameEvents = []engine.GameEvent{
					engine.NewUnitMovedEvent(unitID, engine.Coordinate{X: 1, Y: 2}, engine.Coordinate{X: 2, Y: 2}),
				}
				return roomID, s, tokens[0]
			},
			wantErr: nil,
			validate: func(t *testing.T, turnPhase engine.CPUTurnPhase, planGameEvents, resolveTurnGameEvents []engine.GameEvent, s *ServerStateManager, roomID string) {
				t.Helper()
				room := mustRoom(t, s, roomID)

				if got, want := turnPhase, engine.TurnPhasePlanning; got != want {
					t.Errorf("Expected %v turnPhase return, got %v", want, got)
				}
				if got, want := len(resolveTurnGameEvents), 1; got != want {
					t.Errorf("Expected %d resolveTurnGameEvents returned, got %#v", want, resolveTurnGameEvents)
				}

				if got, want := room.Match.CPU.Phase, engine.TurnPhasePlanning; got != want {
					t.Errorf("Expected CPU.Phase.TurnPhase stays at %v, got %v", want, got)
				}
				if got, want := len(room.Match.CPU.ResolveTurnGameEvents), 1; got != want {
					t.Errorf("Expected CPU.ResolveTurnGameEvents remain unchanged as %d, got %#v", want, room.Match.CPU.ResolveTurnGameEvents)
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
			wantErr: ErrRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				return roomID, s, "dummy-token"
			},
			wantErr: ErrMatchNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s, token := tt.setup(t)
			turnPhase, planGameEvents, resolveTurnGameEvents, err := s.ConsumeCPUStatus(roomID, token)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ConsumeCPUStatus() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, turnPhase, planGameEvents, resolveTurnGameEvents, s, roomID)
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
		name string
		call func(s *ServerStateManager, roomID, token string) error
		// resolved is true when a successful call advances the turn.
		resolved bool
	}{
		{
			name: "SubmitTurnCommand",
			call: func(s *ServerStateManager, roomID, token string) error {
				_, err := s.SubmitTurnCommand(roomID, cpuMove, token)
				return err
			},
		},
		{
			name: "ResetTurn",
			call: func(s *ServerStateManager, roomID, token string) error {
				return s.ResetTurn(roomID, token)
			},
		},
		{
			name: "ResolveTurn",
			call: func(s *ServerStateManager, roomID, token string) error {
				_, _, err := s.ResolveTurn(roomID, token)
				return err
			},
			resolved: true,
		},
	}

	phases := []struct {
		name    string
		phase   engine.CPUTurnPhase
		wantErr error
	}{
		{"Planning", engine.TurnPhasePlanning, ErrCPUTurnInProgress},
		{"Ready", engine.TurnPhaseReady, nil},
		{"Idle", engine.TurnPhaseIdle, nil},
	}

	for _, c := range calls {
		for _, p := range phases {
			t.Run(c.name+"/"+p.name, func(t *testing.T) {
				roomID, tokens, s := newCPUTurnRoom(t)
				room := mustRoom(t, s, roomID)
				room.Match.CPU.Phase = p.phase

				err := c.call(s, roomID, tokens[1])
				if !errors.Is(err, p.wantErr) {
					t.Fatalf("error = %v, want %v", err, p.wantErr)
				}

				wantTurn, wantTeam := 2, 2
				if p.wantErr == nil && c.resolved {
					wantTurn, wantTeam = 3, 1
				}
				if got := room.Match.TrueState.Turn; got != wantTurn {
					t.Errorf("Turn = %d, want %d", got, wantTurn)
				}
				if got := room.Match.TrueState.ActiveTeam; got != wantTeam {
					t.Errorf("ActiveTeam = %d, want %d", got, wantTeam)
				}
			})
		}
	}

	t.Run("Bad token still gets 401", func(t *testing.T) {
		roomID, _, s := newCPUTurnRoom(t)
		mustRoom(t, s, roomID).Match.CPU.Phase = engine.TurnPhasePlanning
		if _, _, err := s.ResolveTurn(roomID, "wrong"); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("error = %v, want %v", err, ErrInvalidToken)
		}
	})
}

func TestServerStateManager_CPUTurnGate_NotGated(t *testing.T) {
	roomID, tokens, s := newCPUTurnRoom(t)
	room := mustRoom(t, s, roomID)
	room.Match.CPU.Phase = engine.TurnPhasePlanning

	if _, _, err := s.StartTurn(roomID, tokens[1]); err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}
	if got, want := room.Match.CPU.Phase, engine.TurnPhasePlanning; got != want {
		t.Errorf("Phase after StartTurn = %v, want %v", got, want)
	}

	phase, _, _, err := s.ConsumeCPUStatus(roomID, tokens[0])
	if err != nil {
		t.Fatalf("ConsumeCPUStatus() error = %v", err)
	}
	if want := engine.TurnPhasePlanning; phase != want {
		t.Errorf("ConsumeCPUStatus phase = %v, want %v", phase, want)
	}

	if _, err := s.Surrender(roomID, 1, tokens[0]); err != nil {
		t.Fatalf("Surrender() error = %v", err)
	}
	if got, want := room.Match.WinnerTeamID, 2; got != want {
		t.Errorf("WinnerTeamID = %d, want %d", got, want)
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
			roomID, _, s := newCPUTurnRoom(t)
			room := mustRoom(t, s, roomID)
			h := &captureHandler{}
			room.Logger = slog.New(h)
			room.Match.CPU.Phase = engine.TurnPhasePlanning
			match := room.Match

			decider := &recordingDecider{
				match:  match,
				plan:   func(*engine.GameState, int) []engine.TurnCommand { return tt.cmds },
				err:    decideErr,
				called: make(chan struct{}, 1),
			}
			s.decide = decider.decide

			s.runCPUTurn(room, match)

			if got, want := decider.calls, 1; got != want {
				t.Errorf("decide calls = %d, want %d", got, want)
			}
			if got, want := match.CPU.Phase, engine.TurnPhaseReady; got != want {
				t.Errorf("Phase = %v, want %v", got, want)
			}
			if got, want := match.TrueState.ActiveTeam, 1; got != want {
				t.Errorf("ActiveTeam = %d, want %d", got, want)
			}
			if got := hasGameEvent(match.CPU.PlanGameEvents, engine.GameEvtUnitMoved, cpuUnitID); got != tt.wantMove {
				t.Errorf("move applied = %v, want %v, events %#v", got, tt.wantMove, match.CPU.PlanGameEvents)
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
