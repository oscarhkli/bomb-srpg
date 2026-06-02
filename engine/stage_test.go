package engine

import "testing"

func TestGameState_IsWithinBounds(t *testing.T) {
	tests := []struct {
		name     string
		grid     [][]Tile
		pos      Coordinate
		expected bool
	}{
		{
			name:     "Empty grid",
			grid:     [][]Tile{},
			pos:      Coordinate{0, 0},
			expected: false,
		},
		{
			name:     "Single tile grid, in bounds",
			grid:     [][]Tile{{{Type: TerrainPlain}}},
			pos:      Coordinate{0, 0},
			expected: true,
		},
		{
			name:     "Single tile grid, out of bounds",
			grid:     [][]Tile{{{Type: TerrainPlain}}},
			pos:      Coordinate{1, 0},
			expected: false,
		},
		{
			name: "3x3 grid, in bounds",
			grid: [][]Tile{
				{{Type: TerrainPlain}, {Type: TerrainPlain}, {Type: TerrainPlain}},
				{{Type: TerrainPlain}, {Type: TerrainPlain}, {Type: TerrainPlain}},
				{{Type: TerrainPlain}, {Type: TerrainPlain}, {Type: TerrainPlain}},
			},
			pos:      Coordinate{1, 1},
			expected: true,
		},
		{
			name: "3x3 grid, out of bounds (negative)",
			grid: [][]Tile{
				{{Type: TerrainPlain}, {Type: TerrainPlain}, {Type: TerrainPlain}},
				{{Type: TerrainPlain}, {Type: TerrainPlain}, {Type: TerrainPlain}},
				{{Type: TerrainPlain}, {Type: TerrainPlain}, {Type: TerrainPlain}},
			},
			pos:      Coordinate{-1, 0},
			expected: false,
		},
		{
			name: "3x3 grid, out of bounds (exceeds width)",
			grid: [][]Tile{
				{{Type: TerrainPlain}, {Type: TerrainPlain}, {Type: TerrainPlain}},
				{{Type: TerrainPlain}, {Type: TerrainPlain}, {Type: TerrainPlain}},
				{{Type: TerrainPlain}, {Type: TerrainPlain}, {Type: TerrainPlain}},
			},
			pos:      Coordinate{3, 0},
			expected: false,
		},
		{
			name: "3x3 grid, out of bounds (exceeds height)",
			grid: [][]Tile{
				{{Type: TerrainPlain}, {Type: TerrainPlain}, {Type: TerrainPlain}},
				{{Type: TerrainPlain}, {Type: TerrainPlain}, {Type: TerrainPlain}},
				{{Type: TerrainPlain}, {Type: TerrainPlain}, {Type: TerrainPlain}},
			},
			pos:      Coordinate{0, 3},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := &GameState{
				Grid: tt.grid,
			}
			result := gs.IsWithinBounds(tt.pos)
			if result != tt.expected {
				t.Errorf("Expected IsWithinBounds(%v) to be %v, got %v", tt.pos, tt.expected, result)
			}
		})
	}
}

func TestGameState_ClearStageTile(t *testing.T) {
	const unitID = 100

	tests := []struct {
		name         string
		clearPos     Coordinate
		verifyPos    Coordinate
		expectedType OccupantType
		expectedID   int
	}{
		{
			name:         "Within bound, can clear",
			clearPos:     Coordinate{1, 0},
			verifyPos:    Coordinate{1, 0},
			expectedType: OccupantNone,
			expectedID:   0,
		},
		{
			name:         "Out of bound, do nothing",
			clearPos:     Coordinate{1000, 1000},
			verifyPos:    Coordinate{1, 0},
			expectedType: OccupantUnit,
			expectedID:   unitID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange: Create a sterile 1x3 grid row for this test case run
			state := &GameState{
				Grid: [][]Tile{{
					{Type: TerrainPlain, OccupantType: OccupantUnit, OccupantID: 123},
					{Type: TerrainPlain, OccupantType: OccupantUnit, OccupantID: unitID},
					{Type: TerrainPlain, OccupantType: OccupantUnit, OccupantID: 456},
				}},
			}

			// Act: Run the target operation
			state.ClearStageTile(tt.clearPos)

			// Assert: Dynamically inspect the precise coordinate vector specified by the test case row
			cell := state.Grid[tt.verifyPos.Y][tt.verifyPos.X]

			if cell.OccupantType != tt.expectedType {
				t.Errorf("ClearStageTile %q failure: expected OccupantType at %v to be %v, got %v",
					tt.name, tt.verifyPos, tt.expectedType, cell.OccupantType)
			}
			if cell.OccupantID != tt.expectedID {
				t.Errorf("ClearStageTile %q failure: expected OccupantID at %v to be %d, got %d",
					tt.name, tt.verifyPos, tt.expectedID, cell.OccupantID)
			}
		})
	}
}

func TestGameState_UpdateStageOccupant(t *testing.T) {
	tests := []struct {
		name         string
		updatePos    Coordinate
		verifyPos    Coordinate
		expectedType OccupantType
		expectedID   int
	}{
		{
			name:         "Within bound, can update",
			updatePos:    Coordinate{1, 0},
			verifyPos:    Coordinate{1, 0},
			expectedType: OccupantBomb,
			expectedID:   200,
		},
		{
			name:         "Out of bound, do nothing",
			updatePos:    Coordinate{1000, 1000},
			verifyPos:    Coordinate{1, 0},
			expectedType: OccupantUnit,
			expectedID:   100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange: Create a sterile 1x3 grid row for this test case run
			state := &GameState{
				Grid: [][]Tile{{
					{Type: TerrainPlain, OccupantType: OccupantUnit, OccupantID: 123},
					{Type: TerrainPlain, OccupantType: OccupantUnit, OccupantID: 100},
					{Type: TerrainPlain, OccupantType: OccupantUnit, OccupantID: 456},
				}},
			}

			// Act: Run the target operation
			state.UpdateStageOccupant(tt.updatePos, tt.expectedType, tt.expectedID)

			// Assert: Dynamically inspect the precise coordinate vector specified by the test case row
			cell := state.Grid[tt.verifyPos.Y][tt.verifyPos.X]

			if cell.OccupantType != tt.expectedType {
				t.Errorf("UpdateStageOccupant %q failure: expected OccupantType at %v to be %v, got %v",
					tt.name, tt.verifyPos, tt.expectedType, cell.OccupantType)
			}
			if cell.OccupantID != tt.expectedID {
				t.Errorf("UpdateStageOccupant %q failure: expected OccupantID at %v to be %d, got %d",
					tt.name, tt.verifyPos, tt.expectedID, cell.OccupantID)
			}
		})
	}
}
