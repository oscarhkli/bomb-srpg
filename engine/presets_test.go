package engine

import (
	"maps"
	"slices"
	"testing"
)

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
			name:           "Existing archetype Bandit",
			inputName:      "Bandit",
			expectedExists: true,
			expectedName:   "Bandit",
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
	archetypes := []string{"King", "Fighter", "Witch", "Bandit"}
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

func TestGetAllArchetypes(t *testing.T) {
	archeTypes := GetAllArchetypes()

	missing := map[string]bool{
		"Fighter": true,
		"Witch":   true,
		"Bandit":  true,
	}

	for _, archeTypes := range archeTypes {
		if missing[archeTypes.Name] {
			delete(missing, archeTypes.Name)
		}
	}

	if len(missing) > 0 {
		t.Errorf("GetAllArchetypes misses Archetype %v", slices.Collect(maps.Keys(missing)))
	}
}

func TestGetStagePreset(t *testing.T) {
	tests := []struct {
		name           string
		inputName      string
		expectedExists bool
	}{
		{
			name:           "Existing stage preset Plain",
			inputName:      "Plain",
			expectedExists: true,
		},
		{
			name:           "Existing stage prset Standard",
			inputName:      "Standard",
			expectedExists: true,
		},
		{
			name:           "Existing stage prset Divided",
			inputName:      "Divided",
			expectedExists: true,
		},
		{
			name:           "Non-existing stage preset",
			inputName:      "NonExistent",
			expectedExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, exists := GetStagePreset(tt.inputName)
			if exists != tt.expectedExists {
				t.Errorf("Expected exists to be %v, got %v", tt.expectedExists, exists)
			}
		})
	}
}

func TestStagePresets(t *testing.T) {
	for _, s := range stagePresetsRegistry() {
		t.Run(s.Name, func(t *testing.T) {
			stagePreset, exists := GetStagePreset(s.Name)
			if !exists {
				t.Fatalf("Stage preset %s should exist", s.Name)
			}
			if stagePreset.Width < 5 {
				t.Errorf("Width for %s should be minimum 5, got %d", s.Name, stagePreset.Width)
			}
			if stagePreset.Height < 5 {
				t.Errorf("Height for %s should be minimum 5, got %d", s.Name, stagePreset.Height)
			}

			// Check if layout grid dimensions match width and height
			if len(stagePreset.LayoutGrid) != stagePreset.Height {
				t.Errorf("LayoutGrid row count for %s should match Height, got %d rows, expected %d", s.Name, len(stagePreset.LayoutGrid), stagePreset.Height)
			} else {
				for i, row := range stagePreset.LayoutGrid {
					if len(row) != stagePreset.Width {
						t.Errorf("LayoutGrid row %d for %s should match Width, got %d columns, expected %d", i, s.Name, len(row), stagePreset.Width)
					}
				}
			}

			// Check if all SoftBlocks are in terrainPlain (.) positions
			for _, softBlock := range stagePreset.SoftBlocks {
				if softBlock.X < 0 || softBlock.X >= stagePreset.Width || softBlock.Y < 0 || softBlock.Y >= stagePreset.Height {
					t.Errorf("SoftBlock at (%d, %d) for %s is out of bounds", softBlock.X, softBlock.Y, s.Name)
				} else if stagePreset.LayoutGrid[softBlock.Y][softBlock.X] != '.' {
					t.Errorf("SoftBlock at (%d, %d) for %s should be on a plain terrain (.), but found '%c'", softBlock.X, softBlock.Y, s.Name, stagePreset.LayoutGrid[softBlock.Y][softBlock.X])
				}
			}
			// Check if all SoftBlocks are not overlapping with each other
			softBlockPositions := make(map[Coordinate]bool)
			for _, softBlock := range stagePreset.SoftBlocks {
				pos := Coordinate{softBlock.X, softBlock.Y}
				if softBlockPositions[pos] {
					t.Errorf("SoftBlock at (%d, %d) for %s is overlapping with another SoftBlock", softBlock.X, softBlock.Y, s.Name)
				}
				softBlockPositions[pos] = true
			}

			// Check if starting positions for P1 and P2 are on plain terrain (.)
			for i, pos := range stagePreset.P1StartingPositions {
				if pos.X < 0 || pos.X >= stagePreset.Width || pos.Y < 0 || pos.Y >= stagePreset.Height {
					t.Errorf("P1 Starting Position %d at (%d, %d) for %s is out of bounds", i, pos.X, pos.Y, s.Name)
				} else if stagePreset.LayoutGrid[pos.Y][pos.X] != '.' {
					t.Errorf("P1 Starting Position %d at (%d, %d) for %s should be on a plain terrain (.), but found '%c'", i, pos.X, pos.Y, s.Name, stagePreset.LayoutGrid[pos.Y][pos.X])
				}
			}
			for i, pos := range stagePreset.P2StartingPositions {
				if pos.X < 0 || pos.X >= stagePreset.Width || pos.Y < 0 || pos.Y >= stagePreset.Height {
					t.Errorf("P2 Starting Position %d at (%d, %d) for %s is out of bounds", i, pos.X, pos.Y, s.Name)
				} else if stagePreset.LayoutGrid[pos.Y][pos.X] != '.' {
					t.Errorf("P2 Starting Position %d at (%d, %d) for %s should be on a plain terrain (.), but found '%c'", i, pos.X, pos.Y, s.Name, stagePreset.LayoutGrid[pos.Y][pos.X])
				}
			}
		})
	}
}

// Stage sanity checks on whether all characters' starting positions can reach the opponent's starting position by walking
func TestStagePrests_Sanity(t *testing.T) {
	rule := MovementRule{
		MaxSteps:        -1,
		Pattern:         PatternCardinal,
		CanTurn:         true,
		PassPermissions: PassUnits | PassSoftBlocks | PassItems,
	}

	for _, s := range stagePresetsRegistry() {
		t.Run(s.Name, func(t *testing.T) {
			gs, err := initGameState(GameCfg{
				StagePreset: s.Name,
				P1Teams:     []string{"King"}, // we don't care this value in this test, as long as it can create a GameState
				P2Teams:     []string{"King"}, // same as above
			})

			if err != nil {
				t.Fatal(err)
			}

			stagePreset, exists := GetStagePreset(s.Name)
			if !exists {
				t.Fatalf("Stage preset %s should exist", s.Name)
			}

			for i, p1Pos := range stagePreset.P1StartingPositions {
				tiles := gs.FindReachableTiles(p1Pos, rule)

				for j, p2Pos := range stagePreset.P2StartingPositions {
					if _, ok := tiles[p2Pos]; !ok {
						t.Errorf("Reachable validation failed for %s: P1 Starting Position %d at (%d, %d) cannot reach P2 Starting Position %d at (%d, %d)",
							s.Name, i, p1Pos.X, p1Pos.Y, j, p2Pos.X, p2Pos.Y)
					}
				}
			}
		})
	}
}
