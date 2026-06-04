package engine

// IsWithinBounds checks if the given coordinate is within the grid boundaries.
// Return false if the grid is empty or if the coordinate is out of bounds.
func (gs *GameState) IsWithinBounds(pos Coordinate) bool {
	if len(gs.Grid) == 0 || len(gs.Grid[0]) == 0 {
		return false
	}
	return pos.X >= 0 && pos.X < len(gs.Grid[0]) && pos.Y >= 0 && pos.Y < len(gs.Grid)
}

func (gs *GameState) ClearStageTile(pos Coordinate) {
	if !gs.IsWithinBounds(pos) {
		return
	}

	gs.Grid[pos.Y][pos.X].OccupantType = OccupantNone
	gs.Grid[pos.Y][pos.X].OccupantID = 0
}

func (gs *GameState) UpdateStageOccupant(pos Coordinate, occupantType OccupantType, id int64) {
	if !gs.IsWithinBounds(pos) {
		return
	}

	gs.Grid[pos.Y][pos.X].OccupantType = occupantType
	gs.Grid[pos.Y][pos.X].OccupantID = id
}

// DeduceBombCountDown inspects the target position and unit's skill and deduces the count down of the bomb.
// Since terrains and skills are in later phase, it always return 5 at the moment
func (gs *GameState) DeduceBombCountDown(pos Coordinate, unit *Unit) int {
	return BombDefaultCountDown
}
