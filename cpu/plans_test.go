package cpu

import (
	"bomb-srpg/engine"
	"slices"
	"testing"
)

func planKeys(plans [][]engine.TurnCommand) []string {
	keys := make([]string, len(plans))
	for i, p := range plans {
		keys[i] = planTag(p)
	}
	slices.Sort(keys)
	return keys
}

func TestPlansFor(t *testing.T) {
	tests := []struct {
		name  string
		setup func() (*engine.GameState, *engine.Unit)
		want  []string
	}{
		{
			name: "Unit fully boxed in by hard blocks",
			setup: func() (*engine.GameState, *engine.Unit) {
				gs := newTestGameState(9, 9)
				center := engine.Coordinate{X: 4, Y: 4}
				unit := addFighter(gs, engine.NewUnitID(1, 1), center)
				for _, off := range crossOffsets(unit.Speed) {
					setHardBlock(gs, engine.Coordinate{X: center.X + off.X, Y: center.Y + off.Y})
				}
				return gs, unit
			},
			want: []string{"Idle"},
		},
		{
			name: "Full combo enumeration in a 1-wide corridor",
			setup: func() (*engine.GameState, *engine.Unit) {
				gs := newTestGameState(7, 1)
				unit := addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0})
				return gs, unit
			},
			want: []string{
				"Idle",
				"move(1,0)", "move(2,0)",
				"placeBomb(1,0)", "placeBomb(2,0)",
				"move(1,0)+placeBomb(0,0)", "move(1,0)+placeBomb(2,0)", "move(1,0)+placeBomb(3,0)",
				"move(2,0)+placeBomb(0,0)", "move(2,0)+placeBomb(1,0)", "move(2,0)+placeBomb(3,0)", "move(2,0)+placeBomb(4,0)",
				"placeBomb(2,0)+move(1,0)",
			},
		},
		{
			name: "Bomb already used this turn",
			setup: func() (*engine.GameState, *engine.Unit) {
				gs := newTestGameState(5, 5)
				unit := addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 2, Y: 2})
				unit.BombUsed = unit.MaxBombCount
				return gs, unit
			},
			want: []string{
				"Idle",
				"move(2,0)", "move(2,1)", "move(2,3)", "move(2,4)",
				"move(0,2)", "move(1,2)", "move(3,2)", "move(4,2)",
			},
		},
		{
			name: "Skill already used this turn",
			setup: func() (*engine.GameState, *engine.Unit) {
				gs := newTestGameState(5, 5)
				unit := addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 2, Y: 2})
				unit.HasUsedSkill = true
				return gs, unit
			},
			want: []string{
				"Idle",
				"move(2,0)", "move(2,1)", "move(2,3)", "move(2,4)",
				"move(0,2)", "move(1,2)", "move(3,2)", "move(4,2)",
			},
		},
		{
			name: "Unit already moved this turn",
			setup: func() (*engine.GameState, *engine.Unit) {
				gs := newTestGameState(5, 5)
				unit := addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 2, Y: 2})
				unit.HasMoved = true
				return gs, unit
			},
			want: []string{
				"Idle",
				"placeBomb(2,0)", "placeBomb(2,1)", "placeBomb(2,3)", "placeBomb(2,4)",
				"placeBomb(0,2)", "placeBomb(1,2)", "placeBomb(3,2)", "placeBomb(4,2)",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs, unit := tt.setup()
			want := slices.Clone(tt.want)
			slices.Sort(want)

			got := planKeys(plansFor(unit, gs))
			if !slices.Equal(got, want) {
				t.Errorf("plansFor() = %v, want %v", got, want)
			}
		})
	}
}

func TestPlansFor_OpenField(t *testing.T) {
	gs := newTestGameState(9, 9)
	center := engine.Coordinate{X: 4, Y: 4}
	unit := addFighter(gs, engine.NewUnitID(1, 1), center)

	plans := plansFor(unit, gs)

	// A placed bomb blocks its own lane, so bomb->move isn't symmetric with move->bomb.
	const wantTotal = 1 + 8 + 8 + 8*8 + (4*6 + 4*7)
	if len(plans) != wantTotal {
		t.Fatalf("plansFor() returned %d plans, want %d", len(plans), wantTotal)
	}

	got := planKeys(plans)
	for _, want := range []string{"move(6,4)", "placeBomb(6,4)", "move(6,4)+placeBomb(8,4)"} {
		if !slices.Contains(got, want) {
			t.Errorf("plansFor() missing expected plan %q", want)
		}
	}
	if !slices.Contains(got, "placeBomb(4,2)+move(4,3)") {
		t.Errorf("plansFor() missing expected plan %q", "placeBomb(4,2)+move(4,3)")
	}
	if slices.Contains(got, "placeBomb(5,4)+move(6,4)") {
		t.Errorf("plansFor() unexpectedly contains %q", "placeBomb(5,4)+move(6,4)")
	}
}

func TestSingleActionPlans_Occupancy(t *testing.T) {
	tests := []struct {
		name          string
		placeOccupant func(gs *engine.GameState, pos engine.Coordinate)
		fn            func(*engine.Unit, *engine.GameState) [][]engine.TurnCommand
		want          []string
	}{
		{
			name:          "Move blocked by soft block occupancy",
			placeOccupant: func(gs *engine.GameState, pos engine.Coordinate) { addSoftBlock(gs, 1, pos) },
			fn:            movePlansFor,
			want:          []string{"move(2,0)", "move(2,1)", "move(2,3)", "move(2,4)", "move(0,2)", "move(1,2)"},
		},
		{
			name:          "Bomb passes through soft block to land beyond it",
			placeOccupant: func(gs *engine.GameState, pos engine.Coordinate) { addSoftBlock(gs, 1, pos) },
			fn:            placeBombPlansFor,
			want: []string{
				"placeBomb(2,0)", "placeBomb(2,1)", "placeBomb(2,3)", "placeBomb(2,4)",
				"placeBomb(0,2)", "placeBomb(1,2)", "placeBomb(4,2)",
			},
		},
		{
			name:          "Move blocked by bomb occupancy",
			placeOccupant: func(gs *engine.GameState, pos engine.Coordinate) { addBomb(gs, 1, pos) },
			fn:            movePlansFor,
			want:          []string{"move(2,0)", "move(2,1)", "move(2,3)", "move(2,4)", "move(0,2)", "move(1,2)"},
		},
		{
			name:          "Bomb passes through bomb to land beyond it",
			placeOccupant: func(gs *engine.GameState, pos engine.Coordinate) { addBomb(gs, 1, pos) },
			fn:            placeBombPlansFor,
			want: []string{
				"placeBomb(2,0)", "placeBomb(2,1)", "placeBomb(2,3)", "placeBomb(2,4)",
				"placeBomb(0,2)", "placeBomb(1,2)", "placeBomb(4,2)",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := newTestGameState(5, 5)
			unit := addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 2, Y: 2})
			tt.placeOccupant(gs, engine.Coordinate{X: 3, Y: 2})

			want := slices.Clone(tt.want)
			slices.Sort(want)

			got := planKeys(tt.fn(unit, gs))
			if !slices.Equal(got, want) {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

func TestSingleCommandPlans_errorPaths(t *testing.T) {
	gs := newTestGameState(3, 3)
	deadUnit := addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 1, Y: 1})
	deadUnit.HP = 0
	liveUnit := addFighter(gs, engine.NewUnitID(1, 2), engine.Coordinate{X: 2, Y: 2})
	missingUnitID := engine.NewUnitID(1, 9)

	tests := []struct {
		name        string
		unit        *engine.Unit
		turnCmdType engine.TurnCmdType
	}{
		{"Failure: Unit not found", &engine.Unit{ID: missingUnitID}, engine.TurnCmdMove},
		{"Failure: Unit is dead", deadUnit, engine.TurnCmdMove},
		{"Failure: Unsupported command type", liveUnit, engine.TurnCmdType("bogus")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := singleCommandPlans(tt.unit, gs, tt.turnCmdType, engine.NewMoveCommand)
			if got != nil {
				t.Errorf("singleCommandPlans() = %v, want nil", got)
			}
		})
	}
}
