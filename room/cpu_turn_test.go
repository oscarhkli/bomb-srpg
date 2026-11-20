package room

import (
	"bomb-srpg/engine"
	"errors"
	"slices"
	"testing"
)

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
			match, err := engine.InitGame(validGameCfg())
			if err != nil {
				t.Fatalf("InitGame() error = %v", err)
			}

			err = applyPlan(match, tt.plan(t, match))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("applyPlan() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, match)
			}
		})
	}
}

func TestRoom_CPUTurnAbandons_NotCPUsTurn(t *testing.T) {
	f := planningFixture(t)
	f.r.mu.Lock()
	f.r.match.WorkingState.ActiveTeam = 1
	f.r.mu.Unlock()

	f.held.runAll()

	if got := *f.decides; got != 0 {
		t.Errorf("Decide calls = %d, want 0", got)
	}
	status, err := f.r.ConsumeCPU(f.tokens[0])
	if err != nil {
		t.Fatalf("ConsumeCPU() error = %v", err)
	}
	if status.Phase != engine.TurnPhaseReady {
		t.Errorf("ConsumeCPU() phase = %v, want %v", status.Phase, engine.TurnPhaseReady)
	}
	if len(status.Plan) != 0 || len(status.Resolve) != 0 {
		t.Errorf("ConsumeCPU() events = %+v and %+v, want none", status.Plan, status.Resolve)
	}
}
