package engine

import "fmt"

// FindReachableTiles runs standard pathfinding using the live, active game grid state.
func (gs *GameState) FindReachableTiles(start Coordinate, rule MovementRule) map[Coordinate]int {
	return gs.findReachableTiles(start, rule, gs.Grid)
}

// FindReachableTilesOnSnapshot runs pathfinding using a frozen, read-only grid snapshot matrix.
// The frozen snapshot lets overlapping explosions evaluate line-of-sight blocks correctly.
func (gs *GameState) FindReachableTilesOnSnapshot(start Coordinate, snapshot [][]Tile, rule MovementRule) map[Coordinate]int {
	return gs.findReachableTiles(start, rule, snapshot)
}

func (gs *GameState) findReachableTiles(startPos Coordinate, rule MovementRule, grid [][]Tile) map[Coordinate]int {
	if len(grid) == 0 || len(grid[0]) == 0 {
		return make(map[Coordinate]int)
	}

	type QueueItem struct {
		Pos  Coordinate
		Step int
		Dir  Coordinate
	}

	steps := map[Coordinate]int{startPos: 0}
	queue := []QueueItem{{Pos: startPos, Step: 0}}
	dirs := []Coordinate{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if rule.MaxSteps != -1 && current.Step >= rule.MaxSteps {
			continue
		}

		for _, dir := range dirs {
			if !rule.CanTurn && current.Step > 0 && dir != current.Dir {
				continue
			}

			nextPos := Coordinate{current.Pos.X + dir.X, current.Pos.Y + dir.Y}
			nextStep := current.Step + 1

			if !gs.IsWithinBounds(nextPos) {
				continue
			}

			canPass, canLand := rule.CheckPassability(grid[nextPos.Y][nextPos.X])
			if !canPass && !canLand {
				continue
			}

			if canLand {
				if oldSteps, ok := steps[nextPos]; !ok || nextStep < oldSteps {
					steps[nextPos] = nextStep
				}
				continue
			}

			if oldSteps, ok := steps[nextPos]; ok && oldSteps <= nextStep {
				continue
			}

			steps[nextPos] = nextStep
			queue = append(queue, QueueItem{Pos: nextPos, Step: nextStep, Dir: dir})
		}
	}

	return steps
}

// CheckPassability reports how the rule treats the tile: canPass = can walk through it;
// canLand = can step on it but must stop; both false = blocked.
// Both true never occurs.
func (mr MovementRule) CheckPassability(tile Tile) (canPass bool, canLand bool) {
	if tile.Type == TerrainBlock && (mr.PassPermissions&PassHardBlocks == 0) {
		return false, false
	}

	if tile.OccupantType == OccupantUnit && (mr.PassPermissions&PassUnits == 0) {
		return false, false
	}

	if tile.OccupantType == OccupantNone || tile.OccupantType == OccupantUnit {
		return true, false
	}

	var permissionFlag PassFlag
	switch tile.OccupantType {
	case OccupantBomb:
		permissionFlag = PassBombs
	case OccupantSoftBlock:
		permissionFlag = PassSoftBlocks
	case OccupantItem:
		permissionFlag = PassItems
	}

	if mr.PassPermissions&permissionFlag != 0 {
		return true, false
	}

	if mr.StopOnNonUnitOccupant {
		return false, true
	}

	return false, false
}

// NewMovementRule builds a snapshot configuration for a unit's movement action.
func (u Unit) NewMovementRule() MovementRule {
	return MovementRule{
		MaxSteps:        u.Speed,
		Pattern:         PatternCardinal,
		PassPermissions: PassItems,
	}
}

// NewBombPlacementRule builds a snapshot configuration for a unit's bomb placement action.
func (u Unit) NewBombPlacementRule() MovementRule {
	return MovementRule{
		MaxSteps:        u.BombMaxRange,
		Pattern:         PatternCardinal,
		PassPermissions: PassUnits | PassSoftBlocks | PassHardBlocks | PassItems | PassBombs,
	}
}

// FindAllowedTiles deduces the tiles that an occupant can reach and land on.
// It returns coordinates with distance from the start.
func (gs *GameState) FindAllowedTiles(start Coordinate, rule MovementRule, occupantType OccupantType) map[Coordinate]int {
	reachable := gs.FindReachableTiles(start, rule)
	allowed := make(map[Coordinate]int)
	for pos, steps := range reachable {
		if gs.IsLandingLegal(pos, occupantType) != nil {
			continue
		}
		allowed[pos] = steps
	}
	return allowed
}

// FindAllowedTilesForCommand returns the tiles a unit can use for the given command type.
func (gs *GameState) FindAllowedTilesForCommand(unitID UnitID, turnCmdType TurnCmdType) (map[Coordinate]int, error) {
	unit, err := gs.findUnit(unitID, true)
	if err != nil {
		return nil, err
	}

	switch turnCmdType {
	case TurnCmdMove:
		return gs.FindAllowedTiles(unit.Position, unit.NewMovementRule(), OccupantUnit), nil
	case TurnCmdPlaceBomb:
		return gs.FindAllowedTiles(unit.Position, unit.NewBombPlacementRule(), OccupantBomb), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedCommand, turnCmdType)
	}
}

func (gs *GameState) findUnit(unitID UnitID, isAlive bool) (*Unit, error) {
	unit, ok := gs.Units[unitID]
	if !ok {
		return nil, fmt.Errorf("%w: unit %#x", ErrUnitNotFound, unitID)
	}
	if isAlive && unit.HP <= 0 {
		return nil, fmt.Errorf("%w: unit %#x", ErrUnitDead, unitID)
	}
	return unit, nil
}
