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

type ObjectType int

const (
	ObjectNone ObjectType = iota
	ObjectCharacter
	ObjectBomb
	ObjectSoftBlock // later implementation
	ObjectPowerUp   // later implementation
)

type Cell struct {
	Type         TerrainType
	OccupantType ObjectType
	OccupantID   int // ID of the occupant for cross reference
}

type MapPreset struct {
	Name                string
	Description         string
	Width               int
	Height              int
	Terrains            [][]TerrainType
	SoftBlocks          []Coordinate
	P1StartingPositions [5]Coordinate // 31024, by default, P1 starts at the bottom side
	P2StartingPositions [5]Coordinate // 42013, by default, P2 starts at the top side
}

type SkillType int

const (
	SkillCanJump SkillType = iota
	SkillCanFly
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

type Unit struct {
	ID           int
	Type         Archetype
	Position     Coordinate
	Speed        int
	BombMaxRange int
	BombMinRange int
	BombPower    int
	MaxBombCount int
	BombUsed     int
	Team         int // 1 = Player 1, 2 = Player 2 / AI
	HP           int // 1 = alive, 0 = dead, could be extended to more HP in later implementation
	Skills       map[SkillType]bool
}

type Bomb struct {
	ID        int
	OwnerID   int // ID of the character who placed the bomb
	Position  Coordinate
	Range     int
	Countdown int // Turns until explosion
}

type GameState struct {
	Grid         [][]Cell
	Units        map[int]*Unit // Keyed by Unit ID
	Bombs        map[int]*Bomb // Keyed by Bomb ID
	TurnCommands []TurnCommand // Commands issued by players for the current turn
	Turn         int           // 1 = Player 1's turn, 2 = Player 2's turn
}

type ActionType int

const (
	ActionMove ActionType = iota
	ActionPlaceBomb
	ActionEndTurn
)

type TurnCommand struct {
	Action         ActionType
	ActorID        int
	TargetPosition Coordinate // For move and place bomb actions
}
