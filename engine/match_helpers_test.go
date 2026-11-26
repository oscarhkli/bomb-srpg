package engine

import (
	"testing"
)

func eventIs[T GameEvent](evt GameEvent) bool {
	_, ok := evt.(T)
	return ok
}

func newTestMatch(width, height int) *Match {
	grid := make([][]Tile, height)
	for y, row := range grid {
		grid[y] = make([]Tile, width)
		for x := range row {
			grid[y][x] = Tile{Type: TerrainPlain, OccupantType: OccupantNone}
		}
	}

	m := &Match{
		GameCfg: GameCfg{MaxTurns: 100},
		WorkingState: &GameState{
			Turn:            1,
			TurnBombCounter: 0,
			Grid:            grid,
			Units:           make(map[UnitID]*Unit),
			Bombs:           make(map[BombID]*Bomb),
			SoftBlocks:      make(map[int]*SoftBlock),
		},
		PlaybackLog: []GameEvent{},
	}
	m.TrueState = m.WorkingState.DeepCopy()

	return m
}

func newActiveMatch(width, height int) *Match {
	m := newTestMatch(width, height)
	m.WorkingState.ActiveTeam = 1
	return m
}

func mustArchetype(t *testing.T, name string) Archetype {
	t.Helper()

	archetype, ok := GetArchetype(name)
	if !ok {
		t.Fatalf("archetype %q does not exist", name)
	}
	return archetype
}

func (m *Match) addUnit(team, idx int, archetype Archetype, role UnitRole, pos Coordinate, hp int) {
	id := NewUnitID(team, idx)
	u := &Unit{
		ID:           id,
		Type:         archetype,
		Position:     pos,
		Speed:        archetype.BaseSpeed,
		BombMaxRange: archetype.BombMaxRange,
		BombMinRange: archetype.BombMinRange,
		BombPower:    archetype.BombPower,
		MaxBombCount: archetype.MaxBombCount,
		BombUsed:     0,
		Team:         team,
		HP:           hp,
		Skills:       archetype.PresetSkills,
		Role:         role,
	}
	m.WorkingState.Units[id] = u
	m.WorkingState.Grid[pos.Y][pos.X] = Tile{
		Type:         TerrainPlain,
		OccupantType: OccupantUnit,
		OccupantID:   int64(id),
	}
}

func (m *Match) addBomb(counter int, owner UnitID, pos Coordinate, range_, cd int) {
	b := &Bomb{
		ID:        NewBombID(1, counter, owner),
		OwnerID:   owner,
		Position:  pos,
		Range:     range_,
		Countdown: cd,
	}
	m.WorkingState.Bombs[b.ID] = b
	m.WorkingState.Grid[pos.Y][pos.X] = Tile{
		Type:         TerrainPlain,
		OccupantType: OccupantBomb,
		OccupantID:   int64(b.ID),
	}
}

func (m *Match) addSoftBlock(id int, pos Coordinate) *SoftBlock {
	sb := &SoftBlock{ID: id, Position: pos}
	m.WorkingState.SoftBlocks[id] = sb
	m.WorkingState.Grid[pos.Y][pos.X] = Tile{
		Type:         TerrainPlain,
		OccupantType: OccupantSoftBlock,
		OccupantID:   int64(id),
	}
	return sb
}

// teamFormation describes one team's roster; a nil King selects a Boss-type team built from BossesAlive.
type teamFormation struct {
	King               *bool
	BossesAlive        []bool
	OrdinaryUnitsAlive []bool
}

func newTestMatchWithTeams(t *testing.T, p1, p2 teamFormation) *Match {
	t.Helper()

	m := newActiveMatch(6, 2)

	king := mustArchetype(t, "King")

	boss := mustArchetype(t, "Prologue")

	fighter := mustArchetype(t, "Fighter")

	hpOf := func(alive bool) int {
		if alive {
			return 1
		}
		return 0
	}

	buildTeam := func(team, row int, tf teamFormation) {
		x := 0
		if tf.King != nil {
			m.addUnit(team, x, king, RoleKing, Coordinate{x, row}, hpOf(*tf.King))
			x++
		} else {
			for _, alive := range tf.BossesAlive {
				m.addUnit(team, x, boss, RoleBoss, Coordinate{x, row}, hpOf(alive))
				x++
			}
		}
		for _, alive := range tf.OrdinaryUnitsAlive {
			m.addUnit(team, x, fighter, RoleNormal, Coordinate{x, row}, hpOf(alive))
			x++
		}
	}

	buildTeam(1, 0, p1)
	buildTeam(2, 1, p2)

	m.TrueState = m.WorkingState.DeepCopy()

	return m
}

func newCloneFixture(t *testing.T) *Match {
	t.Helper()
	m, err := InitGame(GameCfg{
		StagePreset: "Plain",
		P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
		P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Witch", Role: RoleNormal}},
		MaxTurns:    10,
	})
	if err != nil {
		t.Fatalf("InitGame() error = %v", err)
	}
	m.PlaybackLog = []GameEvent{BombExploded{BombID: 1, Position: Coordinate{X: 1, Y: 2}, AffectedPositions: []Coordinate{{X: 7, Y: 7}}}}
	for _, gs := range []*GameState{m.TrueState, m.WorkingState} {
		gs.Bombs[900] = &Bomb{ID: 900, OwnerID: 1, Position: Coordinate{X: 3, Y: 3}, Range: 2, Countdown: 1}
		gs.SoftBlocks[900] = &SoftBlock{ID: 900, Position: Coordinate{X: 4, Y: 4}}
		gs.TurnCommands = []TurnCommand{NewMoveCommand(1, Coordinate{X: 2, Y: 2})}
		gs.TurnStartEvents = []GameEvent{BombExploded{BombID: 2, Position: Coordinate{X: 5, Y: 5}, AffectedPositions: []Coordinate{{X: 6, Y: 6}}}}
	}
	return m
}
