package room

import (
	"bomb-srpg/engine"
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
	r      *Room
	tokens [2]string
	clock  *fakeClock
	held   *heldSpawn
}

var cpuPlan = []engine.TurnCommand{
	engine.NewMoveCommand(engine.NewUnitID(2, 0), engine.Coordinate{X: 4, Y: 1}),
	engine.NewPlaceBombCommand(engine.NewUnitID(2, 0), engine.Coordinate{X: 4, Y: 2}),
}

func newFixture(arrange ...func(*engine.Match) error) fixture {
	clock := newFakeClock()
	held := &heldSpawn{}
	r := New("room", Deps{
		Logger: slog.New(slog.DiscardHandler),
		Decide: func(*engine.GameState) ([]engine.TurnCommand, error) {
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
	return fixture{r: r, clock: clock, held: held}
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
	if status, err := f.r.ConsumeCPU(f.tokens[0]); err != nil || status.Phase != engine.TurnPhasePlanning {
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
