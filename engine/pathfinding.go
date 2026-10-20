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

type pathNode struct {
	Pos  Coordinate
	Step int
	Dir  Coordinate
}

// stepTarget returns the tile one step from node along dir, or false when the rule forbids the
// move: a turn when CanTurn is false, or a target outside the grid.
func (gs *GameState) stepTarget(node pathNode, dir Coordinate, rule MovementRule) (Coordinate, bool) {
	if !rule.CanTurn && node.Step > 0 && dir != node.Dir {
		return Coordinate{}, false
	}
	next := Coordinate{node.Pos.X + dir.X, node.Pos.Y + dir.Y}
	return next, gs.IsWithinBounds(next)
}

func (gs *GameState) findReachableTiles(startPos Coordinate, rule MovementRule, grid [][]Tile) map[Coordinate]int {
	if len(grid) == 0 || len(grid[0]) == 0 {
		return make(map[Coordinate]int)
	}

	steps := map[Coordinate]int{startPos: 0}
	queue := []pathNode{{Pos: startPos}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if rule.MaxSteps != -1 && current.Step >= rule.MaxSteps {
			continue
		}
		queue = gs.expandNode(queue, current, rule, grid, steps)
	}

	return steps
}

// expandNode records the tiles one step from node in steps and appends those the search continues from to queue.
// The queue visits tiles by ascending step, so the first count recorded for a tile is its shortest.
func (gs *GameState) expandNode(queue []pathNode, node pathNode, rule MovementRule, grid [][]Tile, steps map[Coordinate]int) []pathNode {
	dirs := [4]Coordinate{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
	nextStep := node.Step + 1

	for _, dir := range dirs {
		nextPos, ok := gs.stepTarget(node, dir, rule)
		if !ok {
			continue
		}

		canPass, canLand := rule.CheckPassability(grid[nextPos.Y][nextPos.X])
		if !canPass && !canLand {
			continue
		}

		oldSteps, seen := steps[nextPos]
		if seen && (canLand || oldSteps <= nextStep) {
			continue
		}

		steps[nextPos] = nextStep
		if canPass {
			queue = append(queue, pathNode{Pos: nextPos, Step: nextStep, Dir: dir})
		}
	}

	return queue
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

	var permissionFlag PassFlag
	switch tile.OccupantType {
	case OccupantNone, OccupantUnit:
		return true, false
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
