package engine

const (
	// SystemUnitID is the environmental actor, Team 0 Player 0.
	SystemUnitID UnitID = 0

	// UnitLocalShift places the player index in the low nibble of a UnitID.
	UnitLocalShift = 0
	// UnitTeamShift places the team in the high nibble of a UnitID.
	UnitTeamShift = 4

	UnitLocalMask uint8 = 0x0F
	UnitTeamMask  uint8 = 0x0F

	// BombCounterShift places the per-turn counter in the low 16 bits of a BombID.
	BombCounterShift = 0
	// BombTurnShift places the turn above the counter.
	BombTurnShift = 16
	// BombUnitIDShift places the owner UnitID in the top byte.
	BombUnitIDShift = 24

	BombCounterMask uint32 = 0xFFFF
	BombTurnMask    uint32 = 0xFF
	BombUnitIDMask  uint32 = 0xFF
)

// NewUnitID packs a team and a player index into a UnitID.
func NewUnitID(teamID, counter int) UnitID {
	team := uint8(teamID & int(UnitTeamMask))
	local := uint8(counter & int(UnitLocalMask))
	return UnitID((team << UnitTeamShift) | (local << UnitLocalShift))
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
			(uint32(turn&int(BombTurnMask)) << BombTurnShift) |
			(uint32(counter&int(BombCounterMask)) << BombCounterShift),
	)
}

// Decode extracts Turn, Counter, and Owner UnitID from a BombID.
func (id BombID) Decode() (turn int, counter int, ownerID UnitID) {
	ownerID = UnitID((uint32(id) >> BombUnitIDShift) & BombUnitIDMask)
	turn = int((uint32(id) >> BombTurnShift) & BombTurnMask)
	counter = int((uint32(id) >> BombCounterShift) & BombCounterMask)
	return turn, counter, ownerID
}
