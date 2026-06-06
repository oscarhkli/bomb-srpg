package engine

import (
	"fmt"
	"maps"
)

var terrainToken = map[byte]TerrainType{
	'.': TerrainPlain,
	'B': TerrainBlock,
	'T': TerrainTower,
	'W': TerrainWater,
	'L': TerrainLava,
}

func InitGame(gameCfg GameCfg) (*Match, error) {
	gameState, err := initGameState(gameCfg)
	if err != nil {
		return nil, err
	}

	return &Match{
		TrueState:    gameState,
		WorkingState: gameState.DeepCopy(),
		GameCfg:      gameCfg,
		PlaybackLog:  []GameEvent{},
	}, nil
}

func initGameState(gameCfg GameCfg) (*GameState, error) {
	stagePreset, exists := GetStagePreset(gameCfg.StagePreset)
	if !exists {
		return nil, fmt.Errorf("stage preset '%s' not found", gameCfg.StagePreset)
	}

	if len(gameCfg.P1Teams) < 1 || len(gameCfg.P1Teams) > 5 {
		return nil, fmt.Errorf("Player 1 must have between 1 and 5 units, got %d", len(gameCfg.P1Teams))
	}
	if len(gameCfg.P2Teams) < 1 || len(gameCfg.P2Teams) > 5 {
		return nil, fmt.Errorf("Player 2 must have between 1 and 5 units, got %d", len(gameCfg.P2Teams))
	}

	// Only 1 king each team allowed, and must be the first unit if present
	if !hasExactlyOneAndFirstIsKing(gameCfg.P1Teams) {
		return nil, fmt.Errorf("Player 1 must have exactly one King as the first unit")
	}
	if !hasExactlyOneAndFirstIsKing(gameCfg.P2Teams) {
		return nil, fmt.Errorf("Player 2 must have exactly one King as the first unit")
	}

	// verify stage layout dimensions match the specified width and height
	if len(stagePreset.LayoutGrid) != stagePreset.Height {
		return nil, fmt.Errorf("stage preset layout grid row count %d does not match specified height %d", len(stagePreset.LayoutGrid), stagePreset.Height)
	}
	for y, row := range stagePreset.LayoutGrid {
		if len(row) != stagePreset.Width {
			return nil, fmt.Errorf("stage preset layout grid row %d column count %d does not match specified width %d", y, len(row), stagePreset.Width)
		}
	}

	units := map[UnitID]*Unit{}

	err := createUnits(units, gameCfg.P1Teams, stagePreset.P1StartingPositions, 1, gameCfg)
	if err != nil {
		return nil, err
	}
	err = createUnits(units, gameCfg.P2Teams, stagePreset.P2StartingPositions, 2, gameCfg)
	if err != nil {
		return nil, err
	}

	grid := make([][]Tile, stagePreset.Height)
	for y, row := range stagePreset.LayoutGrid {
		grid[y] = make([]Tile, stagePreset.Width)
		for x, char := range row {
			terrain, exists := terrainToken[byte(char)]
			if !exists {
				return nil, fmt.Errorf("invalid terrain character '%c' at (%d, %d)", char, x, y)
			}
			grid[y][x] = Tile{
				Type: terrain,
			}
		}
	}

	softBlocks := make(map[int]*SoftBlock)
	for i, pos := range stagePreset.SoftBlocks {
		softBlocks[i] = &SoftBlock{
			ID:       i,
			Position: pos,
		}
	}

	return &GameState{
		Turn:       1,
		ActiveTeam: 1,
		Grid:       grid,
		Units:      units,
		Bombs:      make(map[BombID]*Bomb),
		SoftBlocks: softBlocks,
	}, nil
}

func hasExactlyOneAndFirstIsKing(team []string) bool {
	if len(team) == 0 || team[0] != "King" {
		return false
	}

	for i := 1; i < len(team); i++ {
		if team[i] == "King" {
			return false
		}
	}
	return true
}

func createUnits(
	units map[UnitID]*Unit,
	teams []string,
	startingPositions [5]Coordinate,
	teamID int,
	gameCfg GameCfg,
) error {
	for i, archetypeName := range teams {
		archetype, exists := GetArchetype(archetypeName)
		if !exists {
			return fmt.Errorf("archetype '%s' for Player %d not found", archetypeName, teamID)
		}
		id := NewUnitID(teamID, i) // Player 1 units have IDs starting from 8, Player 2 units have IDs starting from 16
		units[id] = &Unit{
			ID:           id,
			Type:         archetype,
			Position:     startingPositions[i],
			Speed:        applyGlobalOverride(archetype.BaseSpeed, gameCfg.GlobalSpeedOverride),
			BombMaxRange: applyGlobalOverride(archetype.BombMaxRange, gameCfg.GlobalBombMaxRangeOverride),
			BombMinRange: archetype.BombMinRange,
			BombPower:    archetype.BombPower,
			MaxBombCount: archetype.MaxBombCount,
			BombUsed:     0,
			Team:         teamID,
			HP:           archetype.BaseHP,
			Skills:       archetype.PresetSkills,
		}
	}
	return nil
}

func applyGlobalOverride(orig, newVal int) int {
	if newVal > 0 {
		return newVal
	}
	return orig
}

// DeepCopy creates a deep copy of the GameState.
// This is used to create an independent working state for planning stage, allowing player to reset to the original state if needed without affecting the true state.
func (gs *GameState) DeepCopy() *GameState {
	if gs == nil {
		return nil
	}

	clone := &GameState{
		Turn: gs.Turn,
	}

	if gs.Grid == nil {
		clone.Grid = make([][]Tile, 0)
	} else {
		clone.Grid = make([][]Tile, len(gs.Grid))
		for y := range gs.Grid {
			clone.Grid[y] = make([]Tile, len(gs.Grid[y]))
			copy(clone.Grid[y], gs.Grid[y])
		}
	}

	if gs.Units == nil {
		clone.Units = make(map[UnitID]*Unit)
	} else {
		clone.Units = make(map[UnitID]*Unit, len(gs.Units))
		for id, unit := range gs.Units {
			if unit == nil {
				continue
			}
			clone.Units[id] = &Unit{
				ID:           unit.ID,
				Type:         unit.Type, // Archetype is immutable, can share reference
				Position:     unit.Position,
				Speed:        unit.Speed,
				BombMaxRange: unit.BombMaxRange,
				BombMinRange: unit.BombMinRange,
				BombPower:    unit.BombPower,
				MaxBombCount: unit.MaxBombCount,
				BombUsed:     unit.BombUsed,
				Team:         unit.Team,
				HP:           unit.HP,
				Skills:       make(map[SkillType]bool),
			}
			maps.Copy(clone.Units[id].Skills, unit.Skills)
		}
	}

	if gs.Bombs == nil {
		clone.Bombs = make(map[BombID]*Bomb)
	} else {
		clone.Bombs = make(map[BombID]*Bomb, len(gs.Bombs))
		for id, bomb := range gs.Bombs {
			if bomb == nil {
				continue
			}
			clone.Bombs[id] = &Bomb{
				ID:        bomb.ID,
				OwnerID:   bomb.OwnerID,
				Position:  bomb.Position,
				Range:     bomb.Range,
				Countdown: bomb.Countdown,
			}
		}
	}

	if gs.SoftBlocks == nil {
		clone.SoftBlocks = make(map[int]*SoftBlock)
	} else {
		clone.SoftBlocks = make(map[int]*SoftBlock, len(gs.SoftBlocks))
		for id, sb := range gs.SoftBlocks {
			if sb == nil {
				continue
			}
			clone.SoftBlocks[id] = &SoftBlock{
				ID:       sb.ID,
				Position: sb.Position,
			}
		}
	}

	return clone
}
