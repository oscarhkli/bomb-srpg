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
