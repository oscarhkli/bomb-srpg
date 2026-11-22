package engine

import "testing"

func TestTeamStatus_GoalAndWiped(t *testing.T) {
	tests := []struct {
		name      string
		status    teamStatus
		wantGoal  bool
		wantWiped bool
	}{
		{
			name:     "living Boss is the goal",
			status:   teamStatus{hasBoss: true, bossAlive: true},
			wantGoal: true,
		},
		{
			name:      "dead Boss is wiped even with ordinary units alive",
			status:    teamStatus{hasBoss: true, ordinaryAlive: true},
			wantWiped: true,
		},
		{
			name:     "King with an ordinary unit is the goal",
			status:   teamStatus{kingAlive: true, ordinaryAlive: true},
			wantGoal: true,
		},
		{
			name:   "King alone is neither",
			status: teamStatus{kingAlive: true},
		},
		{
			name:   "ordinary units without a King are neither",
			status: teamStatus{ordinaryAlive: true},
		},
		{
			name:      "Boss-less team with nobody alive is wiped",
			status:    teamStatus{},
			wantWiped: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.goal(); got != tt.wantGoal {
				t.Errorf("goal() = %v, want %v", got, tt.wantGoal)
			}
			if got := tt.status.wiped(); got != tt.wantWiped {
				t.Errorf("wiped() = %v, want %v", got, tt.wantWiped)
			}
		})
	}
}
