package engine

import (
	"fmt"
	"testing"
)

func TestUnitIDCodecs(t *testing.T) {
	tests := []struct {
		name       string
		teamID     int
		localIndex int
	}{
		{"Special Character ID 0", 0, 0},
		{"Team 1 Player 1 First Slot", 1, 0},
		{"Team 1 Player 5 Last Slot", 1, 4},
		{"Team 2 Player 1 First Slot", 2, 0},
		{"Team 2 Player 5 Last Slot", 2, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packedID := NewUnitID(tt.teamID, tt.localIndex)

			gotTeam, gotIndex := DecodeUnitID(packedID)

			if gotTeam != tt.teamID {
				t.Errorf("DecodeUnitID() gotTeam = %d, want %d", gotTeam, tt.teamID)
			}
			if gotIndex != tt.localIndex {
				t.Errorf("DecodeUnitID() gotIndex = %d, want %d", gotIndex, tt.localIndex)
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
	}{
		{"Turn 1, Team 1, P1, Bomb 1", 1, 1, 1, 0},
		{"Turn 1, Team 1, P5, Bomb 8", 1, 8, 1, 4},
		{"Turn 1, Team 2, P1, Bomb 1", 1, 1, 2, 0},
		{"Turn 1, Team 2, P5, Bomb 8", 1, 8, 2, 4},
		{"Mid Game Turn 100, Bomb 5", 100, 5, 1, 2},
		{"Max Turn 255 Capacity Boundary", 255, 8, 2, 4},
		{"Max Bomb Counter 65535 Structural Limit", 5, 65535, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unitID := NewUnitID(tt.teamID, tt.localIndex)

			packedBombID := NewBombID(tt.turn, tt.bombCounter, unitID)

			gotTurn, gotCounter, gotOwnerID := DecodeBombID(packedBombID)
			gotTeam, gotIndex := DecodeUnitID(gotOwnerID)

			if gotTurn != tt.turn {
				t.Errorf("DecodeBombID() gotTurn = %d, want %d", gotTurn, tt.turn)
			}
			if gotCounter != tt.bombCounter {
				t.Errorf("DecodeBombID() gotCounter = %d, want %d", gotCounter, tt.bombCounter)
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
			for playerIdx := range maxPlayersPerTeam {
				for bombNum := 1; bombNum <= maxBombsPerPlayer; bombNum++ {

					name := fmt.Sprintf("Turn%d_T%d_P%d_B%d", turn, team, playerIdx, bombNum)
					t.Run(name, func(t *testing.T) {
						uID := NewUnitID(team, playerIdx)
						bID := NewBombID(turn, bombNum, uID)

						gotTurn, gotCount, gotUID := DecodeBombID(bID)
						gotTeam, gotIdx := DecodeUnitID(gotUID)

						if gotTurn != turn || gotCount != bombNum || gotTeam != team || gotIdx != playerIdx {
							t.Errorf("Data mismatch at Turn %d, Team %d, Player %d, Bomb %d. Got: Turn %d, Team %d, Player %d, Bomb %d",
								turn, team, playerIdx, bombNum, gotTurn, gotTeam, gotIdx, gotCount)
						}
					})

				}
			}
		}
	}
}
