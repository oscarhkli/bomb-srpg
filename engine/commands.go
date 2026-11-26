package engine

type TurnCmdType string

const (
	TurnCmdMove      TurnCmdType = "move"
	TurnCmdPlaceBomb TurnCmdType = "placeBomb"
)

//sumtype:decl
type TurnCommand interface {
	Actor() UnitID
	isTurnCommand()
}

type MoveCommand struct {
	UnitID UnitID
	Target Coordinate
}

type PlaceBombCommand struct {
	UnitID UnitID
	Target Coordinate
}

func (c MoveCommand) Actor() UnitID      { return c.UnitID }
func (c PlaceBombCommand) Actor() UnitID { return c.UnitID }
func (MoveCommand) isTurnCommand()       {}
func (PlaceBombCommand) isTurnCommand()  {}

func NewMoveCommand(unitID UnitID, target Coordinate) MoveCommand {
	return MoveCommand{UnitID: unitID, Target: target}
}

func NewPlaceBombCommand(unitID UnitID, target Coordinate) PlaceBombCommand {
	return PlaceBombCommand{UnitID: unitID, Target: target}
}
