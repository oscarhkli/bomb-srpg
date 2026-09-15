package cpu

import "bomb-srpg/engine"

// reachDistToUnit returns the walking distance from fromPos to target's tile.
// target's own tile is never reachable, so this takes the shortest distance to one of its four neighbors and adds 1.
// Returns -1 if none are reachable.
func reachDistToUnit(gs *engine.GameState, unit *engine.Unit, fromPos engine.Coordinate, target *engine.Unit) int {
	rule := unit.NewMovementRule()
	rule.MaxSteps = -1
	rule.CanTurn = true
	rule.PassPermissions |= engine.PassBombs

	reachable := gs.FindReachableTiles(fromPos, rule)
	dirs := []engine.Coordinate{{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0}}
	best := -1
	for _, dir := range dirs {
		adj := engine.Coordinate{X: target.Position.X + dir.X, Y: target.Position.Y + dir.Y}
		if d, ok := reachable[adj]; ok && (best == -1 || d < best) {
			best = d
		}
	}
	if best == -1 {
		return -1
	}
	return best + 1
}

// nearestAffectedDist returns the walking distance from unit to the nearest tile in affectedTiles.
// Returns -1 if affectedTiles is empty or none of its tiles are reachable.
func nearestAffectedDist(gs *engine.GameState, unit *engine.Unit, affectedTiles map[engine.Coordinate]struct{}) int {
	if len(affectedTiles) == 0 {
		return -1
	}

	rule := unit.NewMovementRule()
	rule.MaxSteps = -1
	rule.CanTurn = true
	rule.PassPermissions |= engine.PassUnits

	reachableTiles := gs.FindReachableTiles(unit.Position, rule)
	nearest := -1
	for tile := range affectedTiles {
		d, ok := reachableTiles[tile]
		if !ok {
			continue
		}
		if nearest == -1 || d < nearest {
			nearest = d
		}
	}
	return nearest
}
