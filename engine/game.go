package engine

import (
	"fmt"
	"slices"
)

const NoUnit string = "NO_UNIT"

const maxTurnsLimit = 100

func terrainToken() map[rune]TerrainType {
	return map[rune]TerrainType{
		'.': TerrainPlain,
		'B': TerrainBlock,
		'T': TerrainTower,
		'W': TerrainWater,
		'L': TerrainLava,
	}
}

// InitGame validates the config, builds the initial GameState, and returns a ready-to-play Match.
func InitGame(gameCfg GameCfg) (*Match, error) {
	gameState, err := initGameState(gameCfg)
	if err != nil {
		return nil, err
	}

	return &Match{
		TrueState:    gameState,
		WorkingState: gameState.DeepCopy(),
		GameCfg:      gameCfg,
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

	err = createUnits(units, grid, gameCfg.P1Slots, stagePreset.P1StartingPositions, Team1, gameCfg)
	if err != nil {
		return nil, err
	}
	err = createUnits(units, grid, gameCfg.P2Slots, stagePreset.P2StartingPositions, Team2, gameCfg)
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

// compileGrid parses a preset's LayoutGrid into a Tile matrix, checking it against the declared size.
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
			terrain, exists := terrainToken()[char]
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
		if t.Archetype != NoUnit && t.Role == role {
			return true
		}
	}
	return false
}

func hasExactlyOneAndFirstIsKing(team []TeamSlot) bool {
	if len(team) == 0 || team[0].Archetype == NoUnit || team[0].Role != RoleKing {
		return false
	}

	for i := 1; i < len(team); i++ {
		if team[i].Archetype != NoUnit && team[i].Role == RoleKing {
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
		if ts.Archetype == NoUnit {
			continue
		}
		archetype, exists := GetArchetype(ts.Archetype)
		if !exists {
			return fmt.Errorf("%w: archetype '%s' for Player %d not found", ErrUnknownArchetype, ts.Archetype, teamID)
		}
		id := NewUnitID(teamID, i)
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

// DeepCopy returns a copy of gs that shares no memory with it.
// Reference fields must be re-cloned here.
func (gs *GameState) DeepCopy() *GameState {
	if gs == nil {
		return nil
	}

	clone := *gs
	clone.Grid = gs.cloneGridSnapshot()
	clone.Units = cloneEntities(gs.Units)
	clone.Bombs = cloneEntities(gs.Bombs)
	clone.SoftBlocks = cloneEntities(gs.SoftBlocks)
	if gs.TurnCommands != nil {
		clone.TurnCommands = slices.Clone(gs.TurnCommands)
	}
	return &clone
}

// V must hold no reference fields.
func cloneEntities[K comparable, V any](src map[K]*V) map[K]*V {
	dst := make(map[K]*V, len(src))
	for id, entity := range src {
		if entity == nil {
			continue
		}
		c := *entity
		dst[id] = &c
	}
	return dst
}
