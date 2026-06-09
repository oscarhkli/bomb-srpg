package server

import "bomb-srpg/engine"

type ArchetypeResponse struct {
	Name         string   `json:"name"`
	BaseSpeed    int      `json:"speed"`
	BombMaxRange int      `json:"bombMaxRange"`
	PresetSkills []string `json:"skills"` // Only the active skills
}

// MapToArchetypeResponse converts a core domain Archetype into a web-friendly response.
func MapToArchetypeResponse(a engine.Archetype) ArchetypeResponse {
	skills := []string{}

	for skill, active := range a.PresetSkills {
		if active {
			skills = append(skills, skill.String())
		}
	}

	return ArchetypeResponse{
		Name:         a.Name,
		BaseSpeed:    a.BaseSpeed,
		BombMaxRange: a.BombMaxRange,
		PresetSkills: skills,
	}
}
