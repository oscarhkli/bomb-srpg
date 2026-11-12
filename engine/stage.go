package engine

func (gs *GameState) isWithinBounds(pos Coordinate) bool {
	if len(gs.Grid) == 0 || len(gs.Grid[0]) == 0 {
		return false
	}
	return pos.X >= 0 && pos.X < len(gs.Grid[0]) && pos.Y >= 0 && pos.Y < len(gs.Grid)
}

func (gs *GameState) clearStageTile(pos Coordinate) {
	if !gs.isWithinBounds(pos) {
		return
	}

	gs.Grid[pos.Y][pos.X].OccupantType = OccupantNone
	gs.Grid[pos.Y][pos.X].OccupantID = 0
}

func (gs *GameState) UpdateStageOccupant(pos Coordinate, occupantType OccupantType, id int64) {
	if !gs.isWithinBounds(pos) {
		return
	}

	gs.Grid[pos.Y][pos.X].OccupantType = occupantType
	gs.Grid[pos.Y][pos.X].OccupantID = id
}

// deduceBombCountDown returns the countdown for a newly placed bomb.
func (gs *GameState) deduceBombCountDown() int {
	return bombDefaultCountDown
}
