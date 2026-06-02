package engine

import (
	"maps"
	"testing"
)

func TestGameState_FindReachableTiles_TerrainAndBoundaries(t *testing.T) {
	state := &GameState{
		Grid: [][]Tile{
			{{Type: TerrainPlain}, {Type: TerrainPlain}, {Type: TerrainPlain}, {Type: TerrainPlain}, {Type: TerrainBlock}},
			{{Type: TerrainPlain}, {Type: TerrainPlain}, {Type: TerrainBlock}, {Type: TerrainPlain}, {Type: TerrainPlain}},
			{{Type: TerrainBlock}, {Type: TerrainBlock}, {Type: TerrainPlain}, {Type: TerrainBlock}, {Type: TerrainPlain}},
			{{Type: TerrainPlain}, {Type: TerrainPlain}, {Type: TerrainPlain}, {Type: TerrainBlock}, {Type: TerrainPlain}},
		},
	}
	startPos := Coordinate{1, 1}

	tests := []struct {
		name     string
		rule     MovementRule
		expected map[Coordinate]int
	}{
		{
			name: "1 step, all impassable",
			rule: MovementRule{
				MaxSteps: 1,
				Pattern:  PatternCardinal,
			},
			expected: map[Coordinate]int{
				{1, 0}: 1,
				{0, 1}: 1, {1, 1}: 0,
			},
		},
		{
			name: "large steps, all impassable",
			rule: MovementRule{
				MaxSteps: 2,
				Pattern:  PatternCardinal,
			},
			expected: map[Coordinate]int{
				{1, 0}: 1,
				{0, 1}: 1, {1, 1}: 0,
			},
		},
		{
			name: "1 step, can pass hard blocks",
			rule: MovementRule{
				MaxSteps:        1,
				Pattern:         PatternCardinal,
				PassPermissions: PassHardBlocks,
			},
			expected: map[Coordinate]int{
				{1, 0}: 1,
				{0, 1}: 1, {1, 1}: 0, {2, 1}: 1,
				{1, 2}: 1,
			},
		},
		{
			name: "2 steps, can pass hard blocks",
			rule: MovementRule{
				MaxSteps:        2,
				Pattern:         PatternCardinal,
				PassPermissions: PassHardBlocks,
			},
			expected: map[Coordinate]int{
				{1, 0}: 1,
				{0, 1}: 1, {1, 1}: 0, {2, 1}: 1, {3, 1}: 2,
				{1, 2}: 1,
				{1, 3}: 2,
			},
		},
		{
			name: "2 steps, can pass all",
			rule: MovementRule{
				MaxSteps:        2,
				Pattern:         PatternCardinal,
				PassPermissions: PassUnits | PassSoftBlocks | PassHardBlocks | PassItems | PassBombs,
			},
			expected: map[Coordinate]int{
				{1, 0}: 1,
				{0, 1}: 1, {1, 1}: 0, {2, 1}: 1, {3, 1}: 2,
				{1, 2}: 1,
				{1, 3}: 2,
			},
		},
		{
			name: "infinite steps, can turn, all impassable",
			rule: MovementRule{
				MaxSteps: -1,
				Pattern:  PatternCardinal,
				CanTurn:  true,
			},
			expected: map[Coordinate]int{
				{0, 0}: 2, {1, 0}: 1, {2, 0}: 2, {3, 0}: 3,
				{0, 1}: 1, {1, 1}: 0, {3, 1}: 4, {4, 1}: 5,
				{4, 2}: 6,
				{4, 3}: 7,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := state.FindReachableTiles(startPos, tt.rule)
			if !maps.Equal(result, tt.expected) {
				extra, missing, mismatched := diffMaps(tt.expected, result)

				t.Errorf("FAIL: %s\n"+
					"  Missing tiles (expected but got none):   %v\n"+
					"  Extra tiles (got but didn't expect):     %v\n"+
					"  Mismatched step counts (coord: exp!=got): %v",
					tt.name, missing, extra, mismatched)
			}
		})
	}
}

func TestGameState_FindReachableTiles_OccupiedTiles(t *testing.T) {
	occupants := [][]ObjectType{
		{ObjectNone, ObjectUnit, ObjectSoftBlock, ObjectUnit, ObjectItem, ObjectBomb, ObjectBomb, ObjectSoftBlock},
		{ObjectUnit, ObjectUnit, ObjectSoftBlock, ObjectUnit, ObjectBomb, ObjectBomb, ObjectItem, ObjectSoftBlock},
		{ObjectUnit, ObjectUnit, ObjectSoftBlock, ObjectUnit, ObjectNone, ObjectNone, ObjectBomb, ObjectBomb},
		{ObjectUnit, ObjectUnit, ObjectUnit, ObjectUnit, ObjectBomb, ObjectSoftBlock, ObjectBomb, ObjectNone},
	}
	grid := make([][]Tile, len(occupants))
	for y, row := range occupants {
		grid[y] = make([]Tile, len(row))
		for x, occupant := range row {
			grid[y][x] = Tile{
				Type:         TerrainPlain,
				OccupantType: occupant,
			}
		}
	}
	state := &GameState{
		Grid: grid,
	}
	startPos := Coordinate{4, 1}

	tests := []struct {
		name     string
		rule     MovementRule
		expected map[Coordinate]int
	}{
		{
			name: "all impassable",
			rule: MovementRule{
				MaxSteps: -1,
				Pattern:  PatternCardinal,
			},
			expected: map[Coordinate]int{
				{4, 1}: 0,
				{4, 2}: 1,
			},
		},
		{
			name: "can pass units",
			rule: MovementRule{
				MaxSteps:        -1,
				Pattern:         PatternCardinal,
				PassPermissions: PassUnits,
			},
			expected: map[Coordinate]int{
				{3, 1}: 1, {4, 1}: 0,
				{4, 2}: 1,
			},
		},
		{
			name: "can pass units, stops on non-unit occupant", // like bomb blast movement
			rule: MovementRule{
				MaxSteps:              -1,
				Pattern:               PatternCardinal,
				PassPermissions:       PassUnits,
				StopOnNonUnitOccupant: true,
			},
			expected: map[Coordinate]int{
				{4, 0}: 1,
				{2, 1}: 2, {3, 1}: 1, {4, 1}: 0, {5, 1}: 1,
				{4, 2}: 1,
				{4, 3}: 2,
			},
		},
		{
			name: "can pass bombs, stops on non-unit occupant",
			rule: MovementRule{
				MaxSteps:              -1,
				Pattern:               PatternCardinal,
				PassPermissions:       PassBombs,
				StopOnNonUnitOccupant: true,
			},
			expected: map[Coordinate]int{
				{4, 0}: 1,
				{4, 1}: 0, {5, 1}: 1, {6, 1}: 2,
				{4, 2}: 1,
				{4, 3}: 2,
			},
		},
		{
			name: "can pass bombs, items and units, can turn",
			rule: MovementRule{
				MaxSteps:        -1,
				Pattern:         PatternCardinal,
				PassPermissions: PassBombs | PassItems | PassUnits,
				CanTurn:         true,
			},
			expected: map[Coordinate]int{
				{0, 0}: 9, {1, 0}: 8, {3, 0}: 2, {4, 0}: 1, {5, 0}: 2, {6, 0}: 3,
				{0, 1}: 8, {1, 1}: 7, {3, 1}: 1, {4, 1}: 0, {5, 1}: 1, {6, 1}: 2,
				{0, 2}: 7, {1, 2}: 6, {3, 2}: 2, {4, 2}: 1, {5, 2}: 2, {6, 2}: 3, {7, 2}: 4,
				{0, 3}: 6, {1, 3}: 5, {2, 3}: 4, {3, 3}: 3, {4, 3}: 2, {6, 3}: 4, {7, 3}: 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := state.FindReachableTiles(startPos, tt.rule)
			if !maps.Equal(result, tt.expected) {
				extra, missing, mismatched := diffMaps(tt.expected, result)

				t.Errorf("FAIL: %s\n"+
					"  Missing tiles (expected but got none):   %v\n"+
					"  Extra tiles (got but didn't expect):     %v\n"+
					"  Mismatched step counts (coord: exp!=got): %v",
					tt.name, missing, extra, mismatched)
			}
		})
	}
}

// diffMaps compares two coordinate maps and cleanly isolates the discrepancies.
func diffMaps(expected, got map[Coordinate]int) (extra, missing map[Coordinate]int, mismatched map[Coordinate][2]int) {
	extra = make(map[Coordinate]int)
	missing = make(map[Coordinate]int)
	mismatched = make(map[Coordinate][2]int)

	// Find missing tiles and mismatched values
	for coord, expStep := range expected {
		gotStep, exists := got[coord]
		if !exists {
			missing[coord] = expStep
			continue
		}
		if expStep != gotStep {
			mismatched[coord] = [2]int{expStep, gotStep}
		}
	}

	// Find extra tiles
	for coord, gotStep := range got {
		if _, exists := expected[coord]; !exists {
			extra[coord] = gotStep
		}
	}

	return extra, missing, mismatched
}

func TestGameState_FindReachableTiles_0HeightDimensionGrid(t *testing.T) {
	state := &GameState{
		Grid: [][]Tile{},
	}
	startPos := Coordinate{0, 0}
	rule := MovementRule{
		MaxSteps: 1,
	}
	result := state.FindReachableTiles(startPos, rule)
	if len(result) != 0 {
		t.Errorf("Expected no reachable tiles for 0-height grid, got %v", result)
	}
}

func TestGameState_FindReachableTiles_0WidthDimensionGrid(t *testing.T) {
	state := &GameState{
		Grid: [][]Tile{
			{},
			{},
		},
	}
	startPos := Coordinate{0, 0}
	rule := MovementRule{
		MaxSteps: 1,
	}
	result := state.FindReachableTiles(startPos, rule)
	if len(result) != 0 {
		t.Errorf("Expected no reachable tiles for 0-width grid, got %v", result)
	}
}

func TestUnit_NewMovementRule_BasicWalking(t *testing.T) {
	unit := Unit{Type: Archetype{Name: "King"}}

	mr := unit.NewMovementRule()

	if mr.MaxSteps != unit.Speed {
		t.Errorf("Expected MaxSteps to match unit speed (%d), got %d", unit.Speed, mr.MaxSteps)
	}

	if mr.Pattern != PatternCardinal {
		t.Errorf("Expected step pattern to be PatternCardinal, got %v", mr.Pattern)
	}

	if mr.CanTurn {
		t.Error("Expected CanTurn to be false for baseline walking profiles")
	}

	if mr.StopOnNonUnitOccupant {
		t.Error("Expected StopOnNonUnitOccupant to be false for character pedestrian walking")
	}

	// Verify that the correct permission bit flag is flipped ON
	if mr.PassPermissions&PassItems == 0 {
		t.Error("Security flaw: Baseline movement rule is missing the PassItems permission gate!")
	}

	// Verify that all other barriers are locked DOWN (default-fail)
	forbiddenFlags := PassUnits | PassSoftBlocks | PassHardBlocks | PassBombs
	if mr.PassPermissions&forbiddenFlags != 0 {
		t.Errorf("Boundary leak: Baseline walking rule was granted unauthorized privileges (Bitmask: %b)",
			mr.PassPermissions)
	}
}
