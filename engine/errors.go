package engine

import "errors"

var (
	ErrUnitNotFound       = errors.New("unit not found")
	ErrUnitDead           = errors.New("unit is dead")
	ErrNotActiveTeam      = errors.New("not active team")
	ErrAlreadyMoved       = errors.New("unit already moved this turn")
	ErrAlreadyUsedSkill   = errors.New("unit already used skill this turn")
	ErrOutOfMoveRange     = errors.New("target out of move range")
	ErrOutOfBombRange     = errors.New("target out of bomb range")
	ErrCellOccupied       = errors.New("cell occupied")
	ErrOutOfBombs         = errors.New("out of bombs")
	ErrUnsupportedCommand = errors.New("unsupported command type")
	ErrInvalidLanding     = errors.New("invalid landing position")

	ErrInvalidStagePreset = errors.New("invalid stage preset")
	ErrInvalidTeamSize    = errors.New("invalid team size")
	ErrMissingKing        = errors.New("missing king")
	ErrInvalidStageLayout = errors.New("invalid stage layout")
	ErrInvalidTerrain     = errors.New("invalid terrain")
	ErrUnknownArchetype   = errors.New("unknown archetype")

	ErrOutOfBounds = errors.New("out of bounds")
	ErrDesynced    = errors.New("desynced")
)
