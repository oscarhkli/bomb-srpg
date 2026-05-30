package engine

import (
	"fmt"
	"strings"
	"testing"
)

func TestInitGameState_TDD_Suite(t *testing.T) {
	tests := []struct {
		name               string
		cfg                GameCfg
		expectError        bool
		errorContains      string
		expectedTotalUnits int
	}{
		{
			name: "Success: Full Teams (5 vs 5) with Plain Stage",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Teams:     []string{"King", "Fighter", "Witch", "Fighter", "Fighter"},
				P2Teams:     []string{"King", "Fighter", "Witch", "Thief", "Witch"},
			},
			expectError:        false,
			expectedTotalUnits: 10, // 5 for each player
		},
		{
			name: "Success: Minimum Teams (1 vs 1) with Plain Stage",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Teams:     []string{"King"},
				P2Teams:     []string{"King"},
			},
			expectError:        false,
			expectedTotalUnits: 2, // 1 for each player
		},
		{
			name: "Success: Mixed Teams (3 vs 2) with Standard Stage",
			cfg: GameCfg{
				StagePreset: "Standard",
				P1Teams:     []string{"King", "Fighter", "Witch"},
				P2Teams:     []string{"King", "Fighter"},
			},
			expectError:        false,
			expectedTotalUnits: 5, // 3 for Player 1, 2 for Player 2
		},
		{
			name: "Failure: Player 1 has no King",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Teams:     []string{"Fighter", "Witch", "Fighter"},
				P2Teams:     []string{"King", "Fighter"},
			},
			expectError:   true,
			errorContains: "Player 1 must have exactly one King as the first unit",
		},
		{
			name: "Failure: Player 2 has no King",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Teams:     []string{"King", "Fighter"},
				P2Teams:     []string{"Fighter", "Witch"},
			},
			expectError:   true,
			errorContains: "Player 2 must have exactly one King as the first unit",
		},
		{
			name: "Failure: Player 1 has more than 1 King",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Teams:     []string{"King", "King", "Fighter"},
				P2Teams:     []string{"King", "Fighter"},
			},
			expectError:   true,
			errorContains: "Player 1 must have exactly one King as the first unit",
		},
		{
			name: "Failure: Player 2 has more than 1 King",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Teams:     []string{"King", "Fighter"},
				P2Teams:     []string{"King", "King", "Fighter"},
			},
			expectError:   true,
			errorContains: "Player 2 must have exactly one King as the first unit",
		},
		{
			name: "Failure: Player 1's King is not the first unit",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Teams:     []string{"Fighter", "King", "Witch"},
				P2Teams:     []string{"King", "Fighter"},
			},
			expectError:   true,
			errorContains: "Player 1 must have exactly one King as the first unit",
		},
		{
			name: "Failure: Player 2's King is not the first unit",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Teams:     []string{"King", "Fighter"},
				P2Teams:     []string{"Fighter", "King", "Witch"},
			},
			expectError:   true,
			errorContains: "Player 2 must have exactly one King as the first unit",
		},
		{
			name: "Failure: Player 1 has more than 5 units",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Teams:     []string{"King", "Fighter", "Witch", "Thief", "Witch", "Fighter"},
				P2Teams:     []string{"King", "Fighter"},
			},
			expectError:   true,
			errorContains: "Player 1 must have between 1 and 5 units",
		},
		{
			name: "Failure: Player 2 has more than 5 units",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Teams:     []string{"King", "Fighter"},
				P2Teams:     []string{"King", "Fighter", "Witch", "Thief", "Witch", "Fighter"},
			},
			expectError:   true,
			errorContains: "Player 2 must have between 1 and 5 units",
		},
		{
			name: "Failure: Player 1 has no units",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Teams:     []string{},
				P2Teams:     []string{"King", "Fighter"},
			},
			expectError:   true,
			errorContains: "Player 1 must have between 1 and 5 units",
		},
		{
			name: "Failure: Player 2 has no units",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Teams:     []string{"King", "Fighter"},
				P2Teams:     []string{},
			},
			expectError:   true,
			errorContains: "Player 2 must have between 1 and 5 units",
		},
		{
			name: "Failure: Player 1 has an invalid archetype",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Teams:     []string{"King", "InvalidArchetype"},
				P2Teams:     []string{"King", "Fighter"},
			},
			expectError:   true,
			errorContains: "archetype 'InvalidArchetype' for Player 1 not found",
		},
		{
			name: "Failure: Player 2 has an invalid archetype",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Teams:     []string{"King", "Fighter"},
				P2Teams:     []string{"King", "InvalidArchetype"},
			},
			expectError:   true,
			errorContains: "archetype 'InvalidArchetype' for Player 2 not found",
		},
		{
			name: "Failure: Invalid stage preset",
			cfg: GameCfg{
				StagePreset: "NonExistentStage",
				P1Teams:     []string{"King", "Fighter"},
				P2Teams:     []string{"King", "Fighter"},
			},
			expectError:   true,
			errorContains: "stage preset 'NonExistentStage' not found",
		},
		{
			name: "Success: With Global Overrides for Speed and Bomb Range Positive",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Teams:     []string{"King", "Fighter"},
				P2Teams:     []string{"King", "Fighter"},
				GlobalSpeedOverride:        10,
				GlobalBombMaxRangeOverride: 5,
			},
			expectError:   false,
			expectedTotalUnits: 4,
		},
		{
			name: "Success: With Global Overrides for Speed and Bomb Range Zero (No Override)",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Teams:     []string{"King", "Fighter"},
				P2Teams:     []string{"King", "Fighter"},
				GlobalSpeedOverride:        0,
				GlobalBombMaxRangeOverride: 0,
			},
			expectError:   false,
			expectedTotalUnits: 4,
		},
		{
			name: "Success: With Global Overrides for Speed and Bomb Range Negative (Treated as No Override)",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Teams:     []string{"King", "Fighter"},
				P2Teams:     []string{"King", "Fighter"},
				GlobalSpeedOverride:        -5,
				GlobalBombMaxRangeOverride: -3,
			},
			expectError:   false,
			expectedTotalUnits: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gameState, err := initGameState(tt.cfg)

			if (err != nil) != tt.expectError {
				t.Fatalf("Expected error: %v, got: %v", tt.expectError, err)
			}

			if tt.expectError {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
				return // No need to check further if we expected an error
			}

			if len(gameState.Units) != tt.expectedTotalUnits {
				t.Errorf("Expected %d total units, got %d", tt.expectedTotalUnits, len(gameState.Units))
			}

			// Additional checks for unit attributes, grid initialization, etc. can be added here
			preset, _ := GetStagePreset(tt.cfg.StagePreset)
			if len(gameState.Grid) != preset.Height {
				t.Errorf("Expected grid height %d, got %d", preset.Height, len(gameState.Grid))
			}
			for i, row := range gameState.Grid {
				if len(row) != preset.Width {
					t.Errorf("Expected grid width %d in row %d, got %d", preset.Width, i, len(row))
				}
			}

			// Verify that all units have valid initial stats and starting positions
			for id, unit := range gameState.Units {
				// Validate every initial stats of the unit against the archetype
				expectedArchetype, exists := GetArchetype(unit.Type.Name)
				if !exists {
					t.Errorf("Unit ID %d has unknown archetype %s", id, unit.Type.Name)
					continue
				}

				t.Run(fmt.Sprintf("Verify initial stats for Unit %d (%s)", id, unit.Type.Name), func(t *testing.T) {
					// Overridable attributes should be checked against the game config overrides if they are set
					if tt.cfg.GlobalSpeedOverride > 0 {
						expectedArchetype.BaseSpeed = tt.cfg.GlobalSpeedOverride
					}
					if unit.Speed != expectedArchetype.BaseSpeed {
						t.Errorf("Expected unit ID %d to have speed %d, got %d", id, expectedArchetype.BaseSpeed, unit.Speed)
					}
					if tt.cfg.GlobalBombMaxRangeOverride > 0 {
						expectedArchetype.BombMaxRange = tt.cfg.GlobalBombMaxRangeOverride
					}

					if unit.BombMaxRange != expectedArchetype.BombMaxRange {
						t.Errorf("Expected unit ID %d to have BombMaxRange %d, got %d", id, expectedArchetype.BombMaxRange, unit.BombMaxRange)
					}
					if unit.BombMinRange != expectedArchetype.BombMinRange {
						t.Errorf("Expected unit ID %d to have BombMinRange %d, got %d", id, expectedArchetype.BombMinRange, unit.BombMinRange)
					}
					if unit.BombPower != expectedArchetype.BombPower {
						t.Errorf("Expected unit ID %d to have BombPower %d, got %d", id, expectedArchetype.BombPower, unit.BombPower)
					}
					if unit.MaxBombCount != expectedArchetype.MaxBombCount {
						t.Errorf("Expected unit ID %d to have MaxBombCount %d, got %d", id, expectedArchetype.MaxBombCount, unit.MaxBombCount)
					}
					if unit.HP != expectedArchetype.BaseHP {
						t.Errorf("Expected unit ID %d to have HP %d, got %d", id, expectedArchetype.BaseHP, unit.HP)
					}

					if unit.BombUsed != 0 {
						t.Errorf("Expected unit ID %d to have BombUsed 0 at game start, got %d", id, unit.BombUsed)
					}
				})

				switch unit.Team {
				case 1:
					expectedPosition := preset.P1StartingPositions[id-8] // Player 1 IDs start from 8
					if unit.Position != expectedPosition {
						t.Errorf("Expected Player 1 unit ID %d to start at (%d,%d), got (%d,%d)", id, expectedPosition.X, expectedPosition.Y, unit.Position.X, unit.Position.Y)
					}
				case 2:
					expectedPosition := preset.P2StartingPositions[id-16] // Player 2 IDs start from 16
					if unit.Position != expectedPosition {
						t.Errorf("Expected Player 2 unit ID %d to start at (%d,%d), got (%d,%d)", id, expectedPosition.X, expectedPosition.Y, unit.Position.X, unit.Position.Y)
					}
				default:
					t.Errorf("Unit ID %d has invalid team %d", id, unit.Team)
				}
			}

			// Verify bombs initialization
			if len(gameState.Bombs) != 0 {
				t.Errorf("Expected no bombs at game start, got %d", len(gameState.Bombs))
			}

			// Verify soft blocks initialization. Will have real tests later when we have stage presets with soft blocks
			if len(gameState.SoftBlocks) != 0 {
				t.Errorf("Expected no soft blocks in 'Plain' stage, got %d", len(gameState.SoftBlocks))
			}

			// Verify turn and max turns
			if gameState.Turn != 0 {
				t.Errorf("Expected turn to start at 0, got %d", gameState.Turn)
			}
		})
	}
}

func TestInitGameState_LayoutGridCompilation(t *testing.T) {
	tests := []struct {
		name          string
		presetName   string
		customPreset StagePreset // mock sandbox layout for testing
		expectError   bool
	}{
		{
			name:       "Success: Compile Diverse Terrain Matrix",
			presetName: "Sandbox3x3",
			customPreset: StagePreset{
				Name:   "Sandbox3x3",
				Width:  3,
				Height: 3,
				LayoutGrid: []string{
					"T.T", // 
					".HH", // 
					".LW", // 
				},
				P1StartingPositions: [5]Coordinate{{1, 0}},
				P2StartingPositions: [5]Coordinate{{0, 2}},
			},
			expectError: false,
		},
		{
			name:       "Failure: Extra Width Layout typo",
			presetName: "BrokenWidth3x3",
			customPreset: StagePreset{
				Name:   "BrokenWidth3x3",
				Width:  3,
				Height: 3,
				LayoutGrid: []string{
					"...",
					"....",
					"...",
				},
				P1StartingPositions: [5]Coordinate{{0, 0}},
				P2StartingPositions: [5]Coordinate{{2, 2}},
			},
			expectError: true,
		},
		{
			name:       "Failure: Extra Height Layout typo",
			presetName: "BrokenHeight3x3",
			customPreset: StagePreset{
				Name:   "BrokenHeight3x3",
				Width:  3,
				Height: 3,
				LayoutGrid: []string{
					"...",
					"...",
					"...",
					"...",
				},
				P1StartingPositions: [5]Coordinate{{0, 0}},
				P2StartingPositions: [5]Coordinate{{2, 2}},
			},
			expectError: true,
		},
		{
			name:       "Failure: Invalid Token Symbol",
			presetName: "InvalidToken3x3",
			customPreset: StagePreset{
				Name:   "InvalidToken3x3",
				Width:  3,
				Height: 3,
				LayoutGrid: []string{
					"...",
					".X.",
					"...",
				},
				P1StartingPositions: [5]Coordinate{{0, 0}},
				P2StartingPositions: [5]Coordinate{{2, 2}},
			},
			expectError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Temporarily add the custom preset to the registry for testing
			stagePresetsRegistry[tt.customPreset.Name] = tt.customPreset
			defer delete(stagePresetsRegistry, tt.customPreset.Name) // Clean up after test

			gameState, err := initGameState(GameCfg{
				StagePreset: tt.presetName,
				P1Teams:     []string{"King"},
				P2Teams:     []string{"King"},
			})
			
			if (err != nil) != tt.expectError {
				t.Fatalf("Expected error: %v, got: %v", tt.expectError, err)
			}

			if tt.expectError {
				return // No need to check further if we expected an error
			}

			expectedMatrix := [][]TerrainType{
				{TerrainTower, TerrainPlain, TerrainTower},
				{TerrainPlain, TerrainBlock, TerrainBlock},
				{TerrainPlain, TerrainLava, TerrainWater},
			}

			for y, row := range gameState.Grid {
				for x, cell := range row {
					if cell.Type != expectedMatrix[y][x] {
						t.Errorf("Expected terrain at (%d,%d) to be %v, got %v", x, y, expectedMatrix[y][x], cell.Type)
					}

					if cell.OccupantType != ObjectNone || cell.OccupantID != 0 {
						t.Errorf("Expected cell at (%d,%d) to have no occupant, got type %v with ID %d", x, y, cell.OccupantType, cell.OccupantID)
					}
				}
			}
		})
	}
}