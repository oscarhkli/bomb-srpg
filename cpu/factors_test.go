package cpu

import (
	"bomb-srpg/engine"
	"fmt"
	"math"
	"testing"
)

func TestAdvanceOpponentKingReachability(t *testing.T) {
	tests := []struct {
		name                string
		setup               func(gs *engine.GameState) (actor, king *engine.Unit)
		destroyedSoftBlocks int
		turn                int
		distToKingBefore    int
		distToKingAfter     int
		want                int
	}{
		{
			name: "No SoftBlock destroyed",
			setup: func(gs *engine.GameState) (*engine.Unit, *engine.Unit) {
				return addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0}),
					addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 4})
			},
			destroyedSoftBlocks: 0,
			distToKingBefore:    5,
			distToKingAfter:     1,
			want:                0,
		},
		{
			name: "Actor dead",
			setup: func(gs *engine.GameState) (*engine.Unit, *engine.Unit) {
				actor := addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0})
				actor.HP = 0
				return actor, addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 4})
			},
			destroyedSoftBlocks: 1,
			distToKingBefore:    5,
			distToKingAfter:     1,
			want:                0,
		},
		{
			name: "King dead",
			setup: func(gs *engine.GameState) (*engine.Unit, *engine.Unit) {
				king := addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 4})
				king.HP = 0
				return addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0}), king
			},
			destroyedSoftBlocks: 1,
			distToKingBefore:    5,
			distToKingAfter:     1,
			want:                0,
		},
		{
			name: "No change",
			setup: func(gs *engine.GameState) (*engine.Unit, *engine.Unit) {
				return addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0}),
					addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 4})
			},
			destroyedSoftBlocks: 1,
			turn:                0,
			distToKingBefore:    5,
			distToKingAfter:     5,
			want:                0,
		},
		{
			name: "Breakthrough: was unreachable, now reachable",
			setup: func(gs *engine.GameState) (*engine.Unit, *engine.Unit) {
				return addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0}),
					addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 4})
			},
			destroyedSoftBlocks: 1,
			turn:                0,
			distToKingBefore:    -1,
			distToKingAfter:     5,
			want:                5,
		},
		{
			name: "Normal improvement: both reachable, got closer",
			setup: func(gs *engine.GameState) (*engine.Unit, *engine.Unit) {
				return addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0}),
					addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 4})
			},
			destroyedSoftBlocks: 1,
			turn:                0,
			distToKingBefore:    5,
			distToKingAfter:     1,
			want:                5,
		},
		{
			name: "Regression: both reachable, got farther",
			setup: func(gs *engine.GameState) (*engine.Unit, *engine.Unit) {
				return addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0}),
					addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 4})
			},
			destroyedSoftBlocks: 1,
			turn:                0,
			distToKingBefore:    1,
			distToKingAfter:     5,
			want:                -5,
		},
		{
			name: "Newly unreachable: was reachable, now unreachable",
			setup: func(gs *engine.GameState) (*engine.Unit, *engine.Unit) {
				return addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0}),
					addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 4})
			},
			destroyedSoftBlocks: 1,
			turn:                0,
			distToKingBefore:    5,
			distToKingAfter:     -1,
			want:                -5,
		},
		{
			name: "Turn decay: same improvement, later turn scores less",
			setup: func(gs *engine.GameState) (*engine.Unit, *engine.Unit) {
				return addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0}),
					addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 4})
			},
			destroyedSoftBlocks: 1,
			turn:                4,
			distToKingBefore:    5,
			distToKingAfter:     1,
			want:                1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := newTestGameState(5, 5)
			actor, king := tt.setup(gs)
			sc := scoreContext{actorID: actor.ID, opponentKingID: king.ID}
			tr := turnResult{
				turn:                tt.turn,
				destroyedSoftBlocks: tt.destroyedSoftBlocks,
				distToKingBefore:    tt.distToKingBefore,
				distToKingAfter:     tt.distToKingAfter,
			}

			if got := advanceOpponentKingReachability(gs, sc, tr); got != tt.want {
				t.Errorf("advanceOpponentKingReachability() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestKillOpponentKing(t *testing.T) {
	opponentKingID := engine.NewUnitID(2, 1)
	tests := []struct {
		name     string
		turn     int
		diedID   engine.UnitID
		unitDied bool
		want     int
	}{
		{name: "King died at T+0", turn: 0, diedID: opponentKingID, unitDied: true, want: killKingScore},
		{name: "King died at T+1: too late, opponent already had a turn", turn: 1, diedID: opponentKingID, unitDied: true, want: 0},
		{name: "Different unit died", turn: 0, diedID: engine.NewUnitID(1, 1), unitDied: true, want: 0},
		{name: "Nobody died", turn: 0, unitDied: false, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := scoreContext{opponentKingID: opponentKingID}
			diedUnits := make(map[engine.UnitID]struct{})
			if tt.unitDied {
				diedUnits[tt.diedID] = struct{}{}
			}
			tr := turnResult{turn: tt.turn, diedUnits: diedUnits}

			if got := killOpponentKing(nil, sc, tr); got != tt.want {
				t.Errorf("killOpponentKing() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestKillOpponents(t *testing.T) {
	tests := []struct {
		name   string
		turn   int
		before int
		after  int
		want   int
	}{
		{name: "Some killed at T+0", turn: 0, before: 2, after: 1, want: killUnitScore / 2},
		{name: "Last one killed at T+0", turn: 0, before: 1, after: 0, want: killUnitScore},
		{name: "None killed", turn: 0, before: 3, after: 3, want: 0},
		{name: "No non-King opponents to begin with", turn: 0, before: 0, after: 0, want: 0},
		{name: "Killed at T+1: too late, opponent already had a turn", turn: 1, before: 2, after: 1, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := turnResult{turn: tt.turn, aliveOpponentsBefore: tt.before, aliveOpponentsAfter: tt.after}

			if got := killOpponents(nil, scoreContext{}, tr); got != tt.want {
				t.Errorf("killOpponents() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestKillAllies(t *testing.T) {
	tests := []struct {
		name   string
		turn   int
		before int
		after  int
		want   int
	}{
		{name: "Some killed at T+0", turn: 0, before: 2, after: 1, want: killUnitScore / 2},
		{name: "Killed at T+1: still certain, ally couldn't act", turn: 1, before: 2, after: 1, want: killUnitScore / 2},
		{name: "Killed at T+2: too late, ally already had a turn", turn: 2, before: 2, after: 1, want: 0},
		{name: "None killed", turn: 0, before: 3, after: 3, want: 0},
		{name: "No non-King allies to begin with", turn: 0, before: 0, after: 0, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := turnResult{turn: tt.turn, aliveAlliesBefore: tt.before, aliveAlliesAfter: tt.after}

			if got := killAllies(nil, scoreContext{}, tr); got != tt.want {
				t.Errorf("killAllies() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestKillAllyKing(t *testing.T) {
	allyKingID := engine.NewUnitID(1, 1)
	tests := []struct {
		name     string
		turn     int
		diedID   engine.UnitID
		unitDied bool
		want     int
	}{
		{name: "King died at T+0", turn: 0, diedID: allyKingID, unitDied: true, want: killKingScore},
		{name: "King died at T+1: still certain, ally couldn't act", turn: 1, diedID: allyKingID, unitDied: true, want: killKingScore},
		{name: "King died at T+2: too late, ally already had a turn", turn: 2, diedID: allyKingID, unitDied: true, want: 0},
		{name: "Different unit died", turn: 0, diedID: engine.NewUnitID(2, 1), unitDied: true, want: 0},
		{name: "Nobody died", turn: 0, unitDied: false, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := scoreContext{allyKingID: allyKingID}
			diedUnits := make(map[engine.UnitID]struct{})
			if tt.unitDied {
				diedUnits[tt.diedID] = struct{}{}
			}
			tr := turnResult{turn: tt.turn, diedUnits: diedUnits}

			if got := killAllyKing(nil, sc, tr); got != tt.want {
				t.Errorf("killAllyKing() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDecayRatio(t *testing.T) {
	tests := []struct {
		name string
		x    int
		t    int
		k    float64
		want float64
	}{
		{name: "x=0: full weight", x: 0, t: 5, k: 0.6, want: 1.0},
		{name: "x=t: fully decayed", x: 5, t: 5, k: 0.6, want: 0.0},
		{name: "x>t: past the boundary", x: 6, t: 5, k: 0.6, want: 0.0},
		{name: "interior, turn axis scale", x: 2, t: 5, k: 0.6, want: 0.8784358579},
		{name: "interior, dist axis scale", x: 3, t: riskFreeDist, k: kDist, want: 0.8807970780},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decayRatio(tt.x, tt.t, tt.k); math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("decayRatio(%d,%d,%v) = %v, want %v", tt.x, tt.t, tt.k, got, tt.want)
			}
		})
	}
}

// onAffectedTile, hasAnOut and nearNotOnIt are shared setups for TestThreatOpponentKing
// and TestRiskAllyKing: same King, only the affected tiles around it differ.
func onAffectedTile(gs *engine.GameState) (*engine.Unit, map[engine.Coordinate]struct{}) {
	king := addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 2, Y: 2})
	return king, map[engine.Coordinate]struct{}{
		{X: 2, Y: 2}: {}, {X: 2, Y: 1}: {}, {X: 2, Y: 3}: {}, {X: 1, Y: 2}: {}, {X: 3, Y: 2}: {},
	}
}

func hasAnOut(gs *engine.GameState) (*engine.Unit, map[engine.Coordinate]struct{}) {
	king := addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 2, Y: 2})
	return king, map[engine.Coordinate]struct{}{{X: 2, Y: 2}: {}}
}

func nearNotOnIt(gs *engine.GameState) (*engine.Unit, map[engine.Coordinate]struct{}) {
	king := addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 2, Y: 2})
	return king, map[engine.Coordinate]struct{}{{X: 2, Y: 5}: {}}
}

func TestEscapable(t *testing.T) {
	tests := []struct {
		name  string
		setup func(gs *engine.GameState) (*engine.Unit, map[engine.Coordinate]struct{})
		want  bool
	}{
		{name: "Has an out", setup: hasAnOut, want: true},
		{name: "Cornered", setup: onAffectedTile, want: false},
		{
			name: "Unit not registered in gs.Units: treated as escapable",
			setup: func(gs *engine.GameState) (*engine.Unit, map[engine.Coordinate]struct{}) {
				return &engine.Unit{ID: engine.NewUnitID(9, 9)}, map[engine.Coordinate]struct{}{}
			},
			want: true,
		},
		{
			name: "Unit dead: treated as escapable",
			setup: func(gs *engine.GameState) (*engine.Unit, map[engine.Coordinate]struct{}) {
				king := addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 2, Y: 2})
				king.HP = 0
				return king, map[engine.Coordinate]struct{}{{X: 2, Y: 2}: {}}
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := newTestGameState(9, 9)
			unit, affectedTiles := tt.setup(gs)
			if got := escapable(gs, unit, affectedTiles); got != tt.want {
				t.Errorf("escapable() = %v, want %v", got, tt.want)
			}
		})
	}
}

// kingExposureCases is shared by TestThreatOpponentKing and TestRiskAllyKing: k is the
// countable tick (turnOffset applied by each test), verified equal for both factors.
var kingExposureCases = []struct {
	name  string
	setup func(gs *engine.GameState) (king *engine.Unit, affectedTiles map[engine.Coordinate]struct{})
	k     int
	want  int
}{
	{
		name: "No affected tile this tick",
		setup: func(gs *engine.GameState) (*engine.Unit, map[engine.Coordinate]struct{}) {
			return addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 2, Y: 2}), nil
		},
		k: 1, want: 0,
	},
	{
		name: "King already dead: still scores frozen-position exposure",
		setup: func(gs *engine.GameState) (*engine.Unit, map[engine.Coordinate]struct{}) {
			king := addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 2, Y: 2})
			king.HP = 0
			return king, map[engine.Coordinate]struct{}{{X: 0, Y: 0}: {}}
		},
		k: 1, want: 1794,
	},
	{
		name: "Affected tile unreachable: King boxed in by TerrainBlocks",
		setup: func(gs *engine.GameState) (*engine.Unit, map[engine.Coordinate]struct{}) {
			king := addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 2, Y: 2})
			setTerrainBlock(gs, engine.Coordinate{X: 2, Y: 1})
			setTerrainBlock(gs, engine.Coordinate{X: 2, Y: 3})
			setTerrainBlock(gs, engine.Coordinate{X: 1, Y: 2})
			setTerrainBlock(gs, engine.Coordinate{X: 3, Y: 2})
			return king, map[engine.Coordinate]struct{}{{X: 0, Y: 0}: {}}
		},
		k: 1, want: 0,
	},
	{
		name: "Beyond risk-free horizon",
		setup: func(gs *engine.GameState) (*engine.Unit, map[engine.Coordinate]struct{}) {
			king := addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 0, Y: 0})
			return king, map[engine.Coordinate]struct{}{{X: 0, Y: riskFreeDist}: {}}
		},
		k: 1, want: 0,
	},
	{name: "On an affected tile, cornered: k=1", setup: onAffectedTile, k: 1, want: 4784},
	{name: "On an affected tile, cornered: k=3", setup: onAffectedTile, k: 3, want: 3677},
	{name: "On an affected tile, has an out: k=1", setup: hasAnOut, k: 1, want: 2392},
	{name: "On an affected tile, has an out: k=3", setup: hasAnOut, k: 3, want: 1838},
	{name: "Near an affected tile, not on it: k=1", setup: nearNotOnIt, k: 1, want: 2107},
	{name: "Near an affected tile, not on it: k=3", setup: nearNotOnIt, k: 3, want: 1619},
}

func TestThreatOpponentKing(t *testing.T) {
	t.Run("At T+0: scores as urgent, not excluded", func(t *testing.T) {
		gs := newTestGameState(9, 9)
		king := addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 2, Y: 2})
		sc := scoreContext{opponentKingID: king.ID}
		tr := turnResult{turn: 0, affectedTiles: map[engine.Coordinate]struct{}{{X: 2, Y: 2}: {}}}
		if got, want := threatOpponentKing(gs, sc, tr), 2500; got != want {
			t.Errorf("threatOpponentKing() = %d, want %d", got, want)
		}
	})
	for _, tt := range kingExposureCases {
		t.Run(tt.name, func(t *testing.T) {
			gs := newTestGameState(9, 9)
			king, affectedTiles := tt.setup(gs)
			sc := scoreContext{opponentKingID: king.ID}
			tr := turnResult{turn: tt.k, affectedTiles: affectedTiles}

			if got := threatOpponentKing(gs, sc, tr); got != tt.want {
				t.Errorf("threatOpponentKing() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRiskAllyKing(t *testing.T) {
	for _, turn := range []int{0, 1} {
		t.Run(fmt.Sprintf("At T+%d: scores as urgent, not excluded", turn), func(t *testing.T) {
			gs := newTestGameState(9, 9)
			king := addKing(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 2, Y: 2})
			sc := scoreContext{allyKingID: king.ID}
			tr := turnResult{turn: turn, affectedTiles: map[engine.Coordinate]struct{}{{X: 2, Y: 2}: {}}}
			if got, want := riskAllyKing(gs, sc, tr), 2500; got != want {
				t.Errorf("riskAllyKing() = %d, want %d", got, want)
			}
		})
	}
	for _, tt := range kingExposureCases {
		t.Run(tt.name, func(t *testing.T) {
			gs := newTestGameState(9, 9)
			king, affectedTiles := tt.setup(gs)
			sc := scoreContext{allyKingID: king.ID}
			tr := turnResult{turn: tt.k + 1, affectedTiles: affectedTiles}

			if got := riskAllyKing(gs, sc, tr); got != tt.want {
				t.Errorf("riskAllyKing() = %d, want %d", got, tt.want)
			}
		})
	}
}

// twoOpponentsAveraged and twoAlliesAveraged build 2 non-King units plus a third unit sharing
// the King's ID, proving the King is excluded from the average rather than merely absent.
func twoOpponentsAveraged(gs *engine.GameState) (unitIDs []engine.UnitID, kingID engine.UnitID, affectedTiles map[engine.Coordinate]struct{}) {
	unitA := addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0})
	unitB := addFighter(gs, engine.NewUnitID(1, 2), engine.Coordinate{X: 0, Y: riskFreeDist})
	king := addFighter(gs, engine.NewUnitID(1, 3), engine.Coordinate{X: 1, Y: 0})
	affectedTiles = map[engine.Coordinate]struct{}{{X: 0, Y: 0}: {}, {X: 1, Y: 0}: {}}
	return []engine.UnitID{unitA.ID, unitB.ID, king.ID}, king.ID, affectedTiles
}

func twoAlliesAveraged(gs *engine.GameState) (unitIDs []engine.UnitID, kingID engine.UnitID, affectedTiles map[engine.Coordinate]struct{}) {
	unitA := addFighter(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 0, Y: 0})
	unitB := addFighter(gs, engine.NewUnitID(2, 2), engine.Coordinate{X: 0, Y: riskFreeDist})
	king := addFighter(gs, engine.NewUnitID(2, 3), engine.Coordinate{X: 1, Y: 0})
	affectedTiles = map[engine.Coordinate]struct{}{{X: 0, Y: 0}: {}, {X: 1, Y: 0}: {}}
	return []engine.UnitID{unitA.ID, unitB.ID, king.ID}, king.ID, affectedTiles
}

func TestThreatOpponents(t *testing.T) {
	t.Run("At T+0: scores as urgent, not excluded", func(t *testing.T) {
		gs := newTestGameState(9, 9)
		unitIDs, kingID, affectedTiles := twoOpponentsAveraged(gs)
		sc := scoreContext{opponentIDs: unitIDs, opponentKingID: kingID}
		tr := turnResult{turn: 0, affectedTiles: affectedTiles, aliveOpponentsAfter: 2}
		if got, want := threatOpponents(gs, sc, tr), 250; got != want {
			t.Errorf("threatOpponents() = %d, want %d", got, want)
		}
	})
	t.Run("No affected tile this tick", func(t *testing.T) {
		gs := newTestGameState(9, 9)
		unitIDs, kingID, _ := twoOpponentsAveraged(gs)
		sc := scoreContext{opponentIDs: unitIDs, opponentKingID: kingID}
		tr := turnResult{turn: 1, aliveOpponentsAfter: 2}
		if got := threatOpponents(gs, sc, tr); got != 0 {
			t.Errorf("threatOpponents() = %d, want 0", got)
		}
	})
	t.Run("Empty opponent roster (King only)", func(t *testing.T) {
		gs := newTestGameState(9, 9)
		_, kingID, affectedTiles := twoOpponentsAveraged(gs)
		sc := scoreContext{opponentIDs: []engine.UnitID{kingID}, opponentKingID: kingID}
		tr := turnResult{turn: 1, affectedTiles: affectedTiles, aliveOpponentsAfter: 0}
		if got := threatOpponents(gs, sc, tr); got != 0 {
			t.Errorf("threatOpponents() = %d, want 0", got)
		}
	})
	t.Run("All opponents dead: still scores their frozen-position exposure", func(t *testing.T) {
		gs := newTestGameState(9, 9)
		unitIDs, kingID, affectedTiles := twoOpponentsAveraged(gs)
		for _, id := range unitIDs {
			if id != kingID {
				gs.Units[id].HP = 0
			}
		}
		sc := scoreContext{opponentIDs: unitIDs, opponentKingID: kingID}
		tr := turnResult{turn: 1, affectedTiles: affectedTiles, aliveOpponentsAfter: 0}
		if got, want := threatOpponents(gs, sc, tr), 239; got != want {
			t.Errorf("threatOpponents() = %d, want %d", got, want)
		}
	})

	tests := []struct {
		name string
		k    int
		want int
	}{
		{name: "Averages across alive opponents, King excluded: k=1", k: 1, want: 239},
		{name: "Averages across alive opponents, King excluded: k=3", k: 3, want: 183},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := newTestGameState(9, 9)
			unitIDs, kingID, affectedTiles := twoOpponentsAveraged(gs)
			sc := scoreContext{opponentIDs: unitIDs, opponentKingID: kingID}
			tr := turnResult{turn: tt.k, affectedTiles: affectedTiles, aliveOpponentsAfter: 2}
			if got := threatOpponents(gs, sc, tr); got != tt.want {
				t.Errorf("threatOpponents() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRiskAllies(t *testing.T) {
	for _, turn := range []int{0, 1} {
		t.Run(fmt.Sprintf("At T+%d: scores as urgent, not excluded", turn), func(t *testing.T) {
			gs := newTestGameState(9, 9)
			unitIDs, kingID, affectedTiles := twoAlliesAveraged(gs)
			sc := scoreContext{allyIDs: unitIDs, allyKingID: kingID}
			tr := turnResult{turn: turn, affectedTiles: affectedTiles, aliveAlliesAfter: 2}
			if got, want := riskAllies(gs, sc, tr), 250; got != want {
				t.Errorf("riskAllies() = %d, want %d", got, want)
			}
		})
	}
	t.Run("No affected tile this tick", func(t *testing.T) {
		gs := newTestGameState(9, 9)
		unitIDs, kingID, _ := twoAlliesAveraged(gs)
		sc := scoreContext{allyIDs: unitIDs, allyKingID: kingID}
		tr := turnResult{turn: 2, aliveAlliesAfter: 2}
		if got := riskAllies(gs, sc, tr); got != 0 {
			t.Errorf("riskAllies() = %d, want 0", got)
		}
	})
	t.Run("Empty ally roster (King only)", func(t *testing.T) {
		gs := newTestGameState(9, 9)
		_, kingID, affectedTiles := twoAlliesAveraged(gs)
		sc := scoreContext{allyIDs: []engine.UnitID{kingID}, allyKingID: kingID}
		tr := turnResult{turn: 2, affectedTiles: affectedTiles, aliveAlliesAfter: 0}
		if got := riskAllies(gs, sc, tr); got != 0 {
			t.Errorf("riskAllies() = %d, want 0", got)
		}
	})
	t.Run("All allies dead: still scores their frozen-position exposure", func(t *testing.T) {
		gs := newTestGameState(9, 9)
		unitIDs, kingID, affectedTiles := twoAlliesAveraged(gs)
		for _, id := range unitIDs {
			if id != kingID {
				gs.Units[id].HP = 0
			}
		}
		sc := scoreContext{allyIDs: unitIDs, allyKingID: kingID}
		tr := turnResult{turn: 2, affectedTiles: affectedTiles, aliveAlliesAfter: 0}
		if got, want := riskAllies(gs, sc, tr), 239; got != want {
			t.Errorf("riskAllies() = %d, want %d", got, want)
		}
	})

	tests := []struct {
		name string
		k    int
		want int
	}{
		{name: "Averages across alive allies, King excluded: k=1", k: 1, want: 239},
		{name: "Averages across alive allies, King excluded: k=3", k: 3, want: 183},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := newTestGameState(9, 9)
			unitIDs, kingID, affectedTiles := twoAlliesAveraged(gs)
			sc := scoreContext{allyIDs: unitIDs, allyKingID: kingID}
			tr := turnResult{turn: tt.k + 1, affectedTiles: affectedTiles, aliveAlliesAfter: 2}
			if got := riskAllies(gs, sc, tr); got != tt.want {
				t.Errorf("riskAllies() = %d, want %d", got, tt.want)
			}
		})
	}
}
