package room

import (
	"bomb-srpg/engine"
	"errors"
	"log/slog"
	"testing"
)

type heldSpawn struct{ fns []func() }

func (h *heldSpawn) spawn(f func()) { h.fns = append(h.fns, f) }

func (h *heldSpawn) runAll() {
	fns := h.fns
	h.fns = nil
	for _, f := range fns {
		f()
	}
}

func cpuTurnPending(match *engine.Match) error {
	match.WorkingState.Turn = 2
	match.WorkingState.ActiveTeam = 2
	match.TrueState = match.WorkingState.DeepCopy()
	return nil
}

type fixture struct {
	r       *Room
	tokens  [2]string
	clock   *fakeClock
	held    *heldSpawn
	decides *int
}

var cpuPlan = []engine.TurnCommand{
	engine.NewMoveCommand(engine.NewUnitID(2, 0), engine.Coordinate{X: 4, Y: 1}),
	engine.NewPlaceBombCommand(engine.NewUnitID(2, 0), engine.Coordinate{X: 4, Y: 2}),
}

func newFixture(arrange ...func(*engine.Match) error) fixture {
	clock := newFakeClock()
	held := &heldSpawn{}
	decides := new(int)
	r := New("room", Deps{
		Logger: slog.New(slog.DiscardHandler),
		Decide: func(*engine.GameState) ([]engine.TurnCommand, error) {
			*decides++
			return cpuPlan, nil
		},
		Spawn: held.spawn,
		NewMatch: func(cfg engine.GameCfg) (*engine.Match, error) {
			match, err := engine.InitGame(cfg)
			if err != nil {
				return nil, err
			}
			for _, f := range arrange {
				if err := f(match); err != nil {
					return nil, err
				}
			}
			return match, nil
		},
		Now: clock.now,
	})
	return fixture{r: r, clock: clock, held: held, decides: decides}
}

func humanTurnCfg() engine.GameCfg {
	cfg := validGameCfg()
	cfg.AllowResetTurn = true
	return cfg
}

func vsCPUCfg() engine.GameCfg {
	cfg := humanTurnCfg()
	cfg.VSCpu = true
	return cfg
}

func startedFixture(t *testing.T, cfg engine.GameCfg, arrange ...func(*engine.Match) error) fixture {
	t.Helper()
	f := newFixture(arrange...)
	tokens, err := f.r.Start(cfg)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	f.tokens = tokens
	return f
}

func humanFixture(t *testing.T) fixture {
	t.Helper()
	return startedFixture(t, humanTurnCfg())
}

func planningFixture(t *testing.T) fixture {
	t.Helper()
	f := startedFixture(t, vsCPUCfg(), cpuTurnPending)
	if status, err := f.r.ConsumeCPU(f.tokens[0]); err != nil || status.Phase != TurnPhasePlanning {
		t.Fatalf("ConsumeCPU() = %v, %v, want phase Planning", status.Phase, err)
	}
	return f
}

func cpuReadyFixture(t *testing.T) fixture {
	t.Helper()
	f := planningFixture(t)
	f.held.runAll()
	return f
}

func validGameCfg() engine.GameCfg {
	return engine.GameCfg{
		StagePreset: "Plain",
		P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
		P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
		MaxTurns:    10,
	}
}

func unitAlreadyMoved(unitID engine.UnitID) func(*engine.Match) error {
	return func(match *engine.Match) error {
		unit, ok := match.WorkingState.Units[unitID]
		if !ok {
			return errors.New("unit not found")
		}
		unit.HasMoved = true
		return nil
	}
}

func bombAboutToExplode(unitID engine.UnitID) func(*engine.Match) error {
	return func(match *engine.Match) error {
		unit, ok := match.WorkingState.Units[unitID]
		if !ok {
			return errors.New("unit not found")
		}
		target, ok := findFreeNeighbour(match.WorkingState, unit.Position)
		if !ok {
			return errors.New("no free neighbour")
		}
		if _, err := match.CommandPlaceBomb(unitID, target); err != nil {
			return err
		}
		for _, bomb := range match.WorkingState.Bombs {
			if bomb.OwnerID == unitID {
				bomb.Countdown = 1
			}
		}
		return nil
	}
}

func suddenDeathMidTurn(match *engine.Match) error {
	match.WorkingState.InSuddenDeath = true
	for _, unit := range match.WorkingState.Units {
		unit.HasMoved = true
		unit.HasUsedSkill = true
	}
	return nil
}

func findFreeNeighbour(gs *engine.GameState, pos engine.Coordinate) (engine.Coordinate, bool) {
	for _, step := range []engine.Coordinate{{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0}} {
		target := engine.Coordinate{X: pos.X + step.X, Y: pos.Y + step.Y}
		if gs.IsLandingLegal(target, engine.OccupantBomb) == nil {
			return target, true
		}
	}
	return engine.Coordinate{}, false
}

func startedRoom(t *testing.T, arrange ...func(*engine.Match) error) (*Room, [2]string) {
	t.Helper()
	f := startedFixture(t, validGameCfg(), arrange...)
	return f.r, f.tokens
}

func mustState(t *testing.T, r *Room) *engine.GameState {
	t.Helper()
	gs, err := r.State()
	if err != nil {
		t.Fatalf("State() error = %v", err)
	}
	return gs
}

func endAndDelete(t *testing.T, r *Room, token string) {
	t.Helper()
	if _, err := r.Surrender(1, token); err != nil {
		t.Fatalf("Surrender() error = %v", err)
	}
	if err := r.Delete(token); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func assertTokensAuthorize(t *testing.T, r *Room, tokens [2]string) {
	t.Helper()
	if err := r.Reset(tokens[0]); err != nil {
		t.Errorf("Reset() with the team 1 token error = %v", err)
	}
	if _, err := r.Surrender(2, tokens[1]); err != nil {
		t.Errorf("Surrender() with the team 2 token error = %v", err)
	}
}
