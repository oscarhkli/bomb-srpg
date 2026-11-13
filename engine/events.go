package engine

type GameEvtType string

const (
	GameEvtUnitMoved            GameEvtType = "unitMoved"
	GameEvtUnitDamaged          GameEvtType = "unitDamaged"
	GameEvtUnitDied             GameEvtType = "unitDied"
	GameEvtBombPlaced           GameEvtType = "bombPlaced"
	GameEvtBombCountdownUpdated GameEvtType = "bombCountdownUpdated"
	GameEvtBombExploded         GameEvtType = "bombExploded"
	GameEvtSoftBlockDestroyed   GameEvtType = "softBlockDestroyed"
	GameEvtMatchEnded           GameEvtType = "matchEnded"
)

// GameEvent's Type selects the event variant; optional fields are omitted when zero.
type GameEvent struct {
	Type              GameEvtType  `json:"type"`
	UnitID            UnitID       `json:"unitId"`
	BombID            BombID       `json:"bombId,omitzero"`
	SoftBlockID       int          `json:"softBlockId,omitzero"`
	ItemID            int          `json:"itemId,omitzero"`
	Position          *Coordinate  `json:"position,omitzero"`
	From              *Coordinate  `json:"from,omitzero"`
	To                *Coordinate  `json:"to,omitzero"`
	NewHP             int          `json:"newHp"`
	Range             int          `json:"range,omitzero"`
	Countdown         int          `json:"countdown"`
	AffectedPositions []Coordinate `json:"affectedPositions,omitempty"`
	WinnerTeamID      int          `json:"winnerTeamId,omitzero"`
}

func NewUnitMovedEvent(unitID UnitID, from, to Coordinate) GameEvent {
	return GameEvent{Type: GameEvtUnitMoved, UnitID: unitID, From: &from, To: &to}
}

func newUnitDamagedEvent(unitID UnitID, pos Coordinate, newHP int) GameEvent {
	return GameEvent{Type: GameEvtUnitDamaged, UnitID: unitID, Position: &pos, NewHP: newHP}
}

func NewUnitDiedEvent(unitID UnitID, pos Coordinate) GameEvent {
	return GameEvent{Type: GameEvtUnitDied, UnitID: unitID, Position: &pos}
}

func newBombPlacedEvent(unitID UnitID, bombID BombID, pos Coordinate, rangeVal, countdown int) GameEvent {
	return GameEvent{Type: GameEvtBombPlaced, UnitID: unitID, BombID: bombID, Position: &pos, Range: rangeVal, Countdown: countdown}
}

func newBombCountdownUpdatedEvent(bombID BombID, pos Coordinate, countdown int) GameEvent {
	return GameEvent{Type: GameEvtBombCountdownUpdated, BombID: bombID, Position: &pos, Countdown: countdown}
}

func newBombExplodedEvent(bombID BombID, pos Coordinate, affected []Coordinate) GameEvent {
	return GameEvent{Type: GameEvtBombExploded, BombID: bombID, Position: &pos, AffectedPositions: affected}
}

func newSoftBlockDestroyedEvent(id int, pos Coordinate) GameEvent {
	return GameEvent{Type: GameEvtSoftBlockDestroyed, SoftBlockID: id, Position: &pos}
}

func NewMatchEndedEvent(winnerTeamID int) GameEvent {
	return GameEvent{Type: GameEvtMatchEnded, WinnerTeamID: winnerTeamID}
}
