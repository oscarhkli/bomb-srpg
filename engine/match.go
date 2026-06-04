package engine

import "fmt"

// ResetTurn discards the mid-turn WorkingState
// and rollback to the beginning of the turn with deep copy of TrueState
func (m *Match) ResetTurn() {
	m.WorkingState = m.TrueState.DeepCopy()
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

// CommandMoveUnit executes a unit relocation after verifying game rule compliance.
// It calculates the active range, updates the board matrix, and commits a UnitMovedEvent.
// Returns an error if the pathing rules are violated or if the target cell is blocked.
func (m *Match) CommandMoveUnit(unitID int, target Coordinate) error {
	unit, err := m.validateActiveUnit(unitID)
	if err != nil {
		return err
	}

	tiles := m.WorkingState.FindReachableTiles(unit.Position, unit.NewMovementRule())

	if _, ok := tiles[target]; !ok {
		return fmt.Errorf("movement restriction: target coordinate is out of moving range")
	}

	// err will always be nil at the moment, not testable until the Skills implementation in Phase 3
	if err = m.WorkingState.IsLandingLegal(target, OccupantUnit); err != nil {
		return fmt.Errorf("movement rejected: %w", err)
	}

	oldPos := unit.Position
	m.WorkingState.ClearStageTile(oldPos)
	m.WorkingState.UpdateStageOccupant(target, OccupantUnit, unitID)

	m.SubmitAction(UnitMovedEvent{
		UnitID: unitID,
		From:   oldPos,
		To:     target,
	})

	return nil
}

// validateActiveUnit performs systemic and structural sanity checks on a requested unit.
// It sequentially validates presence, vitality, phase ownership, bounds, and grid desync.
// Returns a pointer to the verified Unit, or a detailed error blocking action execution.
func (m *Match) validateActiveUnit(unitID int) (*Unit, error) {
	unit, ok := m.WorkingState.Units[unitID]
	if !ok {
		return nil, fmt.Errorf("security violation: unit ID %d does not exist in active sandbox context", unitID)
	}

	if unit.HP <= 0 {
		return nil, fmt.Errorf("tactical restriction: unit %d is dead and cannot declare actions", unitID)
	}

	currentTeamTurn := ((m.WorkingState.Turn - 1) & 1) + 1
	if unit.Team != currentTeamTurn {
		return nil, fmt.Errorf("turn restriction: unit %d belongs to Team %d but it's currently Team %d's turn", unitID, unit.Team, currentTeamTurn)
	}

	if !m.WorkingState.IsWithinBounds(unit.Position) {
		return nil, fmt.Errorf("data corruption: unit %d current position %v is out of stage bounds", unitID, unit.Position)
	}

	cell := m.WorkingState.Grid[unit.Position.Y][unit.Position.X]
	if cell.OccupantType != OccupantUnit || cell.OccupantID != unitID {
		return nil, fmt.Errorf("data desync: grid matrix at %v does not acknowledge unit %d as its occupant (found type: %v, id: %d)",
			unit.Position, unitID, cell.OccupantType, cell.OccupantID)
	}

	return unit, nil
}

// CommandPlaceBomb executes a bomb deployment after verifying unit's bomb availability and grid compliance.
// It validates placement range, registers a new Bomb state tracking instance, and commits a BombPlacedEvent.
// Returns an error if the unit is running out of bombs, the target is out of range, or the cell is blocked.
func (m *Match) CommandPlaceBomb(unitID int, target Coordinate) error {

	// identify the unit and check the availability
	unit, err := m.validateActiveUnit(unitID)
	if err != nil {
		return err
	}

	if unit.BombUsed >= unit.MaxBombCount {
		return fmt.Errorf("unit restriction: unit %d has used up all his bombs", unitID)
	}

	tiles := m.WorkingState.FindReachableTiles(unit.Position, unit.NewBombPlacementRule())

	if _, ok := tiles[target]; !ok {
		return fmt.Errorf("bomb placement restriction: target coordinate is out of placement range")
	}

	if err = m.WorkingState.IsLandingLegal(target, OccupantBomb); err != nil {
		return fmt.Errorf("bomb placement rejected: %w", err)
	}

	m.WorkingState.TurnBombCounter++
	bomb := &Bomb{
		ID:        NewBombID(m.WorkingState.Turn, m.WorkingState.TurnBombCounter, unitID),
		OwnerID:   unitID,
		Position:  target,
		Range:     unit.BombPower,
		Countdown: m.WorkingState.DeduceBombCountDown(target, unit),
	}
	m.WorkingState.Bombs[bomb.ID] = bomb
	m.WorkingState.UpdateStageOccupant(target, OccupantBomb, bomb.ID)

	m.SubmitAction(BombPlacedEvent{
		UnitID:    unitID,
		BombID:    bomb.ID,
		Position:  target,
		Range:     bomb.Range,
		Countdown: bomb.Countdown,
	})

	return nil
}

// IsLandingLegal checks if the target is legal to be landed by a certain occupantType.
// In Phase 1 it's used by placing Bomb only, but in future it will be used for skills like jump
func (gs GameState) IsLandingLegal(target Coordinate, occupantType OccupantType) error {
	if !gs.IsWithinBounds(target) {
		return fmt.Errorf("boundary restriction: coordinate %v is out of stage dimensions", target)
	}

	tile := gs.Grid[target.Y][target.X]

	// Phase 1 only on TerrainPlain. In futur phase it should support TerrainLava as well
	if tile.Type != TerrainPlain {
		return fmt.Errorf("terrain restriction: can only land on plain tile but target is %v", tile.Type)
	}

	// Cell Occupant Collisions
	if tile.OccupantType != OccupantNone {
		return fmt.Errorf("occupant restriction: target cell already contains entity type %v", tile.OccupantType)
	}

	return nil
}

// ResolveTurn controls everything in between turns
// 1. Tick Bomb Countdowns
// 2. Detonate Zero-Timer Bombs & Cascade Chain Reactions
// 3. Calculate Occupant Destruction (Units, SoftBlocks, Items)
// 4. Victory audit guard: Check who has living units left on the board
// 5. Sudden death trigger check: If Turn >= MaxTurn: enter sudden death
// 6. Advance Turn Counter (Turn++)
// 7. Overwrite TrueState with clean DeepCopy
func (m *Match) ResolveTurn() []GameEvent {
	// TODO
	return []GameEvent{}
}
