package engine

const (
	// SystemUnitID is the environmental actor, Team 0 Player 0.
	SystemUnitID UnitID = 0

	// UnitID layout: team in the high nibble, player index in the low nibble.
	UnitLocalShift = 0
	UnitTeamShift  = 4

	UnitLocalMask uint8 = 0x0F
	UnitTeamMask  uint8 = 0x0F

	// BombID layout: owner UnitID, turn and per-turn counter packed into 32 bits.
	BombCounterShift = 0
	BombTurnShift    = 16
	BombUnitIDShift  = 24

	BombCounterMask uint32 = 0xFFFF
	BombTurnMask    uint32 = 0xFF
	BombUnitIDMask  uint32 = 0xFF
)

// NewUnitID packs a team and a player index into a UnitID.
func NewUnitID(teamID, counter int) UnitID {
	return UnitID((uint8(teamID) << UnitTeamShift) | (uint8(counter) << UnitLocalShift))
}

// Decode extracts TeamID and PlayerIndex from a UnitID.
func (id UnitID) Decode() (teamID int, index int) {
	teamID = int((uint8(id) >> UnitTeamShift) & UnitTeamMask)
	index = int(uint8(id) >> uint8(UnitLocalShift) & uint8(UnitLocalMask))
	return
}

// NewBombID packs an owner, a turn and a counter into a BombID.
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
