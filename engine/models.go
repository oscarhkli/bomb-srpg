package engine

import (
	"fmt"
	"math"
	"slices"
)

type TerrainType int

const (
	TerrainPlain TerrainType = iota
	TerrainBlock
	TerrainTower
	TerrainWater
	TerrainLava
)

func (t TerrainType) String() string {
	switch t {
	case TerrainPlain:
		return "TerrainPlain"
	case TerrainBlock:
		return "TerrainBlock"
	case TerrainTower:
		return "TerrainTower"
	case TerrainWater:
		return "TerrainWater"
	case TerrainLava:
		return "TerrainLava"
	default:
		return "TerrainUnknown"
	}
}

// Coordinate is a grid position; (0,0) is the top-left tile.
type Coordinate struct {
	X int
	Y int
}

type OccupantType int

const (
	OccupantNone OccupantType = iota
	OccupantUnit
	OccupantBomb
	OccupantSoftBlock
	OccupantItem
)

func (o OccupantType) String() string {
	switch o {
	case OccupantNone:
		return "OccupantNone"
	case OccupantUnit:
		return "OccupantUnit"
	case OccupantBomb:
		return "OccupantBomb"
	case OccupantSoftBlock:
		return "OccupantSoftBlock"
	case OccupantItem:
		return "OccupantItem"
	default:
		return "OccupantUnknown"
	}
}

const (
	Team1 = 1
	Team2 = 2
)

type Tile struct {
	Type         TerrainType
	OccupantType OccupantType
	OccupantID   int64
}

// unitID returns OccupantID as a UnitID, wrapping values wider than the type.
func (t Tile) unitID() UnitID {
	return UnitID(t.OccupantID & int64(math.MaxUint8))
}

// BombID returns OccupantID as a BombID, wrapping values wider than the type.
func (t Tile) BombID() BombID {
	return BombID(t.OccupantID & int64(math.MaxUint32))
}

type SoftBlock struct {
	ID       int
	Position Coordinate
}

type StagePreset struct {
	Name        string
	Description string
	Width       int
	Height      int
	MaxTurns    int
	// Visual layout matrix; each string is a row (Y), each char a column (X)
	LayoutGrid          []string
	SoftBlocks          []Coordinate
	P1StartingPositions [5]Coordinate
	P2StartingPositions [5]Coordinate
}

type skillType uint32

const skillNone skillType = 0
const (
	skillCanJump skillType = 1 << iota
	skillCanFly
)

func allSkills() []skillType {
	return []skillType{skillCanJump, skillCanFly}
}

func (s skillType) String() string {
	switch s {
	case skillNone:
		return "None"
	case skillCanFly:
		return "Fly"
	case skillCanJump:
		return "Jump"
	default:
		return "Unknown"
	}
}

func (s skillType) Names() []string {
	names := []string{}
	for _, skill := range allSkills() {
		if skill&s != 0 {
			names = append(names, skill.String())
		}
	}
	return names
}

type Archetype struct {
	Name         string
	BaseSpeed    int
	BombMaxRange int
	BombMinRange int
	BombPower    int
	MaxBombCount int
	BaseHP       int
	PresetSkills skillType
	Selectable   bool
}

type UnitRole int

const (
	RoleNormal UnitRole = iota
	RoleKing
	RoleBoss
)

func roleNames() []string {
	return []string{RoleNormal: "Normal", RoleKing: "King", RoleBoss: "Boss"}
}

func (r UnitRole) String() string {
	names := roleNames()
	if r < 0 || int(r) >= len(names) {
		return "Unknown"
	}
	return names[r]
}

func (r *UnitRole) UnmarshalText(text []byte) error {
	i := slices.Index(roleNames(), string(text))
	if i < 0 {
		return fmt.Errorf("%w: %q", errInvalidUnitRole, text)
	}
	*r = UnitRole(i)
	return nil
}

type UnitID uint8

type BombID uint32

type Unit struct {
	ID           UnitID
	Type         Archetype
	Position     Coordinate
	Speed        int
	BombMaxRange int
	BombMinRange int
	BombPower    int
	MaxBombCount int
	BombUsed     int
	// 1 = P1, 2 = P2 / COM
	Team int
	// alive while above 0; every archetype currently has 1
	HP           int
	Skills       skillType
	Role         UnitRole
	HasMoved     bool
	HasUsedSkill bool
}

type Bomb struct {
	ID       BombID
	OwnerID  UnitID
	Position Coordinate
	Range    int
	// Turns remaining until detonation; <0 for non-countdown bombs
	Countdown int
}

type TeamSlot struct {
	Archetype string
	Role      UnitRole
}

type GameCfg struct {
	VSCpu       bool
	StagePreset string
	P1Slots     []TeamSlot
	P2Slots     []TeamSlot
	// Turn limit; 0 = instant sudden death
	MaxTurns                    int
	AllowResetTurn              bool
	GlobalSpeedOverride         int
	GlobalBombCountdownOverride int
	GlobalBombMaxRangeOverride  int
}

type GameState struct {
	Turn            int
	InSuddenDeath   bool
	ActiveTeam      int
	TurnBombCounter int
	// Board matrix [Y][X]
	Grid            [][]Tile
	Units           map[UnitID]*Unit
	Bombs           map[BombID]*Bomb
	SoftBlocks      map[int]*SoftBlock
	TurnCommands    []TurnCommand
	TurnStartEvents []GameEvent
}

type Match struct {
	GameCfg      GameCfg
	TrueState    *GameState
	WorkingState *GameState
	// events since the last ResolveTurn
	PlaybackLog []GameEvent
	// 0 = in progress, 1/2 = winner, -1 = draw
	WinnerTeamID int
}

const (
	matchInProgress = 0
	matchDrawn      = -1
)

type stepPattern int

const (
	patternCardinal stepPattern = iota
)

type PassFlag uint8

const (
	PassUnits PassFlag = 1 << iota
	PassSoftBlocks
	PassHardBlocks
	PassItems
	PassBombs
)

type MovementRule struct {
	// Max steps; -1 = unlimited
	MaxSteps        int
	Pattern         stepPattern
	CanTurn         bool
	PassPermissions PassFlag
	// True = stop on first non-unit (bomb, block, item); False = stop before it
	StopOnNonUnitOccupant bool
}
