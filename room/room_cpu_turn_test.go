package room

import (
	"bomb-srpg/engine"
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

func TestRoom_ConsumeCPU_LaunchesCPUTurn(t *testing.T) {
	humanUnitID := engine.NewUnitID(1, 0)
	cpuUnitID := engine.NewUnitID(2, 0)

	legalCPUMove := []engine.TurnCommand{engine.NewMoveCommand(cpuUnitID, engine.Coordinate{X: 4, Y: 1})}
	rejectedPlan := []engine.TurnCommand{engine.NewMoveCommand(humanUnitID, engine.Coordinate{X: 1, Y: 1})}

	tests := []struct {
		name           string
		vsCpu          bool
		maxTurns       int
		arrange        []func(*engine.Match) error
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
			arrange:  []func(*engine.Match) error{cpuTurnPending},
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
			arrange:  []func(*engine.Match) error{cpuTurnPending},
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
			arrange:  []func(*engine.Match) error{cpuTurnPending},
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
			arrange:  []func(*engine.Match) error{cpuTurnPending},
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
			arrange:  []func(*engine.Match) error{firstTurnResolved},
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
			arrange:        []func(*engine.Match) error{cpuTurnPending, matchEnded},
			tokenIdx:       1,
			wantCPUTurn:    false,
			wantCalls:      0,
			wantActiveTeam: 2,
		},
		{
			name:           "Not VS CPU",
			vsCpu:          false,
			arrange:        []func(*engine.Match) error{cpuTurnPending},
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
			f := startedFixtureWith(t, gameCfg, withDecide(decider.decide), tt.arrange...)
			held := f.held

			wantPhase, wantHeld := TurnPhaseIdle, 0
			if tt.wantCPUTurn {
				wantPhase, wantHeld = TurnPhasePlanning, 1
			}
			phase, planGameEvents, resolveTurnGameEvents := consumeCPUPhase(t, f.r, f.tokens[tt.tokenIdx])
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
				held.runNext(t)
				phase, planGameEvents, _ = consumeCPUPhase(t, f.r, f.tokens[tt.tokenIdx])
				if phase != TurnPhaseReady {
					t.Errorf("ConsumeCPU() after the turn phase = %v, want %v", phase, TurnPhaseReady)
				}
			}

			if got, want := decider.calls, tt.wantCalls; got != want {
				t.Errorf("Expected %d decide calls, got %d", want, got)
			}
			if got, want := mustState(t, f.r).ActiveTeam, tt.wantActiveTeam; got != want {
				t.Errorf("Expected active team %d, got %d", want, got)
			}
			if tt.validate != nil {
				tt.validate(t, planGameEvents, decider)
			}
		})
	}
}

func TestRoom_ConsumeCPU_StaleReadyDropped(t *testing.T) {
	cpuUnitID := engine.NewUnitID(2, 0)
	posA, posB := engine.Coordinate{X: 4, Y: 1}, engine.Coordinate{X: 3, Y: 1}

	planA := moveThenBombPlan(t, cpuUnitID, posA)
	decider := &recordingDecider{plan: func(gs *engine.GameState, call int) []engine.TurnCommand {
		if call == 0 {
			return planA(gs, call)
		}
		return []engine.TurnCommand{engine.NewMoveCommand(cpuUnitID, posB)}
	}}
	f := cpuTurnFixture(t, withDecide(decider.decide))

	consumeCPUPhase(t, f.r, f.tokens[0])
	f.held.runNext(t)

	if _, _, err := f.r.Resolve(f.tokens[0]); err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	phase, planGameEvents, resolveTurnGameEvents := consumeCPUPhase(t, f.r, f.tokens[0])
	if phase != TurnPhasePlanning {
		t.Fatalf("launching consume phase = %v, want %v", phase, TurnPhasePlanning)
	}
	if len(planGameEvents) != 0 || len(resolveTurnGameEvents) != 0 {
		t.Errorf("launching consume events = %#v and %#v, want none", planGameEvents, resolveTurnGameEvents)
	}
	if got, want := f.held.count(), 2; got != want {
		t.Fatalf("held CPU turns = %d, want %d", got, want)
	}

	f.held.runNext(t)
	phase, planGameEvents, _ = consumeCPUPhase(t, f.r, f.tokens[0])
	if phase != TurnPhaseReady {
		t.Fatalf("phase after the second turn = %v, want %v", phase, TurnPhaseReady)
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

func TestRoom_ConsumeCPU_ClearsResolveEvents(t *testing.T) {
	cpuUnitID := engine.NewUnitID(2, 0)
	decider := &recordingDecider{plan: moveThenBombPlan(t, cpuUnitID, engine.Coordinate{X: 4, Y: 1})}
	f := cpuTurnFixture(t, withDecide(decider.decide))

	consumeCPUPhase(t, f.r, f.tokens[0])
	f.held.runNext(t)

	phase, _, resolveTurnGameEvents := consumeCPUPhase(t, f.r, f.tokens[0])
	if phase != TurnPhaseReady {
		t.Fatalf("phase after the turn = %v, want %v", phase, TurnPhaseReady)
	}
	if len(resolveTurnGameEvents) == 0 {
		t.Fatal("Expected the placed bomb to tick in the resolve events, got none")
	}

	phase, planAgain, resolveAgain := consumeCPUPhase(t, f.r, f.tokens[0])
	if phase != TurnPhaseIdle {
		t.Errorf("second consume phase = %v, want %v", phase, TurnPhaseIdle)
	}
	if len(planAgain) != 0 || len(resolveAgain) != 0 {
		t.Errorf("second consume events = %#v and %#v, want none", planAgain, resolveAgain)
	}
}

func TestRoom_ConsumeCPU_InvalidToken(t *testing.T) {
	decider := &recordingDecider{}
	f := cpuTurnFixture(t, withDecide(decider.decide))

	if _, err := f.r.ConsumeCPU("wrong"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("ConsumeCPU() error = %v, want %v", err, ErrInvalidToken)
	}

	if got := f.held.count(); got != 0 {
		t.Errorf("held CPU turns = %d, want 0", got)
	}
	if got := decider.calls; got != 0 {
		t.Errorf("decide calls = %d, want 0", got)
	}
}

func TestRoom_ConsumeCPU_LaunchesOnce(t *testing.T) {
	decider := &recordingDecider{}
	f := cpuTurnFixture(t, withDecide(decider.decide))

	for range 2 {
		if phase, _, _ := consumeCPUPhase(t, f.r, f.tokens[0]); phase != TurnPhasePlanning {
			t.Fatalf("phase = %v, want %v", phase, TurnPhasePlanning)
		}
	}
	if got, want := f.held.count(), 1; got != want {
		t.Fatalf("held CPU turns = %d, want %d", got, want)
	}

	f.held.runNext(t)
	if got, want := decider.calls, 1; got != want {
		t.Errorf("decide calls = %d, want %d", got, want)
	}
}

func TestRoom_CPUTurn_Abandons(t *testing.T) {
	tests := []struct {
		name      string
		act       func(t *testing.T, f fixture)
		wantErr   error
		wantPhase CPUTurnPhase
	}{
		{
			name: "Match Replaced",
			act: func(t *testing.T, f fixture) {
				t.Helper()
				if _, err := f.r.Rematch(f.tokens[0]); err != nil {
					t.Fatalf("Rematch() error = %v", err)
				}
			},
			wantPhase: TurnPhaseIdle,
		},
		{
			name: "Match Deleted",
			act: func(t *testing.T, f fixture) {
				t.Helper()
				endAndDelete(t, f.r, f.tokens[0])
			},
			wantErr: ErrMatchNotFound,
		},
		{
			name: "Match Already Ended",
			act: func(t *testing.T, f fixture) {
				t.Helper()
				if _, err := f.r.Surrender(1, f.tokens[0]); err != nil {
					t.Fatalf("Surrender() error = %v", err)
				}
			},
			wantPhase: TurnPhaseReady,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decider := &recordingDecider{}
			var captured *engine.Match
			firstCPUTurnPending := func(match *engine.Match) error {
				if captured != nil {
					return nil
				}
				captured = match
				return cpuTurnPending(match)
			}
			f := startedFixtureWith(t, cpuTurnCfg(), withDecide(decider.decide), firstCPUTurnPending)

			if phase, _, _ := consumeCPUPhase(t, f.r, f.tokens[0]); phase != TurnPhasePlanning {
				t.Fatalf("launching consume phase = %v, want %v", phase, TurnPhasePlanning)
			}
			tt.act(t, f)
			wantTrueState := captured.TrueState.DeepCopy()
			f.held.runNext(t)

			if got := decider.calls; got != 0 {
				t.Errorf("Expected the abandoned turn to skip planning, got %d decide calls", got)
			}
			if got := captured.TrueState; !reflect.DeepEqual(got, wantTrueState) {
				t.Errorf("Expected the abandoned match's committed state to stay unchanged, got %#v, want %#v", got, wantTrueState)
			}

			status, err := f.r.ConsumeCPU(f.tokens[0])
			phase, planGameEvents, resolveTurnGameEvents := status.Phase, status.Plan, status.Resolve
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ConsumeCPU() error = %v, want %v", err, tt.wantErr)
			}
			if got, want := phase, tt.wantPhase; got != want {
				t.Errorf("ConsumeCPU() phase = %v, want %v", got, want)
			}
			if len(planGameEvents) != 0 || len(resolveTurnGameEvents) != 0 {
				t.Errorf("ConsumeCPU() events = %#v and %#v, want none", planGameEvents, resolveTurnGameEvents)
			}
		})
	}
}

func TestRoom_ReadsDuringCPUTurnDoNotRaceMatch(t *testing.T) {
	cpuUnitID := engine.NewUnitID(2, 0)

	planning, release := make(chan struct{}), make(chan struct{})
	decider := &recordingDecider{plan: func(*engine.GameState, int) []engine.TurnCommand {
		close(planning)
		<-release
		return nil
	}}
	f := cpuTurnFixture(t, func(d *Deps) {
		d.Decide = decider.decide
		d.Spawn = func(run func()) { go run() }
	})

	if phase, _, _ := consumeCPUPhase(t, f.r, f.tokens[0]); phase != TurnPhasePlanning {
		t.Fatalf("phase = %v, want %v", phase, TurnPhasePlanning)
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
				gs, err := f.r.State()
				if err != nil {
					t.Errorf("State() error = %v", err)
					return
				}
				if _, err := json.Marshal(gs); err != nil {
					t.Errorf("Marshal(gameState) error = %v", err)
					return
				}
				if _, err := f.r.AllowedTiles(cpuUnitID, engine.TurnCmdMove); err != nil {
					t.Errorf("AllowedTiles() error = %v", err)
					return
				}
				if _, err := f.r.Config(); err != nil {
					t.Errorf("Config() error = %v", err)
					return
				}
			}
		})
	}

	close(release)
	activeTeam := mustState(t, f.r).ActiveTeam
	close(stop)
	readers.Wait()

	if got, want := activeTeam, 1; got != want {
		t.Errorf("active team after the CPU turn = %d, want %d", got, want)
	}
	if phase, _, _ := consumeCPUPhase(t, f.r, f.tokens[0]); phase != TurnPhaseReady {
		t.Errorf("Expected phase %v after the CPU turn, got %v", TurnPhaseReady, phase)
	}
}

func TestRoom_CPUTurn_PanicReleasesRoomMutex(t *testing.T) {
	f := startedFixtureWith(t, cpuTurnCfg(),
		withDecide(func(*engine.GameState) ([]engine.TurnCommand, error) { panic("decide exploded") }),
		cpuTurnPending, bombPlacedBy(engine.NewUnitID(2, 0)))

	consumeCPUPhase(t, f.r, f.tokens[0])
	f.held.runNext(t)

	if got, want := mustState(t, f.r).Turn, 3; got != want {
		t.Errorf("Expected the forfeited turn to still advance to %d, got %d", want, got)
	}

	turnPhase, _, resolveTurnGameEvents := consumeCPUPhase(t, f.r, f.tokens[0])
	if got, want := turnPhase, TurnPhaseReady; got != want {
		t.Errorf("Expected %v turnPhase return, got %v", want, got)
	}
	if len(resolveTurnGameEvents) == 0 {
		t.Error("Expected the forfeited turn to report the ticking bomb in the resolve events, got none")
	}
}

func TestRoom_ConsumeCPU(t *testing.T) {
	unitID := engine.NewUnitID(2, 0)
	tests := []struct {
		name     string
		setup    func(t *testing.T) (*Room, string)
		validate func(t *testing.T, turnPhase CPUTurnPhase, planGameEvents, resolveTurnGameEvents []engine.GameEvent, r *Room, token string)
	}{
		{
			name: "Success - TurnPhaseReady and clear status",
			setup: func(t *testing.T) (*Room, string) {
				t.Helper()
				decider := &recordingDecider{plan: func(*engine.GameState, int) []engine.TurnCommand {
					return []engine.TurnCommand{engine.NewMoveCommand(unitID, engine.Coordinate{X: 4, Y: 1})}
				}}
				f := cpuTurnFixture(t, withDecide(decider.decide))
				consumeCPUPhase(t, f.r, f.tokens[0])
				f.held.runNext(t)
				return f.r, f.tokens[0]
			},
			validate: func(t *testing.T, turnPhase CPUTurnPhase, planGameEvents, resolveTurnGameEvents []engine.GameEvent, r *Room, token string) {
				t.Helper()

				if got, want := turnPhase, TurnPhaseReady; got != want {
					t.Errorf("Expected %v turnPhase return, got %v", want, got)
				}
				if !hasGameEvent(planGameEvents, engine.GameEvtUnitMoved, unitID) {
					t.Errorf("Expected the move in planGameEvents, got %#v", planGameEvents)
				}

				phase, planAgain, resolveAgain := consumeCPUPhase(t, r, token)
				if got, want := phase, TurnPhaseIdle; got != want {
					t.Errorf("Expected the second consume to return %v, got %v", want, got)
				}
				if len(planAgain) != 0 || len(resolveAgain) != 0 {
					t.Errorf("Expected the second consume to return no events, got %#v and %#v", planAgain, resolveAgain)
				}
			},
		},
		{
			name: "Success - non-TurnPhaseReady and do nothing",
			setup: func(t *testing.T) (*Room, string) {
				t.Helper()
				f := cpuTurnFixture(t, nil)
				return f.r, f.tokens[0]
			},
			validate: func(t *testing.T, turnPhase CPUTurnPhase, planGameEvents, resolveTurnGameEvents []engine.GameEvent, r *Room, token string) {
				t.Helper()

				if got, want := turnPhase, TurnPhasePlanning; got != want {
					t.Errorf("Expected %v turnPhase return, got %v", want, got)
				}
				if len(planGameEvents) != 0 || len(resolveTurnGameEvents) != 0 {
					t.Errorf("Expected no events while planning, got %#v and %#v", planGameEvents, resolveTurnGameEvents)
				}

				if phase, _, _ := consumeCPUPhase(t, r, token); phase != TurnPhasePlanning {
					t.Errorf("Expected the second consume to stay at %v, got %v", TurnPhasePlanning, phase)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, token := tt.setup(t)
			status, err := r.ConsumeCPU(token)
			if err != nil {
				t.Fatalf("ConsumeCPU() error = %v", err)
			}
			tt.validate(t, status.Phase, status.Plan, status.Resolve, r, token)
		})
	}
}

func TestRoom_CPUTurnGate(t *testing.T) {
	cpuMove := engine.NewMoveCommand(engine.NewUnitID(2, 0), engine.Coordinate{X: 4, Y: 1})

	calls := []struct {
		name     string
		call     func(r *Room, token string) error
		resolved bool
	}{
		{
			name: "SubmitTurnCommand",
			call: func(r *Room, token string) error {
				_, err := r.Submit(cpuMove, token)
				return err
			},
		},
		{
			name: "ResetTurn",
			call: func(r *Room, token string) error {
				return r.Reset(token)
			},
		},
		{
			name: "ResolveTurn",
			call: func(r *Room, token string) error {
				_, _, err := r.Resolve(token)
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
				f := cpuTurnFixture(t, withDecide((&recordingDecider{}).decide))

				callToken, wantErr := f.tokens[1], error(nil)
				turn, team := 2, 2
				switch phase {
				case "Planning":
					consumeCPUPhase(t, f.r, f.tokens[0])
					wantErr = ErrCPUTurnInProgress
				case "Ready":
					consumeCPUPhase(t, f.r, f.tokens[0])
					f.held.runNext(t)
					callToken = f.tokens[0]
					turn, team = 3, 1
				}

				err := c.call(f.r, callToken)
				if !errors.Is(err, wantErr) {
					t.Fatalf("error = %v, want %v", err, wantErr)
				}

				if wantErr == nil && c.resolved {
					turn, team = turn+1, 3-team
				}
				gs := mustState(t, f.r)
				if got := gs.Turn; got != turn {
					t.Errorf("Turn = %d, want %d", got, turn)
				}
				if got := gs.ActiveTeam; got != team {
					t.Errorf("ActiveTeam = %d, want %d", got, team)
				}
			})
		}
	}
}

func TestRoom_CPUTurnGate_NotGated(t *testing.T) {
	f := cpuTurnFixture(t, nil)

	phase, _, _ := consumeCPUPhase(t, f.r, f.tokens[0])
	if want := TurnPhasePlanning; phase != want {
		t.Errorf("ConsumeCPU phase = %v, want %v", phase, want)
	}

	surrenderEvents, err := f.r.Surrender(1, f.tokens[0])
	if err != nil {
		t.Fatalf("Surrender() error = %v", err)
	}
	if !slices.ContainsFunc(surrenderEvents, func(evt engine.GameEvent) bool {
		return evt.Type == engine.GameEvtMatchEnded && evt.WinnerTeamID == 2
	}) {
		t.Errorf("Expected a matchEnded event with winner 2, got %+v", surrenderEvents)
	}

	f.held.runNext(t)
	phase, planGameEvents, resolveTurnGameEvents := consumeCPUPhase(t, f.r, f.tokens[0])
	if want := TurnPhaseReady; phase != want {
		t.Errorf("ConsumeCPU phase after the abandoned turn = %v, want %v", phase, want)
	}
	if len(planGameEvents) != 0 || len(resolveTurnGameEvents) != 0 {
		t.Errorf("Expected no events, got %#v and %#v", planGameEvents, resolveTurnGameEvents)
	}
}

func TestRoom_CPUTurn_DecideError(t *testing.T) {
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
			h := &captureHandler{}
			f := cpuTurnFixture(t, func(d *Deps) {
				d.Decide = decider.decide
				d.Logger = slog.New(h)
			})

			consumeCPUPhase(t, f.r, f.tokens[0])
			f.held.runNext(t)
			phase, planGameEvents, _ := consumeCPUPhase(t, f.r, f.tokens[0])

			if got, want := decider.calls, 1; got != want {
				t.Errorf("decide calls = %d, want %d", got, want)
			}
			if got, want := phase, TurnPhaseReady; got != want {
				t.Errorf("Phase = %v, want %v", got, want)
			}
			if got, want := mustState(t, f.r).ActiveTeam, 1; got != want {
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

func TestRoom_CPUTurn_EndsMatch(t *testing.T) {
	gameCfg := cpuTurnCfg()
	gameCfg.MaxTurns = 0
	decider := &recordingDecider{}
	kingID := engine.NewUnitID(2, 0)
	f := startedFixtureWith(t, gameCfg, withDecide(decider.decide), cpuTurnPending, bombAboutToExplode(kingID))
	hazardIDs := func(gs *engine.GameState) []engine.BombID {
		var ids []engine.BombID
		for id, bomb := range gs.Bombs {
			if bomb.OwnerID == engine.SystemUnitID {
				ids = append(ids, id)
			}
		}
		return ids
	}
	hazardsBefore := hazardIDs(mustState(t, f.r))
	if len(hazardsBefore) == 0 {
		t.Fatal("Expected hazards at creation with MaxTurns 0")
	}

	if phase, _, _ := consumeCPUPhase(t, f.r, f.tokens[0]); phase != TurnPhasePlanning {
		t.Fatalf("launching consume phase = %v, want %v", phase, TurnPhasePlanning)
	}
	f.held.runNext(t)

	gs := mustState(t, f.r)
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

	phase, planGameEvents, resolveTurnGameEvents := consumeCPUPhase(t, f.r, f.tokens[0])
	if got, want := phase, TurnPhaseReady; got != want {
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
		phase, planAgain, resolveAgain := consumeCPUPhase(t, f.r, f.tokens[0])
		if got, want := phase, TurnPhaseIdle; got != want {
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
