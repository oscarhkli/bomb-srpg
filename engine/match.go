package engine

import (
	"fmt"
	"math/rand"
)

func (m *Match) ResetTurn() {
	m.WorkingState = m.TrueState.DeepCopy()
	m.PlaybackLog = nil
}

func (m *Match) submitAction(gameEvent GameEvent) {
	m.PlaybackLog = append(m.PlaybackLog, gameEvent)

	if !m.GameCfg.AllowResetTurn {
		m.TrueState = m.WorkingState.DeepCopy()
	}
}

func (m *Match) Surrender(teamID int) []GameEvent {
	if teamID == 1 {
		m.WinnerTeamID = 2
	} else {
		m.WinnerTeamID = 1
	}

	m.PlaybackLog = nil
	m.WorkingState.TurnStartEvents = nil
	m.TrueState.TurnStartEvents = nil

	return []GameEvent{
		NewMatchEndedEvent(m.WinnerTeamID),
	}
}

func (m *Match) ApplyTurnCommand(cmd TurnCommand) ([]GameEvent, error) {
	switch cmd.Type {
	case TurnCmdMove:
		return m.commandMoveUnit(cmd.UnitID, cmd.Target)
	case TurnCmdPlaceBomb:
		return m.CommandPlaceBomb(cmd.UnitID, cmd.Target)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedCommand, cmd.Type)
	}
}

func (m *Match) commandMoveUnit(unitID UnitID, target Coordinate) ([]GameEvent, error) {
	gameEvent, err := m.WorkingState.MoveUnit(unitID, target)
	if err != nil {
		return nil, err
	}
	m.submitAction(gameEvent)
	return []GameEvent{gameEvent}, nil
}

func (gs *GameState) MoveUnit(unitID UnitID, target Coordinate) (GameEvent, error) {
	unit, err := gs.validateActiveUnit(unitID)
	if err != nil {
		return GameEvent{}, err
	}

	if unit.HasMoved {
		return GameEvent{}, fmt.Errorf("%w: unit %#x already moved this turn", ErrAlreadyMoved, unitID)
	}

	tiles := gs.FindReachableTiles(unit.Position, unit.NewMovementRule())

	if _, ok := tiles[target]; !ok {
		return GameEvent{}, ErrOutOfMoveRange
	}

	if err = gs.IsLandingLegal(target, OccupantUnit); err != nil {
		return GameEvent{}, fmt.Errorf("%w: %w", ErrInvalidLanding, err)
	}

	oldPos := unit.Position
	gs.clearStageTile(oldPos)
	gs.UpdateStageOccupant(target, OccupantUnit, int64(unitID))
	unit.Position = target
	unit.HasMoved = true

	return NewUnitMovedEvent(unitID, oldPos, target), nil
}

func (gs *GameState) validateActiveUnit(unitID UnitID) (*Unit, error) {
	unit, ok := gs.Units[unitID]
	if !ok {
		return nil, fmt.Errorf("%w: unit %#x does not exist", ErrUnitNotFound, unitID)
	}
	if unit.HP <= 0 {
		return nil, fmt.Errorf("%w: unit %#x is dead", ErrUnitDead, unitID)
	}
	if unit.Team != gs.ActiveTeam {
		return nil, fmt.Errorf("%w: unit %#x not active team", ErrNotActiveTeam, unitID)
	}
	if !gs.isWithinBounds(unit.Position) {
		return nil, fmt.Errorf("%w: unit %#x out of bounds", ErrOutOfBounds, unitID)
	}
	cell := gs.Grid[unit.Position.Y][unit.Position.X]
	if cell.OccupantType != OccupantUnit || cell.OccupantID != int64(unitID) {
		return nil, fmt.Errorf("%w: unit %#x desynced at %v", ErrDesynced, unitID, unit.Position)
	}
	return unit, nil
}

func (m *Match) CommandPlaceBomb(unitID UnitID, target Coordinate) ([]GameEvent, error) {
	gameEvent, err := m.WorkingState.PlaceBomb(unitID, target)
	if err != nil {
		return nil, err
	}
	m.submitAction(gameEvent)
	return []GameEvent{gameEvent}, nil
}

// PlaceBomb skips validation for SystemUnitID.
func (gs *GameState) PlaceBomb(unitID UnitID, target Coordinate) (GameEvent, error) {
	if unitID == SystemUnitID {
		return gs.dropBomb(unitID, target, bombDefaultPower), nil
	}

	unit, err := gs.validateBombPlacement(unitID, target)
	if err != nil {
		return GameEvent{}, err
	}

	unit.BombUsed++
	unit.HasUsedSkill = true

	return gs.dropBomb(unitID, target, unit.BombPower), nil
}

func (gs *GameState) validateBombPlacement(unitID UnitID, target Coordinate) (*Unit, error) {
	unit, err := gs.validateActiveUnit(unitID)
	if err != nil {
		return nil, err
	}

	if unit.HasUsedSkill {
		return nil, fmt.Errorf("%w: unit %#x already used skill this turn", ErrAlreadyUsedSkill, unitID)
	}

	if unit.BombUsed >= unit.MaxBombCount {
		return nil, fmt.Errorf("%w: unit %#x out of bombs", ErrOutOfBombs, unitID)
	}

	tiles := gs.FindReachableTiles(unit.Position, unit.newBombPlacementRule())

	if _, ok := tiles[target]; !ok {
		return nil, ErrOutOfBombRange
	}

	if err = gs.IsLandingLegal(target, OccupantBomb); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidLanding, err)
	}

	return unit, nil
}

func (gs *GameState) dropBomb(ownerID UnitID, target Coordinate, power int) GameEvent {
	gs.TurnBombCounter++
	bomb := &Bomb{
		ID:        NewBombID(gs.Turn, gs.TurnBombCounter, ownerID),
		OwnerID:   ownerID,
		Position:  target,
		Range:     power,
		Countdown: gs.deduceBombCountDown(),
	}
	gs.Bombs[bomb.ID] = bomb
	gs.UpdateStageOccupant(target, OccupantBomb, int64(bomb.ID))

	return newBombPlacedEvent(ownerID, bomb.ID, target, bomb.Range, bomb.Countdown)
}

func (gs *GameState) IsLandingLegal(target Coordinate, _ OccupantType) error {
	if !gs.isWithinBounds(target) {
		return fmt.Errorf("%w: coordinate %v out of bounds", ErrOutOfBounds, target)
	}

	tile := gs.Grid[target.Y][target.X]

	if tile.Type != TerrainPlain {
		return fmt.Errorf("%w: can only place on plain terrain, got %v", ErrInvalidLanding, tile.Type)
	}

	if tile.OccupantType != OccupantNone {
		return fmt.Errorf("%w: cell occupied by %v", ErrCellOccupied, tile.OccupantType)
	}

	return nil
}

// beginTurn runs once per turn, with PlaybackLog empty.
func (m *Match) beginTurn() {
	m.WorkingState.TurnStartEvents = nil

	if m.WinnerTeamID == 0 && m.evaluateVictoryConditions() == matchInProgress {
		if m.TrueState.Turn <= m.GameCfg.MaxTurns {
			m.WorkingState.InSuddenDeath = false
		} else {
			m.WorkingState.InSuddenDeath = true
			m.injectSuddenDeathHazards()
			m.WorkingState.TurnStartEvents = cloneGameEvents(m.PlaybackLog)
			m.PlaybackLog = nil
		}
	}

	m.TrueState = m.WorkingState.DeepCopy()
}

func (m *Match) injectSuddenDeathHazards() {
	var emptyTilePos []Coordinate
	for y, row := range m.WorkingState.Grid {
		for x, tile := range row {
			if tile.OccupantType == OccupantNone && tile.Type != TerrainBlock {
				emptyTilePos = append(emptyTilePos, Coordinate{x, y})
			}
		}
	}

	rand.Shuffle(len(emptyTilePos), func(i int, j int) {
		emptyTilePos[i], emptyTilePos[j] = emptyTilePos[j], emptyTilePos[i]
	})

	limit := min(len(emptyTilePos), SuddenDeathBombs)

	for _, target := range emptyTilePos[:limit] {
		m.submitAction(m.WorkingState.dropBomb(SystemUnitID, target, bombDefaultPower))
	}
}

// ResolveTurn commits the turn and begins the next, so hazards are already placed on return.
// Returns the events logged during planning and those produced by resolution; MatchEndedEvent lands in the latter.
func (m *Match) ResolveTurn() (planGameEvents, resolveTurnGameEvents []GameEvent) {
	planGameEvents = make([]GameEvent, len(m.PlaybackLog))
	copy(planGameEvents, m.PlaybackLog)
	m.PlaybackLog = nil

	m.resolveBombExplosionAndDamage()

	for _, u := range m.WorkingState.Units {
		u.HasMoved = false
		u.HasUsedSkill = false
	}
	m.WorkingState.TurnBombCounter = 0

	if m.WinnerTeamID == 0 {
		winner := m.evaluateVictoryConditions()

		if winner == 0 {
			m.WorkingState.Turn++
			m.WorkingState.ActiveTeam = ((m.WorkingState.Turn - 1) & 1) + 1
		} else {
			m.WinnerTeamID = winner
			m.PlaybackLog = append(m.PlaybackLog, NewMatchEndedEvent(winner))
		}
	}

	m.TrueState = m.WorkingState.DeepCopy()

	resolveTurnGameEvents = make([]GameEvent, len(m.PlaybackLog))
	copy(resolveTurnGameEvents, m.PlaybackLog)
	m.PlaybackLog = nil

	m.beginTurn()

	return planGameEvents, resolveTurnGameEvents
}

func (m *Match) resolveBombExplosionAndDamage() {
	m.PlaybackLog = append(m.PlaybackLog, m.WorkingState.ResolveBombExplosionAndDamage()...)
}

func (gs *GameState) ResolveBombExplosionAndDamage() []GameEvent {
	var gameEvents []GameEvent

	explosionQueue, ignitedBombs, bombCountdownUpdatedEvents := gs.tickCountdownsAndQueueFuses()
	gameEvents = append(gameEvents, bombCountdownUpdatedEvents...)

	frozenGrid := gs.cloneGridSnapshot()

	damagedUnits := make(map[UnitID]bool)
	destroyedSoftBlocks := make(map[int]bool)
	destroyedItems := make(map[int]bool)

	bombExplodedEvents := gs.processChainDetonations(explosionQueue, ignitedBombs, frozenGrid, damagedUnits, destroyedSoftBlocks, destroyedItems)
	gameEvents = append(gameEvents, bombExplodedEvents...)
	damagedEvents := gs.handleDelayedBatchDamage(damagedUnits, destroyedSoftBlocks)
	gameEvents = append(gameEvents, damagedEvents...)

	return gameEvents
}

func (gs *GameState) tickCountdownsAndQueueFuses() ([]BombID, map[BombID]bool, []GameEvent) {
	var queue []BombID
	ignited := make(map[BombID]bool)

	var gameEvents []GameEvent
	for id, bomb := range gs.Bombs {
		if bomb.Countdown < 0 {
			continue
		}

		bomb.Countdown--
		gameEvents = append(gameEvents, newBombCountdownUpdatedEvent(id, bomb.Position, bomb.Countdown))
		if bomb.Countdown == 0 {
			queue = append(queue, id)
			ignited[id] = true
		}
	}
	return queue, ignited, gameEvents
}

func (gs *GameState) processChainDetonations(
	explosionQueue []BombID,
	ignitedBombs map[BombID]bool,
	frozenGrid [][]Tile,
	damagedUnits map[UnitID]bool,
	destroyedSoftBlocks map[int]bool,
	destroyedItems map[int]bool,
) []GameEvent {
	var gameEvents []GameEvent
	for len(explosionQueue) > 0 {
		currBombID := explosionQueue[0]
		explosionQueue = explosionQueue[1:]

		currBomb, ok := gs.Bombs[currBombID]
		if !ok {
			continue
		}
		if owner, ok := gs.Units[currBomb.OwnerID]; ok {
			owner.BombUsed = max(owner.BombUsed-1, 0)
		}

		affectedTiles := gs.findReachableTilesOnSnapshot(currBomb.Position, frozenGrid, MovementRule{
			MaxSteps:              currBomb.Range,
			Pattern:               patternCardinal,
			PassPermissions:       PassUnits,
			StopOnNonUnitOccupant: true,
		})

		var affectedPos []Coordinate

		for pos := range affectedTiles {
			affectedPos = append(affectedPos, pos)

			tile := &gs.Grid[pos.Y][pos.X]
			switch tile.OccupantType {
			case OccupantBomb:
				nextBombID := tile.BombID()
				if gs.igniteBomb(nextBombID, ignitedBombs) {
					explosionQueue = append(explosionQueue, nextBombID)
				}
			case OccupantUnit:
				damagedUnits[tile.unitID()] = true
			case OccupantSoftBlock:
				destroyedSoftBlocks[int(tile.OccupantID)] = true
			case OccupantItem:
				destroyedItems[int(tile.OccupantID)] = true
			case OccupantNone:
			}
		}

		gs.clearStageTile(currBomb.Position)
		delete(gs.Bombs, currBombID)
		gameEvents = append(gameEvents, newBombExplodedEvent(currBombID, currBomb.Position, affectedPos))
	}
	return gameEvents
}

func (gs *GameState) igniteBomb(id BombID, ignitedBombs map[BombID]bool) bool {
	bomb, ok := gs.Bombs[id]
	if !ok || ignitedBombs[id] {
		return false
	}
	bomb.Countdown = 0
	ignitedBombs[id] = true
	return true
}

func (gs *GameState) cloneGridSnapshot() [][]Tile {
	frozenGrid := make([][]Tile, len(gs.Grid))
	for y := range gs.Grid {
		frozenGrid[y] = make([]Tile, len(gs.Grid[y]))
		copy(frozenGrid[y], gs.Grid[y])
	}
	return frozenGrid
}

func (gs *GameState) handleDelayedBatchDamage(
	damagedUnits map[UnitID]bool,
	destroyedSoftBlocks map[int]bool,
) []GameEvent {
	var gameEvents []GameEvent
	for unitID := range damagedUnits {
		unit, ok := gs.Units[unitID]
		if !ok {
			continue
		}

		unit.HP -= 1
		gameEvents = append(gameEvents, newUnitDamagedEvent(unitID, unit.Position, unit.HP))

		if unit.HP <= 0 {
			gs.clearStageTile(unit.Position)
			gameEvents = append(gameEvents, NewUnitDiedEvent(unitID, unit.Position))
		}
	}

	for softBlockID := range destroyedSoftBlocks {
		softBlock, ok := gs.SoftBlocks[softBlockID]
		if !ok {
			continue
		}

		gs.clearStageTile(softBlock.Position)
		delete(gs.SoftBlocks, softBlockID)
		gameEvents = append(gameEvents, newSoftBlockDestroyedEvent(softBlockID, softBlock.Position))
	}

	return gameEvents
}

type teamStatus struct {
	hasBoss, bossAlive, kingAlive, ordinaryAlive bool
}

func (t *teamStatus) countLiving(unit *Unit) {
	if unit.HP <= 0 {
		return
	}
	t.kingAlive = t.kingAlive || unit.Role == RoleKing
	t.bossAlive = t.bossAlive || unit.Role == RoleBoss
	t.ordinaryAlive = t.ordinaryAlive || (unit.Role != RoleKing && unit.Role != RoleBoss)
}

func (t *teamStatus) goal() bool {
	return (t.hasBoss && t.bossAlive) || (t.kingAlive && t.ordinaryAlive)
}

func (t *teamStatus) wiped() bool {
	return (t.hasBoss && !t.bossAlive) || (!t.hasBoss && !t.kingAlive && !t.ordinaryAlive)
}

// evaluateVictoryConditions returns matchInProgress, 1, 2, or matchDrawn; any team other than Team1 counts as Team2.
// Goal: a living Boss, or a living King plus a living Ordinary unit. Wiped: the Boss dead, or a Boss-less team with no living unit.
// Both at Goal play on; a team wins on Goal or a living King against a Wiped opponent; both or neither is a draw.
func (m *Match) evaluateVictoryConditions() int {
	var p1, p2 teamStatus

	for _, unit := range m.WorkingState.Units {
		team := &p2
		if unit.Team == Team1 {
			team = &p1
		}
		if unit.Role == RoleBoss && (unit.Team == Team1 || unit.Team == Team2) {
			team.hasBoss = true
		}

		team.countLiving(unit)
	}

	if p1.goal() && p2.goal() {
		return matchInProgress
	}

	p1Wins := p1.goal() || (p1.kingAlive && p2.wiped())
	p2Wins := p2.goal() || (p2.kingAlive && p1.wiped())

	switch {
	case p1Wins && p2Wins:
		return matchDrawn
	case p1Wins:
		return Team1
	case p2Wins:
		return Team2
	default:
		return matchDrawn
	}
}
