package engine

const BombDefaultCountDown int = 5 // default countdown for Bomb
const BombDefaultPower int = 2     // default power of Bomb
const SuddenDeathBombs int = 2     // Sudden Death, maximum bombs to drop during Sudden Death

// ArchetypeRegistry stores the base templates. We keep it unexported (lowercase 'a')
// so code outside this package cannot alter the map items directly.
var archetypesRegistry = map[string]Archetype{
	"King": {
		Name:         "King",
		BaseSpeed:    1,
		BombMaxRange: 2,
		BombMinRange: 1,
		BombPower:    2,
		MaxBombCount: 1,
		BaseHP:       1,
		PresetSkills: map[SkillType]bool{},
		Selectable:   false,
	},
	"Fighter": {
		Name:         "Fighter",
		BaseSpeed:    2,
		BombMaxRange: 2,
		BombMinRange: 1,
		BombPower:    2,
		MaxBombCount: 1,
		BaseHP:       1,
		PresetSkills: map[SkillType]bool{},
		Selectable:   true,
	},
	"Witch": {
		Name:         "Witch",
		BaseSpeed:    1,
		BombMaxRange: 3,
		BombMinRange: 1,
		BombPower:    2,
		MaxBombCount: 1,
		BaseHP:       1,
		PresetSkills: map[SkillType]bool{},
		Selectable:   true,
	},
	"Bandit": {
		Name:         "Bandit",
		BaseSpeed:    3,
		BombMaxRange: 1,
		BombMinRange: 1,
		BombPower:    2,
		MaxBombCount: 1,
		BaseHP:       1,
		PresetSkills: map[SkillType]bool{},
		Selectable:   true,
	},
}

// GetArchetype mimics a read-only database query.
// It returns the archetype and a boolean indicating whether the archetype exists.
func GetArchetype(name string) (Archetype, bool) {
	archetype, exists := archetypesRegistry[name]
	return archetype, exists
}

var stagePresetsRegistry = map[string]StagePreset{
	"MAP01": {
		Name:        "Plain",
		Description: "A simple open field with no obstacles.",
		Width:       9,
		Height:      9,
		MaxTurns:    60,
		LayoutGrid: []string{
			".........",
			".........",
			".........",
			".........",
			".........",
			".........",
			".........",
			".........",
			".........",
		},
		SoftBlocks:          []Coordinate{},
		P1StartingPositions: [5]Coordinate{{4, 8}, {3, 8}, {5, 8}, {2, 8}, {6, 8}},
		P2StartingPositions: [5]Coordinate{{4, 0}, {5, 0}, {3, 0}, {6, 0}, {2, 0}},
	},
	"MAP02": {
		Name:        "Standard",
		Description: "A balanced stage with hard blocks evenly distributed.",
		Width:       9,
		Height:      9,
		MaxTurns:    15,
		LayoutGrid: []string{
			".........",
			".B.B.B.B.",
			".........",
			".B.B.B.B.",
			".........",
			".B.B.B.B.",
			".........",
			".B.B.B.B.",
			".........",
		},
		SoftBlocks:          []Coordinate{},
		P1StartingPositions: [5]Coordinate{{4, 8}, {3, 8}, {5, 8}, {2, 8}, {6, 8}},
		P2StartingPositions: [5]Coordinate{{4, 0}, {5, 0}, {3, 0}, {6, 0}, {2, 0}},
	},
	"MAP03": {
		Name:        "Divided",
		Description: "A stage divided by soft block",
		Width:       9,
		Height:      9,
		MaxTurns:    20,
		LayoutGrid: []string{
			".........",
			".........",
			".........",
			".........",
			".B.B.B.B.",
			".........",
			".........",
			".........",
			".........",
		},
		SoftBlocks:          []Coordinate{{0, 4}, {2, 4}, {4, 4}, {6, 4}, {8, 4}},
		P1StartingPositions: [5]Coordinate{{4, 8}, {3, 8}, {5, 8}, {2, 8}, {6, 8}},
		P2StartingPositions: [5]Coordinate{{4, 0}, {5, 0}, {3, 0}, {6, 0}, {2, 0}},
	},
}

// GetAllArchetypes gets all archestypes for game setup.
func GetAllArchetypes() []Archetype {
	results := []Archetype{}
	for _, arch := range archetypesRegistry {
		if arch.Selectable {
			results = append(results, arch)
		}
	}
	return results
}

// GetStagePreset mimics a read-only database query.
// It returns the stage preset and a boolean indicating whether the stage preset exists.
func GetStagePreset(name string) (StagePreset, bool) {
	stagePreset, exists := stagePresetsRegistry[name]
	return stagePreset, exists
}
