package engine

import (
	"testing"
)

func TestMatch_Resolve_VictoryCondition_Suite(t *testing.T) {
	addRoleUnit := func(m *Match, team, idx int, role UnitRole, hp int) {
		id := NewUnitID(team, idx)
		m.WorkingState.Units[id] = &Unit{ID: id, Team: team, Role: role, HP: hp}
	}

	tests := []struct {
		name                string
		p1                  teamFormation
		p2                  teamFormation
		setup               func(m *Match)
		expectedWinningTeam int
	}{
		{
			name:                "Standard gameplay state",
			p1:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{true, true, false, true}},
			p2:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{true, true, false, false}},
			expectedWinningTeam: matchInProgress,
		},
		{
			name:                "P1 Wins: P2 misses non-king",
			p1:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{true, true, false, true}},
			p2:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{false, false, false, false}},
			expectedWinningTeam: 1,
		},
		{
			name:                "P1 Wins: P2 King dead",
			p1:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{true, true, false, true}},
			p2:                  teamFormation{King: new(false), OrdinaryUnitsAlive: []bool{true, false, false, false}},
			expectedWinningTeam: 1,
		},
		{
			name:                "P1 Wins: P2 wiped out",
			p1:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{true, true, false, true}},
			p2:                  teamFormation{King: new(false), OrdinaryUnitsAlive: []bool{false, false, false, false}},
			expectedWinningTeam: 1,
		},
		{
			name:                "P1 Wins: P1 lone king",
			p1:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{false, false, false, false}},
			p2:                  teamFormation{King: new(false), OrdinaryUnitsAlive: []bool{false, false, false, false}},
			expectedWinningTeam: 1,
		},
		{
			name:                "P2 Wins: P1 misses non-king",
			p1:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{false, false, false, false}},
			p2:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{true, true, false, true}},
			expectedWinningTeam: 2,
		},
		{
			name:                "P2 Wins: P1 King dead",
			p1:                  teamFormation{King: new(false), OrdinaryUnitsAlive: []bool{true, true, false, true}},
			p2:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{true, false, false, false}},
			expectedWinningTeam: 2,
		},
		{
			name:                "P2 Wins: P1 wiped out",
			p1:                  teamFormation{King: new(false), OrdinaryUnitsAlive: []bool{false, false, false, false}},
			p2:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{true, true, false, true}},
			expectedWinningTeam: 2,
		},
		{
			name:                "P2 Wins: P2 lone king",
			p1:                  teamFormation{King: new(false), OrdinaryUnitsAlive: []bool{false, false, false, false}},
			p2:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{false, false, false, false}},
			expectedWinningTeam: 2,
		},
		{
			name:                "Draw: Both Kings dead",
			p1:                  teamFormation{King: new(false), OrdinaryUnitsAlive: []bool{false, true, false, false}},
			p2:                  teamFormation{King: new(false), OrdinaryUnitsAlive: []bool{false, false, true, false}},
			expectedWinningTeam: matchDrawn,
		},
		{
			name:                "Draw: Both Kings dead, P2 wiped out",
			p1:                  teamFormation{King: new(false), OrdinaryUnitsAlive: []bool{false, true, false, false}},
			p2:                  teamFormation{King: new(false), OrdinaryUnitsAlive: []bool{false, false, false, false}},
			expectedWinningTeam: matchDrawn,
		},
		{
			name:                "Draw: Both Kings dead, P1 wiped out",
			p1:                  teamFormation{King: new(false), OrdinaryUnitsAlive: []bool{false, false, false, false}},
			p2:                  teamFormation{King: new(false), OrdinaryUnitsAlive: []bool{false, false, true, false}},
			expectedWinningTeam: matchDrawn,
		},
		{
			name:                "Draw: Everyone dead",
			p1:                  teamFormation{King: new(false), OrdinaryUnitsAlive: []bool{false, false, false, false}},
			p2:                  teamFormation{King: new(false), OrdinaryUnitsAlive: []bool{false, false, false, false}},
			expectedWinningTeam: matchDrawn,
		},
		{
			name:                "Draw: Both Lone Kings dead",
			p1:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{false, false, false, false}},
			p2:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{false, false, false, false}},
			expectedWinningTeam: matchDrawn,
		},
		{
			name:                "Standard: Both Boss teams alive",
			p1:                  teamFormation{BossesAlive: []bool{true}},
			p2:                  teamFormation{BossesAlive: []bool{true}},
			expectedWinningTeam: matchInProgress,
		},
		{
			name:                "P2 Wins: P1 Boss dead",
			p1:                  teamFormation{BossesAlive: []bool{false}},
			p2:                  teamFormation{BossesAlive: []bool{true}},
			expectedWinningTeam: 2,
		},
		{
			name:                "P1 Wins: P2 Boss dead",
			p1:                  teamFormation{BossesAlive: []bool{true}},
			p2:                  teamFormation{BossesAlive: []bool{false}},
			expectedWinningTeam: 1,
		},
		{
			name:                "Draw: Both Bosses dead, regardless of ordinary units alive",
			p1:                  teamFormation{BossesAlive: []bool{false}, OrdinaryUnitsAlive: []bool{true}},
			p2:                  teamFormation{BossesAlive: []bool{false}, OrdinaryUnitsAlive: []bool{true, true}},
			expectedWinningTeam: matchDrawn,
		},
		{
			name:                "Standard: P1 Boss alive vs P2 full team",
			p1:                  teamFormation{BossesAlive: []bool{true}},
			p2:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{true}},
			expectedWinningTeam: matchInProgress,
		},
		{
			name:                "P1 Wins: P1 Boss alive vs P2 lone king",
			p1:                  teamFormation{BossesAlive: []bool{true}},
			p2:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{false}},
			expectedWinningTeam: 1,
		},
		{
			name:                "P1 Wins: P1 Boss alive vs P2 King dead",
			p1:                  teamFormation{BossesAlive: []bool{true}},
			p2:                  teamFormation{King: new(false), OrdinaryUnitsAlive: []bool{true}},
			expectedWinningTeam: 1,
		},
		{
			name:                "P2 Wins: P1 Boss dead vs P2 full team",
			p1:                  teamFormation{BossesAlive: []bool{false}},
			p2:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{true}},
			expectedWinningTeam: 2,
		},
		{
			name:                "P2 Wins: P1 Boss dead vs P2 lone king",
			p1:                  teamFormation{BossesAlive: []bool{false}},
			p2:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{false}},
			expectedWinningTeam: 2,
		},
		{
			name:                "Draw: P1 Boss dead vs P2 King dead",
			p1:                  teamFormation{BossesAlive: []bool{false}},
			p2:                  teamFormation{King: new(false), OrdinaryUnitsAlive: []bool{true}},
			expectedWinningTeam: matchDrawn,
		},
		{
			name:                "Standard: P2 Boss alive vs P1 full team",
			p1:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{true}},
			p2:                  teamFormation{BossesAlive: []bool{true}},
			expectedWinningTeam: matchInProgress,
		},
		{
			name:                "P2 Wins: P2 Boss alive vs P1 lone king",
			p1:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{false}},
			p2:                  teamFormation{BossesAlive: []bool{true}},
			expectedWinningTeam: 2,
		},
		{
			name:                "P2 Wins: P2 Boss alive vs P1 King dead",
			p1:                  teamFormation{King: new(false), OrdinaryUnitsAlive: []bool{true}},
			p2:                  teamFormation{BossesAlive: []bool{true}},
			expectedWinningTeam: 2,
		},
		{
			name:                "P1 Wins: P2 Boss dead vs P1 full team",
			p1:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{true}},
			p2:                  teamFormation{BossesAlive: []bool{false}},
			expectedWinningTeam: 1,
		},
		{
			name:                "P1 Wins: P2 Boss dead vs P1 lone king",
			p1:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{false}},
			p2:                  teamFormation{BossesAlive: []bool{false}},
			expectedWinningTeam: 1,
		},
		{
			name:                "Draw: P2 Boss dead vs P1 King dead",
			p1:                  teamFormation{King: new(false), OrdinaryUnitsAlive: []bool{true}},
			p2:                  teamFormation{BossesAlive: []bool{false}},
			expectedWinningTeam: matchDrawn,
		},
		{
			name:                "Standard: P1 multi-Boss, one alive keeps team going",
			p1:                  teamFormation{BossesAlive: []bool{false, true}},
			p2:                  teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{true}},
			expectedWinningTeam: matchInProgress,
		},
		{
			name: "King and Boss on one team, Boss alive",
			p1:   teamFormation{},
			p2:   teamFormation{King: new(true)},
			setup: func(m *Match) {
				addRoleUnit(m, 1, 0, RoleKing, 1)
				addRoleUnit(m, 1, 1, RoleBoss, 1)
			},
			expectedWinningTeam: 1,
		},
		{
			name: "King and Boss on one team, Boss dead, King alive",
			p1:   teamFormation{},
			p2:   teamFormation{King: new(true)},
			setup: func(m *Match) {
				addRoleUnit(m, 1, 0, RoleKing, 1)
				addRoleUnit(m, 1, 1, RoleBoss, 0)
			},
			expectedWinningTeam: 2,
		},
		{
			name: "Team 3 ordinary unit plus team 2 King",
			p1:   teamFormation{King: new(true)},
			p2:   teamFormation{King: new(true)},
			setup: func(m *Match) {
				addRoleUnit(m, 3, 0, RoleNormal, 1)
			},
			expectedWinningTeam: 2,
		},
		{
			name: "Team 3 Boss alone",
			p1:   teamFormation{King: new(true)},
			p2:   teamFormation{},
			setup: func(m *Match) {
				addRoleUnit(m, 3, 0, RoleBoss, 1)
			},
			expectedWinningTeam: 1,
		},
		{
			name: "Goal and Wiped on one team against a lone King",
			p1:   teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{true}},
			p2:   teamFormation{King: new(true)},
			setup: func(m *Match) {
				addRoleUnit(m, 1, 5, RoleBoss, 0)
			},
			expectedWinningTeam: matchDrawn,
		},
		{
			name: "Unknown role counts as an ordinary unit",
			p1:   teamFormation{King: new(true)},
			p2:   teamFormation{King: new(true)},
			setup: func(m *Match) {
				addRoleUnit(m, 1, 5, UnitRole(9), 1)
			},
			expectedWinningTeam: 1,
		},
		{
			name: "King at HP -1 counts as dead",
			p1:   teamFormation{King: new(true)},
			p2:   teamFormation{},
			setup: func(m *Match) {
				addRoleUnit(m, 2, 0, RoleKing, -1)
			},
			expectedWinningTeam: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestMatchWithTeams(t, tt.p1, tt.p2)
			if tt.setup != nil {
				tt.setup(m)
			}

			_, gameEvents := m.ResolveTurn()

			if m.WinnerTeamID != tt.expectedWinningTeam {
				t.Errorf("Victory Guard Failed! Expected WinnerTeamID = %d, got %d", tt.expectedWinningTeam, m.WinnerTeamID)
			}

			var foundEndedEvent bool
			for _, evt := range gameEvents {
				if ended, ok := evt.(MatchEnded); ok {
					foundEndedEvent = true
					if ended.WinnerTeamID != tt.expectedWinningTeam {
						t.Errorf("Malformed MatchEnded.WinnerTeam, got winner %d", ended.WinnerTeamID)
					}
				}
			}

			if tt.expectedWinningTeam == matchInProgress {
				if foundEndedEvent {
					t.Errorf("MatchEndedEvent should not be captured, but got one")
				}
				return
			}
			if !foundEndedEvent {
				t.Error("Missing critical MatchEndedEvent token inside returned telemetry array stream")
			}
		})
	}
}
