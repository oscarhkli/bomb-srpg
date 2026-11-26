package engine

import (
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"
)

func TestMatch_CommandPlaceBomb(t *testing.T) {
	validUnitID := NewUnitID(1, 0)
	origin := Coordinate{1, 1}
	validTarget := Coordinate{1, 2}

	t.Run("Success: delegates to WorkingState.PlaceBomb and logs the event", func(t *testing.T) {
		m := newActiveMatch(3, 3)
		m.WorkingState.Turn = 3
		m.WorkingState.Units[validUnitID] = &Unit{
			ID:           validUnitID,
			HP:           1,
			Team:         1,
			Position:     origin,
			BombPower:    3,
			BombMaxRange: 3,
			MaxBombCount: 2,
		}
		m.WorkingState.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantUnit, OccupantID: int64(validUnitID)}

		gameEvents, err := m.CommandPlaceBomb(validUnitID, validTarget)
		if err != nil {
			t.Fatalf("CommandPlaceBomb() unexpected error: %v", err)
		}
		if len(gameEvents) != 1 || !eventIs[BombPlaced](gameEvents[0]) {
			t.Fatalf("expected 1 BombPlaced GameEvent returned, got %#v", gameEvents)
		}
		if len(m.PlaybackLog) != 1 || !eventIs[BombPlaced](m.PlaybackLog[0]) {
			t.Fatalf("expected event submitted to PlaybackLog, got %#v", m.PlaybackLog)
		}
	})

	t.Run("Failure: PlaceBomb error propagates unchanged", func(t *testing.T) {
		m := newTestMatch(2, 2)

		gameEvents, err := m.CommandPlaceBomb(NewUnitID(3, 3), validTarget)
		if err == nil || !strings.Contains(err.Error(), "does not exist") {
			t.Fatalf("expected propagated not-found error, got %v", err)
		}
		if len(gameEvents) != 0 {
			t.Errorf("expected no GameEvents on error, got %#v", gameEvents)
		}
		if len(m.PlaybackLog) != 0 {
			t.Errorf("expected nothing submitted on error, got %#v", m.PlaybackLog)
		}
	})
}

func TestGameState_PlaceBomb(t *testing.T) {
	type testCase struct {
		name          string
		unitID        UnitID
		target        Coordinate
		setupState    func() *GameState
		wantErr       bool
		errContains   string
		verifyResults func(t *testing.T, gs *GameState, evt BombPlaced)
	}

	validUnitID := NewUnitID(1, 0)
	deadUnitID := NewUnitID(1, 2)
	origin := Coordinate{1, 1}
	validTarget := Coordinate{1, 2}
	outOfRangeTarget := Coordinate{5, 5}

	tests := []testCase{
		{
			name:   "Failure: Unit does not exist",
			unitID: NewUnitID(3, 3),
			target: validTarget,
			setupState: func() *GameState {
				return newTestMatch(2, 2).WorkingState
			},
			wantErr:     true,
			errContains: "unit 0x33 does not exist",
		},
		{
			name:   "Failure: Unit is dead",
			unitID: deadUnitID,
			target: validTarget,
			setupState: func() *GameState {
				gs := newTestMatch(2, 2).WorkingState
				gs.Units[deadUnitID] = &Unit{ID: deadUnitID, HP: 0, Team: 1, Position: origin}
				return gs
			},
			wantErr:     true,
			errContains: "unit 0x12 is dead",
		},
		{
			name:   "Failure: Wrong team turn",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *GameState {
				gs := newActiveMatch(2, 2).WorkingState
				gs.Units[validUnitID] = &Unit{
					ID:       validUnitID,
					HP:       1,
					Team:     2,
					Position: origin,
				}
				return gs
			},
			wantErr:     true,
			errContains: "unit 0x10 not active team",
		},
		{
			name:   "Failure: Data corruption - unit out of bounds",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *GameState {
				gs := newActiveMatch(3, 3).WorkingState
				gs.Units[validUnitID] = &Unit{
					ID:       validUnitID,
					HP:       1,
					Team:     1,
					Position: Coordinate{-5, -5},
				}
				return gs
			},
			wantErr:     true,
			errContains: "unit 0x10 out of bounds",
		},
		{
			name:   "Failure: Grid data desync",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *GameState {
				gs := newActiveMatch(2, 2).WorkingState
				gs.Units[validUnitID] = &Unit{
					ID:       validUnitID,
					HP:       1,
					Team:     1,
					Position: origin,
				}
				gs.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantNone}
				return gs
			},
			wantErr:     true,
			errContains: "unit 0x10 desynced at",
		},
		{
			name:   "Failure: Unit has used up all bombs",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *GameState {
				gs := newActiveMatch(3, 3).WorkingState
				gs.Units[validUnitID] = &Unit{
					ID:           validUnitID,
					HP:           1,
					Team:         1,
					Position:     origin,
					MaxBombCount: 2,
					BombUsed:     2,
				}
				gs.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantUnit, OccupantID: int64(validUnitID)}
				return gs
			},
			wantErr:     true,
			errContains: "unit 0x10 out of bombs",
		},
		{
			name:   "Failure: Target out of placement range",
			unitID: validUnitID,
			target: outOfRangeTarget,
			setupState: func() *GameState {
				gs := newActiveMatch(10, 10).WorkingState
				gs.Units[validUnitID] = &Unit{
					ID:           validUnitID,
					HP:           1,
					Team:         1,
					Position:     origin,
					BombMaxRange: 1,
					MaxBombCount: 1,
					BombUsed:     0,
				}
				gs.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantUnit, OccupantID: int64(validUnitID)}
				return gs
			},
			wantErr:     true,
			errContains: "target out of bomb range",
		},
		{
			name:   "Failure: Illegal target",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *GameState {
				gs := newActiveMatch(3, 3).WorkingState
				gs.Turn = 3
				gs.TurnBombCounter = 0
				gs.Bombs = make(map[BombID]*Bomb)
				gs.Units[validUnitID] = &Unit{
					ID:           validUnitID,
					HP:           1,
					Team:         1,
					Position:     origin,
					BombPower:    3,
					BombMaxRange: 3,
					MaxBombCount: 2,
					BombUsed:     0,
				}
				gs.Grid[validTarget.Y][validTarget.X].Type = TerrainBlock
				gs.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantUnit, OccupantID: int64(validUnitID)}
				return gs
			},
			wantErr:     true,
			errContains: "can only place on plain terrain",
		},
		{
			name:   "Failure: Unit has used skill in the turn",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *GameState {
				gs := newActiveMatch(3, 3).WorkingState
				gs.Turn = 3
				gs.TurnBombCounter = 0
				gs.Bombs = make(map[BombID]*Bomb)
				gs.Units[validUnitID] = &Unit{
					ID:           validUnitID,
					HP:           1,
					Team:         1,
					Position:     origin,
					BombPower:    3,
					BombMaxRange: 3,
					MaxBombCount: 2,
					BombUsed:     0,
					HasUsedSkill: true,
				}
				gs.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantUnit, OccupantID: int64(validUnitID)}
				return gs
			},
			wantErr:     true,
			errContains: "unit 0x10 already used skill this turn",
		},
		{
			name:   "Success: Bomb placed successfully",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *GameState {
				gs := newActiveMatch(3, 3).WorkingState
				gs.Turn = 3
				gs.TurnBombCounter = 0
				gs.Bombs = make(map[BombID]*Bomb)
				gs.Units[validUnitID] = &Unit{
					ID:           validUnitID,
					HP:           1,
					Team:         1,
					Position:     origin,
					BombPower:    3,
					BombMaxRange: 3,
					MaxBombCount: 2,
					BombUsed:     0,
				}
				gs.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantUnit, OccupantID: int64(validUnitID)}
				return gs
			},
			wantErr: false,
			verifyResults: func(t *testing.T, gs *GameState, evt BombPlaced) {
				t.Helper()
				if gs.TurnBombCounter != 1 {
					t.Errorf("expected TurnBombCounter to be 1, got %d", gs.TurnBombCounter)
				}

				expectedBombID := NewBombID(3, 1, validUnitID)
				bomb, exists := gs.Bombs[expectedBombID]
				if !exists {
					t.Fatalf("expected bomb tracking map entry under ID %#X missing", expectedBombID)
				}
				if bomb.OwnerID != validUnitID || bomb.Position != validTarget || bomb.Range != 3 || bomb.Countdown != 5 {
					t.Errorf("registered bomb structural parameters mismatched: %+v", bomb)
				}

				unit := gs.Units[validUnitID]
				if got, want := unit.BombUsed, 1; got != want {
					t.Errorf("expected unit bombUsed increased to %v, got: %v", want, got)
				}
				if !unit.HasUsedSkill {
					t.Errorf("expected unit HasUsedSkill to be set")
				}

				targetCell := gs.Grid[validTarget.Y][validTarget.X]
				if targetCell.OccupantType != OccupantBomb || targetCell.OccupantID != int64(expectedBombID) {
					t.Errorf("expected target grid tile to hold bomb entity, got type %v, id %d", targetCell.OccupantType, targetCell.OccupantID)
				}

				originCell := gs.Grid[origin.Y][origin.X]
				if originCell.OccupantType != OccupantUnit || originCell.OccupantID != int64(validUnitID) {
					t.Errorf("expected origin unit tile to remain intact, got type %v, id %d", originCell.OccupantType, originCell.OccupantID)
				}

				if want := (BombPlaced{UnitID: validUnitID, BombID: expectedBombID, Position: validTarget, Range: 3, Countdown: 5}); evt != want {
					t.Errorf("malformed BombPlacedEvent returned: %+v", evt)
				}
			},
		},
		{
			name:   "Success: SystemUnitID bypasses unit validation",
			unitID: SystemUnitID,
			target: validTarget,
			setupState: func() *GameState {
				gs := newTestMatch(3, 3).WorkingState
				gs.Turn = 3
				gs.TurnBombCounter = 0
				gs.Bombs = make(map[BombID]*Bomb)
				return gs
			},
			wantErr: false,
			verifyResults: func(t *testing.T, gs *GameState, evt BombPlaced) {
				t.Helper()
				expectedBombID := NewBombID(3, 1, SystemUnitID)
				bomb, exists := gs.Bombs[expectedBombID]
				if !exists {
					t.Fatalf("expected sudden-death bomb registered under ID %#X", expectedBombID)
				}
				if bomb.Range != bombDefaultPower {
					t.Errorf("expected system bomb to use BombDefaultPower %d, got %d", bombDefaultPower, bomb.Range)
				}
				if evt.UnitID != SystemUnitID {
					t.Errorf("malformed system BombPlacedEvent: %+v", evt)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := tt.setupState()

			evt, err := gs.PlaceBomb(tt.unitID, tt.target)

			if (err != nil) != tt.wantErr {
				t.Fatalf("PlaceBomb() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
			}
			if tt.verifyResults != nil {
				tt.verifyResults(t, gs, evt)
			}
		})
	}
}

func TestGameState_IsLandingLegal_OccupantBomb(t *testing.T) {
	state := &GameState{
		Grid: [][]Tile{
			{{Type: TerrainPlain}, {Type: TerrainPlain, OccupantType: OccupantUnit}, {Type: TerrainBlock}},
		},
	}

	tests := []struct {
		name          string
		pos           Coordinate
		expectError   bool
		errorContains string
	}{
		{
			name:        "Success: Within bound, TerrainPlain, on OccupantNone",
			pos:         Coordinate{0, 0},
			expectError: false,
		},
		{
			name:          "Failure: Out of bound",
			pos:           Coordinate{100, 0},
			expectError:   true,
			errorContains: "out of bounds",
		},
		{
			name:          "Failure: Within bound, non-TerrainPlain",
			pos:           Coordinate{2, 0},
			expectError:   true,
			errorContains: "can only place on plain terrain",
		},
		{
			name:          "Failure: Within bound, TerrainPlain, on non-OccupantNone",
			pos:           Coordinate{1, 0},
			expectError:   true,
			errorContains: "cell occupied by",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := state.IsLandingLegal(tt.pos, OccupantBomb)

			if tt.expectError {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, got '%s'", err.Error())
			}
		})
	}
}

func TestMatch_ResolveTurn_ExplosionAndBlast(t *testing.T) {
	king := mustArchetype(t, "King")

	fighter := mustArchetype(t, "Fighter")

	t.Run("Natural countdown tick reduces for countdown bombs and does not detonate early", func(t *testing.T) {
		m := newTestMatch(16, 16)
		u1 := NewUnitID(1, 0)
		u2 := NewUnitID(2, 0)
		u3 := NewUnitID(1, 1)
		u4 := NewUnitID(2, 1)
		b1 := NewBombID(1, 1, u1)
		b2 := NewBombID(1, 2, u2)
		m.WorkingState.Units[u1] = &Unit{ID: u1, Team: 1, HP: 1, Type: king, Role: RoleKing}
		m.WorkingState.Units[u2] = &Unit{ID: u2, Team: 2, HP: 1, Type: king, Role: RoleKing}
		m.WorkingState.Units[u3] = &Unit{ID: u3, Team: 1, HP: 1, Type: fighter}
		m.WorkingState.Units[u4] = &Unit{ID: u4, Team: 2, HP: 1, Type: fighter}
		m.WorkingState.Bombs[b1] = &Bomb{ID: b1, Countdown: 3, Range: 2, Position: Coordinate{5, 5}}
		m.WorkingState.Bombs[b2] = &Bomb{ID: b2, Countdown: -1, Range: 2, Position: Coordinate{15, 15}}
		m.WorkingState.Grid[5][5] = Tile{OccupantType: OccupantBomb, OccupantID: int64(b1)}
		m.WorkingState.Grid[15][15] = Tile{OccupantType: OccupantBomb, OccupantID: int64(b2)}

		_, gameEvents := m.ResolveTurn()

		if len(gameEvents) != 1 {
			t.Errorf("expected 1 GameEvent returned, got %d", len(gameEvents))
		}
		resEvt := gameEvents[0]
		if want := (BombCountdownUpdated{BombID: b1, Position: Coordinate{5, 5}, Countdown: 2}); resEvt != want {
			t.Errorf("malformed EvtBombCountdownUpdated returned: %+v", resEvt)
		}
		if m.WorkingState.Bombs[b1].Countdown != 2 {
			t.Errorf("Expected Bomb %#X to reduce to 2, got %d", b1, m.WorkingState.Bombs[b1].Countdown)
		}
		if m.WorkingState.Bombs[b2].Countdown != -1 {
			t.Errorf("Expected Bomb %#X to remains at -1, got %d", b2, m.WorkingState.Bombs[b2].Countdown)
		}
		if m.WorkingState.Grid[5][5].OccupantType != OccupantBomb {
			t.Errorf("Grid corruption: Ticking bomb cleared prematurely from cell footprint")
		}
		if m.WorkingState.Grid[15][15].OccupantType != OccupantBomb {
			t.Errorf("Grid corruption: Ticking bomb cleared prematurely from cell footprint")
		}
		if len(m.PlaybackLog) != 0 {
			t.Errorf("Expectd clean slice array from PlaybackLog, got %d items", len(m.PlaybackLog))
		}
	})

	t.Run("Units caught in overlapping blast patterns receive exactly 1 flat HP damage max", func(t *testing.T) {
		m := newTestMatch(16, 16)
		u1 := NewUnitID(1, 0)
		m.WorkingState.Units[u1] = &Unit{ID: u1, HP: 3, Position: Coordinate{2, 2}, BombUsed: 2, HasMoved: true}
		m.WorkingState.Grid[2][2] = Tile{OccupantType: OccupantUnit, OccupantID: int64(u1)}

		u2 := NewUnitID(2, 0)
		m.WorkingState.Units[u2] = &Unit{ID: u2, HP: 1, Position: Coordinate{1, 0}, BombUsed: 2, HasMoved: true, HasUsedSkill: true}
		m.WorkingState.Grid[0][1] = Tile{OccupantType: OccupantUnit, OccupantID: int64(u2)}

		b1 := NewBombID(1, 1, u1)
		m.WorkingState.Bombs[b1] = &Bomb{ID: b1, OwnerID: u1, Countdown: 1, Range: 3, Position: Coordinate{2, 0}}
		m.WorkingState.Grid[0][2] = Tile{OccupantType: OccupantBomb, OccupantID: int64(b1)}

		b2 := NewBombID(1, 2, u1)
		m.WorkingState.Bombs[b2] = &Bomb{ID: b2, OwnerID: u1, Countdown: 1, Range: 3, Position: Coordinate{0, 2}}
		m.WorkingState.Grid[2][0] = Tile{OccupantType: OccupantBomb, OccupantID: int64(b2)}

		sb1 := 1
		m.WorkingState.SoftBlocks[sb1] = &SoftBlock{ID: sb1, Position: Coordinate{0, 3}}
		m.WorkingState.Grid[3][0] = Tile{OccupantType: OccupantSoftBlock, OccupantID: int64(sb1)}

		_, gameEvents := m.ResolveTurn()

		if got, want := m.WorkingState.Units[u1].BombUsed, 0; got != want {
			t.Errorf("expect Unit %#X bombUsed = %v, got %v", u1, got, want)
		}
		if got, want := m.WorkingState.Units[u2].BombUsed, 2; got != want {
			t.Errorf("expect Unit %#X bombUsed = %v, got %v", u2, got, want)
		}

		for uid, u := range m.WorkingState.Units {
			if u.HasMoved || u.HasUsedSkill {
				t.Errorf("expect Unit %#X HasMoved and HasUsedSkill guard reset to false, got HasMoved = %v, HasUsedSkill = %v", uid, u.HasMoved, u.HasUsedSkill)
			}
		}

		if m.WorkingState.Units[u1].HP != 2 {
			t.Errorf("Flat injury rule failed for Unit %#X! Expected Unit HP = 2, got %d", u1, m.WorkingState.Units[u1].HP)
		}
		if m.WorkingState.Units[u2].HP != 0 {
			t.Errorf("Flat injury rule failed for Unit %#X! Expected Unit HP = 0, got %d", u2, m.WorkingState.Units[u2].HP)
		}

		damageEventsCount := 0
		unitDiedEventsCount := 0
		softBlockDestroyedEventsCount := 0
		for _, evt := range gameEvents {
			switch evt := evt.(type) {
			case UnitDamaged:
				damageEventsCount++
				if wantPos := m.WorkingState.Units[evt.UnitID].Position; evt.Position != wantPos {
					t.Errorf("UnitDamaged for %#X: expected Position %v, got %v", evt.UnitID, wantPos, evt.Position)
				}
			case UnitDied:
				unitDiedEventsCount++
				if wantPos := m.WorkingState.Units[evt.UnitID].Position; evt.Position != wantPos {
					t.Errorf("UnitDied for %#X: expected Position %v, got %v", evt.UnitID, wantPos, evt.Position)
				}
			case SoftBlockDestroyed:
				softBlockDestroyedEventsCount++
			case UnitMoved, BombPlaced, BombCountdownUpdated, BombExploded, MatchEnded:
			}
		}
		if damageEventsCount != 2 {
			t.Errorf("Expected exactly 2 damage log packet event, found %d", damageEventsCount)
		}
		if unitDiedEventsCount != 1 {
			t.Errorf("Expected exactly 1 casulty log packet event, found %d", unitDiedEventsCount)
		}
		if softBlockDestroyedEventsCount != 1 {
			t.Errorf("Expected exactly 1 softBlock destroyed log packet event, found %d", softBlockDestroyedEventsCount)
		}
		if m.WorkingState.Grid[0][2].OccupantType != OccupantNone {
			t.Errorf("Grid Clearance Bug: Exploded bomb positions %#v failed to revert to OccupantNone, got %v", Coordinate{2, 0}, m.WorkingState.Grid[0][2].OccupantType)
		}
		if m.WorkingState.Grid[2][0].OccupantType != OccupantNone {
			t.Errorf("Grid Clearance Bug: Exploded bomb positions %#v failed to revert to OccupantNone, got %v", Coordinate{0, 2}, m.WorkingState.Grid[2][0].OccupantType)
		}
		if m.WorkingState.Grid[2][2].OccupantType != OccupantUnit {
			t.Errorf("Grid Cleared Erroneously: Living unit was scrubbed from position %#v footprint, got %v", Coordinate{2, 2}, m.WorkingState.Grid[2][2].OccupantType)
		}
		if m.WorkingState.Grid[0][1].OccupantType != OccupantNone {
			t.Errorf("Grid Clearance Bug: Dead unit positions %#v failed to revert to OccupantNone, got %v", Coordinate{1, 0}, m.WorkingState.Grid[0][1].OccupantType)
		}
		if m.WorkingState.Grid[3][0].OccupantType != OccupantNone {
			t.Errorf("Grid Clearance Bug: Destroyed softBlock positions %#v failed to revert to OccupantNone, got %v", Coordinate{0, 3}, m.WorkingState.Grid[3][0].OccupantType)
		}
		if len(m.PlaybackLog) != 0 {
			t.Errorf("Expectd clean slice array from PlaybackLog, got %d items", len(m.PlaybackLog))
		}
	})
}

func TestMatch_ResolveTurn_CascadingChainReactions(t *testing.T) {
	t.Run("Ticking bomb triggers secondary explosive via proximity chain reaction", func(t *testing.T) {
		m := newTestMatch(16, 16)
		unitID := NewUnitID(1, 0)

		b1 := NewBombID(1, 1, unitID)
		m.WorkingState.Bombs[b1] = &Bomb{ID: b1, Countdown: 1, Range: 2, Position: Coordinate{1, 1}}
		m.WorkingState.Grid[1][1] = Tile{OccupantType: OccupantBomb, OccupantID: int64(b1)}

		b2 := NewBombID(1, 2, unitID)
		m.WorkingState.Bombs[b2] = &Bomb{ID: b2, Countdown: 3, Range: 2, Position: Coordinate{1, 2}}
		m.WorkingState.Grid[2][1] = Tile{OccupantType: OccupantBomb, OccupantID: int64(b2)}

		_, gameEvents := m.ResolveTurn()

		if _, exists := m.WorkingState.Bombs[b1]; exists {
			t.Error("Bomb 1 failed to clear")
		}
		if _, exists := m.WorkingState.Bombs[b2]; exists {
			t.Error("Bomb 2 failed to ignite and clear via chain reaction!")
		}
		if m.WorkingState.Grid[1][1].OccupantType != OccupantNone {
			t.Errorf("Grid Clearance Bug: Exploded bomb positions %#v failed to revert to OccupantNone, got %v", Coordinate{1, 1}, m.WorkingState.Grid[1][1].OccupantType)
		}
		if m.WorkingState.Grid[2][1].OccupantType != OccupantNone {
			t.Errorf("Grid Clearance Bug: Exploded bomb positions %#v failed to revert to OccupantNone, got %v", Coordinate{1, 2}, m.WorkingState.Grid[1][2].OccupantType)
		}

		wantExplosionPos := map[BombID]Coordinate{b1: {1, 1}, b2: {1, 2}}
		explosionPackets := 0
		for _, evt := range gameEvents {
			if exploded, isExploded := evt.(BombExploded); isExploded {
				explosionPackets++
				wantPos, ok := wantExplosionPos[exploded.BombID]
				if !ok || exploded.Position != wantPos {
					t.Errorf("BombExploded for %#X: expected Position %v, got %v", exploded.BombID, wantPos, exploded.Position)
				}
			}
		}
		if explosionPackets != 2 {
			t.Errorf("Expected 2 separate explosion log streams inside replay, found %d", explosionPackets)
		}
	})

	t.Run("Soft blocks act as solid line of sight obstacles shielding behind tiles during turn", func(t *testing.T) {
		m := newTestMatch(16, 16)
		unitID := NewUnitID(1, 0)

		bombID := NewBombID(1, 1, unitID)
		m.WorkingState.Bombs[bombID] = &Bomb{ID: bombID, Countdown: 1, Range: 5, Position: Coordinate{0, 0}}
		m.WorkingState.Grid[0][0] = Tile{OccupantType: OccupantBomb, OccupantID: int64(bombID)}

		softBlockID := 55
		m.WorkingState.SoftBlocks[softBlockID] = &SoftBlock{ID: softBlockID, Position: Coordinate{1, 0}}
		m.WorkingState.Grid[0][1] = Tile{OccupantType: OccupantSoftBlock, OccupantID: int64(softBlockID)}

		b2 := NewBombID(1, 2, unitID)
		m.WorkingState.Bombs[b2] = &Bomb{ID: b2, Countdown: 3, Range: 2, Position: Coordinate{2, 0}}
		m.WorkingState.Grid[0][2] = Tile{OccupantType: OccupantBomb, OccupantID: int64(b2)}

		_, gameEvents := m.ResolveTurn()

		// The soft block is destroyed but still shields Bomb 2 from the blast within this pass.
		if _, ok := m.WorkingState.SoftBlocks[softBlockID]; ok {
			t.Error("Soft block failed to be destroyed by direct ray hit")
		}
		if _, ok := m.WorkingState.Bombs[b2]; !ok {
			t.Error("Security Breach: Bomb 2 ignited through a solid shielding soft block!")
		}
		if m.WorkingState.Grid[0][1].OccupantType != OccupantNone {
			t.Errorf("Grid Clearance Bug: Exploded SoftBlock positions %#v failed to revert to OccupantNone, got %v", Coordinate{1, 0}, m.WorkingState.Grid[0][1].OccupantType)
		}

		softBlockDestroyedPackets := 0
		for _, evt := range gameEvents {
			if eventIs[SoftBlockDestroyed](evt) {
				softBlockDestroyedPackets++
			}
		}
		if softBlockDestroyedPackets != 1 {
			t.Errorf("Expected 1 soft block destroyed log streams inside replay, found %d", softBlockDestroyedPackets)
		}
	})
}

func TestGameState_ProcessChain_Edges(t *testing.T) {
	owner := NewUnitID(1, 0)
	victim := NewUnitID(2, 0)
	bombA := NewBombID(1, 1, owner)
	bombB := NewBombID(1, 2, owner)
	ghost := NewBombID(1, 9, owner)

	addOwner := func(m *Match, bombUsed int) {
		m.WorkingState.Units[owner] = &Unit{ID: owner, Team: 1, HP: 1, BombUsed: bombUsed}
	}
	type queued struct {
		queue   []BombID
		ignited map[BombID]bool
	}
	fresh := func(ids ...BombID) queued {
		q := queued{queue: ids, ignited: map[BombID]bool{}}
		for _, id := range ids {
			q.ignited[id] = true
		}
		return q
	}

	tests := []struct {
		name          string
		setup         func(m *Match) queued
		wantExploded  []BombID
		wantIgnited   map[BombID]bool
		wantDamaged   map[UnitID]bool
		wantBlocks    map[int]bool
		wantItems     map[int]bool
		wantBombsLeft []BombID
		wantBombUsed  int
	}{
		{
			name: "tile holds an already-ignited bomb",
			setup: func(m *Match) queued {
				addOwner(m, 2)
				m.addBomb(1, owner, Coordinate{0, 1}, 2, 0)
				m.addBomb(2, owner, Coordinate{1, 1}, 2, 3)
				q := fresh(bombA)
				q.ignited[bombB] = true
				return q
			},
			wantExploded:  []BombID{bombA},
			wantIgnited:   map[BombID]bool{bombA: true, bombB: true},
			wantBombsLeft: []BombID{bombB},
			wantBombUsed:  1,
		},
		{
			name: "tile holds a bomb ID missing from Bombs",
			setup: func(m *Match) queued {
				addOwner(m, 2)
				m.addBomb(1, owner, Coordinate{0, 1}, 2, 0)
				m.WorkingState.Grid[1][1] = Tile{Type: TerrainPlain, OccupantType: OccupantBomb, OccupantID: int64(ghost)}
				return fresh(bombA)
			},
			wantExploded: []BombID{bombA},
			wantIgnited:  map[BombID]bool{bombA: true},
			wantBombUsed: 1,
		},
		{
			name: "queued bomb ID missing from Bombs",
			setup: func(m *Match) queued {
				addOwner(m, 2)
				return fresh(ghost)
			},
			wantIgnited:  map[BombID]bool{ghost: true},
			wantBombUsed: 2,
		},
		{
			name: "bomb owner missing from Units",
			setup: func(m *Match) queued {
				m.addBomb(1, owner, Coordinate{0, 1}, 2, 0)
				return fresh(bombA)
			},
			wantExploded: []BombID{bombA},
			wantIgnited:  map[BombID]bool{bombA: true},
		},
		{
			name: "BombUsed already 0",
			setup: func(m *Match) queued {
				addOwner(m, 0)
				m.addBomb(1, owner, Coordinate{0, 1}, 2, 0)
				return fresh(bombA)
			},
			wantExploded: []BombID{bombA},
			wantIgnited:  map[BombID]bool{bombA: true},
			wantBombUsed: 0,
		},
		{
			name: "BombUsed decrements by one",
			setup: func(m *Match) queued {
				addOwner(m, 2)
				m.addBomb(1, owner, Coordinate{0, 1}, 2, 0)
				return fresh(bombA)
			},
			wantExploded: []BombID{bombA},
			wantIgnited:  map[BombID]bool{bombA: true},
			wantBombUsed: 1,
		},
		{
			name: "blast hits a unit",
			setup: func(m *Match) queued {
				addOwner(m, 1)
				m.addBomb(1, owner, Coordinate{0, 1}, 2, 0)
				m.addUnit(2, 0, Archetype{}, RoleNormal, Coordinate{1, 1}, 1)
				return fresh(bombA)
			},
			wantExploded: []BombID{bombA},
			wantIgnited:  map[BombID]bool{bombA: true},
			wantDamaged:  map[UnitID]bool{victim: true},
		},
		{
			name: "blast hits a soft block",
			setup: func(m *Match) queued {
				addOwner(m, 1)
				m.addBomb(1, owner, Coordinate{0, 1}, 2, 0)
				m.addSoftBlock(7, Coordinate{1, 1})
				return fresh(bombA)
			},
			wantExploded: []BombID{bombA},
			wantIgnited:  map[BombID]bool{bombA: true},
			wantBlocks:   map[int]bool{7: true},
		},
		{
			name: "blast hits an item",
			setup: func(m *Match) queued {
				addOwner(m, 1)
				m.addBomb(1, owner, Coordinate{0, 1}, 2, 0)
				m.WorkingState.Grid[1][1] = Tile{Type: TerrainPlain, OccupantType: OccupantItem, OccupantID: 5}
				return fresh(bombA)
			},
			wantExploded: []BombID{bombA},
			wantIgnited:  map[BombID]bool{bombA: true},
			wantItems:    map[int]bool{5: true},
		},
		{
			name: "blast reaches a second bomb",
			setup: func(m *Match) queued {
				addOwner(m, 2)
				m.addBomb(1, owner, Coordinate{0, 1}, 2, 0)
				m.addBomb(2, owner, Coordinate{1, 1}, 1, 3)
				return fresh(bombA)
			},
			wantExploded: []BombID{bombA, bombB},
			wantIgnited:  map[BombID]bool{bombA: true, bombB: true},
			wantBombUsed: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestMatch(8, 3)
			q := tt.setup(m)
			gs := m.WorkingState
			damaged, blocks, items := map[UnitID]bool{}, map[int]bool{}, map[int]bool{}

			gameEvents := gs.processChainDetonations(q.queue, q.ignited, gs.cloneGridSnapshot(), damaged, blocks, items)

			var gotExploded []BombID
			for _, evt := range gameEvents {
				if exploded, ok := evt.(BombExploded); ok {
					gotExploded = append(gotExploded, exploded.BombID)
				}
			}
			if !slices.Equal(gotExploded, tt.wantExploded) {
				t.Errorf("exploded bombs = %v, want %v", gotExploded, tt.wantExploded)
			}
			if !maps.Equal(q.ignited, tt.wantIgnited) {
				t.Errorf("ignited = %v, want %v", q.ignited, tt.wantIgnited)
			}
			if !maps.Equal(damaged, tt.wantDamaged) {
				t.Errorf("damaged units = %v, want %v", damaged, tt.wantDamaged)
			}
			if !maps.Equal(blocks, tt.wantBlocks) {
				t.Errorf("destroyed soft blocks = %v, want %v", blocks, tt.wantBlocks)
			}
			if !maps.Equal(items, tt.wantItems) {
				t.Errorf("destroyed items = %v, want %v", items, tt.wantItems)
			}
			gotLeft := slices.Sorted(maps.Keys(gs.Bombs))
			if !slices.Equal(gotLeft, tt.wantBombsLeft) {
				t.Errorf("remaining bombs = %v, want %v", gotLeft, tt.wantBombsLeft)
			}
			if u, ok := gs.Units[owner]; ok && u.BombUsed != tt.wantBombUsed {
				t.Errorf("owner BombUsed = %d, want %d", u.BombUsed, tt.wantBombUsed)
			}
		})
	}
}

func TestGameState_PlaceBomb_ErrorOrder(t *testing.T) {
	unitID := NewUnitID(1, 0)
	origin := Coordinate{1, 1}
	target := Coordinate{1, 2}

	tests := []struct {
		name   string
		mutate func(gs *GameState, u *Unit)
		target Coordinate
		want   error
	}{
		{
			name:   "skill used and out of bombs reports the skill",
			mutate: func(_ *GameState, u *Unit) { u.HasUsedSkill, u.BombUsed = true, u.MaxBombCount },
			target: target,
			want:   ErrAlreadyUsedSkill,
		},
		{
			name:   "out of bombs and out of range reports the bombs",
			mutate: func(_ *GameState, u *Unit) { u.BombUsed = u.MaxBombCount },
			target: Coordinate{1, 6},
			want:   ErrOutOfBombs,
		},
		{
			name:   "out of range and illegal landing reports the range",
			mutate: func(gs *GameState, _ *Unit) { gs.Grid[6][1].Type = TerrainBlock },
			target: Coordinate{1, 6},
			want:   ErrOutOfBombRange,
		},
		{
			name:   "dead unit that used its skill reports the death",
			mutate: func(_ *GameState, u *Unit) { u.HP, u.HasUsedSkill = 0, true },
			target: target,
			want:   ErrUnitDead,
		},
		{
			name:   "wrong-team unit that used its skill reports the team",
			mutate: func(_ *GameState, u *Unit) { u.Team, u.HasUsedSkill = 2, true },
			target: target,
			want:   ErrNotActiveTeam,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := newActiveMatch(8, 8).WorkingState
			unit := &Unit{ID: unitID, HP: 1, Team: 1, Position: origin, BombPower: 3, BombMaxRange: 3, MaxBombCount: 2}
			gs.Units[unitID] = unit
			gs.Grid[origin.Y][origin.X] = Tile{Type: TerrainPlain, OccupantType: OccupantUnit, OccupantID: int64(unitID)}
			tt.mutate(gs, unit)
			wantUsed, wantSkill := unit.BombUsed, unit.HasUsedSkill

			_, err := gs.PlaceBomb(unitID, tt.target)

			if !errors.Is(err, tt.want) {
				t.Fatalf("PlaceBomb() error = %v, want %v", err, tt.want)
			}
			if unit.BombUsed != wantUsed || unit.HasUsedSkill != wantSkill {
				t.Errorf("counters changed on failure: BombUsed %d, HasUsedSkill %v", unit.BombUsed, unit.HasUsedSkill)
			}
			if len(gs.Bombs) != 0 || gs.TurnBombCounter != 0 {
				t.Errorf("failed placement left bombs %v, counter %d", gs.Bombs, gs.TurnBombCounter)
			}
		})
	}
}

func TestGameState_PlaceBomb_SystemUnit(t *testing.T) {
	bystander := NewUnitID(1, 0)
	tests := []struct {
		name   string
		target Coordinate
		setup  func(gs *GameState)
	}{
		{
			name:   "occupied target",
			target: Coordinate{1, 1},
			setup: func(gs *GameState) {
				gs.Grid[1][1] = Tile{Type: TerrainPlain, OccupantType: OccupantUnit, OccupantID: int64(bystander)}
			},
		},
		{
			name:   "out-of-range target",
			target: Coordinate{7, 7},
			setup:  func(*GameState) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := newTestMatch(8, 8).WorkingState
			gs.Turn = 2
			gs.Units[bystander] = &Unit{ID: bystander, HP: 1, Team: 1, Position: Coordinate{0, 0}}
			tt.setup(gs)

			evt, err := gs.PlaceBomb(SystemUnitID, tt.target)

			if err != nil {
				t.Fatalf("PlaceBomb(SystemUnitID) error = %v, want nil", err)
			}
			wantID := NewBombID(2, 1, SystemUnitID)
			bomb, ok := gs.Bombs[wantID]
			if !ok || bomb.Position != tt.target || bomb.Range != bombDefaultPower {
				t.Fatalf("bomb %#X = %+v, want at %v with range %d", wantID, bomb, tt.target, bombDefaultPower)
			}
			if evt.BombID != wantID {
				t.Errorf("event = %+v, want BombPlaced for %#X", evt, wantID)
			}
			if tile := gs.Grid[tt.target.Y][tt.target.X]; tile.OccupantType != OccupantBomb || tile.OccupantID != int64(wantID) {
				t.Errorf("target tile = %+v, want bomb %#X", tile, wantID)
			}
			if u := gs.Units[bystander]; u.BombUsed != 0 || u.HasUsedSkill {
				t.Errorf("bystander counters changed: %+v", u)
			}
		})
	}
}
