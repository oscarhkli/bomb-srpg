package engine

// FindReachableTiles runs standard pathfinding using the live, active game grid state.
func (gs *GameState) FindReachableTiles(start Coordinate, rule MovementRule) map[Coordinate]int {
	return gs.findReachableTiles(start, rule, gs.Grid)
}

// FindReachableTilesOnSnapshot runs pathfinding using a frozen, read-only grid snapshot matrix.
// This is what allows overlapping explosions to evaluate line-of-sight blocks correctly.
func (gs *GameState) FindReachableTilesOnSnapshot(start Coordinate, snapshot [][]Tile, rule MovementRule) map[Coordinate]int {
	return gs.findReachableTiles(start, rule, snapshot)
}

// findReachableTiles uses a breadth-first search to find all tiles reachable from the starting position.
func (gs *GameState) findReachableTiles(startPos Coordinate, rule MovementRule, grid [][]Tile) map[Coordinate]int {
	if len(grid) == 0 || len(grid[0]) == 0 {
		return make(map[Coordinate]int)
	}

	type QueueItem struct {
		Pos  Coordinate
		Step int
		Dir  Coordinate // used when working on straight-line movement
	}

	steps := map[Coordinate]int{startPos: 0}
	queue := []QueueItem{{Pos: startPos, Step: 0}}
	dirs := []Coordinate{{0, -1}, {0, 1}, {-1, 0}, {1, 0}} // Up, Down, Left, Right

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if rule.MaxSteps != -1 && current.Step >= rule.MaxSteps {
			continue
		}

		for _, dir := range dirs {
			// Straight-line validation
			if !rule.CanTurn && current.Step > 0 && dir != current.Dir {
				continue
			}

			nextPos := Coordinate{current.Pos.X + dir.X, current.Pos.Y + dir.Y}
			nextStep := current.Step + 1

			if !gs.IsWithinBounds(nextPos) {
				continue
			}

			canPass, canLand := rule.CheckPassability(grid[nextPos.Y][nextPos.X])
			// can't pass and can't land means we skip this tile entirely
			if !canPass && !canLand {
				continue
			}

			// can land but can't pass: identify as reachable then stop exploration
			if canLand {
				if oldSteps, ok := steps[nextPos]; !ok || nextStep < oldSteps {
					steps[nextPos] = nextStep
				}
				continue
			}

			// short-circuit if we've already found a shorter or equal path to this tile
			if oldSteps, ok := steps[nextPos]; ok && oldSteps <= nextStep {
				continue
			}

			steps[nextPos] = nextStep
			queue = append(queue, QueueItem{Pos: nextPos, Step: nextStep, Dir: dir})
		}
	}

	return steps
}

// CheckPassability evaluates how a movement rule interacts with a tile's terrain and occupant.
// It returns:
// - canPass: true if the pathfinder can step through this tile and continue moving.
// - canLand: true if the pathfinder can step onto this tile but must stop immediately.
// Either one can be true, both can be false, but they cannot both be true at the same time.
// canPass = true, canLand = false - it's an open tile that can be moved through.
// canPass = false, canLand = true - it's a tile that can be stepped on but not passed through.
// canPass = false, canLand = false - it's a tile that cannot be stepped on or passed through.
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
// Currently restricted to simple Phase 1 walking rules.
func (u Unit) NewMovementRule() MovementRule {
	return MovementRule{
		MaxSteps:        u.Speed,
		Pattern:         PatternCardinal,
		PassPermissions: PassItems,
	}
}

// NewBombPlacementRule builds a snapshot configuration for a unit's bomb placement action.
// It covers almost all situation, unless we want to add MinStep in far future
func (u Unit) NewBombPlacementRule() MovementRule {
	return MovementRule{
		MaxSteps: u.BombMaxRange,
		Pattern:  PatternCardinal,
		// All pass, but can't land on any occupant. Landing is handled in other place.
		PassPermissions: PassUnits | PassSoftBlocks | PassHardBlocks | PassItems | PassBombs,
	}
}
