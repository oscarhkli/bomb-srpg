package cpu

import (
	"bomb-srpg/engine"
	"testing"
)

func TestReachDist(t *testing.T) {
	tests := []struct {
		name       string
		newUnit    func(gs *engine.GameState, id engine.UnitID, pos engine.Coordinate) *engine.Unit
		unitPos    engine.Coordinate
		setupBoard func(gs *engine.GameState, unit *engine.Unit)
		target     engine.Coordinate
		want       int
	}{
		{
			name:       "Reachable in a straight line",
			newUnit:    addFighter,
			unitPos:    engine.Coordinate{X: 2, Y: 2},
			setupBoard: func(gs *engine.GameState, unit *engine.Unit) {},
			target:     engine.Coordinate{X: 2, Y: 0},
			want:       2,
		},
		{
			name:       "Reachable around a corner",
			newUnit:    addFighter,
			unitPos:    engine.Coordinate{X: 2, Y: 2},
			setupBoard: func(gs *engine.GameState, unit *engine.Unit) {},
			target:     engine.Coordinate{X: 0, Y: 0},
			want:       4,
		},
		{
			name:       "Unit already standing on target",
			newUnit:    addFighter,
			unitPos:    engine.Coordinate{X: 2, Y: 2},
			setupBoard: func(gs *engine.GameState, unit *engine.Unit) {},
			target:     engine.Coordinate{X: 2, Y: 2},
			want:       0,
		},
		{
			name:    "Unreachable: unit fully boxed in by TerrainBlocks",
			newUnit: addFighter,
			unitPos: engine.Coordinate{X: 2, Y: 2},
			setupBoard: func(gs *engine.GameState, unit *engine.Unit) {
				setTerrainBlock(gs, engine.Coordinate{X: 2, Y: 1})
				setTerrainBlock(gs, engine.Coordinate{X: 2, Y: 3})
				setTerrainBlock(gs, engine.Coordinate{X: 1, Y: 2})
				setTerrainBlock(gs, engine.Coordinate{X: 3, Y: 2})
			},
			target: engine.Coordinate{X: 0, Y: 0},
			want:   -1,
		},
		{
			name:    "Unreachable: unit fully boxed in by softBlocks",
			newUnit: addFighter,
			unitPos: engine.Coordinate{X: 2, Y: 2},
			setupBoard: func(gs *engine.GameState, unit *engine.Unit) {
				addSoftBlock(gs, 1, engine.Coordinate{X: 2, Y: 1})
				addSoftBlock(gs, 2, engine.Coordinate{X: 2, Y: 3})
				addSoftBlock(gs, 3, engine.Coordinate{X: 1, Y: 2})
				addSoftBlock(gs, 4, engine.Coordinate{X: 3, Y: 2})
			},
			target: engine.Coordinate{X: 0, Y: 0},
			want:   -1,
		},
		{
			name:    "Reachable through a bomb on the path",
			newUnit: addFighter,
			unitPos: engine.Coordinate{X: 2, Y: 2},
			setupBoard: func(gs *engine.GameState, unit *engine.Unit) {
				addBomb(gs, engine.NewBombID(1, 0, unit.ID), engine.Coordinate{X: 2, Y: 1})
			},
			target: engine.Coordinate{X: 2, Y: 0},
			want:   2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := newTestGameState(5, 5)
			unit := tt.newUnit(gs, engine.NewUnitID(1, 1), tt.unitPos)
			tt.setupBoard(gs, unit)

			if got := reachDist(gs, unit, tt.target); got != tt.want {
				t.Errorf("reachDist() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNearestAffectedDist(t *testing.T) {
	tests := []struct {
		name       string
		newUnit    func(gs *engine.GameState, id engine.UnitID, pos engine.Coordinate) *engine.Unit
		unitPos    engine.Coordinate
		setupBoard func(gs *engine.GameState, unit *engine.Unit)
		affected   []engine.Coordinate
		want       int
	}{
		{
			name:       "Single affected tile, reachable in a straight line",
			newUnit:    addFighter,
			unitPos:    engine.Coordinate{X: 2, Y: 2},
			setupBoard: func(gs *engine.GameState, unit *engine.Unit) {},
			affected:   []engine.Coordinate{{X: 2, Y: 0}},
			want:       2,
		},
		{
			name:       "Multiple affected tiles picks the nearest",
			newUnit:    addFighter,
			unitPos:    engine.Coordinate{X: 2, Y: 2},
			setupBoard: func(gs *engine.GameState, unit *engine.Unit) {},
			affected:   []engine.Coordinate{{X: 2, Y: 0}, {X: 2, Y: 1}},
			want:       1,
		},
		{
			name:       "Unit already standing on an affected tile",
			newUnit:    addFighter,
			unitPos:    engine.Coordinate{X: 2, Y: 2},
			setupBoard: func(gs *engine.GameState, unit *engine.Unit) {},
			affected:   []engine.Coordinate{{X: 2, Y: 2}, {X: 0, Y: 0}},
			want:       0,
		},
		{
			name:       "Empty affectedTiles",
			newUnit:    addFighter,
			unitPos:    engine.Coordinate{X: 2, Y: 2},
			setupBoard: func(gs *engine.GameState, unit *engine.Unit) {},
			affected:   nil,
			want:       -1,
		},
		{
			name:    "Non-empty affectedTiles, all unreachable: unit fully boxed in by TerrainBlocks",
			newUnit: addFighter,
			unitPos: engine.Coordinate{X: 2, Y: 2},
			setupBoard: func(gs *engine.GameState, unit *engine.Unit) {
				setTerrainBlock(gs, engine.Coordinate{X: 2, Y: 1})
				setTerrainBlock(gs, engine.Coordinate{X: 2, Y: 3})
				setTerrainBlock(gs, engine.Coordinate{X: 1, Y: 2})
				setTerrainBlock(gs, engine.Coordinate{X: 3, Y: 2})
			},
			affected: []engine.Coordinate{{X: 0, Y: 0}},
			want:     -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := newTestGameState(5, 5)
			unit := tt.newUnit(gs, engine.NewUnitID(1, 1), tt.unitPos)
			tt.setupBoard(gs, unit)

			affectedTiles := make(map[engine.Coordinate]struct{}, len(tt.affected))
			for _, pos := range tt.affected {
				affectedTiles[pos] = struct{}{}
			}

			if got := nearestAffectedDist(gs, unit, affectedTiles); got != tt.want {
				t.Errorf("nearestAffectedDist() = %d, want %d", got, tt.want)
			}
		})
	}
}
