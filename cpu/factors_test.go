package cpu

import (
	"bomb-srpg/engine"
	"testing"
)

func TestAdvanceOpponentKingReachability(t *testing.T) {
	tests := []struct {
		name                string
		setup               func(gs *engine.GameState) (actor, king *engine.Unit)
		destroyedSoftBlocks int
		turn                int
		distToKingBefore    int
		distToKingAfter     int
		want                int
	}{
		{
			name: "No SoftBlock destroyed",
			setup: func(gs *engine.GameState) (*engine.Unit, *engine.Unit) {
				return addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0}),
					addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 4})
			},
			destroyedSoftBlocks: 0,
			distToKingBefore:    5,
			distToKingAfter:     1,
			want:                0,
		},
		{
			name: "Actor dead",
			setup: func(gs *engine.GameState) (*engine.Unit, *engine.Unit) {
				actor := addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0})
				actor.HP = 0
				return actor, addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 4})
			},
			destroyedSoftBlocks: 1,
			distToKingBefore:    5,
			distToKingAfter:     1,
			want:                0,
		},
		{
			name: "King dead",
			setup: func(gs *engine.GameState) (*engine.Unit, *engine.Unit) {
				king := addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 4})
				king.HP = 0
				return addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0}), king
			},
			destroyedSoftBlocks: 1,
			distToKingBefore:    5,
			distToKingAfter:     1,
			want:                0,
		},
		{
			name: "No change",
			setup: func(gs *engine.GameState) (*engine.Unit, *engine.Unit) {
				return addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0}),
					addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 4})
			},
			destroyedSoftBlocks: 1,
			turn:                0,
			distToKingBefore:    5,
			distToKingAfter:     5,
			want:                0,
		},
		{
			name: "Breakthrough: was unreachable, now reachable",
			setup: func(gs *engine.GameState) (*engine.Unit, *engine.Unit) {
				return addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0}),
					addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 4})
			},
			destroyedSoftBlocks: 1,
			turn:                0,
			distToKingBefore:    -1,
			distToKingAfter:     5,
			want:                5,
		},
		{
			name: "Normal improvement: both reachable, got closer",
			setup: func(gs *engine.GameState) (*engine.Unit, *engine.Unit) {
				return addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0}),
					addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 4})
			},
			destroyedSoftBlocks: 1,
			turn:                0,
			distToKingBefore:    5,
			distToKingAfter:     1,
			want:                5,
		},
		{
			name: "Regression: both reachable, got farther",
			setup: func(gs *engine.GameState) (*engine.Unit, *engine.Unit) {
				return addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0}),
					addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 4})
			},
			destroyedSoftBlocks: 1,
			turn:                0,
			distToKingBefore:    1,
			distToKingAfter:     5,
			want:                -5,
		},
		{
			name: "Newly unreachable: was reachable, now unreachable",
			setup: func(gs *engine.GameState) (*engine.Unit, *engine.Unit) {
				return addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0}),
					addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 4})
			},
			destroyedSoftBlocks: 1,
			turn:                0,
			distToKingBefore:    5,
			distToKingAfter:     -1,
			want:                -5,
		},
		{
			name: "Turn decay: same improvement, later turn scores less",
			setup: func(gs *engine.GameState) (*engine.Unit, *engine.Unit) {
				return addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0}),
					addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 4})
			},
			destroyedSoftBlocks: 1,
			turn:                4,
			distToKingBefore:    5,
			distToKingAfter:     1,
			want:                1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := newTestGameState(5, 5)
			actor, king := tt.setup(gs)
			sc := scoreContext{actorID: actor.ID, opponentKingID: king.ID}
			tr := turnResult{
				turn:                tt.turn,
				destroyedSoftBlocks: tt.destroyedSoftBlocks,
				distToKingBefore:    tt.distToKingBefore,
				distToKingAfter:     tt.distToKingAfter,
			}

			if got := advanceOpponentKingReachability(gs, sc, tr); got != tt.want {
				t.Errorf("advanceOpponentKingReachability() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestKillOpponentKing(t *testing.T) {
	opponentKingID := engine.NewUnitID(2, 1)
	tests := []struct {
		name     string
		turn     int
		diedID   engine.UnitID
		unitDied bool
		want     int
	}{
		{name: "King died at T+0", turn: 0, diedID: opponentKingID, unitDied: true, want: killKingScore},
		{name: "King died at T+1: too late, opponent already had a turn", turn: 1, diedID: opponentKingID, unitDied: true, want: 0},
		{name: "Different unit died", turn: 0, diedID: engine.NewUnitID(1, 1), unitDied: true, want: 0},
		{name: "Nobody died", turn: 0, unitDied: false, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := scoreContext{opponentKingID: opponentKingID}
			diedUnits := make(map[engine.UnitID]struct{})
			if tt.unitDied {
				diedUnits[tt.diedID] = struct{}{}
			}
			tr := turnResult{turn: tt.turn, diedUnits: diedUnits}

			if got := killOpponentKing(nil, sc, tr); got != tt.want {
				t.Errorf("killOpponentKing() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestKillOpponents(t *testing.T) {
	tests := []struct {
		name   string
		turn   int
		before int
		after  int
		want   int
	}{
		{name: "Some killed at T+0", turn: 0, before: 2, after: 1, want: killUnitScore / 2},
		{name: "Last one killed at T+0", turn: 0, before: 1, after: 0, want: killUnitScore},
		{name: "None killed", turn: 0, before: 3, after: 3, want: 0},
		{name: "No non-King opponents to begin with", turn: 0, before: 0, after: 0, want: 0},
		{name: "Killed at T+1: too late, opponent already had a turn", turn: 1, before: 2, after: 1, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := turnResult{turn: tt.turn, aliveOpponentsBefore: tt.before, aliveOpponentsAfter: tt.after}

			if got := killOpponents(nil, scoreContext{}, tr); got != tt.want {
				t.Errorf("killOpponents() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestKillAllies(t *testing.T) {
	tests := []struct {
		name   string
		turn   int
		before int
		after  int
		want   int
	}{
		{name: "Some killed at T+0", turn: 0, before: 2, after: 1, want: killUnitScore / 2},
		{name: "Killed at T+1: still certain, ally couldn't act", turn: 1, before: 2, after: 1, want: killUnitScore / 2},
		{name: "Killed at T+2: too late, ally already had a turn", turn: 2, before: 2, after: 1, want: 0},
		{name: "None killed", turn: 0, before: 3, after: 3, want: 0},
		{name: "No non-King allies to begin with", turn: 0, before: 0, after: 0, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := turnResult{turn: tt.turn, aliveAlliesBefore: tt.before, aliveAlliesAfter: tt.after}

			if got := killAllies(nil, scoreContext{}, tr); got != tt.want {
				t.Errorf("killAllies() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestKillAllyKing(t *testing.T) {
	allyKingID := engine.NewUnitID(1, 1)
	tests := []struct {
		name     string
		turn     int
		diedID   engine.UnitID
		unitDied bool
		want     int
	}{
		{name: "King died at T+0", turn: 0, diedID: allyKingID, unitDied: true, want: killKingScore},
		{name: "King died at T+1: still certain, ally couldn't act", turn: 1, diedID: allyKingID, unitDied: true, want: killKingScore},
		{name: "King died at T+2: too late, ally already had a turn", turn: 2, diedID: allyKingID, unitDied: true, want: 0},
		{name: "Different unit died", turn: 0, diedID: engine.NewUnitID(2, 1), unitDied: true, want: 0},
		{name: "Nobody died", turn: 0, unitDied: false, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := scoreContext{allyKingID: allyKingID}
			diedUnits := make(map[engine.UnitID]struct{})
			if tt.unitDied {
				diedUnits[tt.diedID] = struct{}{}
			}
			tr := turnResult{turn: tt.turn, diedUnits: diedUnits}

			if got := killAllyKing(nil, sc, tr); got != tt.want {
				t.Errorf("killAllyKing() = %d, want %d", got, tt.want)
			}
		})
	}
}
