package engine

// BombDefaultCountDown is the countdown in turns of a newly placed bomb.
const BombDefaultCountDown int = 5

// BombDefaultPower is the default explosion range in tiles.
const BombDefaultPower int = 2

// SuddenDeathBombs is the maximum number of bombs dropped per Sudden Death turn.
const SuddenDeathBombs int = 2

// archetypeRegistry stores the base templates of Archetypes.
func archetypesRegistry() []Archetype {
	return []Archetype{
		{
			Name:         "King",
			BaseSpeed:    1,
			BombMaxRange: 2,
			BombMinRange: 1,
			BombPower:    2,
			MaxBombCount: 1,
			BaseHP:       1,
			PresetSkills: SkillNone,
			Selectable:   false,
		},
		{
			Name:         "Fighter",
			BaseSpeed:    2,
			BombMaxRange: 2,
			BombMinRange: 1,
			BombPower:    2,
			MaxBombCount: 1,
			BaseHP:       1,
			PresetSkills: SkillNone,
			Selectable:   true,
		},
		{
			Name:         "Witch",
			BaseSpeed:    1,
			BombMaxRange: 3,
			BombMinRange: 1,
			BombPower:    2,
			MaxBombCount: 1,
			BaseHP:       1,
			PresetSkills: SkillNone,
			Selectable:   true,
		},
		{
			Name:         "Bandit",
			BaseSpeed:    3,
			BombMaxRange: 1,
			BombMinRange: 1,
			BombPower:    2,
			MaxBombCount: 1,
			BaseHP:       1,
			PresetSkills: SkillNone,
			Selectable:   true,
		},
		{
			Name:         "Prologue",
			BaseSpeed:    2,
			BombMaxRange: 1,
			BombMinRange: 1,
			BombPower:    2,
			MaxBombCount: 1,
			BaseHP:       1,
			PresetSkills: SkillNone,
			Selectable:   false,
		},
	}
}

// GetArchetype returns the archetype and whether it exists.
func GetArchetype(name string) (Archetype, bool) {
	for _, a := range archetypesRegistry() {
		if a.Name == name {
			return a, true
		}
	}
	return Archetype{}, false
}

// GetAllArchetypes gets all archetypes for game setup.
func GetAllArchetypes() []Archetype {
	results := []Archetype{}
	for _, a := range archetypesRegistry() {
		if a.Selectable {
			results = append(results, a)
		}
	}
	return results
}

// stagePresetsRegistry stores the base templates of stagePresets.
func stagePresetsRegistry() []StagePreset {
	return []StagePreset{
		{
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
		{
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
		{
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
}

// GetStagePreset returns the stage preset and whether it exists.
func GetStagePreset(name string) (StagePreset, bool) {
	for _, s := range stagePresetsRegistry() {
		if s.Name == name {
			return s, true
		}
	}
	return StagePreset{}, false
}

// GetStagePresets gets all stagePresets for game setup.
func GetAllStagePresets() []StagePreset {
	return stagePresetsRegistry()
}
