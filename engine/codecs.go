package engine

const (
	// SystemUnitID is the environmental actor, Team 0 Player 0.
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

func NewUnitID(teamID, counter int) UnitID {
	team := uint8(teamID & int(UnitTeamMask))
	local := uint8(counter & int(UnitLocalMask))
	return UnitID((team << UnitTeamShift) | (local << UnitLocalShift))
}

func (id UnitID) Decode() (teamID int, index int) {
	teamID = int((uint8(id) >> UnitTeamShift) & UnitTeamMask)
	index = int(uint8(id) >> uint8(UnitLocalShift) & uint8(UnitLocalMask))
	return
}

func NewBombID(turn int, counter int, unitID UnitID) BombID {
	return BombID(
		(uint32(unitID) << BombUnitIDShift) |
			(uint32(turn&int(BombTurnMask)) << BombTurnShift) |
			(uint32(counter&int(BombCounterMask)) << BombCounterShift),
	)
}

func (id BombID) Decode() (turn int, counter int, ownerID UnitID) {
	ownerID = UnitID((uint32(id) >> BombUnitIDShift) & BombUnitIDMask)
	turn = int((uint32(id) >> BombTurnShift) & BombTurnMask)
	counter = int((uint32(id) >> BombCounterShift) & BombCounterMask)
	return turn, counter, ownerID
}
