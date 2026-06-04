package engine

import "testing"

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
