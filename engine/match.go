package engine

import (
	"fmt"
	"math/rand"
)

// ResetTurn discards the mid-turn WorkingState and PlaybackLog
// and rollback to the beginning of the turn with deep copy of TrueState
func (m *Match) ResetTurn() {
	m.WorkingState = m.TrueState.DeepCopy()
	m.PlaybackLog = []GameEvent{}
}

// SubmitAction registers a validated mid-turn planning step permanently.
// - In Rollback Mode: It just logs the event but waits for a final commit command.
// - In Non-Rollback Mode: The server calls this immediately, making the move irreversible.
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

	// discard all the logs in rollback mode
	// as opponent doesn't need to know what steps were taken lead to surrender
	m.PlaybackLog = []GameEvent{}

	// broadcast it
	return []GameEvent{
		NewMatchEndedEvent(m.WinnerTeamID, false),
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
		return []GameEvent{}, fmt.Errorf("%w: %s", ErrUnsupportedCommand, cmd.Type)
	}
}

// CommandMoveUnit executes a unit relocation after verifying game rule compliance.
// It calculates the active range, updates the board matrix, and commits a UnitMovedEvent.
// Returns GameEvents produced or an error if the pathing rules are violated or if the target cell is blocked.
func (m *Match) CommandMoveUnit(unitID UnitID, target Coordinate) ([]GameEvent, error) {
	unit, err := m.validateActiveUnit(unitID)
	if err != nil {
		return []GameEvent{}, err
	}

	if unit.HasMoved {
		return []GameEvent{}, fmt.Errorf("%w: unit %#x already moved this turn", ErrAlreadyMoved, unitID)
	}

	tiles := m.WorkingState.FindReachableTiles(unit.Position, unit.NewMovementRule())

	if _, ok := tiles[target]; !ok {
		return []GameEvent{}, ErrOutOfMoveRange
	}

	// err will always be nil at the moment, not testable until the Skills implementation in Phase 4
	if err = m.WorkingState.IsLandingLegal(target, OccupantUnit); err != nil {
		return []GameEvent{}, fmt.Errorf("%w: %w", ErrInvalidLanding, err)
	}

	oldPos := unit.Position
	m.WorkingState.ClearStageTile(oldPos)
	m.WorkingState.UpdateStageOccupant(target, OccupantUnit, int64(unitID))
	unit.Position = target
	unit.HasMoved = true

	gameEvent := NewUnitMovedEvent(unitID, oldPos, target)
	m.SubmitAction(gameEvent)

	return []GameEvent{gameEvent}, nil
}

func (m *Match) validateActiveUnit(unitID UnitID) (*Unit, error) {
	unit, ok := m.WorkingState.Units[unitID]
	if !ok {
		return nil, fmt.Errorf("%w: unit %#x does not exist", ErrUnitNotFound, unitID)
	}
	if unit.HP <= 0 {
		return nil, fmt.Errorf("%w: unit %#x is dead", ErrUnitDead, unitID)
	}
	if unit.Team != m.WorkingState.ActiveTeam {
		return nil, fmt.Errorf("%w: unit %#x not active team", ErrNotActiveTeam, unitID)
	}
	if !m.WorkingState.IsWithinBounds(unit.Position) {
		return nil, fmt.Errorf("%w: unit %#x out of bounds", ErrOutOfBounds, unitID)
	}
	cell := m.WorkingState.Grid[unit.Position.Y][unit.Position.X]
	if cell.OccupantType != OccupantUnit || cell.OccupantID != int64(unitID) {
		return nil, fmt.Errorf("%w: unit %#x desynced at %v", ErrDesynced, unitID, unit.Position)
	}
	return unit, nil
}

// CommandPlaceBomb executes a bomb deployment after verifying unit's bomb availability and grid compliance.
// It validates placement range, registers a new Bomb state tracking instance, and commits a BombPlacedEvent.
// Returns GameEvents produced or an error if the unit is running out of bombs, the target is out of range, or the cell is blocked.
func (m *Match) CommandPlaceBomb(unitID UnitID, target Coordinate) ([]GameEvent, error) {
	// identify the unit and check the availability
	unit, err := m.validateActiveUnit(unitID)
	if err != nil {
		return []GameEvent{}, err
	}

	if unit.HasUsedSkill {
		return []GameEvent{}, fmt.Errorf("%w: unit %#x already used skill this turn", ErrAlreadyUsedSkill, unitID)
	}

	if unit.BombUsed >= unit.MaxBombCount {
		return []GameEvent{}, fmt.Errorf("%w: unit %#x out of bombs", ErrOutOfBombs, unitID)
	}

	tiles := m.WorkingState.FindReachableTiles(unit.Position, unit.NewBombPlacementRule())

	if _, ok := tiles[target]; !ok {
		return []GameEvent{}, ErrOutOfBombRange
	}

	if err = m.WorkingState.IsLandingLegal(target, OccupantBomb); err != nil {
		return []GameEvent{}, fmt.Errorf("%w: %w", ErrInvalidLanding, err)
	}

	gameEvents := m.placeBomb(unitID, target, unit.BombPower)
	unit.HasUsedSkill = true

	return gameEvents, nil
}

func (m *Match) placeBomb(unitID UnitID, target Coordinate, bombPower int) []GameEvent {
	m.WorkingState.TurnBombCounter++
	bomb := &Bomb{
		ID:        NewBombID(m.WorkingState.Turn, m.WorkingState.TurnBombCounter, unitID),
		OwnerID:   unitID,
		Position:  target,
		Range:     bombPower,
		Countdown: m.WorkingState.DeduceBombCountDown(target),
	}
	m.WorkingState.Bombs[bomb.ID] = bomb
	m.WorkingState.UpdateStageOccupant(target, OccupantBomb, int64(bomb.ID))

	evt := NewBombPlacedEvent(unitID, bomb.ID, target, bomb.Range, bomb.Countdown)
	m.SubmitAction(evt)

	return []GameEvent{evt}
}

// IsLandingLegal checks if the target is legal to be landed by a certain occupantType.
// In Phase 1 it's used by placing Bomb only, but in future it will be used for skills like jump.
func (gs GameState) IsLandingLegal(target Coordinate, occupantType OccupantType) error {
	if !gs.IsWithinBounds(target) {
		return fmt.Errorf("%w: coordinate %v out of bounds", ErrOutOfBounds, target)
	}

	tile := gs.Grid[target.Y][target.X]

	// Phase 1 only on TerrainPlain. In futur phase it should support TerrainLava as well
	if tile.Type != TerrainPlain {
		return fmt.Errorf("%w: can only place on plain terrain, got %v", ErrInvalidLanding, tile.Type)
	}

	// Cell Occupant Collisions
	if tile.OccupantType != OccupantNone {
		return fmt.Errorf("%w: cell occupied by %v", ErrCellOccupied, tile.OccupantType)
	}

	return nil
}

// StartTurn sets up the environmental boundaries for the upcoming round.
// Returns GameEvents when the match enters SuddenDeath
func (m *Match) StartTurn() []GameEvent {
	victoryResult, _ := m.evaluateVictoryConditions()
	if victoryResult != MatchInProgress {
		return []GameEvent{} // Match has reached a conclusion; abort round initialization
	}

	if !(m.GameCfg.SuddenDeath && m.TrueState.Turn > m.GameCfg.MaxTurns) {
		m.WorkingState.InSuddenDeath = false
		return []GameEvent{}
	}

	m.WorkingState.InSuddenDeath = true
	m.injectSuddenDeathHazards()
	m.TrueState = m.WorkingState.DeepCopy()

	// Flush animation log arrays from the sandbox replay history buffer to the caller
	gameEvents := make([]GameEvent, len(m.PlaybackLog))
	copy(gameEvents, m.PlaybackLog)
	m.PlaybackLog = []GameEvent{}

	return gameEvents
}

// injectSuddenDeathHazards picks 2 random unoccupied tiles and drop bombs there.
func (m *Match) injectSuddenDeathHazards() {
	emptyTilePos := []Coordinate{}
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
		m.placeBomb(SystemUnitID, target, BombDefaultPower)
	}
}

// ResolveTurn controls everything in between turns:
// 1. Tick Bomb Countdowns
// 2. Detonate Zero-Timer Bombs & Cascade Chain Reactions
// 3. Calculate Occupant Destruction (Units, SoftBlocks, Items)
// 4. Victory audit guard: Check who has living units left on the board
// 5. Advance Turn Counter (Turn++)
// 6. Reset move & skills limits
// 7. Overwrite TrueState with clean DeepCopy
func (m *Match) ResolveTurn() []GameEvent {
	m.resolveBombExplosionAndDamage()

	if m.WinnerTeamID == 0 {
		result, winner := m.evaluateVictoryConditions()

		switch result {
		case MatchDraw:
			m.WinnerTeamID = -1
			m.PlaybackLog = append(m.PlaybackLog, NewMatchEndedEvent(-1, true))

		case MatchWin:
			m.WinnerTeamID = winner
			m.PlaybackLog = append(m.PlaybackLog, NewMatchEndedEvent(winner, false))

		case MatchInProgress:
			m.WorkingState.Turn++
			m.WorkingState.ActiveTeam = ((m.WorkingState.Turn - 1) & 1) + 1
			for _, unit := range m.WorkingState.Units {
				unit.HasMoved = false
				unit.HasUsedSkill = false
			}
		}
	}

	m.TrueState = m.WorkingState.DeepCopy()

	// Flush animation log arrays from the sandbox replay history buffer to the caller
	gameEvents := make([]GameEvent, len(m.PlaybackLog))
	copy(gameEvents, m.PlaybackLog)
	m.PlaybackLog = []GameEvent{}

	return gameEvents
}

// resolveBombExplosionAndDamage resolves bomb explosion, chain reaction and the damages made, and fire related GameEvents.
// Note: All cascading chain reactions occur at the exact same physical millisecond within a turn.
// Soft block / Item destroyed by an early explosion must continue to exist as a solid, ray-blocking obstacle for all subsequent waves of bombs
// until the entire chain reaction loop is completely finished.
func (m *Match) resolveBombExplosionAndDamage() {
	explosionQueue, ignitedBombs := m.tickCountdownsAndQueueFuses()

	frozenGrid := m.WorkingState.cloneGridSnapshot()

	// Setup delayed batch damange handling
	damagedUnits := make(map[UnitID]bool)
	destroyedSoftBlocks := make(map[int]bool)
	destroyedItems := make(map[int]bool)

	m.processChainDetonations(explosionQueue, ignitedBombs, frozenGrid, damagedUnits, destroyedSoftBlocks, destroyedItems)
	m.handleDelayedBatchDamage(damagedUnits, destroyedSoftBlocks)
}

func (m *Match) tickCountdownsAndQueueFuses() ([]BombID, map[BombID]bool) {
	var queue []BombID
	ignited := make(map[BombID]bool)

	for id, bomb := range m.WorkingState.Bombs {
		if bomb.Countdown < 0 {
			continue // Skip non-countdown bombs
		}

		bomb.Countdown--
		m.PlaybackLog = append(m.PlaybackLog, NewBombCountdownUpdatedEvent(id, bomb.Countdown))
		if bomb.Countdown == 0 {
			queue = append(queue, id)
			ignited[id] = true
		}
	}
	return queue, ignited
}

// processChainDetonations handles Occupant Destruction & Chain reaction.
func (m *Match) processChainDetonations(
	explosionQueue []BombID,
	ignitedBombs map[BombID]bool,
	frozenGrid [][]Tile,
	damagedUnits map[UnitID]bool,
	destroyedSoftBlocks map[int]bool,
	destroyedItems map[int]bool,
) {
	for len(explosionQueue) > 0 {
		currBombID := explosionQueue[0]
		explosionQueue = explosionQueue[1:]

		currBomb, ok := m.WorkingState.Bombs[currBombID]
		if !ok {
			continue
		}

		affectedTiles := m.WorkingState.FindReachableTilesOnSnapshot(currBomb.Position, frozenGrid, MovementRule{
			MaxSteps:              currBomb.Range,
			Pattern:               PatternCardinal,
			PassPermissions:       PassUnits,
			StopOnNonUnitOccupant: true,
		})

		affectedPos := []Coordinate{}

		for pos := range affectedTiles {
			affectedPos = append(affectedPos, pos)

			tile := &m.WorkingState.Grid[pos.Y][pos.X]
			switch tile.OccupantType {
			case OccupantBomb:
				// chain reaction
				nextBombID := BombID(tile.OccupantID)
				if ignitedBombs[nextBombID] {
					continue
				}

				nextBomb, ok := m.WorkingState.Bombs[nextBombID]
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

		m.WorkingState.ClearStageTile(currBomb.Position)
		delete(m.WorkingState.Bombs, currBombID)
		m.PlaybackLog = append(m.PlaybackLog, NewBombExplodedEvent(currBombID, affectedPos))
	}
}

// cloneGridSnapshot captures the exact state of the board before any bomb goes off.
// This ensures soft blocks continue to block rays for the entire duration of the turn.
func (gs *GameState) cloneGridSnapshot() [][]Tile {
	frozenGrid := make([][]Tile, len(gs.Grid))
	for y := range gs.Grid {
		frozenGrid[y] = make([]Tile, len(gs.Grid[y]))
		copy(frozenGrid[y], gs.Grid[y])
	}
	return frozenGrid
}

// handleDelayedBatchDamage handles delayed batch damange after all ignited bombs detonated
func (m *Match) handleDelayedBatchDamage(
	damagedUnits map[UnitID]bool,
	destroyedSoftBlocks map[int]bool,
	// destroyedItems map[int]bool,
) {
	for unitID := range damagedUnits {
		unit, ok := m.WorkingState.Units[unitID]
		if !ok {
			continue
		}

		unit.HP -= 1
		m.PlaybackLog = append(m.PlaybackLog, NewUnitDamagedEvent(unitID, unit.HP))

		if unit.HP <= 0 {
			m.WorkingState.ClearStageTile(unit.Position)
			m.PlaybackLog = append(m.PlaybackLog, NewUnitDiedEvent(unitID))
		}
	}

	for softBlockID := range destroyedSoftBlocks {
		softBlock, ok := m.WorkingState.SoftBlocks[softBlockID]
		if !ok {
			continue
		}

		m.WorkingState.ClearStageTile(softBlock.Position)
		delete(m.WorkingState.SoftBlocks, softBlockID)
		m.PlaybackLog = append(m.PlaybackLog, NewSoftBlockDestroyedEvent(softBlockID, softBlock.Position))
	}

	// TODO: Item destruction in future phase
}

// evaluateVictoryConditions calculate the VictoryResult with the below truth table:
//
// # P1 King | P1 Non | P2 King | P2 Non | Expected Result
//
// ------------------------------------------------------
//
//	T    |   T    |    T    |   T    | MatchInProgress, 0
//
// ------------------------------------------------------
//
//	T    |   T    |    T    |   F    | MatchWin, 1 (P2 misses non-king)
//	T    |   T    |    F    |   T    | MatchWin, 1 (P2 King dead)
//	T    |   T    |    F    |   F    | MatchWin, 1 (P2 wiped out)
//	T    |   F    |    F    |   F    | MatchWin, 1 (P1 lone king)
//
// ------------------------------------------------------
//
//	F    |   T    |    T    |   T    | MatchWin, 2 (P1 King dead)
//	F    |   F    |    T    |   T    | MatchWin, 2 (P1 wiped out)
//	F    |   F    |    T    |   F    | MatchWin, 2 (P2 lone King)
//	T    |   F    |    T    |   T    | MatchWin, 2 (P1 misses non-king)
//
// ------------------------------------------------------
//
//	F    |   T    |    F    |   T    | MatchDraw, -1 (Both Kings dead)
//	F    |   T    |    F    |   F    | MatchDraw, -1 (Both Kings dead)
//	F    |   F    |    F    |   T    | MatchDraw, -1 (Both Kings dead)
//	F    |   F    |    F    |   F    | MatchDraw, -1 (Everyone dead)
//	T    |   F    |    T    |   F    | MatchDraw, -1 (Mutual lone Kings)*
func (m *Match) evaluateVictoryConditions() (VictoryResult, int) {
	p1King, p1NonKing := false, false
	p2King, p2NonKing := false, false

	for _, unit := range m.WorkingState.Units {
		if unit.HP <= 0 {
			continue
		}
		if unit.Team == 1 {
			if unit.Type.Name == "King" {
				p1King = true
			} else {
				p1NonKing = true
			}
		} else {
			if unit.Type.Name == "King" {
				p2King = true
			} else {
				p2NonKing = true
			}
		}
	}

	// Standard goals
	p1Goal := p1King && p1NonKing
	p2Goal := p2King && p2NonKing

	// Opponent fully defeated conditions
	p1Wiped := !p1King && !p1NonKing
	p2Wiped := !p2King && !p2NonKing

	// 1. Both teams are still strong -> The fight continues
	if p1Goal && p2Goal {
		return MatchInProgress, 0
	}

	// 2. P1 wins if they meet their goal, OR if they have a King and P2 is wiped
	if p1Goal || (p1King && p2Wiped) {
		// Double check mutual wipe out edge case
		if p2Goal || (p2King && p1Wiped) {
			return MatchDraw, -1
		}
		return MatchWin, 1
	}

	// 3. P2 wins if they meet their goal, OR if they have a King and P1 is wiped
	if p2Goal || (p2King && p1Wiped) {
		return MatchWin, 2
	}

	// 4. Anything else is a Draw
	return MatchDraw, -1
}
