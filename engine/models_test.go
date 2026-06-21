package engine

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTerrainTypeString(t *testing.T) {
	tests := []struct {
		name    string
		terrain TerrainType
		want    string
	}{
		{"Plain Tile", TerrainPlain, "TerrainPlain"},
		{"Block Tile", TerrainBlock, "TerrainBlock"},
		{"Tower Tile", TerrainTower, "TerrainTower"},
		{"Water Tile", TerrainWater, "TerrainWater"},
		{"Lava Tile", TerrainLava, "TerrainLava"},
		{"Invalid Value", TerrainType(-1), "TerrainUnknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.terrain.String(); got != tt.want {
				t.Errorf("TerrainType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOccupantTypeString(t *testing.T) {
	tests := []struct {
		name     string
		occupant OccupantType
		want     string
	}{
		{"No Occupant", OccupantNone, "OccupantNone"},
		{"Unit Occupant", OccupantUnit, "OccupantUnit"},
		{"Bomb Occupant", OccupantBomb, "OccupantBomb"},
		{"SoftBlock Occupant", OccupantSoftBlock, "OccupantSoftBlock"},
		{"Item Occupant", OccupantItem, "OccupantItem"},
		{"Invalid Value", OccupantType(99), "OccupantUnknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.occupant.String(); got != tt.want {
				t.Errorf("OccupantType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSkillTypeString(t *testing.T) {
	tests := []struct {
		name  string
		skill SkillType
		want  string
	}{
		{"Skill Fly", SkillCanFly, "Fly"},
		{"Skill Jump", SkillCanJump, "Jump"},
		{"Invalid Value", SkillType(0), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.skill.String(); got != tt.want {
				t.Errorf("SkillType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGameStateJSONSerialization(t *testing.T) {
	tests := []struct {
		name string
		gs   *GameState
		want []string
	}{
		{
			name: "TerrainType enums",
			gs: &GameState{
				Grid: [][]Tile{{
					{Type: TerrainPlain, OccupantType: OccupantNone, OccupantID: 0},
					{Type: TerrainBlock, OccupantType: OccupantNone, OccupantID: 0},
					{Type: TerrainTower, OccupantType: OccupantNone, OccupantID: 0},
					{Type: TerrainWater, OccupantType: OccupantNone, OccupantID: 0},
					{Type: TerrainLava, OccupantType: OccupantNone, OccupantID: 0},
				}},
			},
			want: []string{
				"TerrainPlain", "TerrainBlock", "TerrainTower", "TerrainWater", "TerrainLava",
				"OccupantNone",
			},
		},
		{
			name: "OccupantType enums",
			gs: &GameState{
				Grid: [][]Tile{{
					{Type: TerrainPlain, OccupantType: OccupantNone, OccupantID: 0},
					{Type: TerrainPlain, OccupantType: OccupantUnit, OccupantID: 1},
					{Type: TerrainPlain, OccupantType: OccupantBomb, OccupantID: 2},
					{Type: TerrainPlain, OccupantType: OccupantSoftBlock, OccupantID: 3},
					{Type: TerrainPlain, OccupantType: OccupantItem, OccupantID: 4},
				}},
			},
			want: []string{
				"OccupantNone", "OccupantUnit", "OccupantBomb", "OccupantSoftBlock", "OccupantItem",
				"TerrainPlain",
			},
		},
		{
			name: "TurnCmdType enums",
			gs: &GameState{
				TurnCommands: []TurnCommand{
					{Type: TurnCmdMove, UnitID: 0x11, Target: Coordinate{X: 1, Y: 2}},
					{Type: TurnCmdPlaceBomb, UnitID: 0x22, Target: Coordinate{X: 3, Y: 4}},
				},
			},
			want: []string{"move", "placeBomb", "unitId", "target"},
		},
		{
			name: "Unit Archetype name",
			gs: &GameState{
				Units: map[UnitID]*Unit{
					0x11: {ID: 0x11, Type: Archetype{Name: "King"}, Position: Coordinate{X: 0, Y: 0}, Team: 1, HP: 1, Skills: map[SkillType]bool{}},
					0x21: {ID: 0x21, Type: Archetype{Name: "Fighter"}, Position: Coordinate{X: 1, Y: 0}, Team: 1, HP: 1, Skills: map[SkillType]bool{SkillCanFly: true}},
					0x31: {ID: 0x31, Type: Archetype{Name: "Witch"}, Position: Coordinate{X: 2, Y: 0}, Team: 2, HP: 1, Skills: map[SkillType]bool{SkillCanJump: true}},
				},
			},
			want: []string{"King", "Fighter", "Witch", "Fly", "Jump", "hasMoved", "hasUsedSkill"},
		},
		{
			name: "Bomb fields",
			gs: &GameState{
				Bombs: map[BombID]*Bomb{
					0x110001: {ID: 0x110001, OwnerID: 0x11, Position: Coordinate{X: 2, Y: 3}, Range: 2, Countdown: 3},
				},
			},
			want: []string{"ownerId", "range", "countdown"},
		},
		{
			name: "SoftBlock fields",
			gs: &GameState{
				SoftBlocks: map[int]*SoftBlock{
					0: {ID: 0, Position: Coordinate{X: 4, Y: 5}},
				},
			},
			want: []string{"id", "position", "x", "y"},
		},
		{
			name: "GameState top-level fields",
			gs: &GameState{
				Turn:       5,
				ActiveTeam: 2,
				Grid:       [][]Tile{{{Type: TerrainPlain, OccupantType: OccupantNone, OccupantID: 0}}},
			},
			want: []string{"turn", "activeTeam", "grid", "units", "bombs", "softBlocks", "turnCommands"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.gs)
			if err != nil {
				t.Fatalf("MarshalJSON failed: %v", err)
			}
			s := string(body)
			for _, sub := range tt.want {
				if !strings.Contains(s, sub) {
					t.Errorf("JSON missing %q: got %s", sub, s)
				}
			}
		})
	}
}
