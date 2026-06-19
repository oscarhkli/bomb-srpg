package engine

// TurnCmdType represents a player command type during a turn.
type TurnCmdType string

const (
	// TurnCmdMove instructs a unit to move to a target coordinate.
	TurnCmdMove TurnCmdType = "move"
	// TurnCmdPlaceBomb instructs a unit to place a bomb at a target coordinate.
	TurnCmdPlaceBomb TurnCmdType = "placeBomb"
)

// TurnCommand is a player action submitted during the planning phase.
// Type determines the action; UnitID and Target are always required.
type TurnCommand struct {
	Type   TurnCmdType `json:"type"`
	UnitID UnitID      `json:"unitId"`
	Target Coordinate  `json:"target"`
}

// NewMoveCommand creates a move command for the given unit to the target coordinate.
func NewMoveCommand(unitID UnitID, target Coordinate) TurnCommand {
	return TurnCommand{Type: TurnCmdMove, UnitID: unitID, Target: target}
}

// NewPlaceBombCommand creates a bomb placement command for the given unit at the target coordinate.
func NewPlaceBombCommand(unitID UnitID, target Coordinate) TurnCommand {
	return TurnCommand{Type: TurnCmdPlaceBomb, UnitID: unitID, Target: target}
}
