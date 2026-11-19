package room

import (
	"bomb-srpg/engine"
	"bytes"
	"errors"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func validGameCfg() engine.GameCfg {
	return engine.GameCfg{
		StagePreset: "Plain",
		P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
		P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
		MaxTurns:    10,
	}
}

func hasGameEvent(gameEvents []engine.GameEvent, evtType engine.GameEvtType, unitID engine.UnitID) bool {
	return slices.ContainsFunc(gameEvents, func(evt engine.GameEvent) bool {
		return evt.Type == evtType && evt.UnitID == unitID
	})
}

func TestRoom_Rematch_KeepsMatchWhenBuildFails(t *testing.T) {
	buildErr := errors.New("build failed")
	builds := 0
	r := New("room", Deps{
		Logger: slog.New(slog.DiscardHandler),
		NewMatch: func(cfg engine.GameCfg) (*engine.Match, error) {
			builds++
			if builds > 1 {
				return nil, buildErr
			}
			return engine.InitGame(cfg)
		},
	})
	tokens, err := r.Start(validGameCfg())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	want, err := r.State()
	if err != nil {
		t.Fatalf("State() error = %v", err)
	}

	_, err = r.Rematch(tokens[0])
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("Rematch() error = %v, want %v", err, ErrInvalidConfig)
	}

	got, err := r.State()
	if err != nil {
		t.Fatalf("State() after failed Rematch error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("State() after failed Rematch = %+v, want the first match's %+v", got, want)
	}
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

func TestRoom_Rematch_ReplacesMatchAndKeepsTokens(t *testing.T) {
	r := New("room", Deps{Logger: slog.New(slog.DiscardHandler), NewMatch: engine.InitGame})
	tokens, err := r.Start(validGameCfg())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	first := r.match

	got, err := r.Rematch(tokens[0])
	if err != nil {
		t.Fatalf("Rematch() error = %v", err)
	}

	if got != tokens {
		t.Errorf("Rematch() tokens = %v, want %v", got, tokens)
	}
	if r.match == nil || r.match == first {
		t.Errorf("Match after Rematch = %p, want a new match distinct from %p", r.match, first)
	}
	if _, err := r.State(); err != nil {
		t.Errorf("State() after Rematch error = %v", err)
	}
}

func TestRoom_Rematch_WithoutConfig(t *testing.T) {
	r := New("room", Deps{Logger: slog.New(slog.DiscardHandler), NewMatch: engine.InitGame})
	tokens, err := r.Start(validGameCfg())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	r.mu.Lock()
	r.cfg = nil
	r.mu.Unlock()

	_, err = r.Rematch(tokens[0])
	if !errors.Is(err, ErrMatchNotFound) {
		t.Errorf("Rematch() error = %v, want %v", err, ErrMatchNotFound)
	}
}

func TestRoom_CPUTurnAbandons_NotCPUsTurn(t *testing.T) {
	f := planningFixture(t)
	f.r.mu.Lock()
	f.r.match.WorkingState.ActiveTeam = 1
	f.r.mu.Unlock()

	f.held.runAll()

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

func TestRoom_LogsCarryRoomID(t *testing.T) {
	var out bytes.Buffer
	r := New("room-42", Deps{Logger: slog.New(slog.NewTextHandler(&out, nil)), NewMatch: engine.InitGame})

	if _, err := r.State(); !errors.Is(err, ErrMatchNotFound) {
		t.Fatalf("State() error = %v, want %v", err, ErrMatchNotFound)
	}

	if got := out.String(); !strings.Contains(got, "roomID=room-42") {
		t.Errorf("log output = %q, want it to carry roomID=room-42", got)
	}
}
