package engine

//sumtype:decl
type GameEvent interface {
	isGameEvent()
}

type UnitMoved struct {
	UnitID UnitID
	From   Coordinate
	To     Coordinate
}

type UnitDamaged struct {
	UnitID   UnitID
	Position Coordinate
	NewHP    int
}

type UnitDied struct {
	UnitID   UnitID
	Position Coordinate
}

type BombPlaced struct {
	UnitID    UnitID
	BombID    BombID
	Position  Coordinate
	Range     int
	Countdown int
}

type BombCountdownUpdated struct {
	BombID    BombID
	Position  Coordinate
	Countdown int
}

type BombExploded struct {
	BombID            BombID
	Position          Coordinate
	AffectedPositions []Coordinate
}

type SoftBlockDestroyed struct {
	SoftBlockID int
	Position    Coordinate
}

type MatchEnded struct {
	WinnerTeamID int
}

func (UnitMoved) isGameEvent()            {}
func (UnitDamaged) isGameEvent()          {}
func (UnitDied) isGameEvent()             {}
func (BombPlaced) isGameEvent()           {}
func (BombCountdownUpdated) isGameEvent() {}
func (BombExploded) isGameEvent()         {}
func (SoftBlockDestroyed) isGameEvent()   {}
func (MatchEnded) isGameEvent()           {}
