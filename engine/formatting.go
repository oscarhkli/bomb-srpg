package engine

// String converts a TerrainType integer value into a human-readable text string.
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

// String converts an OccupantType integer value into a human-readable text string.
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

// String converts an SkillType integer value into a human-readable text string.
// It's a placeholder for Phase 4+, may be changed into other from.
func (s SkillType) String() string {
	switch s {
	case SkillCanFly:
		return "Fly"
	case SkillCanJump:
		return "Jump"
	default:
		return "Unknown"
	}
}
