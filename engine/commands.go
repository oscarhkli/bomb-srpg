package engine

type TurnCmdType string

const (
	TurnCmdMove      TurnCmdType = "move"
	TurnCmdPlaceBomb TurnCmdType = "placeBomb"
)

// TurnCommand's Type determines the action; UnitID and Target are always required.
type TurnCommand struct {
	Type   TurnCmdType `json:"type"`
	UnitID UnitID      `json:"unitId"`
	Target Coordinate  `json:"target"`
}

func NewMoveCommand(unitID UnitID, target Coordinate) TurnCommand {
	return TurnCommand{Type: TurnCmdMove, UnitID: unitID, Target: target}
}

func NewPlaceBombCommand(unitID UnitID, target Coordinate) TurnCommand {
	return TurnCommand{Type: TurnCmdPlaceBomb, UnitID: unitID, Target: target}
}
