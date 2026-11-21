package room

import (
	"bomb-srpg/engine"
	"errors"
	"log/slog"
	"reflect"
	"slices"
	"testing"
)

func cloneEvents(gameEvents []engine.GameEvent) []engine.GameEvent {
	out := make([]engine.GameEvent, len(gameEvents))
	for i, evt := range gameEvents {
		if evt.Position != nil {
			pos := *evt.Position
			evt.Position = &pos
		}
		if evt.From != nil {
			from := *evt.From
			evt.From = &from
		}
		if evt.To != nil {
			to := *evt.To
			evt.To = &to
		}
		evt.AffectedPositions = slices.Clone(evt.AffectedPositions)
		out[i] = evt
	}
	return out
}

func scribbleEvents(t *testing.T, gameEvents []engine.GameEvent) {
	t.Helper()
	if len(gameEvents) == 0 {
		t.Fatal("no events to overwrite")
	}
	for i := range gameEvents {
		evt := &gameEvents[i]
		evt.Type = "scribbled"
		evt.UnitID++
		for _, pos := range []*engine.Coordinate{evt.Position, evt.From, evt.To} {
			if pos != nil {
				pos.X, pos.Y = -1, -1
			}
		}
		for j := range evt.AffectedPositions {
			evt.AffectedPositions[j] = engine.Coordinate{X: -1, Y: -1}
		}
	}
}

func scribbleState(gs *engine.GameState) {
	gs.Turn += 100
	gs.ActiveTeam += 5
	for _, unit := range gs.Units {
		unit.HasMoved = !unit.HasMoved
	}
	for id := range gs.Units {
		delete(gs.Units, id)
		break
	}
}

func TestRoom_NoAlias(t *testing.T) {
	covered := map[string]bool{}

	t.Run("State", func(t *testing.T) {
		covered["State"] = true
		f := humanFixture(t)
		got, err := f.r.State()
		if err != nil {
			t.Fatalf("State() error = %v", err)
		}
		want := got.DeepCopy()

		scribbleState(got)

		after, err := f.r.State()
		if err != nil {
			t.Fatalf("State() error = %v", err)
		}
		if !reflect.DeepEqual(after, want) {
			t.Errorf("State() changed after the caller wrote to a returned state: got %+v, want %+v", after, want)
		}
	})

	t.Run("Config", func(t *testing.T) {
		covered["Config"] = true
		f := humanFixture(t)
		got, err := f.r.Config()
		if err != nil {
			t.Fatalf("Config() error = %v", err)
		}
		want := got
		want.P1Slots = slices.Clone(got.P1Slots)
		want.P2Slots = slices.Clone(got.P2Slots)

		got.P1Slots[0].Archetype = "scribbled"
		got.P2Slots[0].Archetype = "scribbled"

		after, err := f.r.Config()
		if err != nil {
			t.Fatalf("Config() error = %v", err)
		}
		if !reflect.DeepEqual(after, want) {
			t.Errorf("Config() changed after the caller wrote to a returned config: got %+v, want %+v", after, want)
		}
	})

	t.Run("Submit", func(t *testing.T) {
		covered["Submit"] = true
		f := humanFixture(t)
		gameEvents, err := f.r.Submit(humanMove, f.tokens[0])
		if err != nil {
			t.Fatalf("Submit() error = %v", err)
		}
		want := cloneEvents(f.r.match.PlaybackLog)

		scribbleEvents(t, gameEvents)

		if !reflect.DeepEqual(f.r.match.PlaybackLog, want) {
			t.Errorf("playback log changed after the caller wrote to the returned events: got %+v, want %+v", f.r.match.PlaybackLog, want)
		}
	})

	t.Run("ConsumeCPU", func(t *testing.T) {
		covered["ConsumeCPU"] = true
		f := cpuReadyFixture(t)
		status, err := f.r.ConsumeCPU(f.tokens[0])
		if err != nil {
			t.Fatalf("ConsumeCPU() error = %v", err)
		}
		if status.Phase != TurnPhaseReady {
			t.Fatalf("ConsumeCPU() phase = %v, want %v", status.Phase, TurnPhaseReady)
		}

		scribbleEvents(t, status.Plan)
		scribbleEvents(t, status.Resolve)

		if f.r.cpu.plan != nil || f.r.cpu.resolve != nil {
			t.Errorf("room still holds the returned events: plan %+v, resolve %+v", f.r.cpu.plan, f.r.cpu.resolve)
		}
	})

	t.Run("Start does not keep the caller's slot slices", func(t *testing.T) {
		covered["Start"] = true
		cfg := validGameCfg()
		f := newFixture()
		if _, err := f.r.Start(cfg); err != nil {
			t.Fatalf("Start() error = %v", err)
		}

		cfg.P1Slots[0].Archetype = "scribbled"
		cfg.P2Slots[0].Archetype = "scribbled"

		got, err := f.r.Config()
		if err != nil {
			t.Fatalf("Config() error = %v", err)
		}
		if want := validGameCfg(); !reflect.DeepEqual(got.P1Slots, want.P1Slots) || !reflect.DeepEqual(got.P2Slots, want.P2Slots) {
			t.Errorf("Config() slots = %+v and %+v, want %+v and %+v", got.P1Slots, got.P2Slots, want.P1Slots, want.P2Slots)
		}
	})

	t.Run("every exported method is listed", func(t *testing.T) {
		noSharedMemory := map[string]string{
			"Rematch":      "returns an array of tokens and an error",
			"Delete":       "returns only an error",
			"Reset":        "returns only an error",
			"Expired":      "returns a boolean",
			"AllowedTiles": "builds a new slice from a map on every call",
			"Resolve":      "the engine returns copies and no live holder of them remains",
			"Surrender":    "the engine returns copies and no live holder of them remains",
		}
		typ := reflect.TypeFor[*Room]()
		for method := range typ.Methods() {
			name := method.Name
			if _, ok := noSharedMemory[name]; !ok && !covered[name] {
				t.Errorf("exported method %s has no alias row and is not in noSharedMemory", name)
			}
		}
	})
}

func TestRoom_NewMatch_RoomKeepsNoPointerTheFactoryHolds(t *testing.T) {
	var built []*engine.Match
	var received []engine.GameCfg
	r := New("room", Deps{
		Logger: slog.New(slog.DiscardHandler),
		NewMatch: func(cfg engine.GameCfg) (*engine.Match, error) {
			match, err := engine.InitGame(cfg)
			if err == nil {
				built = append(built, match)
				received = append(received, cfg)
			}
			return match, err
		},
	})
	tokens, err := r.Start(validGameCfg())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if _, err := r.Rematch(tokens[0]); err != nil {
		t.Fatalf("Rematch() error = %v", err)
	}
	want, err := r.State()
	if err != nil {
		t.Fatalf("State() error = %v", err)
	}

	for _, match := range built {
		scribbleState(match.WorkingState)
		match.WinnerTeamID = 1
	}
	for _, cfg := range received {
		cfg.P1Slots[0].Archetype = "scribbled"
		cfg.P2Slots[0].Archetype = "scribbled"
	}

	got, err := r.State()
	if err != nil {
		t.Fatalf("State() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("State() turn = %d, active team = %d after the factory edited its matches, want %d and %d", got.Turn, got.ActiveTeam, want.Turn, want.ActiveTeam)
	}
	if err := r.Delete(tokens[0]); !errors.Is(err, ErrMatchInProgress) {
		t.Errorf("Delete() error = %v, want %v: the factory's winner edit reached the room", err, ErrMatchInProgress)
	}
	if _, err := r.Rematch(tokens[0]); err != nil {
		t.Fatalf("Rematch() error = %v", err)
	}
	if got, want := received[len(received)-1], validGameCfg(); !reflect.DeepEqual(got, want) {
		t.Errorf("cfg the room passed to the factory after it edited its slot slices = %+v, want %+v", got, want)
	}
}
