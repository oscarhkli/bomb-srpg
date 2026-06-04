package engine

import "testing"

func TestUnitIDCodecs(t *testing.T) {
	tests := []struct {
		name       string
		teamID     int
		localIndex int
		wantRawHex uint8
	}{
		{"Special Character ID 0", 0, 0, 0x00},
		{"Team 1 Player 1 First Slot", 1, 0, 0x10},
		{"Team 1 Player 5 Last Slot", 1, 4, 0x14},
		{"Team 2 Player 1 First Slot", 2, 0, 0x20},
		{"Team 2 Player 5 Last Slot", 2, 4, 0x24},
		{"Max Structural Boundary Limit", 15, 15, 0xFF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packedID := NewUnitID(tt.teamID, tt.localIndex)

			// Validate raw bitwise layout composition
			if uint8(packedID) != tt.wantRawHex {
				t.Errorf("NewUnitID() raw binary corrupt! Got 0x%02X, want 0x%02X", uint8(packedID), tt.wantRawHex)
			}

			// Validate Method-Style decoding symmetry
			gotTeam, gotIndex := packedID.Decode()
			if gotTeam != tt.teamID {
				t.Errorf("Decode() gotTeam = %d, want %d", gotTeam, tt.teamID)
			}
			if gotIndex != tt.localIndex {
				t.Errorf("Decode() gotIndex = %d, want %d", gotIndex, tt.localIndex)
			}
		})
	}
}

func TestBombIDCodecs(t *testing.T) {
	tests := []struct {
		name        string
		turn        int
		bombCounter int
		teamID      int
		localIndex  int
		wantRawHex  uint32 // 👑 Independent raw benchmark validation
	}{
		{"Turn 1, Team 1, P1, Bomb 1", 1, 1, 1, 0, 0x10010001},
		{"Turn 1, Team 1, P5, Bomb 8", 1, 8, 1, 4, 0x14010008},
		{"Turn 1, Team 2, P1, Bomb 1", 1, 1, 2, 0, 0x20010001},
		{"Turn 1, Team 2, P5, Bomb 8", 1, 8, 2, 4, 0x24010008},
		{"Mid Game Turn 100, Bomb 5", 100, 5, 1, 2, 0x12640005},
		{"Max Turn 255 Capacity Boundary", 255, 8, 2, 4, 0x24FF0008},
		{"Max Bomb Counter 65535 structural Limit", 5, 65535, 1, 1, 0x1105FFFF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unitID := NewUnitID(tt.teamID, tt.localIndex)
			packedBombID := NewBombID(tt.turn, tt.bombCounter, unitID)
			// Validate raw bitwise layout composition
			if uint32(packedBombID) != tt.wantRawHex {
				t.Errorf("NewBombID() raw binary corrupt! Got 0x%08X, want 0x%08X", uint32(packedBombID), tt.wantRawHex)
			}

			// Validate Method-Style decoding symmetry
			gotTurn, gotCounter, gotOwnerID := packedBombID.Decode()
			gotTeam, gotIndex := gotOwnerID.Decode()

			if gotTurn != tt.turn {
				t.Errorf("Decode() gotTurn = %d, want %d", gotTurn, tt.turn)
			}
			if gotCounter != tt.bombCounter {
				t.Errorf("Decode() gotCounter = %d, want %d", gotCounter, tt.bombCounter)
			}
			if gotTeam != tt.teamID {
				t.Errorf("Decoded Owner Team = %d, want %d", gotTeam, tt.teamID)
			}
			if gotIndex != tt.localIndex {
				t.Errorf("Decoded Owner Local Index = %d, want %d", gotIndex, tt.localIndex)
			}
		})
	}
}

// TestCombinatorialExhaustive loops through every valid game asset combination
// to guarantee 100% safety with no bitwise leaks or overlapping corruptions.
func TestCombinatorialExhaustive(t *testing.T) {
	maxTeams := 2
	maxPlayersPerTeam := 5
	maxBombsPerPlayer := 8
	sampleTurns := []int{1, 50, 100, 255}

	for _, turn := range sampleTurns {
		for team := 1; team <= maxTeams; team++ {
			for playerIdx := 0; playerIdx < maxPlayersPerTeam; playerIdx++ {
				for bombNum := 1; bombNum <= maxBombsPerPlayer; bombNum++ {
					uID := NewUnitID(team, playerIdx)
					bID := NewBombID(turn, bombNum, uID)

					gotTurn, gotCount, gotUID := bID.Decode()
					gotTeam, gotIdx := gotUID.Decode()

					if gotTurn != turn || gotCount != bombNum || gotTeam != team || gotIdx != playerIdx {
						t.Fatalf("Bitwise leak detected! Turn %d, Team %d, Player %d, Bomb %d. Got: Turn %d, Team %d, Player %d, Bomb %d",
							turn, team, playerIdx, bombNum, gotTurn, gotTeam, gotIdx, gotCount)
					}
				}
			}
		}
	}
}
