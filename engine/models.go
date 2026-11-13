package engine

import (
	"encoding/json/v2"
	"fmt"
	"maps"
	"math"
	"slices"
)

type TerrainType int

const (
	TerrainPlain TerrainType = iota
	// TerrainBlock is a solid wall; not walkable, flyable, or jumpable.
	TerrainBlock
	// TerrainTower is a high wall; not walkable, not flyable, not jumpable.
	TerrainTower
	// TerrainWater is not walkable but flyable/jumpable; bombs disappear on contact.
	TerrainWater
	// TerrainLava is not walkable but flyable/jumpable; bomb countdown forced to 1.
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

func (t TerrainType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// Coordinate is a grid position; (0,0) is the top-left tile.
type Coordinate struct {
	X int `json:"x"`
	Y int `json:"y"`
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

func (o OccupantType) MarshalText() ([]byte, error) {
	return []byte(o.String()), nil
}

const (
	Team1 = 1
	Team2 = 2
)

type Tile struct {
	Type         TerrainType  `json:"type"`
	OccupantType OccupantType `json:"occupantType"`
	OccupantID   int64        `json:"occupantId"`
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
	ID       int        `json:"id"`
	Position Coordinate `json:"position"`
}

type StagePreset struct {
	Name                string        `json:"name"`
	Description         string        `json:"description"`
	Width               int           `json:"width"`
	Height              int           `json:"height"`
	MaxTurns            int           `json:"maxTurns"`
	LayoutGrid          []string      `json:"-"` // Visual layout matrix; each string is a row (Y), each char a column (X)
	SoftBlocks          []Coordinate  `json:"-"`
	P1StartingPositions [5]Coordinate `json:"-"`
	P2StartingPositions [5]Coordinate `json:"-"`
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

// MarshalJSON encodes the set as the names of its skills, never null.
func (s skillType) MarshalJSON() ([]byte, error) {
	var names []string
	for _, skill := range allSkills() {
		if skill&s != 0 {
			names = append(names, skill.String())
		}
	}
	return json.Marshal(names)
}

type Archetype struct {
	Name         string    `json:"name"`
	BaseSpeed    int       `json:"speed"`
	BombMaxRange int       `json:"bombMaxRange"`
	BombMinRange int       `json:"-"`
	BombPower    int       `json:"-"`
	MaxBombCount int       `json:"-"`
	BaseHP       int       `json:"-"`
	PresetSkills skillType `json:"skills"`
	Selectable   bool      `json:"-"`
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

func (r UnitRole) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

func (r *UnitRole) UnmarshalText(text []byte) error {
	i := slices.Index(roleNames(), string(text))
	if i < 0 {
		return fmt.Errorf("%w: %q", errInvalidUnitRole, text)
	}
	*r = UnitRole(i)
	return nil
}

// UnitID packs a team and a player index; see NewUnitID.
type UnitID uint8

// BombID packs owner, turn and counter; see NewBombID.
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
	Team         int // 1 = P1, 2 = P2 / COM
	HP           int // alive while above 0; every archetype currently has 1
	Skills       skillType
	Role         UnitRole
	HasMoved     bool
	HasUsedSkill bool // set after placing a bomb or using a skill; reset each turn
}

func (u Unit) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID           UnitID     `json:"id"`
		Type         string     `json:"type"`
		Position     Coordinate `json:"position"`
		Speed        int        `json:"speed"`
		BombMaxRange int        `json:"bombMaxRange"`
		BombPower    int        `json:"bombPower"`
		MaxBombCount int        `json:"maxBombCount"`
		BombUsed     int        `json:"bombUsed"`
		Team         int        `json:"team"`
		HP           int        `json:"hp"`
		Skills       skillType  `json:"skills"`
		HasMoved     bool       `json:"hasMoved"`
		HasUsedSkill bool       `json:"hasUsedSkill"`
	}{
		u.ID,
		u.Type.Name,
		u.Position,
		u.Speed,
		u.BombMaxRange,
		u.BombPower,
		u.MaxBombCount,
		u.BombUsed,
		u.Team,
		u.HP,
		u.Skills,
		u.HasMoved,
		u.HasUsedSkill,
	})
}

type Bomb struct {
	ID        BombID     `json:"id"`
	OwnerID   UnitID     `json:"ownerId"`
	Position  Coordinate `json:"position"`
	Range     int        `json:"range"`
	Countdown int        `json:"countdown"` // Turns remaining until detonation; <0 for non-countdown bombs
}

type TeamSlot struct {
	Archetype string   `json:"archetype"`
	Role      UnitRole `json:"role"`
}

type GameCfg struct {
	VSCpu                       bool       `json:"vsCpu"`
	StagePreset                 string     `json:"stagePreset"`
	P1Slots                     []TeamSlot `json:"p1Slots"`
	P2Slots                     []TeamSlot `json:"p2Slots"`
	MaxTurns                    int        `json:"maxTurns"` // Turn limit; 0 = instant sudden death
	AllowResetTurn              bool       `json:"allowResetTurn"`
	GlobalSpeedOverride         int        `json:"-"`
	GlobalBombCountdownOverride int        `json:"-"`
	GlobalBombMaxRangeOverride  int        `json:"-"` // Test override for bomb max range (0 = disabled)
}

type GameState struct {
	Turn            int // Current turn number (starts at 1)
	InSuddenDeath   bool
	ActiveTeam      int      // Team whose turn it is (1 or 2)
	TurnBombCounter int      // Bombs placed this turn (for BombID generation)
	Grid            [][]Tile // Board matrix [Y][X]
	Units           map[UnitID]*Unit
	Bombs           map[BombID]*Bomb
	SoftBlocks      map[int]*SoftBlock
	TurnCommands    []TurnCommand
	TurnStartEvents []GameEvent // Hazard events of the current turn start
}

// MarshalJSON emits the client-facing shape, with the entity maps flattened to arrays.
func (gs *GameState) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Turn            int           `json:"turn"`
		InSuddenDeath   bool          `json:"inSuddenDeath"`
		ActiveTeam      int           `json:"activeTeam"`
		Grid            [][]Tile      `json:"grid"`
		Units           []*Unit       `json:"units"`
		Bombs           []*Bomb       `json:"bombs"`
		SoftBlocks      []*SoftBlock  `json:"softBlocks"`
		TurnCommands    []TurnCommand `json:"turnCommands"`
		TurnStartEvents []GameEvent   `json:"turnStartEvents"`
	}{
		gs.Turn,
		gs.InSuddenDeath,
		gs.ActiveTeam,
		gs.Grid,
		slices.Collect(maps.Values(gs.Units)),
		slices.Collect(maps.Values(gs.Bombs)),
		slices.Collect(maps.Values(gs.SoftBlocks)),
		gs.TurnCommands,
		gs.TurnStartEvents,
	})
}

type CPUTurnPhase int

const (
	TurnPhaseIdle CPUTurnPhase = iota // CPU is idling, or no CPU involved at all
	TurnPhasePlanning
	TurnPhaseReady
)

func (c CPUTurnPhase) String() string {
	switch c {
	case TurnPhaseIdle:
		return "TurnPhaseIdle"
	case TurnPhasePlanning:
		return "TurnPhasePlanning"
	case TurnPhaseReady:
		return "TurnPhaseReady"
	default:
		return "TurnPhaseUnknown"
	}
}

func (c CPUTurnPhase) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

type cpuState struct {
	Phase                 CPUTurnPhase
	PlanGameEvents        []GameEvent
	ResolveTurnGameEvents []GameEvent
}

type Match struct {
	GameCfg      GameCfg
	TrueState    *GameState
	WorkingState *GameState  // Sandbox for mid-turn planning
	PlaybackLog  []GameEvent // events since the last ResolveTurn
	CPU          cpuState
	WinnerTeamID int // 0 = in progress, 1/2 = winner, -1 = draw
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
	PassUnits      PassFlag = 1 << iota // walk through other units
	PassSoftBlocks                      // walk through soft blocks
	PassHardBlocks                      // walk through hard blocks (TerrainBlock)
	PassItems                           // walk through items
	PassBombs                           // walk through bombs
)

type MovementRule struct {
	MaxSteps              int // Max steps; -1 = unlimited
	Pattern               stepPattern
	CanTurn               bool     // True = can change direction mid-path
	PassPermissions       PassFlag // obstacle types the mover may walk through
	StopOnNonUnitOccupant bool     // True = stop on first non-unit (bomb, block, item); False = stop before it
}
