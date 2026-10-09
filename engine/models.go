package engine

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
)

// TerrainType represents the base terrain of a tile.
type TerrainType int

const (
	// TerrainPlain is walkable by all units.
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

// String returns the name used in logs and JSON.
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

// MarshalJSON encodes the terrain as its String name.
func (o TerrainType) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.String())
}

// Coordinate represents a grid position using (X, Y) where (0,0) is top-left.
type Coordinate struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// OccupantType represents what occupies a tile (if anything).
type OccupantType int

const (
	OccupantNone OccupantType = iota
	OccupantUnit
	OccupantBomb
	OccupantSoftBlock
	OccupantItem
)

// String returns the name used in logs and JSON.
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

// MarshalJSON encodes the occupant type as its String name.
func (o OccupantType) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.String())
}

// Tile represents a single cell on the game board combining terrain and occupant.
type Tile struct {
	Type         TerrainType  `json:"type"`
	OccupantType OccupantType `json:"occupantType"`
	OccupantID   int64        `json:"occupantId"` // UnitID, BombID or SoftBlockID, per OccupantType
}

// SoftBlock represents a destructible block that may hide an item.
type SoftBlock struct {
	ID       int        `json:"id"`
	Position Coordinate `json:"position"`
}

// StagePreset defines a complete map layout including terrain, soft blocks, and starting positions.
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

// SkillType is a bitmask for unit abilities (jump, fly, etc.).
type SkillType uint32

const SkillNone SkillType = 0
const (
	SkillCanJump SkillType = 1 << iota
	SkillCanFly
)

func allSkills() []SkillType {
	return []SkillType{SkillCanJump, SkillCanFly}
}

// String returns the name used in logs and JSON.
func (s SkillType) String() string {
	switch s {
	case SkillNone:
		return "None"
	case SkillCanFly:
		return "Fly"
	case SkillCanJump:
		return "Jump"
	default:
		return "Unknown"
	}
}

// Archetype defines the base template for a unit class (King, Fighter, Witch, etc.).
type Archetype struct {
	Name         string
	BaseSpeed    int
	BombMaxRange int
	BombMinRange int
	BombPower    int
	MaxBombCount int
	BaseHP       int
	PresetSkills SkillType
	Selectable   bool
}

// MarshalJSON emits the client-facing shape.
func (a Archetype) MarshalJSON() ([]byte, error) {
	skills := []string{}
	for _, skill := range allSkills() {
		if (skill & a.PresetSkills) != 0 {
			skills = append(skills, skill.String())
		}
	}
	return json.Marshal(struct {
		Name         string   `json:"name"`
		BaseSpeed    int      `json:"speed"`
		BombMaxRange int      `json:"bombMaxRange"`
		Skills       []string `json:"skills"`
	}{a.Name, a.BaseSpeed, a.BombMaxRange, skills})
}

// UnitRoles defines if the Unit is King, Boss, or just a Normal Unit
type UnitRole int

const (
	RoleNormal UnitRole = iota
	RoleKing
	RoleBoss
)

// String returns the name used in logs and JSON.
func (o UnitRole) String() string {
	switch o {
	case RoleNormal:
		return "Normal"
	case RoleKing:
		return "King"
	case RoleBoss:
		return "Boss"
	default:
		return "Unknown"
	}
}

// MarshalJSON encodes the role as its String name.
func (o UnitRole) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.String())
}

// UnmarshalJSON decodes a role from its String name.
func (r *UnitRole) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch s {
	case "Normal":
		*r = RoleNormal
	case "King":
		*r = RoleKing
	case "Boss":
		*r = RoleBoss
	default:
		return fmt.Errorf("%w: %q", ErrInvalidUnitRole, s)
	}
	return nil
}

// UnitID packs a team and a player index; see NewUnitID.
type UnitID uint8

// BombID packs owner, turn and counter; see NewBombID.
type BombID uint32

// Unit represents a single controllable character on the board.
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
	Skills       SkillType
	Role         UnitRole
	HasMoved     bool
	HasUsedSkill bool // set after placing a bomb or using a skill; reset each turn
}

// MarshalJSON emits the client-facing shape.
func (u Unit) MarshalJSON() ([]byte, error) {
	skills := []string{}
	for _, skill := range allSkills() {
		if (skill & u.Skills) != 0 {
			skills = append(skills, skill.String())
		}
	}
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
		Skills       []string   `json:"skills"`
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
		skills,
		u.HasMoved,
		u.HasUsedSkill,
	})
}

// Bomb represents an active explosive on the board.
type Bomb struct {
	ID        BombID     `json:"id"`
	OwnerID   UnitID     `json:"ownerId"` // unit that placed the bomb
	Position  Coordinate `json:"position"`
	Range     int        `json:"range"`     // Explosion radius in tiles
	Countdown int        `json:"countdown"` // Turns remaining until detonation; <0 for non-countdown bombs
}

// TeamSlot pairs an archetype name with the role it fills in a team.
type TeamSlot struct {
	Archetype string   `json:"archetype"`
	Role      UnitRole `json:"role"`
}

// GameCfg holds all configuration for a match.
type GameCfg struct {
	VSCpu                       bool       `json:"vsCpu"`       // true = the second team is CPU-controlled
	StagePreset                 string     `json:"stagePreset"` // stage preset name, e.g. "Plain"
	P1Slots                     []TeamSlot `json:"p1Slots"`
	P2Slots                     []TeamSlot `json:"p2Slots"`
	MaxTurns                    int        `json:"maxTurns"`       // Turn limit; 0 = instant sudden death
	AllowResetTurn              bool       `json:"allowResetTurn"` // true = players can undo planned actions before commit
	GlobalSpeedOverride         int        `json:"-"`              // Test override for all unit speeds (0 = disabled)
	GlobalBombCountdownOverride int        `json:"-"`              // Test override for bomb countdown (0 = disabled)
	GlobalBombMaxRangeOverride  int        `json:"-"`              // Test override for bomb max range (0 = disabled)
}

// GameState is the complete snapshot of a match at a point in time.
type GameState struct {
	Turn            int                // Current turn number (starts at 1)
	InSuddenDeath   bool               // Indicate if the current turn is in Sudden Death
	ActiveTeam      int                // Team whose turn it is (1 or 2)
	TurnBombCounter int                // Bombs placed this turn (for BombID generation)
	Grid            [][]Tile           // Board matrix [Y][X]
	Units           map[UnitID]*Unit   // All units by ID
	Bombs           map[BombID]*Bomb   // Active bombs by ID
	SoftBlocks      map[int]*SoftBlock // Soft blocks by ID
	TurnCommands    []TurnCommand      // Pending commands for current turn
}

func nonNilValues[K comparable, V any](m map[K]V) []V {
	return slices.AppendSeq(make([]V, 0, len(m)), maps.Values(m))
}

// MarshalJSON emits the client-facing shape; every slice is non-nil so it encodes as [].
func (gs GameState) MarshalJSON() ([]byte, error) {
	units := nonNilValues(gs.Units)
	bombs := nonNilValues(gs.Bombs)
	softBlocks := nonNilValues(gs.SoftBlocks)
	turnCommands := append(make([]TurnCommand, 0, len(gs.TurnCommands)), gs.TurnCommands...)
	return json.Marshal(struct {
		Turn          int           `json:"turn"`
		InSuddenDeath bool          `json:"inSuddenDeath"`
		ActiveTeam    int           `json:"activeTeam"`
		Grid          [][]Tile      `json:"grid"`
		Units         []*Unit       `json:"units"`
		Bombs         []*Bomb       `json:"bombs"`
		SoftBlocks    []*SoftBlock  `json:"softBlocks"`
		TurnCommands  []TurnCommand `json:"turnCommands"`
	}{
		gs.Turn,
		gs.InSuddenDeath,
		gs.ActiveTeam,
		gs.Grid,
		units,
		bombs,
		softBlocks,
		turnCommands,
	})
}

// CPUTurnPhase represents what CPU is operating in a Match. VS CPU only.
type CPUTurnPhase int

const (
	TurnPhaseIdle CPUTurnPhase = iota // CPU is idling, or no CPU involved at all
	TurnPhasePlanning
	TurnPhaseReady
)

// String returns the name used in logs and JSON.
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

// MarshalJSON encodes the phase as its String name.
func (o CPUTurnPhase) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.String())
}

// CPUState manages the TurnPhase state and pending GameEvent for VS CPU.
type CPUState struct {
	Phase                 CPUTurnPhase
	PlanGameEvents        []GameEvent
	ResolveTurnGameEvents []GameEvent
}

// Match orchestrates a full game session: state, config, and event log.
type Match struct {
	GameCfg      GameCfg
	TrueState    *GameState  // Committed state
	WorkingState *GameState  // Sandbox for mid-turn planning
	PlaybackLog  []GameEvent // events since the last ResolveTurn
	CPU          CPUState
	WinnerTeamID int // 0 = in progress, 1/2 = winner, -1 = draw
}

const (
	MatchInProgress = 0
	MatchDrawn      = -1
)

// StepPattern defines the movement topology.
type StepPattern int

const (
	PatternCardinal StepPattern = iota
)

// PassFlag is a bitmask for pathfinding passability rules.
type PassFlag uint8

const (
	PassUnits      PassFlag = 1 << iota // walk through other units
	PassSoftBlocks                      // walk through soft blocks
	PassHardBlocks                      // walk through hard blocks (TerrainBlock)
	PassItems                           // walk through items
	PassBombs                           // walk through bombs
)

// MovementRule configures pathfinding for a specific action (move, bomb placement, skill).
type MovementRule struct {
	MaxSteps              int // Max steps; -1 = unlimited
	Pattern               StepPattern
	CanTurn               bool     // True = can change direction mid-path
	PassPermissions       PassFlag // obstacle types the mover may walk through
	StopOnNonUnitOccupant bool     // True = stop on first non-unit (bomb, block, item); False = stop before it
}
