package engine

import "testing"

func TestGetArchetype(t *testing.T) {
	tests := []struct {
		name           string
		inputName      string
		expectedExists bool
		expectedName   string
	}{
		{
			name:           "Existing archetype King",
			inputName:      "King",
			expectedExists: true,
			expectedName:   "King",
		},
		{
			name:           "Existing archetype Fighter",
			inputName:      "Fighter",
			expectedExists: true,
			expectedName:   "Fighter",
		},
		{
			name:           "Existing archetype Witch",
			inputName:      "Witch",
			expectedExists: true,
			expectedName:   "Witch",
		},
		{
			name:           "Existing archetype Thief",
			inputName:      "Thief",
			expectedExists: true,
			expectedName:   "Thief",
		},
		{
			name:           "Non-existing archetype",
			inputName:      "NonExistent",
			expectedExists: false,
			expectedName:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archetype, exists := GetArchetype(tt.inputName)
			if exists != tt.expectedExists {
				t.Errorf("Expected exists to be %v, got %v", tt.expectedExists, exists)
			}
			if exists && archetype.Name != tt.expectedName {
				t.Errorf("Expected archetype name to be %s, got %s", tt.expectedName, archetype.Name)
			}
		})
	}
}

func TestStatBoundaries(t *testing.T) {
	archetypes := []string{"King", "Fighter", "Witch", "Thief"}
	for _, name := range archetypes {
		t.Run(name, func(t *testing.T) {
			archetype, exists := GetArchetype(name)
			if !exists {
				t.Fatalf("Archetype %s should exist", name)
			}
			if archetype.BaseSpeed < 0 {
				t.Errorf("BaseSpeed for %s should be non-negative, got %d", name, archetype.BaseSpeed)
			}
			if archetype.BombMaxRange < 0 {
				t.Errorf("BombMaxRange for %s should be non-negative, got %d", name, archetype.BombMaxRange)
			}
			if archetype.BombMinRange < 0 {
				t.Errorf("BombMinRange for %s should be non-negative, got %d", name, archetype.BombMinRange)
			}
			if archetype.BombMinRange > archetype.BombMaxRange {
				t.Errorf("BombMinRange for %s should not exceed BombMaxRange, got Min: %d, Max: %d", name, archetype.BombMinRange, archetype.BombMaxRange)
			}
			if archetype.BombPower < 0 {
				t.Errorf("BombPower for %s should be non-negative, got %d", name, archetype.BombPower)
			}
			if archetype.MaxBombCount < 0 {
				t.Errorf("MaxBombCount for %s should be non-negative, got %d", name, archetype.MaxBombCount)
			}
			if archetype.BaseHP < 0 {
				t.Errorf("BaseHP for %s should be non-negative, got %d", name, archetype.BaseHP)
			}
		})
	}
}

func TestGetStagePreset(t *testing.T) {
	tests := []struct {
		name           string
		inputName      string
		expectedExists bool
		expectedName   string
	}{
		{
			name:           "Existing stage preset Plain",
			inputName:      "Plain",
			expectedExists: true,
			expectedName:   "Plain",
		},
		{
			name:           "Existing stage prset Standard",
			inputName:      "Standard",
			expectedExists: true,
			expectedName:   "Standard",
		},
		{
			name:           "Non-existing stage preset",
			inputName:      "NonExistent",
			expectedExists: false,
			expectedName:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stagePreset, exists := GetStagePreset(tt.inputName)
			if exists != tt.expectedExists {
				t.Errorf("Expected exists to be %v, got %v", tt.expectedExists, exists)
			}
			if exists && stagePreset.Name != tt.expectedName {
				t.Errorf("Expected stage preset name to be %s, got %s", tt.expectedName, stagePreset.Name)
			}
		})
	}
}

func TestStagePresets(t *testing.T) {
	stagePresets := []string{"Plain", "Standard"}
	for _, name := range stagePresets {
		t.Run(name, func(t *testing.T) {
			stagePreset, exists := GetStagePreset(name)
			if !exists {
				t.Fatalf("Stage preset %s should exist", name)
			}
			if stagePreset.Width < 5 {
				t.Errorf("Width for %s should be minimum 5, got %d", name, stagePreset.Width)
			}
			if stagePreset.Height < 5 {
				t.Errorf("Height for %s should be minimum 5, got %d", name, stagePreset.Height)
			}
			// Check if layout grid dimensions match width and height
			if len(stagePreset.LayoutGrid) != stagePreset.Height {
				t.Errorf("LayoutGrid row count for %s should match Height, got %d rows, expected %d", name, len(stagePreset.LayoutGrid), stagePreset.Height)
			} else {
				for i, row := range stagePreset.LayoutGrid {
					if len(row) != stagePreset.Width {
						t.Errorf("LayoutGrid row %d for %s should match Width, got %d columns, expected %d", i, name, len(row), stagePreset.Width)
					}
				}
			}
			// Check if all SoftBlocks are in terrainPlain (.) positions
			for _, softBlock := range stagePreset.SoftBlocks {
				if softBlock.X < 0 || softBlock.X >= stagePreset.Width || softBlock.Y < 0 || softBlock.Y >= stagePreset.Height {
					t.Errorf("SoftBlock at (%d, %d) for %s is out of bounds", softBlock.X, softBlock.Y, name)
				} else if stagePreset.LayoutGrid[softBlock.Y][softBlock.X] != '.' {
					t.Errorf("SoftBlock at (%d, %d) for %s should be on a plain terrain (.), but found '%c'", softBlock.X, softBlock.Y, name, stagePreset.LayoutGrid[softBlock.Y][softBlock.X])
				}
			}
			// Check if all SoftBlocks are not overlapping with each other
			softBlockPositions := make(map[Coordinate]bool)
			for _, softBlock := range stagePreset.SoftBlocks {
				pos := Coordinate{X: softBlock.X, Y: softBlock.Y}
				if softBlockPositions[pos] {
					t.Errorf("SoftBlock at (%d, %d) for %s is overlapping with another SoftBlock", softBlock.X, softBlock.Y, name)
				}
				softBlockPositions[pos] = true
			}
			// Check if starting positions for P1 and P2 are on plain terrain (.)
			for i, pos := range stagePreset.P1StartingPositions {
				if pos.X < 0 || pos.X >= stagePreset.Width || pos.Y < 0 || pos.Y >= stagePreset.Height {
					t.Errorf("P1 Starting Position %d at (%d, %d) for %s is out of bounds", i, pos.X, pos.Y, name)
				} else if stagePreset.LayoutGrid[pos.Y][pos.X] != '.' {
					t.Errorf("P1 Starting Position %d at (%d, %d) for %s should be on a plain terrain (.), but found '%c'", i, pos.X, pos.Y, name, stagePreset.LayoutGrid[pos.Y][pos.X])
				}
			}
			for i, pos := range stagePreset.P2StartingPositions {
				if pos.X < 0 || pos.X >= stagePreset.Width || pos.Y < 0 || pos.Y >= stagePreset.Height {
					t.Errorf("P2 Starting Position %d at (%d, %d) for %s is out of bounds", i, pos.X, pos.Y, name)
				} else if stagePreset.LayoutGrid[pos.Y][pos.X] != '.' {
					t.Errorf("P2 Starting Position %d at (%d, %d) for %s should be on a plain terrain (.), but found '%c'", i, pos.X, pos.Y, name, stagePreset.LayoutGrid[pos.Y][pos.X])
				}
			}
		})
	}
}

// TODO: Stage sanity checks on whether all characters' starting positions can reach the opponent's starting position by walking
