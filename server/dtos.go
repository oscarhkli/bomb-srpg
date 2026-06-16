package server

import "bomb-srpg/engine"

// ArchetypeResponse is the public representation of a unit archetype for the client.
type ArchetypeResponse struct {
	Name         string   `json:"name"`
	BaseSpeed    int      `json:"speed"`
	BombMaxRange int      `json:"bombMaxRange"`
	PresetSkills []string `json:"skills"` // Only the active skills
}

// MapToArchetypeResponse converts a core engine Archetype to its API response form.
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

// CreateMatchRoomResponse is returned when a new match room is created.
type CreateMatchRoomResponse struct {
	ID string `json:"id"`
}
