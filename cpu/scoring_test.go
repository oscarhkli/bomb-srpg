package cpu

import (
	"bomb-srpg/engine"
	"errors"
	"testing"
)

func TestAliveCount(t *testing.T) {
	tests := []struct {
		name  string
		setup func(gs *engine.GameState) (ids []engine.UnitID, kingID engine.UnitID)
		want  int
	}{
		{
			name: "Normal count excluding King",
			setup: func(gs *engine.GameState) ([]engine.UnitID, engine.UnitID) {
				f1 := addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0})
				f2 := addFighter(gs, engine.NewUnitID(1, 2), engine.Coordinate{X: 1, Y: 0})
				king := addKing(gs, engine.NewUnitID(1, 3), engine.Coordinate{X: 2, Y: 0})
				return []engine.UnitID{f1.ID, f2.ID, king.ID}, king.ID
			},
			want: 2,
		},
		{
			name: "Dead unit not counted",
			setup: func(gs *engine.GameState) ([]engine.UnitID, engine.UnitID) {
				f1 := addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0})
				f2 := addFighter(gs, engine.NewUnitID(1, 2), engine.Coordinate{X: 1, Y: 0})
				f2.HP = 0
				king := addKing(gs, engine.NewUnitID(1, 3), engine.Coordinate{X: 2, Y: 0})
				return []engine.UnitID{f1.ID, f2.ID, king.ID}, king.ID
			},
			want: 1,
		},
		{
			name: "ID not present in gs.Units not counted",
			setup: func(gs *engine.GameState) ([]engine.UnitID, engine.UnitID) {
				f1 := addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0})
				missingID := engine.NewUnitID(9, 9)
				return []engine.UnitID{f1.ID, missingID}, engine.NewUnitID(9, 1)
			},
			want: 1,
		},
		{
			name: "ids empty",
			setup: func(gs *engine.GameState) ([]engine.UnitID, engine.UnitID) {
				return nil, engine.NewUnitID(1, 1)
			},
			want: 0,
		},
		{
			name: "King's own ID appears in ids but is skipped",
			setup: func(gs *engine.GameState) ([]engine.UnitID, engine.UnitID) {
				king := addKing(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0})
				return []engine.UnitID{king.ID}, king.ID
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := newTestGameState(5, 5)
			ids, kingID := tt.setup(gs)

			if got := aliveCount(gs, ids, kingID); got != tt.want {
				t.Errorf("aliveCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() (*engine.GameState, scoreContext, []engine.TurnCommand)
		wantErr   error
		wantTotal int
		wantTag   string
	}{
		{
			name: "Idle, no hazards",
			setup: func() (*engine.GameState, scoreContext, []engine.TurnCommand) {
				gs := newTestGameState(5, 5)
				actor := addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0})
				opponent := addFighter(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 4})
				allyKing := addKing(gs, engine.NewUnitID(1, 2), engine.Coordinate{X: 0, Y: 4})
				opponentKing := addKing(gs, engine.NewUnitID(2, 2), engine.Coordinate{X: 4, Y: 0})
				sc := scoreContext{
					actorID:        actor.ID,
					allyIDs:        []engine.UnitID{actor.ID},
					opponentIDs:    []engine.UnitID{opponent.ID},
					allyKingID:     allyKing.ID,
					opponentKingID: opponentKing.ID,
				}
				return gs, sc, nil
			},
			wantTotal: 0,
			wantTag:   "Idle",
		},
		{
			name: "Immediate kill, isolated from other factors",
			setup: func() (*engine.GameState, scoreContext, []engine.TurnCommand) {
				gs := newTestGameState(16, 16)
				actor := addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 10, Y: 10})
				opponent := addFighter(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 2, Y: 2})
				allyKing := addKing(gs, engine.NewUnitID(1, 2), engine.Coordinate{X: 15, Y: 15})
				opponentKing := addKing(gs, engine.NewUnitID(2, 2), engine.Coordinate{X: 0, Y: 15})

				bombID := engine.NewBombID(0, 0, actor.ID)
				bombPos := engine.Coordinate{X: 1, Y: 2}
				gs.Bombs[bombID] = &engine.Bomb{ID: bombID, OwnerID: actor.ID, Position: bombPos, Range: 1, Countdown: 1}
				gs.Grid[bombPos.Y][bombPos.X] = engine.Tile{Type: engine.TerrainPlain, OccupantType: engine.OccupantBomb, OccupantID: int64(bombID)}

				sc := scoreContext{
					actorID:        actor.ID,
					allyIDs:        []engine.UnitID{actor.ID},
					opponentIDs:    []engine.UnitID{opponent.ID},
					allyKingID:     allyKing.ID,
					opponentKingID: opponentKing.ID,
				}
				return gs, sc, nil
			},
			wantTotal: killUnitScore + 500,
			wantTag:   "Idle",
		},
		{
			name: "Error propagation from applyCandidate",
			setup: func() (*engine.GameState, scoreContext, []engine.TurnCommand) {
				gs := newTestGameState(5, 5)
				actor := addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0})
				cmds := []engine.TurnCommand{{Type: engine.TurnCmdType("bogus"), UnitID: actor.ID, Target: engine.Coordinate{X: 0, Y: 0}}}
				return gs, scoreContext{actorID: actor.ID}, cmds
			},
			wantErr: engine.ErrUnsupportedCommand,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs, sc, cmds := tt.setup()

			got, err := evaluate(sc, gs, cmds)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("evaluate() err = %v, want %v", err, tt.wantErr)
				}
				if got.score != 0 || got.tag != "" || got.turnCommands != nil {
					t.Errorf("evaluate() = %+v, want zero value", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("evaluate() unexpected err = %v", err)
			}
			if got.score != tt.wantTotal {
				t.Errorf("evaluate() score = %d, want %d", got.score, tt.wantTotal)
			}
			if got.tag != tt.wantTag {
				t.Errorf("evaluate() tag = %q, want %q", got.tag, tt.wantTag)
			}
		})
	}
}

func TestEvaluate_RiskAllyKingExcludesOwnBomb(t *testing.T) {
	tests := []struct {
		name      string
		bombOwner func(allyID, opponentID engine.UnitID) engine.UnitID
		want      int
	}{
		{
			name:      "Ally-owned bomb: not scored as risk",
			bombOwner: func(allyID, _ engine.UnitID) engine.UnitID { return allyID },
			want:      0,
		},
		{
			name:      "Opponent-owned bomb: scored as risk",
			bombOwner: func(_, opponentID engine.UnitID) engine.UnitID { return opponentID },
			want:      -1500,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := newTestGameState(9, 9)
			allyKing := addKing(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 2, Y: 2})
			allyKing.HP = 5
			opponentKing := addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 8, Y: 8})
			owner := tt.bombOwner(allyKing.ID, opponentKing.ID)

			bombID := engine.NewBombID(0, 0, owner)
			bombPos := engine.Coordinate{X: 3, Y: 2}
			gs.Bombs[bombID] = &engine.Bomb{ID: bombID, OwnerID: owner, Position: bombPos, Range: 1, Countdown: 1}
			gs.Grid[bombPos.Y][bombPos.X] = engine.Tile{Type: engine.TerrainPlain, OccupantType: engine.OccupantBomb, OccupantID: int64(bombID)}

			sc := scoreContext{
				actorID:        allyKing.ID,
				allyIDs:        []engine.UnitID{allyKing.ID},
				opponentIDs:    []engine.UnitID{opponentKing.ID},
				allyKingID:     allyKing.ID,
				opponentKingID: opponentKing.ID,
				actorOrigin:    allyKing.Position,
			}

			got, err := evaluate(sc, gs, nil)
			if err != nil {
				t.Fatalf("evaluate() unexpected err = %v", err)
			}
			if got.score != tt.want {
				t.Errorf("evaluate() score = %d, want %d", got.score, tt.want)
			}
		})
	}
}

func TestEvaluate_SuicideUncappedByTurn(t *testing.T) {
	tests := []struct {
		name      string
		bombOwner func(allyID, opponentID engine.UnitID) engine.UnitID
		want      int
	}{
		{
			name:      "Ally-owned bomb: certain self-destruction vetoed regardless of fuse length",
			bombOwner: func(allyID, _ engine.UnitID) engine.UnitID { return allyID },
			want:      -killKingScore,
		},
		{
			name:      "Opponent-owned bomb: not scored as self-destruction",
			bombOwner: func(_, opponentID engine.UnitID) engine.UnitID { return opponentID },
			want:      -782,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := newTestGameState(9, 9)
			allyKing := addKing(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 2, Y: 2})
			allyKing.HP = 1
			opponentKing := addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 8, Y: 8})
			owner := tt.bombOwner(allyKing.ID, opponentKing.ID)

			bombID := engine.NewBombID(0, 0, owner)
			bombPos := engine.Coordinate{X: 3, Y: 2}
			gs.Bombs[bombID] = &engine.Bomb{ID: bombID, OwnerID: owner, Position: bombPos, Range: 1, Countdown: 5}
			gs.Grid[bombPos.Y][bombPos.X] = engine.Tile{Type: engine.TerrainPlain, OccupantType: engine.OccupantBomb, OccupantID: int64(bombID)}

			sc := scoreContext{
				actorID:        allyKing.ID,
				allyIDs:        []engine.UnitID{allyKing.ID},
				opponentIDs:    []engine.UnitID{opponentKing.ID},
				allyKingID:     allyKing.ID,
				opponentKingID: opponentKing.ID,
				actorOrigin:    allyKing.Position,
			}

			got, err := evaluate(sc, gs, nil)
			if err != nil {
				t.Fatalf("evaluate() unexpected err = %v", err)
			}
			if got.score != tt.want {
				t.Errorf("evaluate() score = %d, want %d", got.score, tt.want)
			}
		})
	}
}

func TestEvaluate_MutualKingTradeNotDoubleCounted(t *testing.T) {
	gs := newTestGameState(9, 9)
	allyKing := addKing(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 2, Y: 2})
	allyKing.HP = 1
	opponentKing := addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 2})
	opponentKing.HP = 1

	bombID := engine.NewBombID(0, 0, allyKing.ID)
	bombPos := engine.Coordinate{X: 3, Y: 2}
	gs.Bombs[bombID] = &engine.Bomb{ID: bombID, OwnerID: allyKing.ID, Position: bombPos, Range: 2, Countdown: 1}
	gs.Grid[bombPos.Y][bombPos.X] = engine.Tile{Type: engine.TerrainPlain, OccupantType: engine.OccupantBomb, OccupantID: int64(bombID)}

	sc := scoreContext{
		actorID:        allyKing.ID,
		allyIDs:        []engine.UnitID{allyKing.ID},
		opponentIDs:    []engine.UnitID{opponentKing.ID},
		allyKingID:     allyKing.ID,
		opponentKingID: opponentKing.ID,
		actorOrigin:    allyKing.Position,
	}

	got, err := evaluate(sc, gs, nil)
	if err != nil {
		t.Fatalf("evaluate() unexpected err = %v", err)
	}
	if got.score <= 0 {
		t.Errorf("evaluate() score = %d, want a net-positive score for a winning King trade", got.score)
	}
}

func TestEvaluate_OriginalGameStateUntouched(t *testing.T) {
	gs := newTestGameState(5, 5)
	actor := addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 2, Y: 2})
	allyKing := addKing(gs, engine.NewUnitID(1, 2), engine.Coordinate{X: 0, Y: 4})
	opponentKing := addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 4})
	sc := scoreContext{actorID: actor.ID, allyKingID: allyKing.ID, opponentKingID: opponentKing.ID}
	cmds := []engine.TurnCommand{engine.NewPlaceBombCommand(actor.ID, engine.Coordinate{X: 2, Y: 0})}

	if _, err := evaluate(sc, gs, cmds); err != nil {
		t.Fatalf("evaluate() unexpected err = %v", err)
	}

	if len(gs.Bombs) != 0 {
		t.Errorf("gs.Bombs mutated by evaluate(): %+v", gs.Bombs)
	}
	if gs.Units[actor.ID].BombUsed != 0 {
		t.Errorf("gs.Units[actor.ID].BombUsed = %d, want 0", gs.Units[actor.ID].BombUsed)
	}
}
