package engine

const (
	// SystemUnitID represents environmental or game-engine authoritative actions
	// mapped explicitly to Team 0, Player 0 for sudden-death bomb drops.
	SystemUnitID UnitID = 0

	UnitLocalShift = 0
	UnitTeamShift  = 4

	UnitLocalMask uint8 = 0x0F
	UnitTeamMask  uint8 = 0x0F

	BombCounterShift = 0
	BombTurnShift    = 16
	BombUnitIDShift  = 24

	BombCounterMask uint32 = 0xFFFF
	BombTurnMask    uint32 = 0xFF
	BombUnitIDMask  uint32 = 0xFF
)

// NewUnitID constructs a UnitID from team and player index using (TeamID << 4) | Index.
func NewUnitID(teamID, counter int) UnitID {
	return UnitID((uint8(teamID) << UnitTeamShift) | (uint8(counter) << UnitLocalShift))
}

// Decode extracts TeamID and PlayerIndex from a UnitID.
func (id UnitID) Decode() (teamID int, index int) {
	teamID = int((uint8(id) >> UnitTeamShift) & UnitTeamMask)
	index = int(uint8(id) >> uint8(UnitLocalShift) & uint8(UnitLocalMask))
	return
}

// NewBombID constructs a BombID from turn, counter, and owner UnitID using (UnitID << 24) | (Turn << 16) | Counter.
func NewBombID(turn int, counter int, unitID UnitID) BombID {
	return BombID(
		(uint32(unitID) << BombUnitIDShift) |
			(uint32(turn) << BombTurnShift) |
			(uint32(counter) << BombCounterShift),
	)
}

// Decode extracts Turn, Counter, and Owner UnitID from a BombID.
func (id BombID) Decode() (turn int, counter int, ownerID UnitID) {
	ownerID = UnitID((uint32(id) >> BombUnitIDShift) & BombUnitIDMask)
	turn = int((uint32(id) >> BombTurnShift) & BombTurnMask)
	counter = int((uint32(id) >> BombCounterShift) & BombCounterMask)
	return turn, counter, ownerID
}
