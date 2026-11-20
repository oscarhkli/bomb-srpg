package server

import (
	"bomb-srpg/engine"
	"bomb-srpg/room"
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

func TestServerStateManager_ConsumeCPUStatus_LaunchesCPUTurn(t *testing.T) {
	humanUnitID := engine.NewUnitID(1, 0)
	cpuUnitID := engine.NewUnitID(2, 0)

	legalCPUMove := []engine.TurnCommand{engine.NewMoveCommand(cpuUnitID, engine.Coordinate{X: 4, Y: 1})}
	rejectedPlan := []engine.TurnCommand{engine.NewMoveCommand(humanUnitID, engine.Coordinate{X: 1, Y: 1})}

	tests := []struct {
		name           string
		vsCpu          bool
		maxTurns       int
		arrange        []matchArrangement
		tokenIdx       int
		plan           func(gs *engine.GameState, call int) []engine.TurnCommand
		wantCPUTurn    bool
		wantCalls      int
		wantActiveTeam int
		validate       func(t *testing.T, planGameEvents []engine.GameEvent, d *recordingDecider)
	}{
		{
			name:     "Plan Applied",
			vsCpu:    true,
			arrange:  []matchArrangement{cpuTurnPending},
			tokenIdx: 1,
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
			name:     "Empty Plan",
			vsCpu:    true,
			arrange:  []matchArrangement{cpuTurnPending},
			tokenIdx: 1,
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
			name:     "Replan On Rejection",
			vsCpu:    true,
			arrange:  []matchArrangement{cpuTurnPending},
			tokenIdx: 1,
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
			name:     "Replan Cap Exhausted",
			vsCpu:    true,
			arrange:  []matchArrangement{cpuTurnPending},
			tokenIdx: 1,
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
			arrange:  []matchArrangement{firstTurnResolved},
			tokenIdx: 1,
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
			name:           "Match Ended",
			vsCpu:          true,
			arrange:        []matchArrangement{cpuTurnPending, matchEnded},
			tokenIdx:       1,
			wantCPUTurn:    false,
			wantCalls:      0,
			wantActiveTeam: 2,
		},
		{
			name:           "Not VS CPU",
			vsCpu:          false,
			arrange:        []matchArrangement{cpuTurnPending},
			tokenIdx:       1,
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
			decider := &recordingDecider{plan: tt.plan}
			held := holdCPUTurn(t)
			roomID, tokens, s := createTestRoomWithCfg(t, gameCfg, WithDecider(decider.decide), WithSpawner(held.spawn), arrangedMatch(tt.arrange...))

			wantPhase, wantHeld := room.TurnPhaseIdle, 0
			if tt.wantCPUTurn {
				wantPhase, wantHeld = room.TurnPhasePlanning, 1
			}
			phase, planGameEvents, resolveTurnGameEvents := consumeCPUPhase(t, s, roomID, tokens[tt.tokenIdx])
			if phase != wantPhase {
				t.Errorf("ConsumeCPU() phase = %v, want %v", phase, wantPhase)
			}
			if len(planGameEvents) != 0 || len(resolveTurnGameEvents) != 0 {
				t.Errorf("ConsumeCPU() events = %#v and %#v, want none", planGameEvents, resolveTurnGameEvents)
			}
			if got := held.count(); got != wantHeld {
				t.Errorf("held CPU turns = %d, want %d", got, wantHeld)
			}

			if tt.wantCPUTurn {
				held.runOutsideRoomLock()
				phase, planGameEvents, _ = consumeCPUPhase(t, s, roomID, tokens[tt.tokenIdx])
				if phase != room.TurnPhaseReady {
					t.Errorf("ConsumeCPU() after the turn phase = %v, want %v", phase, room.TurnPhaseReady)
				}
			}

			if got, want := decider.calls, tt.wantCalls; got != want {
				t.Errorf("Expected %d decide calls, got %d", want, got)
			}
			if got, want := mustMatchState(t, s, roomID).ActiveTeam, tt.wantActiveTeam; got != want {
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

	planA := moveThenBombPlan(t, cpuUnitID, posA)
	decider := &recordingDecider{plan: func(gs *engine.GameState, call int) []engine.TurnCommand {
		if call == 0 {
			return planA(gs, call)
		}
		return []engine.TurnCommand{engine.NewMoveCommand(cpuUnitID, posB)}
	}}
	held := holdCPUTurn(t)
	roomID, tokens, s := newCPUTurnRoom(t, WithDecider(decider.decide), WithSpawner(held.spawn))

	consumeCPUPhase(t, s, roomID, tokens[0])
	held.runOutsideRoomLock()

	if _, _, err := mustRoom(t, s, roomID).Resolve(tokens[0]); err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	phase, planGameEvents, resolveTurnGameEvents := consumeCPUPhase(t, s, roomID, tokens[0])
	if phase != room.TurnPhasePlanning {
		t.Fatalf("launching consume phase = %v, want %v", phase, room.TurnPhasePlanning)
	}
	if len(planGameEvents) != 0 || len(resolveTurnGameEvents) != 0 {
		t.Errorf("launching consume events = %#v and %#v, want none", planGameEvents, resolveTurnGameEvents)
	}
	if got, want := held.count(), 2; got != want {
		t.Fatalf("held CPU turns = %d, want %d", got, want)
	}

	held.runOutsideRoomLock()
	phase, planGameEvents, _ = consumeCPUPhase(t, s, roomID, tokens[0])
	if phase != room.TurnPhaseReady {
		t.Fatalf("phase after the second turn = %v, want %v", phase, room.TurnPhaseReady)
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
	held := holdCPUTurn(t)
	decider := &recordingDecider{plan: moveThenBombPlan(t, cpuUnitID, engine.Coordinate{X: 4, Y: 1})}
	roomID, tokens, s := newCPUTurnRoom(t, WithDecider(decider.decide), WithSpawner(held.spawn))

	consumeCPUPhase(t, s, roomID, tokens[0])
	held.runOutsideRoomLock()

	phase, _, resolveTurnGameEvents := consumeCPUPhase(t, s, roomID, tokens[0])
	if phase != room.TurnPhaseReady {
		t.Fatalf("phase after the turn = %v, want %v", phase, room.TurnPhaseReady)
	}
	if len(resolveTurnGameEvents) == 0 {
		t.Fatal("Expected the placed bomb to tick in the resolve events, got none")
	}

	phase, planAgain, resolveAgain := consumeCPUPhase(t, s, roomID, tokens[0])
	if phase != room.TurnPhaseIdle {
		t.Errorf("second consume phase = %v, want %v", phase, room.TurnPhaseIdle)
	}
	if len(planAgain) != 0 || len(resolveAgain) != 0 {
		t.Errorf("second consume events = %#v and %#v, want none", planAgain, resolveAgain)
	}
}

func TestServerStateManager_ConsumeCPUStatus_InvalidToken(t *testing.T) {
	decider := &recordingDecider{}
	held := holdCPUTurn(t)
	roomID, _, s := newCPUTurnRoom(t, WithDecider(decider.decide), WithSpawner(held.spawn))

	if _, err := mustRoom(t, s, roomID).ConsumeCPU("wrong"); !errors.Is(err, room.ErrInvalidToken) {
		t.Fatalf("ConsumeCPUStatus() error = %v, want %v", err, room.ErrInvalidToken)
	}

	if got := held.count(); got != 0 {
		t.Errorf("held CPU turns = %d, want 0", got)
	}
	if got := decider.calls; got != 0 {
		t.Errorf("decide calls = %d, want 0", got)
	}
}

func TestServerStateManager_ConsumeCPUStatus_LaunchesOnce(t *testing.T) {
	decider := &recordingDecider{}
	held := holdCPUTurn(t)
	roomID, tokens, s := newCPUTurnRoom(t, WithDecider(decider.decide), WithSpawner(held.spawn))

	for range 2 {
		if phase, _, _ := consumeCPUPhase(t, s, roomID, tokens[0]); phase != room.TurnPhasePlanning {
			t.Fatalf("phase = %v, want %v", phase, room.TurnPhasePlanning)
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

func TestServerStateManager_runCPUTurn_Abandons(t *testing.T) {
	tests := []struct {
		name      string
		act       func(t *testing.T, s *ServerStateManager, roomID string, tokens [2]string)
		wantErr   error
		wantPhase room.CPUTurnPhase
	}{
		{
			name: "Match Replaced",
			act: func(t *testing.T, s *ServerStateManager, roomID string, tokens [2]string) {
				t.Helper()
				if _, err := mustRoom(t, s, roomID).Rematch(tokens[0]); err != nil {
					t.Fatalf("Rematch() error = %v", err)
				}
			},
			wantPhase: room.TurnPhaseIdle,
		},
		{
			name: "Match Deleted",
			act: func(t *testing.T, s *ServerStateManager, roomID string, tokens [2]string) {
				t.Helper()
				mustEndAndDeleteMatch(t, s, roomID, tokens[0])
			},
			wantErr: room.ErrMatchNotFound,
		},
		{
			name: "Match Already Ended",
			act: func(t *testing.T, s *ServerStateManager, roomID string, tokens [2]string) {
				t.Helper()
				if _, err := mustRoom(t, s, roomID).Surrender(1, tokens[0]); err != nil {
					t.Fatalf("Surrender() error = %v", err)
				}
			},
			wantPhase: room.TurnPhaseReady,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decider := &recordingDecider{}
			held := holdCPUTurn(t)
			var captured *engine.Match
			firstCPUTurnPending := WithMatchFactory(func(gameCfg engine.GameCfg) (*engine.Match, error) {
				match, err := engine.InitGame(gameCfg)
				if err != nil || captured != nil {
					return match, err
				}
				captured = match
				return match, cpuTurnPending(match)
			})
			roomID, tokens, s := newCPUTurnRoom(t, WithDecider(decider.decide), WithSpawner(held.spawn), firstCPUTurnPending)

			if phase, _, _ := consumeCPUPhase(t, s, roomID, tokens[0]); phase != room.TurnPhasePlanning {
				t.Fatalf("launching consume phase = %v, want %v", phase, room.TurnPhasePlanning)
			}
			tt.act(t, s, roomID, tokens)
			wantTrueState := captured.TrueState.DeepCopy()
			held.runOutsideRoomLock()

			if got := decider.calls; got != 0 {
				t.Errorf("Expected the abandoned turn to skip planning, got %d decide calls", got)
			}
			if got := captured.TrueState; !reflect.DeepEqual(got, wantTrueState) {
				t.Errorf("Expected the abandoned match's committed state to stay unchanged, got %#v, want %#v", got, wantTrueState)
			}

			status, err := mustRoom(t, s, roomID).ConsumeCPU(tokens[0])
			phase, planGameEvents, resolveTurnGameEvents := status.Phase, status.Plan, status.Resolve
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

	planning, release := make(chan struct{}), make(chan struct{})
	decide := func(*engine.GameState) ([]engine.TurnCommand, error) {
		close(planning)
		<-release
		return nil, nil
	}
	roomID, tokens, s := newCPUTurnRoom(t, WithDecider(decide))

	if phase, _, _ := consumeCPUPhase(t, s, roomID, tokens[0]); phase != room.TurnPhasePlanning {
		t.Fatalf("phase = %v, want %v", phase, room.TurnPhasePlanning)
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
				gs, err := mustRoom(t, s, roomID).State()
				if err != nil {
					t.Errorf("GetMatchState() error = %v", err)
					return
				}
				if _, err := json.Marshal(gs); err != nil {
					t.Errorf("Marshal(gameState) error = %v", err)
					return
				}
				if _, err := mustRoom(t, s, roomID).AllowedTiles(cpuUnitID, engine.TurnCmdMove); err != nil {
					t.Errorf("GetAllowedTiles() error = %v", err)
					return
				}
				if _, err := mustRoom(t, s, roomID).Config(); err != nil {
					t.Errorf("GetMatchConfig() error = %v", err)
					return
				}
			}
		})
	}

	close(release)
	activeTeam := mustMatchState(t, s, roomID).ActiveTeam
	close(stop)
	readers.Wait()

	if got, want := activeTeam, 1; got != want {
		t.Errorf("active team after the CPU turn = %d, want %d", got, want)
	}
	if phase, _, _ := consumeCPUPhase(t, s, roomID, tokens[0]); phase != room.TurnPhaseReady {
		t.Errorf("Expected phase %v after the CPU turn, got %v", room.TurnPhaseReady, phase)
	}
}

func TestServerStateManager_runCPUTurn_Panic(t *testing.T) {
	held := holdCPUTurn(t)
	roomID, tokens, s := newCPUTurnRoom(t,
		WithDecider(func(*engine.GameState) ([]engine.TurnCommand, error) { panic("decide exploded") }),
		WithSpawner(held.spawn),
		arrangedMatch(cpuTurnPending, bombPlacedBy(engine.NewUnitID(2, 0))))

	consumeCPUPhase(t, s, roomID, tokens[0])
	held.runOutsideRoomLock()

	if got, want := mustMatchState(t, s, roomID).Turn, 3; got != want {
		t.Errorf("Expected the forfeited turn to still advance to %d, got %d", want, got)
	}

	// Proves the room mutex was released rather than left held by the unwinding goroutine.
	turnPhase, _, resolveTurnGameEvents := consumeCPUPhase(t, s, roomID, tokens[0])
	if got, want := turnPhase, room.TurnPhaseReady; got != want {
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
		validate func(t *testing.T, turnPhase room.CPUTurnPhase, planGameEvents, resolveTurnGameEvents []engine.GameEvent, s *ServerStateManager, roomID, token string)
	}{
		{
			name: "Success - TurnPhaseReady and clear status",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				t.Helper()
				decider := &recordingDecider{plan: func(*engine.GameState, int) []engine.TurnCommand {
					return []engine.TurnCommand{engine.NewMoveCommand(unitID, engine.Coordinate{X: 4, Y: 1})}
				}}
				held := holdCPUTurn(t)
				roomID, tokens, s := newCPUTurnRoom(t, WithDecider(decider.decide), WithSpawner(held.spawn))
				consumeCPUPhase(t, s, roomID, tokens[0])
				held.runOutsideRoomLock()
				return roomID, s, tokens[0]
			},
			wantErr: nil,
			validate: func(t *testing.T, turnPhase room.CPUTurnPhase, planGameEvents, resolveTurnGameEvents []engine.GameEvent, s *ServerStateManager, roomID, token string) {
				t.Helper()

				if got, want := turnPhase, room.TurnPhaseReady; got != want {
					t.Errorf("Expected %v turnPhase return, got %v", want, got)
				}
				if !hasGameEvent(planGameEvents, engine.GameEvtUnitMoved, unitID) {
					t.Errorf("Expected the move in planGameEvents, got %#v", planGameEvents)
				}

				phase, planAgain, resolveAgain := consumeCPUPhase(t, s, roomID, token)
				if got, want := phase, room.TurnPhaseIdle; got != want {
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
				roomID, tokens, s := newCPUTurnRoom(t, WithSpawner(holdCPUTurn(t).spawn))
				return roomID, s, tokens[0]
			},
			wantErr: nil,
			validate: func(t *testing.T, turnPhase room.CPUTurnPhase, planGameEvents, resolveTurnGameEvents []engine.GameEvent, s *ServerStateManager, roomID, token string) {
				t.Helper()

				if got, want := turnPhase, room.TurnPhasePlanning; got != want {
					t.Errorf("Expected %v turnPhase return, got %v", want, got)
				}
				if len(planGameEvents) != 0 || len(resolveTurnGameEvents) != 0 {
					t.Errorf("Expected no events while planning, got %#v and %#v", planGameEvents, resolveTurnGameEvents)
				}

				if phase, _, _ := consumeCPUPhase(t, s, roomID, token); phase != room.TurnPhasePlanning {
					t.Errorf("Expected the second consume to stay at %v, got %v", room.TurnPhasePlanning, phase)
				}
			},
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				return roomID, s, "dummy-token"
			},
			wantErr: room.ErrMatchNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s, token := tt.setup(t)
			status, err := mustRoom(t, s, roomID).ConsumeCPU(token)
			turnPhase, planGameEvents, resolveTurnGameEvents := status.Phase, status.Plan, status.Resolve
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
				_, err := mustRoom(t, s, roomID).Submit(cpuMove, token)
				return err
			},
		},
		{
			name: "ResetTurn",
			call: func(s *ServerStateManager, roomID, token string) error {
				return mustRoom(t, s, roomID).Reset(token)
			},
		},
		{
			name: "ResolveTurn",
			call: func(s *ServerStateManager, roomID, token string) error {
				_, _, err := mustRoom(t, s, roomID).Resolve(token)
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
				held := holdCPUTurn(t)
				roomID, tokens, s := newCPUTurnRoom(t, WithDecider((&recordingDecider{}).decide), WithSpawner(held.spawn))

				callToken, wantErr := tokens[1], error(nil)
				turn, team := 2, 2
				switch phase {
				case "Planning":
					consumeCPUPhase(t, s, roomID, tokens[0])
					wantErr = room.ErrCPUTurnInProgress
				case "Ready":
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
				gs := mustMatchState(t, s, roomID)
				if got := gs.Turn; got != turn {
					t.Errorf("Turn = %d, want %d", got, turn)
				}
				if got := gs.ActiveTeam; got != team {
					t.Errorf("ActiveTeam = %d, want %d", got, team)
				}
			})
		}
	}

	t.Run("Bad token still gets 401", func(t *testing.T) {
		roomID, tokens, s := newCPUTurnRoom(t, WithSpawner(holdCPUTurn(t).spawn))
		consumeCPUPhase(t, s, roomID, tokens[0])
		if _, _, err := mustRoom(t, s, roomID).Resolve("wrong"); !errors.Is(err, room.ErrInvalidToken) {
			t.Errorf("error = %v, want %v", err, room.ErrInvalidToken)
		}
	})
}

func TestServerStateManager_CPUTurnGate_NotGated(t *testing.T) {
	held := holdCPUTurn(t)
	roomID, tokens, s := newCPUTurnRoom(t, WithSpawner(held.spawn))

	phase, _, _ := consumeCPUPhase(t, s, roomID, tokens[0])
	if want := room.TurnPhasePlanning; phase != want {
		t.Errorf("ConsumeCPUStatus phase = %v, want %v", phase, want)
	}

	surrenderEvents, err := mustRoom(t, s, roomID).Surrender(1, tokens[0])
	if err != nil {
		t.Fatalf("Surrender() error = %v", err)
	}
	if !slices.ContainsFunc(surrenderEvents, func(evt engine.GameEvent) bool {
		return evt.Type == engine.GameEvtMatchEnded && evt.WinnerTeamID == 2
	}) {
		t.Errorf("Expected a matchEnded event with winner 2, got %+v", surrenderEvents)
	}

	held.runOutsideRoomLock()
	phase, planGameEvents, resolveTurnGameEvents := consumeCPUPhase(t, s, roomID, tokens[0])
	if want := room.TurnPhaseReady; phase != want {
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
			decider := &recordingDecider{
				plan: func(*engine.GameState, int) []engine.TurnCommand { return tt.cmds },
				err:  decideErr,
			}
			held := holdCPUTurn(t)
			h := &captureHandler{}
			roomID, tokens, s := newCPUTurnRoom(t, WithDecider(decider.decide), WithSpawner(held.spawn), WithLogger(slog.New(h)))

			consumeCPUPhase(t, s, roomID, tokens[0])
			held.runOutsideRoomLock()
			phase, planGameEvents, _ := consumeCPUPhase(t, s, roomID, tokens[0])

			if got, want := decider.calls, 1; got != want {
				t.Errorf("decide calls = %d, want %d", got, want)
			}
			if got, want := phase, room.TurnPhaseReady; got != want {
				t.Errorf("Phase = %v, want %v", got, want)
			}
			if got, want := mustMatchState(t, s, roomID).ActiveTeam, 1; got != want {
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
	decider := &recordingDecider{}
	held := holdCPUTurn(t)
	kingID := engine.NewUnitID(2, 0)
	roomID, tokens, s := createTestRoomWithCfg(t, gameCfg, WithDecider(decider.decide), WithSpawner(held.spawn),
		arrangedMatch(cpuTurnPending, bombAboutToExplode(kingID)))
	hazardIDs := func(gs *engine.GameState) []engine.BombID {
		var ids []engine.BombID
		for id, bomb := range gs.Bombs {
			if bomb.OwnerID == engine.SystemUnitID {
				ids = append(ids, id)
			}
		}
		return ids
	}
	hazardsBefore := hazardIDs(mustMatchState(t, s, roomID))
	if len(hazardsBefore) == 0 {
		t.Fatal("Expected hazards at creation with MaxTurns 0")
	}

	if phase, _, _ := consumeCPUPhase(t, s, roomID, tokens[0]); phase != room.TurnPhasePlanning {
		t.Fatalf("launching consume phase = %v, want %v", phase, room.TurnPhasePlanning)
	}
	held.runOutsideRoomLock()

	gs := mustMatchState(t, s, roomID)
	if got, want := gs.Turn, 2; got != want {
		t.Errorf("Expected turn %d to stay, got %d", want, got)
	}
	if got, want := gs.ActiveTeam, 2; got != want {
		t.Errorf("Expected active team %d to stay, got %d", want, got)
	}
	for _, id := range hazardIDs(gs) {
		if !slices.Contains(hazardsBefore, id) {
			t.Errorf("Expected no new hazards, found bomb %v", id)
		}
	}

	phase, planGameEvents, resolveTurnGameEvents := consumeCPUPhase(t, s, roomID, tokens[0])
	if got, want := phase, room.TurnPhaseReady; got != want {
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
		if got, want := phase, room.TurnPhaseIdle; got != want {
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
