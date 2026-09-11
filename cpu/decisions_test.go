package cpu

import (
	"bomb-srpg/engine"
	"errors"
	"reflect"
	"slices"
	"testing"
)

func TestApplyCandidate(t *testing.T) {
	tests := []struct {
		name         string
		turnCommands func(unit *engine.Unit) []engine.TurnCommand
		wantErr      error
	}{
		{
			name: "Move succeeds",
			turnCommands: func(unit *engine.Unit) []engine.TurnCommand {
				return []engine.TurnCommand{engine.NewMoveCommand(unit.ID, engine.Coordinate{X: 2, Y: 0})}
			},
		},
		{
			name: "PlaceBomb succeeds",
			turnCommands: func(unit *engine.Unit) []engine.TurnCommand {
				return []engine.TurnCommand{engine.NewPlaceBombCommand(unit.ID, engine.Coordinate{X: 2, Y: 0})}
			},
		},
		{
			name: "Move then PlaceBomb",
			turnCommands: func(unit *engine.Unit) []engine.TurnCommand {
				return []engine.TurnCommand{
					engine.NewMoveCommand(unit.ID, engine.Coordinate{X: 2, Y: 0}),
					engine.NewPlaceBombCommand(unit.ID, engine.Coordinate{X: 2, Y: 2}),
				}
			},
		},
		{
			name: "Unsupported command type",
			turnCommands: func(unit *engine.Unit) []engine.TurnCommand {
				return []engine.TurnCommand{{Type: engine.TurnCmdType("bogus"), UnitID: unit.ID, Target: engine.Coordinate{X: 0, Y: 0}}}
			},
			wantErr: engine.ErrUnsupportedCommand,
		},
		{
			name: "Move to unreachable target",
			turnCommands: func(unit *engine.Unit) []engine.TurnCommand {
				return []engine.TurnCommand{engine.NewMoveCommand(unit.ID, engine.Coordinate{X: 4, Y: 4})}
			},
			wantErr: engine.ErrOutOfMoveRange,
		},
		{
			name: "Second command fails: first command's effect still stands",
			turnCommands: func(unit *engine.Unit) []engine.TurnCommand {
				return []engine.TurnCommand{
					engine.NewMoveCommand(unit.ID, engine.Coordinate{X: 2, Y: 0}),
					engine.NewMoveCommand(unit.ID, engine.Coordinate{X: 2, Y: 1}),
				}
			},
			wantErr: engine.ErrAlreadyMoved,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := newTestGameState(5, 5)
			unit := addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0})

			err := applyCandidate(gs, candidate{turnCommands: tt.turnCommands(unit)})

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("applyCandidate() err = %v, want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("applyCandidate() unexpected err = %v", err)
			}
		})
	}

	t.Run("Move succeeds: position updated", func(t *testing.T) {
		gs := newTestGameState(5, 5)
		unit := addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0})
		want := engine.Coordinate{X: 2, Y: 0}

		if err := applyCandidate(gs, candidate{turnCommands: []engine.TurnCommand{engine.NewMoveCommand(unit.ID, want)}}); err != nil {
			t.Fatalf("applyCandidate() unexpected err = %v", err)
		}
		if got := gs.Units[unit.ID].Position; got != want {
			t.Errorf("Position = %+v, want %+v", got, want)
		}
	})

	t.Run("PlaceBomb succeeds: bomb registered", func(t *testing.T) {
		gs := newTestGameState(5, 5)
		unit := addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0})
		target := engine.Coordinate{X: 2, Y: 0}

		if err := applyCandidate(gs, candidate{turnCommands: []engine.TurnCommand{engine.NewPlaceBombCommand(unit.ID, target)}}); err != nil {
			t.Fatalf("applyCandidate() unexpected err = %v", err)
		}
		found := false
		for _, b := range gs.Bombs {
			if b.Position == target && b.OwnerID == unit.ID {
				found = true
			}
		}
		if !found {
			t.Errorf("gs.Bombs = %+v, want an entry owned by %v at %+v", gs.Bombs, unit.ID, target)
		}
	})

	t.Run("Second command fails: first command's move is not rolled back", func(t *testing.T) {
		gs := newTestGameState(5, 5)
		unit := addFighter(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0})
		first := engine.Coordinate{X: 2, Y: 0}
		cmds := []engine.TurnCommand{
			engine.NewMoveCommand(unit.ID, first),
			engine.NewMoveCommand(unit.ID, engine.Coordinate{X: 2, Y: 1}),
		}

		err := applyCandidate(gs, candidate{turnCommands: cmds})
		if !errors.Is(err, engine.ErrAlreadyMoved) {
			t.Fatalf("applyCandidate() err = %v, want %v", err, engine.ErrAlreadyMoved)
		}
		if got := gs.Units[unit.ID].Position; got != first {
			t.Errorf("Position = %+v, want %+v (first command's effect should stand)", got, first)
		}
	})
}

func TestBestCandidateFor(t *testing.T) {
	t.Run("Picks the highest-scoring plan", func(t *testing.T) {
		gs := newTestGameState(9, 1)
		allyKing := addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 0, Y: 0})
		actor := addFighter(gs, engine.NewUnitID(2, 2), engine.Coordinate{X: 2, Y: 0})
		actor.BombUsed = actor.MaxBombCount
		opponentKing := addKing(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 8, Y: 0})

		bombID := engine.BombID(1)
		bombPos := engine.Coordinate{X: 4, Y: 0}
		gs.Bombs[bombID] = &engine.Bomb{ID: bombID, OwnerID: engine.NewUnitID(9, 9), Position: bombPos, Range: 2, Countdown: 1}
		gs.Grid[bombPos.Y][bombPos.X] = engine.Tile{Type: engine.TerrainPlain, OccupantType: engine.OccupantBomb, OccupantID: int64(bombID)}

		sc := scoreContext{actorID: actor.ID, allyIDs: []engine.UnitID{actor.ID}, allyKingID: allyKing.ID, opponentKingID: opponentKing.ID}

		got, err := bestCandidateFor(sc, gs)
		if err != nil {
			t.Fatalf("bestCandidateFor() unexpected err = %v", err)
		}

		// move(1,0) is the only reachable plan outside the pre-existing bomb's blast radius.
		if want := "move(1,0)"; got.tag != want {
			t.Errorf("bestCandidateFor() tag = %q, want %q (score %d)", got.tag, want, got.score)
		}
		if want := -491; got.score != want {
			t.Errorf("bestCandidateFor() score = %d, want %d", got.score, want)
		}
	})

	t.Run("Fully boxed in: only Idle available", func(t *testing.T) {
		gs := newTestGameState(9, 9)
		allyKing := addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 0, Y: 0})
		center := engine.Coordinate{X: 4, Y: 4}
		actor := addFighter(gs, engine.NewUnitID(2, 2), center)
		radius := max(actor.Speed, actor.BombMaxRange)
		for _, off := range crossOffsets(radius) {
			setTerrainBlock(gs, engine.Coordinate{X: center.X + off.X, Y: center.Y + off.Y})
		}
		opponentKing := addKing(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 8, Y: 8})

		sc := scoreContext{actorID: actor.ID, allyIDs: []engine.UnitID{actor.ID}, allyKingID: allyKing.ID, opponentKingID: opponentKing.ID}

		got, err := bestCandidateFor(sc, gs)
		if err != nil {
			t.Fatalf("bestCandidateFor() unexpected err = %v", err)
		}
		if want := "Idle"; got.tag != want {
			t.Errorf("bestCandidateFor() tag = %q, want %q", got.tag, want)
		}
		if got.score != 0 {
			t.Errorf("bestCandidateFor() score = %d, want %d", got.score, 0)
		}
	})
}

func TestDecide_NoKingOnASide_ReturnsEmpty(t *testing.T) {
	gs := newTestGameState(5, 5)
	addFighter(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 0, Y: 0})
	addKing(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 4, Y: 4})
	addFighter(gs, engine.NewUnitID(1, 2), engine.Coordinate{X: 4, Y: 0})

	if got := Decide(gs); len(got) != 0 {
		t.Errorf("Decide() = %+v, want empty", got)
	}
}

func TestDecide_DeadAllyExcludedFromPlanning(t *testing.T) {
	gs := newTestGameState(9, 9)
	addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 0, Y: 0})
	deadAlly := addFighter(gs, engine.NewUnitID(2, 2), engine.Coordinate{X: 0, Y: 8})
	deadAlly.HP = 0
	addKing(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 8, Y: 8})

	// Nothing scores above neutral for the sole live actor, confirming the dead Fighter is excluded.
	if got := Decide(gs); len(got) != 0 {
		t.Errorf("Decide() = %+v, want empty", got)
	}
}

// corridorWithBreakthrough builds a 9x2 board where the sole scoring action is an ally
// Fighter's bomb clearing a SoftBlock toward the opponent King. The ally King's pocket at
// (8,1) sits exactly at riskFreeDist, so it's kept out of the blast.
func corridorWithBreakthrough(gs *engine.GameState) (allyFighter *engine.Unit) {
	for x := range 9 {
		setTerrainBlock(gs, engine.Coordinate{X: x, Y: 1})
	}
	allyFighter = addFighter(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 0, Y: 0})
	addSoftBlock(gs, 1, engine.Coordinate{X: 3, Y: 0})
	addKing(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 8, Y: 0})
	addKing(gs, engine.NewUnitID(2, 2), engine.Coordinate{X: 8, Y: 1})
	return allyFighter
}

// twinCorridorWithBreakthrough is two corridorWithBreakthrough-style setups (rows 0 and 3).
// Only Fighter1's corridor sits near the opponent King, so it alone scores threatOpponentKing;
// Fighter2's corridor instead scores threatOpponents via a nearby opponent Fighter.
func twinCorridorWithBreakthrough(gs *engine.GameState) (fighter1, fighter2 *engine.Unit) {
	fighter1 = addFighter(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 0, Y: 0})
	fighter1.HasMoved = true
	addSoftBlock(gs, 1, engine.Coordinate{X: 3, Y: 0})
	addKing(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 8, Y: 0})

	for x := range 8 {
		setTerrainBlock(gs, engine.Coordinate{X: x, Y: 1})
		if x != 2 {
			setTerrainBlock(gs, engine.Coordinate{X: x, Y: 2})
		}
	}

	fighter2 = addFighter(gs, engine.NewUnitID(2, 2), engine.Coordinate{X: 0, Y: 3})
	fighter2.HasMoved = true
	addSoftBlock(gs, 2, engine.Coordinate{X: 3, Y: 3})
	// (2,2) sits close to Fighter2's blast without blocking Fighter2's own row-3 path to the King.
	addFighter(gs, engine.NewUnitID(1, 2), engine.Coordinate{X: 2, Y: 2})

	for x := range 9 {
		setTerrainBlock(gs, engine.Coordinate{X: x, Y: 4})
	}
	allyKing := addKing(gs, engine.NewUnitID(2, 3), engine.Coordinate{X: 8, Y: 4})
	allyKing.HasMoved = true
	allyKing.BombUsed = allyKing.MaxBombCount

	return fighter1, fighter2
}

func TestDecide_AppliesMoreThanOneRound(t *testing.T) {
	gs := newTestGameState(9, 5)
	fighter1, fighter2 := twinCorridorWithBreakthrough(gs)
	before := gs.DeepCopy()

	got := Decide(gs)

	if len(got) != 2 {
		t.Fatalf("Decide() = %+v, want 2 commands", got)
	}
	if got[0].Type != engine.TurnCmdPlaceBomb || got[0].UnitID != fighter1.ID {
		t.Errorf("Decide()[0] = %+v, want a placeBomb from %v (higher-scoring, picked first)", got[0], fighter1.ID)
	}
	if got[1].Type != engine.TurnCmdPlaceBomb || got[1].UnitID != fighter2.ID {
		t.Errorf("Decide()[1] = %+v, want a placeBomb from %v (picked in the second round)", got[1], fighter2.ID)
	}
	if !reflect.DeepEqual(gs, before) {
		t.Errorf("Decide() mutated its input GameState")
	}
}

// bombThreatensBothKings pre-places a Countdown-3 bomb whose exposure scores apply
// regardless of the actor's action. Bombing is disabled on both allies, so moving
// farther from the blast is the actor's only lever.
func bombThreatensBothKings(gs *engine.GameState) (actor *engine.Unit) {
	allyKing := addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 8, Y: 1})
	allyKing.BombUsed = allyKing.MaxBombCount
	actor = addFighter(gs, engine.NewUnitID(2, 2), engine.Coordinate{X: 5, Y: 0})
	actor.BombUsed = actor.MaxBombCount
	addKing(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 2, Y: 1})

	bombID := engine.BombID(1)
	bombPos := engine.Coordinate{X: 0, Y: 0}
	gs.Bombs[bombID] = &engine.Bomb{ID: bombID, OwnerID: engine.NewUnitID(0, 0), Position: bombPos, Range: 2, Countdown: 3}
	gs.Grid[bombPos.Y][bombPos.X] = engine.Tile{Type: engine.TerrainPlain, OccupantType: engine.OccupantBomb, OccupantID: int64(bombID)}

	return actor
}

func TestDecide_MoveCommandWinsOnItsOwn(t *testing.T) {
	gs := newTestGameState(9, 2)
	actor := bombThreatensBothKings(gs)
	before := gs.DeepCopy()

	want := []engine.TurnCommand{engine.NewMoveCommand(actor.ID, engine.Coordinate{X: 7, Y: 0})}

	got := Decide(gs)

	if !slices.Equal(got, want) {
		t.Errorf("Decide() = %+v, want %+v", got, want)
	}
	if !reflect.DeepEqual(gs, before) {
		t.Errorf("Decide() mutated its input GameState")
	}
}

func TestDecide_MovesEvenWhenEveryPlanScoresBelowNeutral(t *testing.T) {
	gs := newTestGameState(20, 20)
	allyKing := addKing(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 4})
	allyKing.HP = 5
	allyKing.BombUsed = allyKing.MaxBombCount
	addKing(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 19, Y: 19})

	bombID := engine.BombID(1)
	bombPos := engine.Coordinate{X: 4, Y: 5}
	gs.Bombs[bombID] = &engine.Bomb{ID: bombID, OwnerID: engine.NewUnitID(0, 0), Position: bombPos, Range: 1, Countdown: 3}
	gs.Grid[bombPos.Y][bombPos.X] = engine.Tile{Type: engine.TerrainPlain, OccupantType: engine.OccupantBomb, OccupantID: int64(bombID)}

	before := gs.DeepCopy()

	got := Decide(gs)

	if len(got) != 1 || got[0].Type != engine.TurnCmdMove || got[0].UnitID != allyKing.ID {
		t.Fatalf("Decide() = %+v, want a single move from %v", got, allyKing.ID)
	}
	if !reflect.DeepEqual(gs, before) {
		t.Errorf("Decide() mutated its input GameState")
	}
}

func TestDecide_PrefersFartherSafeTile(t *testing.T) {
	gs := newTestGameState(20, 20)
	boss := addUnit(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 4, Y: 2}, "Prologue", engine.RoleBoss)
	boss.HP = 1
	boss.BombUsed = boss.MaxBombCount
	addKing(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 19, Y: 19})

	bombA := engine.BombID(1)
	bombAPos := engine.Coordinate{X: 4, Y: 3}
	gs.Bombs[bombA] = &engine.Bomb{ID: bombA, OwnerID: engine.NewUnitID(1, 0), Position: bombAPos, Range: 1, Countdown: 1}
	gs.Grid[bombAPos.Y][bombAPos.X] = engine.Tile{Type: engine.TerrainPlain, OccupantType: engine.OccupantBomb, OccupantID: int64(bombA)}

	bombB := engine.BombID(2)
	bombBPos := engine.Coordinate{X: 2, Y: 2}
	gs.Bombs[bombB] = &engine.Bomb{ID: bombB, OwnerID: engine.NewUnitID(1, 0), Position: bombBPos, Range: 2, Countdown: 2}
	gs.Grid[bombBPos.Y][bombBPos.X] = engine.Tile{Type: engine.TerrainPlain, OccupantType: engine.OccupantBomb, OccupantID: int64(bombB)}

	got := Decide(gs)

	if len(got) != 1 || got[0].Type != engine.TurnCmdMove || got[0].UnitID != boss.ID {
		t.Fatalf("Decide() = %+v, want a single move from %v", got, boss.ID)
	}
	if got[0].Target == (engine.Coordinate{X: 5, Y: 2}) {
		t.Errorf("Decide() moved to %+v, want a tile farther from bombB's blast", got[0].Target)
	}
}

func TestDecide_MovesToFarthestSafeTile(t *testing.T) {
	gs := newTestGameState(20, 20)
	boss := addUnit(gs, engine.NewUnitID(2, 1), engine.Coordinate{X: 5, Y: 3}, "Prologue", engine.RoleBoss)
	boss.HP = 5
	boss.BombUsed = boss.MaxBombCount
	addKing(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 19, Y: 19})

	bombB := engine.BombID(1)
	bombBPos := engine.Coordinate{X: 5, Y: 4}
	gs.Bombs[bombB] = &engine.Bomb{ID: bombB, OwnerID: boss.ID, Position: bombBPos, Range: 2, Countdown: 3}
	gs.Grid[bombBPos.Y][bombBPos.X] = engine.Tile{Type: engine.TerrainPlain, OccupantType: engine.OccupantBomb, OccupantID: int64(bombB)}

	bombA := engine.BombID(2)
	bombAPos := engine.Coordinate{X: 3, Y: 3}
	gs.Bombs[bombA] = &engine.Bomb{ID: bombA, OwnerID: engine.NewUnitID(1, 0), Position: bombAPos, Range: 2, Countdown: 4}
	gs.Grid[bombAPos.Y][bombAPos.X] = engine.Tile{Type: engine.TerrainPlain, OccupantType: engine.OccupantBomb, OccupantID: int64(bombA)}

	farTiles := []engine.Coordinate{{X: 5, Y: 1}, {X: 7, Y: 3}}

	got := Decide(gs)

	if len(got) != 1 || got[0].Type != engine.TurnCmdMove || got[0].UnitID != boss.ID || !slices.Contains(farTiles, got[0].Target) {
		t.Errorf("Decide() = %+v, want a single move from %v to one of %+v", got, boss.ID, farTiles)
	}
}

func TestDecide(t *testing.T) {
	tests := []struct {
		name  string
		setup func() *engine.GameState
		want  []engine.TurnCommand
	}{
		{
			name:  "No units at all",
			setup: func() *engine.GameState { return newTestGameState(5, 5) },
			want:  nil,
		},
		{
			name: "Both allies fully boxed in: no positive option",
			setup: func() *engine.GameState {
				gs := newTestGameState(9, 9)
				center := engine.Coordinate{X: 4, Y: 4}
				king := addKing(gs, engine.NewUnitID(2, 1), center)
				for _, off := range crossOffsets(max(king.Speed, king.BombMaxRange)) {
					setTerrainBlock(gs, engine.Coordinate{X: center.X + off.X, Y: center.Y + off.Y})
				}
				addKing(gs, engine.NewUnitID(1, 1), engine.Coordinate{X: 0, Y: 0})
				return gs
			},
			want: nil,
		},
		{
			name: "Single ally: takes the one plan scoring above neutral",
			setup: func() *engine.GameState {
				gs := newTestGameState(9, 2)
				corridorWithBreakthrough(gs)
				return gs
			},
			want: []engine.TurnCommand{{Type: engine.TurnCmdPlaceBomb, UnitID: engine.NewUnitID(2, 1)}},
		},
		{
			name: "Two allies: only the scoring one contributes",
			setup: func() *engine.GameState {
				gs := newTestGameState(9, 2)
				corridorWithBreakthrough(gs)
				addFighter(gs, engine.NewUnitID(2, 3), engine.Coordinate{X: 0, Y: 1})
				return gs
			},
			want: []engine.TurnCommand{{Type: engine.TurnCmdPlaceBomb, UnitID: engine.NewUnitID(2, 1)}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := tt.setup()
			before := gs.DeepCopy()

			got := Decide(gs)

			// The exact tile among tied-score options isn't asserted; tie-break order isn't a contract.
			if len(got) != len(tt.want) {
				t.Fatalf("Decide() = %+v, want %+v", got, tt.want)
			}
			for i, cmd := range got {
				if cmd.Type != tt.want[i].Type || cmd.UnitID != tt.want[i].UnitID {
					t.Errorf("Decide()[%d] = %+v, want Type %v UnitID %v", i, cmd, tt.want[i].Type, tt.want[i].UnitID)
				}
			}
			if !reflect.DeepEqual(gs, before) {
				t.Errorf("Decide() mutated its input GameState")
			}
		})
	}
}
