package engine

// GameEvtType represents a game event type emitted during turn resolution.
type GameEvtType string

const (
	// GameEvtUnitMoved signals a unit relocated.
	GameEvtUnitMoved GameEvtType = "unitMoved"
	// GameEvtUnitDamaged signals a unit took damage.
	GameEvtUnitDamaged GameEvtType = "unitDamaged"
	// GameEvtUnitDied signals a unit was eliminated.
	GameEvtUnitDied GameEvtType = "unitDied"
	// GameEvtBombPlaced signals a new bomb was deployed.
	GameEvtBombPlaced GameEvtType = "bombPlaced"
	// GameEvtBombCountdownUpdated signals a bomb updated its countdown, for both increase or decrease
	GameEvtBombCountdownUpdated GameEvtType = "bombCountdownUpdated"
	// GameEvtBombExploded signals a bomb detonated and lists affected tiles.
	GameEvtBombExploded GameEvtType = "bombExploded"
	// GameEvtSoftBlockDestroyed signals a soft block was destroyed.
	GameEvtSoftBlockDestroyed GameEvtType = "softBlockDestroyed"
	// GameEvtMatchEnded signals the match concluded.
	GameEvtMatchEnded GameEvtType = "matchEnded"
)

// GameEvent is a turn resolution event emitted during commit.
// Type determines the event variant; optional fields are omitted when zero.
type GameEvent struct {
	Type              GameEvtType  `json:"type"`
	UnitID            UnitID       `json:"unitId"`
	BombID            BombID       `json:"bombId,omitempty"`
	SoftBlockID       int          `json:"softBlockId,omitempty"`
	ItemID            int          `json:"itemId,omitempty"`
	Position          *Coordinate  `json:"position,omitempty"`
	From              *Coordinate  `json:"from,omitempty"`
	To                *Coordinate  `json:"to,omitempty"`
	NewHP             int          `json:"newHp"`
	Range             int          `json:"range,omitempty"`
	Countdown         int          `json:"countdown"`
	AffectedPositions []Coordinate `json:"affectedPositions,omitempty"`
	WinnerTeamID      int          `json:"winnerTeamId,omitempty"`
	IsDraw            bool         `json:"isDraw,omitempty"`
}

// NewUnitMovedEvent creates a unit moved event.
func NewUnitMovedEvent(unitID UnitID, from, to Coordinate) GameEvent {
	return GameEvent{Type: GameEvtUnitMoved, UnitID: unitID, From: &from, To: &to}
}

// NewUnitDamagedEvent creates a unit damaged event.
func NewUnitDamagedEvent(unitID UnitID, newHP int) GameEvent {
	return GameEvent{Type: GameEvtUnitDamaged, UnitID: unitID, NewHP: newHP}
}

// NewUnitDiedEvent creates a unit died event.
func NewUnitDiedEvent(unitID UnitID) GameEvent {
	return GameEvent{Type: GameEvtUnitDied, UnitID: unitID}
}

// NewBombPlacedEvent creates a bomb placed event.
func NewBombPlacedEvent(unitID UnitID, bombID BombID, pos Coordinate, rangeVal, countdown int) GameEvent {
	return GameEvent{Type: GameEvtBombPlaced, UnitID: unitID, BombID: bombID, Position: &pos, Range: rangeVal, Countdown: countdown}
}

// NewBombCountdownUpdatedEvent creats a bomb countdown updated event.
func NewBombCountdownUpdatedEvent(bombID BombID, countdown int) GameEvent {
	return GameEvent{Type: GameEvtBombCountdownUpdated, BombID: bombID, Countdown: countdown}
}

// NewBombExplodedEvent creates a bomb exploded event.
func NewBombExplodedEvent(bombID BombID, affected []Coordinate) GameEvent {
	return GameEvent{Type: GameEvtBombExploded, BombID: bombID, AffectedPositions: affected}
}

// NewSoftBlockDestroyedEvent creates a soft block destroyed event.
func NewSoftBlockDestroyedEvent(id int, pos Coordinate) GameEvent {
	return GameEvent{Type: GameEvtSoftBlockDestroyed, SoftBlockID: id, Position: &pos}
}

// NewMatchEndedEvent creates a match ended event.
func NewMatchEndedEvent(winnerTeamID int, isDraw bool) GameEvent {
	return GameEvent{Type: GameEvtMatchEnded, WinnerTeamID: winnerTeamID, IsDraw: isDraw}
}
