package engine

import (
	"encoding/json/v2"
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
		{"Skill None", SkillNone, "None"},
		{"Skill Fly", SkillCanFly, "Fly"},
		{"Skill Jump", SkillCanJump, "Jump"},
		{"Invalid Value", SkillType(123), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.skill.String(); got != tt.want {
				t.Errorf("SkillType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSkillUnitRoleString(t *testing.T) {
	tests := []struct {
		name  string
		skill UnitRole
		want  string
	}{
		{"UnitRole Normal", RoleNormal, "Normal"},
		{"UnitRole King", RoleKing, "King"},
		{"UnitRole Boss", RoleBoss, "Boss"},
		{"Invalid Value", UnitRole(123), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.skill.String(); got != tt.want {
				t.Errorf("UnitRole.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUnitRole_UnmarshalText(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    UnitRole
		wantErr bool
	}{
		{"Role Normal", `"Normal"`, RoleNormal, false},
		{"Role King", `"King"`, RoleKing, false},
		{"Role Boss", `"Boss"`, RoleBoss, false},
		{"Role Unknown", `"ABC"`, RoleNormal, true},
		{"Wrong JSON Type", `2`, RoleNormal, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got UnitRole
			err := json.Unmarshal([]byte(tt.input), &got)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("Unmarshal(%s) error = nil, want error", tt.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unmarshal(%s) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("Unmarshal(%s) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCPUTurnPhaseString(t *testing.T) {
	tests := []struct {
		name      string
		turnPhase CPUTurnPhase
		want      string
	}{
		{"TurnPhase Idle", TurnPhaseIdle, "TurnPhaseIdle"},
		{"TurnPhase Planning", TurnPhasePlanning, "TurnPhasePlanning"},
		{"TurnPhase Ready", TurnPhaseReady, "TurnPhaseReady"},
		{"Invalid Value", CPUTurnPhase(99), "TurnPhaseUnknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.turnPhase.String(); got != tt.want {
				t.Errorf("CPUTurnPhase.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
