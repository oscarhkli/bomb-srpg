package engine

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestInitGameState_Suite(t *testing.T) {
	tests := []struct {
		name               string
		cfg                GameCfg
		wantErr            error
		wantPlayer         string
		expectedTotalUnits int
	}{
		{
			name: "Success: Full Teams (5 vs 5) with Plain Stage",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}, {Archetype: "Witch", Role: RoleNormal}, {Archetype: "Fighter", Role: RoleNormal}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}, {Archetype: "Witch", Role: RoleNormal}, {Archetype: "Bandit", Role: RoleNormal}, {Archetype: "Witch", Role: RoleNormal}},
			},
			expectedTotalUnits: 10, // 5 for each player
		},
		{
			name: "Success: Minimum Teams (2 vs 2) with Plain Stage",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			expectedTotalUnits: 4, // 2 for each player
		},
		{
			name: "Success: Mixed Teams (3 vs 2) with Standard Stage",
			cfg: GameCfg{
				StagePreset: "Standard",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}, {Archetype: "Witch", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			expectedTotalUnits: 5, // 3 for Player 1, 2 for Player 2
		},
		{
			name: "Success: With NO_UNIT",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: NoUnit}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			expectedTotalUnits: 4, // 2 for each player
		},
		{
			name: "Failure: Player 1 has no King",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "Fighter", Role: RoleNormal}, {Archetype: "Witch", Role: RoleNormal}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			wantErr:    ErrMissingKing,
			wantPlayer: "Player 1",
		},
		{
			name: "Failure: Player 2 has no King",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "Fighter", Role: RoleNormal}, {Archetype: "Witch", Role: RoleNormal}},
			},
			wantErr:    ErrMissingKing,
			wantPlayer: "Player 2",
		},
		{
			name: "Failure: Player 1 has more than 1 King",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			wantErr:    ErrMissingKing,
			wantPlayer: "Player 1",
		},
		{
			name: "Failure: Player 2 has more than 1 King",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			wantErr:    ErrMissingKing,
			wantPlayer: "Player 2",
		},
		{
			name: "Failure: Player 1's King is not the first unit",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "Fighter", Role: RoleNormal}, {Archetype: "King", Role: RoleKing}, {Archetype: "Witch", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			wantErr:    ErrMissingKing,
			wantPlayer: "Player 1",
		},
		{
			name: "Failure: Player 2's King is not the first unit",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "Fighter", Role: RoleNormal}, {Archetype: "King", Role: RoleKing}, {Archetype: "Witch", Role: RoleNormal}},
			},
			wantErr:    ErrMissingKing,
			wantPlayer: "Player 2",
		},
		{
			name: "Failure: Player 1 has more than 5 units",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}, {Archetype: "Witch", Role: RoleNormal}, {Archetype: "Bandit", Role: RoleNormal}, {Archetype: "Witch", Role: RoleNormal}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			wantErr:    ErrInvalidTeamSize,
			wantPlayer: "Player 1",
		},
		{
			name: "Failure: Player 2 has more than 5 units",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}, {Archetype: "Witch", Role: RoleNormal}, {Archetype: "Bandit", Role: RoleNormal}, {Archetype: "Witch", Role: RoleNormal}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			wantErr:    ErrInvalidTeamSize,
			wantPlayer: "Player 2",
		},
		{
			name: "Failure: Player 1 has 1 unit",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			wantErr:    ErrInvalidTeamSize,
			wantPlayer: "Player 1",
		},
		{
			name: "Failure: Player 2 has 1 unit",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}},
			},
			wantErr:    ErrInvalidTeamSize,
			wantPlayer: "Player 2",
		},
		{
			name: "Failure: Player 1 has no units",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			wantErr:    ErrInvalidTeamSize,
			wantPlayer: "Player 1",
		},
		{
			name: "Failure: Player 2 has no units",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{},
			},
			wantErr:    ErrInvalidTeamSize,
			wantPlayer: "Player 2",
		},
		{
			name: "Failure: Player 1 has an invalid archetype",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "InvalidArchetype"}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			wantErr:    ErrUnknownArchetype,
			wantPlayer: "Player 1",
		},
		{
			name: "Failure: Player 2 has an invalid archetype",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "InvalidArchetype"}},
			},
			wantErr:    ErrUnknownArchetype,
			wantPlayer: "Player 2",
		},
		{
			name: "Failure: Player 1 has 1 unit as NO_UNIT doesn't count",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: NoUnit}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			wantErr:    ErrInvalidTeamSize,
			wantPlayer: "Player 1",
		},
		{
			name: "Failure: Invalid stage preset",
			cfg: GameCfg{
				StagePreset: "NonExistentStage",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			wantErr: ErrInvalidStagePreset,
		},
		{
			name: "Success: With Global Overrides for Speed and Bomb Range Positive",
			cfg: GameCfg{
				StagePreset:                "Plain",
				P1Slots:                    []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:                    []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
				GlobalSpeedOverride:        10,
				GlobalBombMaxRangeOverride: 5,
			},
			expectedTotalUnits: 4,
		},
		{
			name: "Success: With Global Overrides for Speed and Bomb Range Zero (No Override)",
			cfg: GameCfg{
				StagePreset:                "Plain",
				P1Slots:                    []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:                    []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
				GlobalSpeedOverride:        0,
				GlobalBombMaxRangeOverride: 0,
			},
			expectedTotalUnits: 4,
		},
		{
			name: "Success: With Global Overrides for Speed and Bomb Range Negative (Treated as No Override)",
			cfg: GameCfg{
				StagePreset:                "Plain",
				P1Slots:                    []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:                    []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
				GlobalSpeedOverride:        -5,
				GlobalBombMaxRangeOverride: -3,
			},
			expectedTotalUnits: 4,
		},
		{
			name: "Success: Player 1 With Boss type Archetype but not King",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "Prologue", Role: RoleBoss}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			expectedTotalUnits: 4,
		},
		{
			name: "Success: Player 2 With Boss type Archetype but not King",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "Prologue", Role: RoleBoss}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			expectedTotalUnits: 4,
		},
		{
			name: "Success: Player 1 With Boss type Archetype alone",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "Prologue", Role: RoleBoss}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			expectedTotalUnits: 3,
		},
		{
			name: "Success: Player 2 With Boss type Archetype alone",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "Prologue", Role: RoleBoss}},
			},
			expectedTotalUnits: 3,
		},
		{
			name: "Success: Multiple Boss type Archetypes",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "Prologue", Role: RoleBoss}, {Archetype: "Prologue", Role: RoleBoss}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			expectedTotalUnits: 4,
		},
		{
			name: "Failure: Player 1 With Boss type Archetype and King concurrently",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Prologue", Role: RoleBoss}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			wantErr:    ErrInvalidTeamFormation,
			wantPlayer: "Player 1",
		},
		{
			name: "Failure: Player 2 With Boss type Archetype and King concurrently",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Prologue", Role: RoleBoss}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			wantErr:    ErrInvalidTeamFormation,
			wantPlayer: "Player 2",
		},
		{
			name: "Failure: Player 1 With Boss type Archetype alone, but Player 2 has 1 unit",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "Prologue", Role: RoleBoss}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}},
			},
			wantErr:    ErrInvalidTeamSize,
			wantPlayer: "Player 2",
		},
		{
			name: "Success: Player 2 With Boss type Archetype alone, but Player 1 has 1 unit",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}},
				P2Slots:     []TeamSlot{{Archetype: "Prologue", Role: RoleBoss}},
			},
			wantErr:    ErrInvalidTeamSize,
			wantPlayer: "Player 1",
		},
		{
			name: "Failure: Player 1 With Boss type Archetype, more than 5 units",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots: []TeamSlot{
					{Archetype: "Prologue", Role: RoleBoss}, {Archetype: "Prologue", Role: RoleBoss},
					{Archetype: "Prologue", Role: RoleBoss}, {Archetype: "Prologue", Role: RoleBoss},
					{Archetype: "Prologue", Role: RoleBoss}, {Archetype: "Prologue", Role: RoleBoss},
				},
				P2Slots: []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			wantErr:    ErrInvalidTeamSize,
			wantPlayer: "Player 1",
		},
		{
			name: "Success: Player 1 With Boss type Archetype, exactly 5 units",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots: []TeamSlot{
					{Archetype: "Prologue", Role: RoleBoss}, {Archetype: "Prologue", Role: RoleBoss},
					{Archetype: "Prologue", Role: RoleBoss}, {Archetype: "Prologue", Role: RoleBoss},
					{Archetype: "Prologue", Role: RoleBoss},
				},
				P2Slots: []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			expectedTotalUnits: 7,
		},
		{
			name: "Failure: Player 1 with only a NO_UNIT slot has no real Boss, falls back to King-mode size check",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: NoUnit, Role: RoleBoss}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			wantErr:    ErrInvalidTeamSize,
			wantPlayer: "Player 1",
		},
		{
			name: "Success: Player 1 has a NO_UNIT slot tagged RoleBoss alongside a real King",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: NoUnit, Role: RoleBoss}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			expectedTotalUnits: 4,
		},
		{
			name: "Failure: Player 1's first slot is NO_UNIT tagged RoleKing, real King is elsewhere",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: NoUnit, Role: RoleKing}, {Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			wantErr:    ErrMissingKing,
			wantPlayer: "Player 1",
		},
		{
			name: "Failure: Player 1's King-archetype first slot has no explicit Role",
			cfg: GameCfg{
				StagePreset: "Plain",
				P1Slots:     []TeamSlot{{Archetype: "King"}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
			},
			wantErr:    ErrMissingKing,
			wantPlayer: "Player 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gameState, err := initGameState(tt.cfg)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Expected error %v, got %v", tt.wantErr, err)
			}

			if tt.wantErr != nil {
				if !strings.Contains(err.Error(), tt.wantPlayer) {
					t.Errorf("Expected error to name %s, got %q", tt.wantPlayer, err.Error())
				}
				return
			}

			// Verify turn starts at 1
			if gameState.Turn != 1 {
				t.Errorf("Expected turn to start at 1, got %d", gameState.Turn)
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

				var expectedPosition Coordinate
				_, index := id.Decode()
				switch unit.Team {
				case 1:
					expectedPosition = preset.P1StartingPositions[index]
				case 2:
					expectedPosition = preset.P2StartingPositions[index]
				default:
					t.Errorf("Unit ID %d has invalid team %d", id, unit.Team)
				}
				if unit.Position != expectedPosition {
					t.Errorf("Expected Player %d unit ID %d to start at (%d,%d), got (%d,%d)", unit.Team, id, expectedPosition.X, expectedPosition.Y, unit.Position.X, unit.Position.Y)
				}

				tile := gameState.Grid[unit.Position.Y][unit.Position.X]
				if tile.OccupantType != OccupantUnit && tile.OccupantID != int64(unit.ID) {
					t.Errorf("Expected tile at (%d,%d) contains OccupantUnit with unit ID %d, got %#v", unit.Position.X, unit.Position.Y, unit.ID, tile)
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
		})
	}
}

func TestInitGameState_LayoutGridCompilation(t *testing.T) {
	tests := []struct {
		name         string
		customPreset StagePreset // mock sandbox layout for testing
		expectError  bool
	}{
		{
			name: "Success: Compile Diverse Terrain Matrix",
			customPreset: StagePreset{
				Name:   "Sandbox3x3",
				Width:  3,
				Height: 3,
				LayoutGrid: []string{
					"T.T", //
					".BB", //
					".LW", //
				},
			},
		},
		{
			name: "Failure: Extra Width Layout typo",
			customPreset: StagePreset{
				Name:   "BrokenWidth3x3",
				Width:  3,
				Height: 3,
				LayoutGrid: []string{
					"...",
					"....",
					"...",
				},
			},
			expectError: true,
		},
		{
			name: "Failure: Extra Height Layout typo",
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
			},
			expectError: true,
		},
		{
			name: "Failure: Invalid Token Symbol",
			customPreset: StagePreset{
				Name:   "InvalidToken3x3",
				Width:  3,
				Height: 3,
				LayoutGrid: []string{
					"...",
					".X.",
					"...",
				},
			},
			expectError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grid, err := compileGrid(tt.customPreset)

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

			for y, row := range grid {
				for x, tile := range row {
					if tile.Type != expectedMatrix[y][x] {
						t.Errorf("Expected terrain at (%d,%d) to be %v, got %v", x, y, expectedMatrix[y][x], tile.Type)
					}

					if tile.Type != TerrainPlain && (tile.OccupantType != OccupantNone || tile.OccupantID != 0) {
						t.Errorf("Expected tile at (%d,%d) to have no occupant, got type %v with ID %d", x, y, tile.OccupantType, tile.OccupantID)
					}
				}
			}
		})
	}
}

func TestInitGame_AllGood(t *testing.T) {
	gameCfg := GameCfg{
		StagePreset: "Plain",
		P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
		P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Bandit", Role: RoleNormal}},
	}
	match, err := InitGame(gameCfg)
	if err != nil {
		t.Fatalf("Expected game initialization to succeed, got error: %v", err)
	}
	if match.TrueState == match.WorkingState {
		t.Errorf("Expected TrueState and WorkingState to be different instances, but they are the same")
	}
	if !reflect.DeepEqual(match.GameCfg, gameCfg) {
		t.Errorf("Expected GameCfg to be preserved in Match, got %+v", match.GameCfg)
	}
}

func TestInitGame_ErrorConditions(t *testing.T) {
	invalidCfgs := GameCfg{
		StagePreset: "NonExistentStage",
		P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}},
		P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}},
	}
	_, err := InitGame(invalidCfgs)

	if err == nil {
		t.Fatalf("Expected game initialization to fail due to invalid config, but it succeeded")
	}

	expectedErrorMessage := "invalid stage preset: stage preset 'NonExistentStage' not found"
	if err.Error() != expectedErrorMessage {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrorMessage, err.Error())
	}
}

func TestGameStateDeepCopy_Isolation(t *testing.T) {
	original := &GameState{
		Turn:       1,
		ActiveTeam: 1,
		Grid:       [][]Tile{},
		Units:      make(map[UnitID]*Unit),
		Bombs:      make(map[BombID]*Bomb),
		SoftBlocks: make(map[int]*SoftBlock),
	}
	original.Grid = append(original.Grid, []Tile{{Type: TerrainPlain, OccupantType: OccupantNone, OccupantID: 0}})
	original.Units[1] = &Unit{ID: 1, Type: Archetype{Name: "King"}, Team: 1, Position: Coordinate{0, 0}, HP: 3}
	original.Bombs[1] = &Bomb{ID: 1, OwnerID: 1, Position: Coordinate{1, 1}, Range: 2, Countdown: 3}
	original.SoftBlocks[1] = &SoftBlock{ID: 1, Position: Coordinate{2, 2}}

	clone := original.DeepCopy()

	clone.Turn = 2
	clone.ActiveTeam = 2
	clone.TurnBombCounter = 10
	clone.Grid[0][0].Type = TerrainTower
	clone.Units[1].HP = 100
	clone.Units[1].Position = Coordinate{5, 5}
	clone.Bombs[1].Range = 10
	clone.SoftBlocks[1].Position = Coordinate{10, 10}

	if original.Turn == clone.Turn {
		t.Errorf("Expected original Turn to be unaffected by changes to clone, got %d", original.Turn)
	}
	if original.ActiveTeam == clone.ActiveTeam {
		t.Errorf("Expected original ActiveTeam to be unaffected by changes to clone, got %d", original.ActiveTeam)
	}
	if original.TurnBombCounter == clone.TurnBombCounter {
		t.Errorf("Expected original TurnBombCounter to be unaffected by changes to clone, got %d", original.TurnBombCounter)
	}
	if original.Grid[0][0].Type == clone.Grid[0][0].Type {
		t.Errorf("Expected original Grid tile to be unaffected by changes to clone, got %v", original.Grid[0][0].Type)
	}
	if original.Units[1].HP == clone.Units[1].HP {
		t.Errorf("Expected original unit HP to be unaffected by changes to clone, got %d", original.Units[1].HP)
	}
	if original.Units[1].Position == clone.Units[1].Position {
		t.Errorf("Expected original unit Position to be unaffected by changes to clone, got (%d,%d)", original.Units[1].Position.X, original.Units[1].Position.Y)
	}
	if original.Bombs[1].Range == clone.Bombs[1].Range {
		t.Errorf("Expected original bomb Range to be unaffected by changes to clone, got %d", original.Bombs[1].Range)
	}
	if original.SoftBlocks[1].Position == clone.SoftBlocks[1].Position {
		t.Errorf("Expected original soft block Position to be unaffected by changes to clone, got (%d,%d)", original.SoftBlocks[1].Position.X, original.SoftBlocks[1].Position.Y)
	}
}

func TestGameStateDeepCopy_Fidelity(t *testing.T) {
	original := &GameState{
		Turn:            7,
		InSuddenDeath:   true,
		ActiveTeam:      2,
		TurnBombCounter: 3,
		Grid:            [][]Tile{{{Type: TerrainTower, OccupantType: OccupantSoftBlock, OccupantID: 9}}},
		Units: map[UnitID]*Unit{
			1: {
				ID:           1,
				Type:         Archetype{Name: "King"},
				Position:     Coordinate{2, 3},
				Speed:        4,
				BombMaxRange: 5,
				BombMinRange: 1,
				BombPower:    2,
				MaxBombCount: 3,
				BombUsed:     1,
				Team:         2,
				HP:           2,
				Skills:       SkillNone,
				Role:         RoleKing,
				HasMoved:     true,
				HasUsedSkill: true,
			},
		},
		Bombs:        map[BombID]*Bomb{1: {ID: 1, OwnerID: 1, Position: Coordinate{1, 1}, Range: 2, Countdown: 3}},
		SoftBlocks:   map[int]*SoftBlock{1: {ID: 1, Position: Coordinate{2, 2}}},
		TurnCommands: []TurnCommand{NewMoveCommand(1, Coordinate{4, 4})},
	}

	clone := original.DeepCopy()

	if !reflect.DeepEqual(original, clone) {
		t.Errorf("Expected the clone to carry every field, got %+v, want %+v", clone, original)
	}
}

func TestGameStateDeepCopy_MemoryHandling(t *testing.T) {
	original := &GameState{
		Turn:       1,
		ActiveTeam: 1,
	}

	clone := original.DeepCopy()

	if original.Turn != clone.Turn {
		t.Errorf("Expected colone Turn to be the same as original, got %d", clone.Turn)
	}
	if clone.Grid == nil {
		t.Error("Allocation boundary breach: clone Grid was uninitialized or returned as nil.")
	}
	if clone.Bombs == nil {
		t.Error("Panic guard failure: original.Bombs was nil, but clone.Bombs failed to initialize into an active writable map.")
	}
	if clone.SoftBlocks == nil {
		t.Error("Panic guard failure: original.SoftBlocks was nil, but clone.SoftBlocks failed to initialize into an active writable map.")
	}
}
