package engine

const (
	// SystemUnitID represents environmental or game-engine authoritative actions
	// mapped explicitly to Team 0, Player 0 for sudden-death bomb drops.
	SystemUnitID UnitID = 0

	// UnitID Configuration
	UnitLocalShift = 0
	UnitTeamShift  = 4

	UnitLocalMask uint8 = 0x0F // Isolates the 4 bits for local unit index
	UnitTeamMask  uint8 = 0x0F // Isolates the 4 bits for team after shifting

	// BombID Configuration
	BombCounterShift = 0
	BombTurnShift    = 16
	BombUnitIDShift  = 24

	BombCounterMask uint32 = 0xFFFF // Isolates the 16 bits for counter
	BombTurnMask    uint32 = 0xFF   // Isolates the 8 bits for turn after shifting
	BombUnitIDMask  uint32 = 0xFF   // Isolates the 8 bits for unit ID after shifting
)

// NewUnitID combines variables cleanly into the standard layout
func NewUnitID(teamID, counter int) UnitID {
	return UnitID((uint8(teamID) << UnitTeamShift) | (uint8(counter) << UnitLocalShift))
}

// Decode breaks down the UintID into teamID and index
func (id UnitID) Decode() (teamID int, index int) {
	teamID = int((uint8(id) >> UnitTeamShift) & UnitTeamMask)
	index = int(uint8(id) >> uint8(UnitLocalShift) & uint8(UnitLocalMask))
	return
}

// NewBombID combines variables cleanly into the standard layout
func NewBombID(turn int, counter int, unitID UnitID) BombID {
	return BombID(
		(uint32(unitID) << BombUnitIDShift) |
			(uint32(turn) << BombTurnShift) |
			(uint32(counter) << BombCounterShift),
	)
}

// Decode breaks down the BombID efficiently into clear values in one line
func (id BombID) Decode() (turn int, counter int, ownerID UnitID) {
	ownerID = UnitID((uint32(id) >> BombUnitIDShift) & BombUnitIDMask)
	turn = int((uint32(id) >> BombTurnShift) & BombTurnMask)
	counter = int((uint32(id) >> BombCounterShift) & BombCounterMask)
	return turn, counter, ownerID
}
