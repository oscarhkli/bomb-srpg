package engine

import "fmt"

var terrainToken = map[byte]TerrainType{
	'.': TerrainPlain,
	'H': TerrainBlock,
	'T': TerrainTower,
	'W': TerrainWater,
	'L': TerrainLava,
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

	units := map[int]*Unit{}

	err := createUnits(units, gameCfg.P1Teams, stagePreset.P1StartingPositions, 1, gameCfg)
	if err != nil {
		return nil, err
	}
	err = createUnits(units, gameCfg.P2Teams, stagePreset.P2StartingPositions, 2, gameCfg)
	if err != nil {
		return nil, err
	}

	grid := make([][]Cell, stagePreset.Height)
	for y, row := range stagePreset.LayoutGrid {
		grid[y] = make([]Cell, stagePreset.Width)
		for x, char := range row {
			terrain, exists := terrainToken[byte(char)]
			if !exists {
				return nil, fmt.Errorf("invalid terrain character '%c' at (%d, %d)", char, x, y)
			}
			grid[y][x] = Cell{
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
		Grid:       grid,
		Units:      units,
		Bombs:      make(map[int]*Bomb),
		SoftBlocks: softBlocks,
		Turn:       0,
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
	units map[int]*Unit,
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
		id := teamID<<3 | i // Player 1 units have IDs starting from 8, Player 2 units have IDs starting from 16
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
