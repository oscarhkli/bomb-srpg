package cpu

import "bomb-srpg/engine"

func newTestGameState(width, height int) *engine.GameState {
	grid := make([][]engine.Tile, height)
	for y := range grid {
		grid[y] = make([]engine.Tile, width)
		for x := range grid[y] {
			grid[y][x] = engine.Tile{Type: engine.TerrainPlain}
		}
	}
	return &engine.GameState{
		ActiveTeam: 1,
		Grid:       grid,
		Units:      make(map[engine.UnitID]*engine.Unit),
		Bombs:      make(map[engine.BombID]*engine.Bomb),
		SoftBlocks: make(map[int]*engine.SoftBlock),
	}
}

func addUnit(gs *engine.GameState, id engine.UnitID, pos engine.Coordinate, archetypeName string, role engine.UnitRole) *engine.Unit {
	archetype, _ := engine.GetArchetype(archetypeName)
	u := &engine.Unit{
		ID:           id,
		Type:         archetype,
		Position:     pos,
		Speed:        archetype.BaseSpeed,
		BombMaxRange: archetype.BombMaxRange,
		BombMinRange: archetype.BombMinRange,
		BombPower:    archetype.BombPower,
		MaxBombCount: archetype.MaxBombCount,
		Team:         1,
		HP:           archetype.BaseHP,
		Role:         role,
	}
	gs.Units[id] = u
	gs.Grid[pos.Y][pos.X] = engine.Tile{Type: engine.TerrainPlain, OccupantType: engine.OccupantUnit, OccupantID: int64(id)}
	return u
}

func addFighter(gs *engine.GameState, id engine.UnitID, pos engine.Coordinate) *engine.Unit {
	return addUnit(gs, id, pos, "Fighter", engine.RoleNormal)
}

func addKing(gs *engine.GameState, id engine.UnitID, pos engine.Coordinate) *engine.Unit {
	return addUnit(gs, id, pos, "King", engine.RoleKing)
}

func setTerrainBlock(gs *engine.GameState, pos engine.Coordinate) {
	gs.Grid[pos.Y][pos.X] = engine.Tile{Type: engine.TerrainBlock}
}

func addSoftBlock(gs *engine.GameState, id int, pos engine.Coordinate) {
	gs.SoftBlocks[id] = &engine.SoftBlock{ID: id, Position: pos}
	gs.Grid[pos.Y][pos.X] = engine.Tile{Type: engine.TerrainPlain, OccupantType: engine.OccupantSoftBlock, OccupantID: int64(id)}
}

func addBomb(gs *engine.GameState, id engine.BombID, pos engine.Coordinate) {
	gs.Bombs[id] = &engine.Bomb{ID: id, Position: pos}
	gs.Grid[pos.Y][pos.X] = engine.Tile{Type: engine.TerrainPlain, OccupantType: engine.OccupantBomb, OccupantID: int64(id)}
}

// crossOffsets returns the straight cardinal-line offsets reachable up to radius steps
// in each of the 4 directions (movement/bomb rules in this repo don't allow turning mid-path).
func crossOffsets(radius int) []engine.Coordinate {
	dirs := []engine.Coordinate{{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0}}
	var offsets []engine.Coordinate
	for _, dir := range dirs {
		for step := 1; step <= radius; step++ {
			offsets = append(offsets, engine.Coordinate{X: dir.X * step, Y: dir.Y * step})
		}
	}
	return offsets
}
