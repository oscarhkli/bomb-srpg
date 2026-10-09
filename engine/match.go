package engine

import (
	"fmt"
	"math/rand"
)

// ResetTurn discards WorkingState and PlaybackLog, restoring a copy of TrueState.
func (m *Match) ResetTurn() {
	m.WorkingState = m.TrueState.DeepCopy()
	m.PlaybackLog = nil
}

// SubmitAction logs gameEvent; without AllowResetTurn it also commits WorkingState to TrueState.
func (m *Match) SubmitAction(gameEvent GameEvent) {
	m.PlaybackLog = append(m.PlaybackLog, gameEvent)

	if !m.GameCfg.AllowResetTurn {
		m.TrueState = m.WorkingState.DeepCopy()
	}
}

// Surrender ends the match by setting the winner and sending MatchEndedEvent.
func (m *Match) Surrender(teamID int) []GameEvent {
	if teamID == 1 {
		m.WinnerTeamID = 2
	} else {
		m.WinnerTeamID = 1
	}

	m.PlaybackLog = nil

	return []GameEvent{
		NewMatchEndedEvent(m.WinnerTeamID),
	}
}

// ApplyTurnCommand accepts any packaged action and forwards it to the true match logic.
func (m *Match) ApplyTurnCommand(cmd TurnCommand) ([]GameEvent, error) {
	switch cmd.Type {
	case TurnCmdMove:
		return m.CommandMoveUnit(cmd.UnitID, cmd.Target)
	case TurnCmdPlaceBomb:
		return m.CommandPlaceBomb(cmd.UnitID, cmd.Target)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedCommand, cmd.Type)
	}
}

// CommandMoveUnit applies MoveUnit to WorkingState and logs the resulting event.
func (m *Match) CommandMoveUnit(unitID UnitID, target Coordinate) ([]GameEvent, error) {
	gameEvent, err := m.WorkingState.MoveUnit(unitID, target)
	if err != nil {
		return nil, err
	}
	m.SubmitAction(gameEvent)
	return []GameEvent{gameEvent}, nil
}

// MoveUnit relocates a unit within its move range and returns the UnitMovedEvent.
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
	gs.ClearStageTile(oldPos)
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
	if !gs.IsWithinBounds(unit.Position) {
		return nil, fmt.Errorf("%w: unit %#x out of bounds", ErrOutOfBounds, unitID)
	}
	cell := gs.Grid[unit.Position.Y][unit.Position.X]
	if cell.OccupantType != OccupantUnit || cell.OccupantID != int64(unitID) {
		return nil, fmt.Errorf("%w: unit %#x desynced at %v", ErrDesynced, unitID, unit.Position)
	}
	return unit, nil
}

// CommandPlaceBomb applies PlaceBomb to WorkingState and logs the resulting event.
func (m *Match) CommandPlaceBomb(unitID UnitID, target Coordinate) ([]GameEvent, error) {
	gameEvent, err := m.WorkingState.PlaceBomb(unitID, target)
	if err != nil {
		return nil, err
	}
	m.SubmitAction(gameEvent)
	return []GameEvent{gameEvent}, nil
}

// PlaceBomb drops a bomb for the unit and returns the BombPlacedEvent. SystemUnitID skips validation.
// Otherwise it returns an error if the unit has no bombs or skill use left, the target is out of
// bomb range, or the cell is not landable.
func (gs *GameState) PlaceBomb(unitID UnitID, target Coordinate) (GameEvent, error) {
	bombPower := BombDefaultPower
	var unit *Unit

	if unitID != SystemUnitID {
		var err error
		unit, err = gs.validateActiveUnit(unitID)
		if err != nil {
			return GameEvent{}, err
		}

		if unit.HasUsedSkill {
			return GameEvent{}, fmt.Errorf("%w: unit %#x already used skill this turn", ErrAlreadyUsedSkill, unitID)
		}

		if unit.BombUsed >= unit.MaxBombCount {
			return GameEvent{}, fmt.Errorf("%w: unit %#x out of bombs", ErrOutOfBombs, unitID)
		}

		tiles := gs.FindReachableTiles(unit.Position, unit.NewBombPlacementRule())

		if _, ok := tiles[target]; !ok {
			return GameEvent{}, ErrOutOfBombRange
		}

		if err = gs.IsLandingLegal(target, OccupantBomb); err != nil {
			return GameEvent{}, fmt.Errorf("%w: %w", ErrInvalidLanding, err)
		}

		bombPower = unit.BombPower
	}

	if unit != nil {
		unit.BombUsed++
		unit.HasUsedSkill = true
	}

	return gs.dropBomb(unitID, target, bombPower), nil
}

func (gs *GameState) dropBomb(ownerID UnitID, target Coordinate, power int) GameEvent {
	gs.TurnBombCounter++
	bomb := &Bomb{
		ID:        NewBombID(gs.Turn, gs.TurnBombCounter, ownerID),
		OwnerID:   ownerID,
		Position:  target,
		Range:     power,
		Countdown: gs.DeduceBombCountDown(target),
	}
	gs.Bombs[bomb.ID] = bomb
	gs.UpdateStageOccupant(target, OccupantBomb, int64(bomb.ID))

	return NewBombPlacedEvent(ownerID, bomb.ID, target, bomb.Range, bomb.Countdown)
}

// IsLandingLegal checks if the target is legal to be landed by a certain occupantType.
func (gs *GameState) IsLandingLegal(target Coordinate, occupantType OccupantType) error {
	if !gs.IsWithinBounds(target) {
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

// StartTurn sets up the environmental boundaries for the upcoming round.
// Returns GameEvents when the match enters SuddenDeath
func (m *Match) StartTurn() []GameEvent {
	winner := m.evaluateVictoryConditions()
	if winner != MatchInProgress {
		return []GameEvent{}
	}

	if m.TrueState.Turn <= m.GameCfg.MaxTurns {
		m.WorkingState.InSuddenDeath = false
		return []GameEvent{}
	}

	m.WorkingState.InSuddenDeath = true
	m.injectSuddenDeathHazards()
	m.TrueState = m.WorkingState.DeepCopy()

	gameEvents := make([]GameEvent, len(m.PlaybackLog))
	copy(gameEvents, m.PlaybackLog)
	m.PlaybackLog = nil

	return gameEvents
}

// injectSuddenDeathHazards drops up to SuddenDeathBombs bombs on random empty tiles.
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
		m.SubmitAction(m.WorkingState.dropBomb(SystemUnitID, target, BombDefaultPower))
	}
}

// ResolveTurn detonates due bombs, then advances the turn or ends the match, and commits WorkingState to TrueState.
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

	return planGameEvents, resolveTurnGameEvents
}

func (m *Match) resolveBombExplosionAndDamage() {
	m.PlaybackLog = append(m.PlaybackLog, m.WorkingState.ResolveBombExplosionAndDamage()...)
}

// ResolveBombExplosionAndDamage ticks bomb countdowns, detonates due bombs with their chain reactions, and applies the damage.
// Returns the resulting GameEvents.
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
		gameEvents = append(gameEvents, NewBombCountdownUpdatedEvent(id, bomb.Position, bomb.Countdown))
		if bomb.Countdown == 0 {
			queue = append(queue, id)
			ignited[id] = true
		}
	}
	return queue, ignited, gameEvents
}

// processChainDetonations detonates the queued bombs and any bombs their blasts reach, recording hits into the given sets.
// Returns the BombExplodedEvents.
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

		affectedTiles := gs.FindReachableTilesOnSnapshot(currBomb.Position, frozenGrid, MovementRule{
			MaxSteps:              currBomb.Range,
			Pattern:               PatternCardinal,
			PassPermissions:       PassUnits,
			StopOnNonUnitOccupant: true,
		})

		var affectedPos []Coordinate

		for pos := range affectedTiles {
			affectedPos = append(affectedPos, pos)

			tile := &gs.Grid[pos.Y][pos.X]
			switch tile.OccupantType {
			case OccupantBomb:
				nextBombID := BombID(tile.OccupantID)
				if ignitedBombs[nextBombID] {
					continue
				}

				nextBomb, ok := gs.Bombs[nextBombID]
				if !ok {
					continue
				}

				nextBomb.Countdown = 0
				explosionQueue = append(explosionQueue, nextBombID)
				ignitedBombs[nextBombID] = true

			case OccupantUnit:
				damagedUnits[UnitID(tile.OccupantID)] = true
			case OccupantSoftBlock:
				destroyedSoftBlocks[int(tile.OccupantID)] = true
			case OccupantItem:
				destroyedItems[int(tile.OccupantID)] = true
			}
		}

		gs.ClearStageTile(currBomb.Position)
		delete(gs.Bombs, currBombID)
		gameEvents = append(gameEvents, NewBombExplodedEvent(currBombID, currBomb.Position, affectedPos))
	}
	return gameEvents
}

// cloneGridSnapshot returns a copy of the grid for casting blast rays against the pre-explosion board.
func (gs *GameState) cloneGridSnapshot() [][]Tile {
	frozenGrid := make([][]Tile, len(gs.Grid))
	for y := range gs.Grid {
		frozenGrid[y] = make([]Tile, len(gs.Grid[y]))
		copy(frozenGrid[y], gs.Grid[y])
	}
	return frozenGrid
}

// handleDelayedBatchDamage handles delayed batch damage after all ignited bombs detonated.
// Returns GameEvents related to damaging.
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
		gameEvents = append(gameEvents, NewUnitDamagedEvent(unitID, unit.Position, unit.HP))

		if unit.HP <= 0 {
			gs.ClearStageTile(unit.Position)
			gameEvents = append(gameEvents, NewUnitDiedEvent(unitID, unit.Position))
		}
	}

	for softBlockID := range destroyedSoftBlocks {
		softBlock, ok := gs.SoftBlocks[softBlockID]
		if !ok {
			continue
		}

		gs.ClearStageTile(softBlock.Position)
		delete(gs.SoftBlocks, softBlockID)
		gameEvents = append(gameEvents, NewSoftBlockDestroyedEvent(softBlockID, softBlock.Position))
	}

	return gameEvents
}

// evaluateVictoryConditions returns MatchInProgress, 1, 2, or MatchDrawn.
// Goal: a living Boss, or a living King plus a living Ordinary unit. Wiped: the Boss dead, or a Boss-less team with no living unit.
// Both teams at Goal play on. A team wins by meeting Goal or by holding a living King against a Wiped opponent;
// if both or neither qualify, it is a draw.
func (m *Match) evaluateVictoryConditions() int {
	p1HasBoss, p1BossAlive := false, false
	p2HasBoss, p2BossAlive := false, false
	p1KingAlive, p1OrdinaryAlive := false, false
	p2KingAlive, p2OrdinaryAlive := false, false

	for _, unit := range m.WorkingState.Units {
		p1HasBoss = p1HasBoss || (unit.Team == 1 && unit.Role == RoleBoss)
		p2HasBoss = p2HasBoss || (unit.Team == 2 && unit.Role == RoleBoss)

		if unit.HP <= 0 {
			continue
		}
		if unit.Team == 1 {
			p1KingAlive = p1KingAlive || (unit.Role == RoleKing)
			p1BossAlive = p1BossAlive || (unit.Role == RoleBoss)
			p1OrdinaryAlive = p1OrdinaryAlive || (unit.Role != RoleKing && unit.Role != RoleBoss)
		} else {
			p2KingAlive = p2KingAlive || (unit.Role == RoleKing)
			p2BossAlive = p2BossAlive || (unit.Role == RoleBoss)
			p2OrdinaryAlive = p2OrdinaryAlive || (unit.Role != RoleKing && unit.Role != RoleBoss)
		}
	}

	p1Goal := (p1HasBoss && p1BossAlive) || (p1KingAlive && p1OrdinaryAlive)
	p2Goal := (p2HasBoss && p2BossAlive) || (p2KingAlive && p2OrdinaryAlive)

	p1Wiped := (p1HasBoss && !p1BossAlive) || (!p1HasBoss && !p1KingAlive && !p1OrdinaryAlive)
	p2Wiped := (p2HasBoss && !p2BossAlive) || (!p2HasBoss && !p2KingAlive && !p2OrdinaryAlive)

	if p1Goal && p2Goal {
		return MatchInProgress
	}

	if p1Goal || (p1KingAlive && p2Wiped) {
		if p2Goal || (p2KingAlive && p1Wiped) {
			return MatchDrawn
		}
		return 1
	}

	if p2Goal || (p2KingAlive && p1Wiped) {
		return 2
	}

	return MatchDrawn
}
