package room

import (
	"bomb-srpg/engine"
	"errors"
	"reflect"
	"slices"
	"testing"
)

func hasUnitMoved(gameEvents []engine.GameEvent, unitID engine.UnitID) bool {
	return slices.ContainsFunc(gameEvents, func(evt engine.GameEvent) bool {
		moved, ok := evt.(engine.UnitMoved)
		return ok && moved.UnitID == unitID
	})
}

func hasBombPlaced(gameEvents []engine.GameEvent, unitID engine.UnitID) bool {
	return slices.ContainsFunc(gameEvents, func(evt engine.GameEvent) bool {
		placed, ok := evt.(engine.BombPlaced)
		return ok && placed.UnitID == unitID
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
				if !hasUnitMoved(m.PlaybackLog, moverUnitID) {
					t.Errorf("Expected unit %#x to have moved, got %#v", moverUnitID, m.PlaybackLog)
				}
				if !hasBombPlaced(m.PlaybackLog, bomberUnitID) {
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
				if hasBombPlaced(m.PlaybackLog, bomberUnitID) {
					t.Errorf("Expected command after the rejected one to be skipped, got %#v", m.PlaybackLog)
				}
			},
		},
		{
			name: "Unsupported Command",
			plan: func(t *testing.T, m *engine.Match) []engine.TurnCommand {
				t.Helper()
				return []engine.TurnCommand{nil}
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
	if status.Phase != TurnPhaseReady {
		t.Errorf("ConsumeCPU() phase = %v, want %v", status.Phase, TurnPhaseReady)
	}
	if len(status.Plan) != 0 || len(status.Resolve) != 0 {
		t.Errorf("ConsumeCPU() events = %+v and %+v, want none", status.Plan, status.Resolve)
	}
}

func TestCPUTurnPhaseString(t *testing.T) {
	tests := []struct {
		name      string
		turnPhase CPUTurnPhase
		want      string
	}{
		{"TurnPhase Idle", TurnPhaseIdle, "TurnPhaseIdle"},
		{"TurnPhase Planning", TurnPhasePlanning, "TurnPhasePlanning"},
		{"TurnPhase Ready", TurnPhaseReady, "TurnPhaseReady"},
		{"Invalid Value", CPUTurnPhase(99), "TurnPhaseUnknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.turnPhase.String(); got != tt.want {
				t.Errorf("CPUTurnPhase.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRoom_CPUStateReset(t *testing.T) {
	tests := []struct {
		name      string
		act       func(t *testing.T, f fixture)
		wantReset bool
	}{
		{
			name: "Rematch",
			act: func(t *testing.T, f fixture) {
				t.Helper()
				if _, err := f.r.Rematch(f.tokens[0]); err != nil {
					t.Fatalf("Rematch() error = %v", err)
				}
			},
			wantReset: true,
		},
		{
			name: "Delete",
			act: func(t *testing.T, f fixture) {
				t.Helper()
				endAndDelete(t, f.r, f.tokens[0])
			},
			wantReset: true,
		},
		{
			name: "Surrender then Rematch",
			act: func(t *testing.T, f fixture) {
				t.Helper()
				if _, err := f.r.Surrender(1, f.tokens[0]); err != nil {
					t.Fatalf("Surrender() error = %v", err)
				}
				if _, err := f.r.Rematch(f.tokens[0]); err != nil {
					t.Fatalf("Rematch() error = %v", err)
				}
			},
			wantReset: true,
		},
		{
			name: "second Start",
			act: func(t *testing.T, f fixture) {
				t.Helper()
				if _, err := f.r.Start(vsCPUCfg()); !errors.Is(err, ErrMatchExists) {
					t.Fatalf("Start() error = %v, want %v", err, ErrMatchExists)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := cpuReadyFixture(t)
			before := f.r.cpu
			if before.phase != TurnPhaseReady || len(before.plan) == 0 {
				t.Fatalf("setup cpu state = %+v, want phase Ready with buffered events", before)
			}

			tt.act(t, f)

			want := before
			if tt.wantReset {
				want = cpuTurn{}
			}
			if !reflect.DeepEqual(f.r.cpu, want) {
				t.Errorf("cpu state = %+v, want %+v", f.r.cpu, want)
			}
		})
	}
}
