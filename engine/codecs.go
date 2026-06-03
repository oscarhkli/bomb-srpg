package engine

const (
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
// TODO: convert to UnitID type
func NewUnitID(teamID, counter int) int {
	return int((uint8(teamID) << UnitTeamShift) | (uint8(counter) << UnitLocalShift))
}

// DecodeUnitID breaks down the UintID into teamID and index
// TODO: convert to UnitID type
func DecodeUnitID(unitID int) (teamID int, index int) {
	teamID = int((uint8(unitID) >> UnitTeamShift) & UnitTeamMask)
	index = int(uint8(unitID) >> uint8(UnitLocalShift) & uint8(UnitLocalMask))
	return
}

// NewBombID combines variables cleanly into the standard layout
// TODO: convert to BombID type
func NewBombID(turn int, counter int, unitID int) int {
	return int(
		(uint32(unitID) << BombUnitIDShift) |
			(uint32(turn) << BombTurnShift) |
			(uint32(counter) << BombCounterShift),
	)
}

// DecodeBombID breaks down the BombID efficiently into clear values in one line
// TODO: convert to BombID type
func DecodeBombID(bombID int) (turn int, counter int, ownerID int) {
	ownerID = int((uint32(bombID) >> BombUnitIDShift) & BombUnitIDMask)
	turn = int((uint32(bombID) >> BombTurnShift) & BombTurnMask)
	counter = int((uint32(bombID) >> BombCounterShift) & BombCounterMask)
	return turn, counter, ownerID
}
