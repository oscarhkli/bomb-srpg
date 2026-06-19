package engine

import "encoding/json"

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

// Coordinate represents a grid position using (X, Y) where (0,0) is top-left.
type Coordinate struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func (o TerrainType) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.String())
}

// OccupantType represents what occupies a tile (if anything).
type OccupantType int

const (
	OccupantNone      OccupantType = iota // Empty tile
	OccupantUnit                          // Player unit
	OccupantBomb                          // Active bomb
	OccupantSoftBlock                     // Destructible block
	OccupantItem                          // Item pickup (hidden in soft block)
)

func (o OccupantType) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.String())
}

// Tile represents a single cell on the game board combining terrain and occupant.
type Tile struct {
	Type         TerrainType  `json:"type"`
	OccupantType OccupantType `json:"occupantType"`
	OccupantID   int64        `json:"occupantId"` // Cross-reference ID for the occupant (UnitID, BombID, or SoftBlockID)
}

// SoftBlock represents a destructible block that may hide an item.
type SoftBlock struct {
	ID         int        `json:"id"`
	Position   Coordinate `json:"position"`
	HiddenItem string     `json:"-"` // Reserved for future item system
}

// StagePreset defines a complete map layout including terrain, soft blocks, and starting positions.
type StagePreset struct {
	Name                string
	Description         string
	Width               int
	Height              int
	LayoutGrid          []string // Visual layout matrix; each string is a row (Y), each char a column (X)
	SoftBlocks          []Coordinate
	P1StartingPositions [5]Coordinate // Default order: 3,1,0,2,4 (bottom side)
	P2StartingPositions [5]Coordinate // Default order: 4,2,0,1,3 (top side)
}

// SkillType is a bitmask for unit abilities (jump, fly, etc.).
type SkillType uint32

const (
	SkillCanJump SkillType = 1 << iota
	SkillCanFly
)

// Archetype defines the base template for a unit class (King, Fighter, Witch, etc.).
type Archetype struct {
	Name         string             `json:"name"`
	BaseSpeed    int                `json:"speed"`
	BombMaxRange int                `json:"bombMaxRange"`
	BombMinRange int                `json:"-"`
	BombPower    int                `json:"-"`
	MaxBombCount int                `json:"-"`
	BaseHP       int                `json:"-"`
	PresetSkills map[SkillType]bool `json:"-"`
	IsSelectable bool               `json:"-"`
}

func (a Archetype) MarshalJSON() ([]byte, error) {
	skills := []string{}
	for s, ok := range a.PresetSkills {
		if ok {
			skills = append(skills, s.String())
		}
	}
	return json.Marshal(struct {
		Name         string   `json:"name"`
		BaseSpeed    int      `json:"speed"`
		BombMaxRange int      `json:"bombMaxRange"`
		Skills       []string `json:"skills"`
	}{a.Name, a.BaseSpeed, a.BombMaxRange, skills})
}

// UnitID encodes (TeamID << 4) | PlayerIndex. Max 15 teams, 15 units per team.
// Value 0 is reserved for SystemUnitID (environmental actor).
type UnitID uint8

// BombID encodes (UnitID << 24) | (Turn << 16) | Counter. Unique per bomb placement.
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
	HP           int // 1 = alive, 0 = dead; extensible for multi-HP units
	Skills       map[SkillType]bool
	HasMoved     bool
	HasUsedSkill bool // True after placing bomb or using skill; resets each turn
}

func (u Unit) MarshalJSON() ([]byte, error) {
	skills := []string{}
	for s, ok := range u.Skills {
		if ok {
			skills = append(skills, s.String())
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
	OwnerID   UnitID     `json:"ownerId"` // Unit that placed this bomb
	Position  Coordinate `json:"position"`
	Range     int        `json:"range"`     // Explosion radius in tiles
	Countdown int        `json:"countdown"` // Turns remaining until detonation; <0 for non-countdown bombs
}

// GameCfg holds all configuration for a match.
type GameCfg struct {
	StagePreset                 string   `json:"stagePreset"`    // Stage preset name (e.g., "MAP01")
	P1Teams                     []string `json:"p1Teams"`        // Archetype names for Player 1 (1-5 units, first must be King)
	P2Teams                     []string `json:"p2Teams"`        // Archetype names for Player 2 (1-5 units, first must be King)
	MaxTurns                    int      `json:"maxTurns"`       // Turn limit; 0 = instant sudden death
	AllowResetTurn              bool     `json:"allowResetTurn"` // True = players can undo actions before committing
	SuddenDeath                 bool     `json:"suddenDeath"`    // True = spawn hazards after MaxTurns; False = draw at limit
	GlobalSpeedOverride         int      `json:"-"`              // Test override for all unit speeds (0 = disabled)
	GlobalBombCountdownOverride int      `json:"-"`              // Test override for bomb countdown (0 = disabled)
	GlobalBombMaxRangeOverride  int      `json:"-"`              // Test override for bomb max range (0 = disabled)
}

// GameState is the complete snapshot of a match at a point in time.
type GameState struct {
	Turn            int                // Current turn number (starts at 1)
	ActiveTeam      int                // Team whose turn it is (1 or 2)
	TurnBombCounter int                // Bombs placed this turn (for BombID generation)
	Grid            [][]Tile           // Board matrix [Y][X]
	Units           map[UnitID]*Unit   // All units by ID
	Bombs           map[BombID]*Bomb   // Active bombs by ID
	SoftBlocks      map[int]*SoftBlock // Soft blocks by ID
	TurnCommands    []TurnCommand      // Pending commands for current turn
}

func (gs GameState) MarshalJSON() ([]byte, error) {
	units := make([]*Unit, 0, len(gs.Units))
	for _, u := range gs.Units {
		units = append(units, u)
	}
	bombs := make([]*Bomb, 0, len(gs.Bombs))
	for _, b := range gs.Bombs {
		bombs = append(bombs, b)
	}
	softBlocks := make([]*SoftBlock, 0, len(gs.SoftBlocks))
	for _, sb := range gs.SoftBlocks {
		softBlocks = append(softBlocks, sb)
	}
	return json.Marshal(struct {
		Turn         int           `json:"turn"`
		ActiveTeam   int           `json:"activeTeam"`
		Grid         [][]Tile      `json:"grid"`
		Units        []*Unit       `json:"units"`
		Bombs        []*Bomb       `json:"bombs"`
		SoftBlocks   []*SoftBlock  `json:"softBlocks"`
		TurnCommands []TurnCommand `json:"turnCommands"`
	}{
		gs.Turn,
		gs.ActiveTeam,
		gs.Grid,
		units,
		bombs,
		softBlocks,
		gs.TurnCommands,
	})
}

// TurnCommand is the unified interface for all player actions during a turn.
type TurnCommand interface {
	isTurnCommand()
}

// MoveCommand instructs a unit to move to a target coordinate.
type MoveCommand struct {
	UnitID UnitID     `json:"unitId"`
	Target Coordinate `json:"target"`
}

func (MoveCommand) isTurnCommand() {}

// PlaceBombCommand instructs a unit to place a bomb at a target coordinate.
type PlaceBombCommand struct {
	UnitID UnitID     `json:"unitId"`
	Target Coordinate `json:"target"`
}

func (PlaceBombCommand) isTurnCommand() {}

// GameEvent is the unified interface for all events emitted during turn resolution.
type GameEvent interface {
	isGameEvent()
}

// UnitMovedEvent signals a unit relocated.
type UnitMovedEvent struct {
	UnitID UnitID     `json:"unitId"`
	From   Coordinate `json:"from"`
	To     Coordinate `json:"to"`
}

func (UnitMovedEvent) isGameEvent() {}

// UnitDamagedEvent signals a unit took damage.
type UnitDamagedEvent struct {
	UnitID UnitID `json:"unitId"`
	NewHP  int    `json:"newHp"`
}

func (UnitDamagedEvent) isGameEvent() {}

// UnitDiedEvent signals a unit was eliminated.
type UnitDiedEvent struct {
	UnitID UnitID `json:"unitId"`
}

func (UnitDiedEvent) isGameEvent() {}

// BombPlacedEvent signals a new bomb was deployed.
type BombPlacedEvent struct {
	UnitID    UnitID     `json:"unitId"`
	BombID    BombID     `json:"bombId"`
	Position  Coordinate `json:"position"`
	Range     int        `json:"range"`
	Countdown int        `json:"countdown"`
}

func (BombPlacedEvent) isGameEvent() {}

// BombExplodedEvent signals a bomb detonated and lists affected tiles.
type BombExplodedEvent struct {
	BombID            BombID       `json:"bombId"`
	AffectedPositions []Coordinate `json:"affectedPositions"`
}

func (BombExplodedEvent) isGameEvent() {}

// SoftBlockDestroyedEvent signals a soft block was destroyed.
type SoftBlockDestroyedEvent struct {
	SoftBlockID int        `json:"softBlockId"`
	Position    Coordinate `json:"position"`
}

func (SoftBlockDestroyedEvent) isGameEvent() {}

// ItemDestroyedEvent signals an item was destroyed (future phase).
type ItemDestroyedEvent struct {
	ItemID   int        `json:"itemId"`
	Position Coordinate `json:"position"`
}

func (ItemDestroyedEvent) isGameEvent() {}

// MatchEndedEvent signals the match concluded.
type MatchEndedEvent struct {
	WinnerTeamID int  `json:"winnerTeamId"`
	IsDraw       bool `json:"isDraw"`
}

func (MatchEndedEvent) isGameEvent() {}

// Match orchestrates a full game session: state, config, and event log.
type Match struct {
	GameCfg      GameCfg
	TrueState    *GameState  // Committed state
	WorkingState *GameState  // Sandbox for mid-turn planning
	PlaybackLog  []GameEvent // Events since last ResolveTurn
	WinnerTeamID int         // 0 = in progress, 1/2 = winner, -1 = draw
}

// VictoryResult represents the outcome of a victory check.
type VictoryResult int

const (
	MatchInProgress VictoryResult = iota
	MatchWin
	MatchDraw
)

// StepPattern defines the movement topology.
type StepPattern int

const (
	PatternCardinal StepPattern = iota // 4-directional (up/down/left/right)
)

// PassFlag is a bitmask for pathfinding passability rules.
type PassFlag uint8

const (
	PassUnits      PassFlag = 1 << iota // Can pass through other units
	PassSoftBlocks                      // Can pass through soft blocks
	PassHardBlocks                      // Can pass through hard blocks (TerrainBlock)
	PassItems                           // Can pass through items
	PassBombs                           // Can pass through bombs
)

// MovementRule configures pathfinding for a specific action (move, bomb placement, skill).
type MovementRule struct {
	MaxSteps              int // Max steps; -1 = unlimited
	Pattern               StepPattern
	CanTurn               bool     // True = can change direction mid-path
	PassPermissions       PassFlag // Bitmask of passable obstacle types
	StopOnNonUnitOccupant bool     // True = stop on first non-unit (bomb, block, item); False = stop before it
}
