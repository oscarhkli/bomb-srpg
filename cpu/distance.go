package cpu

import "bomb-srpg/engine"

// reachDist returns the walking distance from unit to target.
// Returns -1 if target is unreachable.
func reachDist(gs *engine.GameState, unit *engine.Unit, target engine.Coordinate) int {
	rule := unit.NewMovementRule()
	rule.MaxSteps = -1
	rule.CanTurn = true

	if d, ok := gs.FindReachableTiles(unit.Position, rule)[target]; ok {
		return d
	}
	return -1
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
