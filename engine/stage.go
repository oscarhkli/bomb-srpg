package engine

// IsWithinBounds checks if the given coordinate is within the grid boundaries.
// Return false if the grid is empty or if the coordinate is out of bounds.
func (gs *GameState) IsWithinBounds(pos Coordinate) bool {
	if len(gs.Grid) == 0 || len(gs.Grid[0]) == 0 {
		return false
	}
	return pos.X >= 0 && pos.X < len(gs.Grid[0]) && pos.Y >= 0 && pos.Y < len(gs.Grid)
}
