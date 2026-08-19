package engine

import (
	"fmt"
)

const NoUnit string = "NO_UNIT"

func terrainToken() map[byte]TerrainType {
	return map[byte]TerrainType{
		'.': TerrainPlain,
		'B': TerrainBlock,
		'T': TerrainTower,
		'W': TerrainWater,
		'L': TerrainLava,
	}
}

// InitGame validates the config, builds the initial GameState, and returns a ready-to-play Match.
// It enforces: 1-5 units per team, King as first unit, valid stage preset, and grid dimensions.
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

func teamSize(team []TeamSlot) int {
	count := 0
	for _, v := range team {
		if v.Archetype != NoUnit {
			count++
		}
	}
	return count
}

func initGameState(gameCfg GameCfg) (*GameState, error) {
	stagePreset, exists := GetStagePreset(gameCfg.StagePreset)
	if !exists {
		return nil, fmt.Errorf("%w: stage preset '%s' not found", ErrInvalidStagePreset, gameCfg.StagePreset)
	}

	if err := validateTeamComposition(gameCfg.P1Slots, "Player 1"); err != nil {
		return nil, err
	}
	if err := validateTeamComposition(gameCfg.P2Slots, "Player 2"); err != nil {
		return nil, err
	}

	grid, err := compileGrid(stagePreset)
	if err != nil {
		return nil, err
	}

	units := map[UnitID]*Unit{}

	err = createUnits(units, grid, gameCfg.P1Slots, stagePreset.P1StartingPositions, 1, gameCfg)
	if err != nil {
		return nil, err
	}
	err = createUnits(units, grid, gameCfg.P2Slots, stagePreset.P2StartingPositions, 2, gameCfg)
	if err != nil {
		return nil, err
	}

	softBlocks := make(map[int]*SoftBlock)
	for i, pos := range stagePreset.SoftBlocks {
		softBlocks[i] = &SoftBlock{
			ID:       i,
			Position: pos,
		}

		grid[pos.Y][pos.X].OccupantType = OccupantSoftBlock
		grid[pos.Y][pos.X].OccupantID = int64(i)
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

// compileGrid validates a stage preset's LayoutGrid against its declared dimensions
// and parses it into a Tile matrix.
func compileGrid(preset StagePreset) ([][]Tile, error) {
	if len(preset.LayoutGrid) != preset.Height {
		return nil, fmt.Errorf("%w: stage preset layout grid row count %d does not match specified height %d", ErrInvalidStageLayout, len(preset.LayoutGrid), preset.Height)
	}
	for y, row := range preset.LayoutGrid {
		if len(row) != preset.Width {
			return nil, fmt.Errorf("%w: stage preset layout grid row %d column count %d does not match specified width %d", ErrInvalidStageLayout, y, len(row), preset.Width)
		}
	}

	grid := make([][]Tile, preset.Height)
	for y, row := range preset.LayoutGrid {
		grid[y] = make([]Tile, preset.Width)
		for x, char := range row {
			terrain, exists := terrainToken()[byte(char)]
			if !exists {
				return nil, fmt.Errorf("%w: invalid terrain character '%c' at (%d, %d)", ErrInvalidTerrain, char, x, y)
			}
			grid[y][x] = Tile{
				Type: terrain,
			}
		}
	}
	return grid, nil
}

// validateTeamComposition enforces roster-shape rules for a single team:
// a Boss-role team is 1-5 units with no King, a King-role team is 2-5 units led by King.
func validateTeamComposition(slots []TeamSlot, playerLabel string) error {
	hasBoss := hasRole(slots, RoleBoss)
	if hasBoss && hasRole(slots, RoleKing) {
		return fmt.Errorf("%w: %s must have either Boss or King, got %v", ErrInvalidTeamFormation, playerLabel, slots)
	}

	minTeamSize := 2
	if hasBoss {
		minTeamSize = 1
	}
	if t := teamSize(slots); t < minTeamSize || t > 5 {
		return fmt.Errorf("%w: %s must have between %d and 5 units, got %d", ErrInvalidTeamSize, playerLabel, minTeamSize, t)
	}
	if !hasBoss && !hasExactlyOneAndFirstIsKing(slots) {
		return fmt.Errorf("%w: %s must have exactly one King as the first unit", ErrMissingKing, playerLabel)
	}
	return nil
}

func hasRole(team []TeamSlot, role UnitRole) bool {
	for _, t := range team {
		if t.Role == role {
			return true
		}
	}
	return false
}

func hasExactlyOneAndFirstIsKing(team []TeamSlot) bool {
	if len(team) == 0 || team[0].Role != RoleKing {
		return false
	}

	for i := 1; i < len(team); i++ {
		if team[i].Role == RoleKing {
			return false
		}
	}
	return true
}

func createUnits(
	units map[UnitID]*Unit,
	grid [][]Tile,
	teams []TeamSlot,
	startingPositions [5]Coordinate,
	teamID int,
	gameCfg GameCfg,
) error {
	for i, ts := range teams {
		if ts.Archetype == NoUnit { // allowing non-full team in a specific location
			continue
		}
		archetype, exists := GetArchetype(ts.Archetype)
		if !exists {
			return fmt.Errorf("%w: archetype '%s' for Player %d not found", ErrUnknownArchetype, ts.Archetype, teamID)
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
			Role:         ts.Role,
		}

		grid[startingPositions[i].Y][startingPositions[i].X].OccupantType = OccupantUnit
		grid[startingPositions[i].Y][startingPositions[i].X].OccupantID = int64(id)
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
		Turn:       gs.Turn,
		ActiveTeam: gs.ActiveTeam,
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
				Skills:       unit.Skills,
				Role:         unit.Role,
			}
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
