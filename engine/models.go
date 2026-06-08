package engine

type TerrainType int

const (
	TerrainPlain TerrainType = iota // Walkable
	TerrainBlock                    // Block; not walkable; flyable/jumpable
	TerrainTower                    // Tower; not walkable; not flyable/jumpable
	TerrainWater                    // Water not walkable; flyable/jumpable; bomb will be disppeared
	TerrainLava                     // Lava; not walkable; flyable/jumpable; bomb countdown will be set to 1
)

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

type Tile struct {
	Type         TerrainType
	OccupantType OccupantType
	OccupantID   int64 // ID of the occupant for cross reference
}

type SoftBlock struct {
	ID         int
	Position   Coordinate
	HiddenItem string // later implementation
}

type StagePreset struct {
	Name                string
	Description         string
	Width               int
	Height              int
	LayoutGrid          []string // Visual layout matrix. Each string in the slice represents a row of the grid
	SoftBlocks          []Coordinate
	P1StartingPositions [5]Coordinate // 31024, by default, P1 starts at the bottom side
	P2StartingPositions [5]Coordinate // 42013, by default, P2 starts at the top side
}

type SkillType uint32 // 32-bit is forseenable

const (
	SkillCanJump SkillType = 1 << iota // 1 (binary 00000001)
	SkillCanFly                        // 2 (binary 00000010)
)

type Archetype struct {
	Name         string
	BaseSpeed    int
	BombMaxRange int
	BombMinRange int
	BombPower    int
	MaxBombCount int
	BaseHP       int
	PresetSkills map[SkillType]bool
}

type UnitID uint8 // will be used later on
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
	HP           int // 1 = alive, 0 = dead, could be extended to more HP in later implementation
	Skills       map[SkillType]bool
	HasMoved     bool
	HasUsedSkill bool // a unit can only place a bomb or use a skill in a turn, but not both
}

type Bomb struct {
	ID        BombID
	OwnerID   UnitID // ID of the character who placed the bomb
	Position  Coordinate
	Range     int
	Countdown int // Turns until explosion
}

type GameCfg struct {
	StagePreset                 string   // Name of the stage preset to use
	P1Teams                     []string // List of archetype names for Player 1's units
	P2Teams                     []string // List of archetype names for Player 2's units
	MaxTurns                    int
	AllowResetTurn              bool // false = no way back. Multiplaying gaming experience will be changed accordingly
	SuddenDeath                 bool // false = draw if Turn >= MaxTurn
	GlobalSpeedOverride         int  // For testing purposes, allows overriding the default speed for all units. 0 means no override.
	GlobalBombCountdownOverride int  // For testing purposes, allows overriding the default bomb countdown. 0 means no override.
	GlobalBombMaxRangeOverride  int  // For testing purposes, allows overriding the default max bomb range. 0 means no override.
}

type GameState struct {
	Turn            int // Turn counter, starting from 1
	ActiveTeam      int
	TurnBombCounter int // To record how many bombs placed during the turn
	Grid            [][]Tile
	Units           map[UnitID]*Unit   // Keyed by Unit ID
	Bombs           map[BombID]*Bomb   // Keyed by Bomb ID
	SoftBlocks      map[int]*SoftBlock // Keyed by SoftBlock ID
	TurnCommands    []TurnCommand      // Commands issued by players for the current turn
}

// TurnCommand is the unified interface for all GameState actions.
type TurnCommand interface {
	isTurnCommand()
}

// MoveCommand packages everything needed to walk across the board.
type MoveCommand struct {
	UnitID UnitID
	Target Coordinate
}

func (MoveCommand) isTurnCommand() {}

// PlaceBombCommand packages everything needed to deploy bomb.
type PlaceBombCommand struct {
	UnitID UnitID
	Target Coordinate
}

func (PlaceBombCommand) isTurnCommand() {}

// GameEvent is the unified interface for all PlaybackLogs actions for frontend rendering.
type GameEvent interface {
	isGameEvent()
}

type UnitMovedEvent struct {
	UnitID UnitID
	From   Coordinate
	To     Coordinate
}

func (UnitMovedEvent) isGameEvent() {}

type UnitDamagedEvent struct {
	UnitID UnitID
	NewHP  int
}

func (UnitDamagedEvent) isGameEvent() {}

type UnitDiedEvent struct {
	UnitID UnitID
}

func (UnitDiedEvent) isGameEvent() {}

type BombPlacedEvent struct {
	UnitID    UnitID
	BombID    BombID
	Position  Coordinate
	Range     int
	Countdown int
}

func (BombPlacedEvent) isGameEvent() {}

type BombExplodedEvent struct {
	BombID            BombID
	AffectedPositions []Coordinate
}

func (BombExplodedEvent) isGameEvent() {}

type SoftBlockDestroyedEvent struct {
	SoftBlockID int
	Position    Coordinate
}

func (SoftBlockDestroyedEvent) isGameEvent() {}

type ItemDestroyedEvent struct {
	ItemID   int
	Position Coordinate
}

func (ItemDestroyedEvent) isGameEvent() {}

type MatchEndedEvent struct {
	WinnerTeamID int // TeamID, -1 means draw game
	IsDraw       bool
}

func (MatchEndedEvent) isGameEvent() {}

type Match struct {
	GameCfg      GameCfg
	TrueState    *GameState
	WorkingState *GameState
	PlaybackLog  []GameEvent
	WinnerTeamID int
}

type VictoryResult int

const (
	MatchInProgress VictoryResult = iota
	MatchWin
	MatchDraw
)

type StepPattern int

const (
	PatternCardinal StepPattern = iota // Up, Down, Left, Right
)

type PassFlag uint8

const (
	PassUnits      PassFlag = 1 << iota // 1  (binary 00000001)
	PassSoftBlocks                      // 2  (binary 00000010)
	PassHardBlocks                      // 4  (binary 00000100)
	PassItems                           // 8  (binary 00001000)
	PassBombs                           // 16 (binary 00010000)
)

type MovementRule struct {
	MaxSteps              int // Maximum number of steps allowed; -1 for unlimited
	Pattern               StepPattern
	CanTurn               bool     // If true, the unit can change direction at each step; if false, the unit must move in a straight line
	PassPermissions       PassFlag // Bitmask for what types of obstacles the unit can pass through
	StopOnNonUnitOccupant bool     // If true, the unit will stop if it encounters any non-unit occupant; if false, it will stop before the tile with the non-unit occupant
}
