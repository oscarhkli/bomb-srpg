package engine

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestMatch_ResetTurn(t *testing.T) {
	unitID := NewUnitID(1, 0)
	m := Match{
		TrueState: &GameState{
			Turn: 1,
			Units: map[UnitID]*Unit{
				unitID: {HP: 5},
			},
		},
		PlaybackLog: []GameEvent{UnitMoved{UnitID: 16, From: Coordinate{X: 1, Y: 2}, To: Coordinate{X: 1, Y: 3}}},
	}
	m.WorkingState = m.TrueState.DeepCopy()

	m.WorkingState.Units[unitID].HP = 2
	m.WorkingState.Units[unitID].HasMoved = true

	m.ResetTurn()

	if m.WorkingState.Units[unitID].HP != 5 || m.WorkingState.Units[unitID].HasMoved {
		t.Errorf("Rollback invariant broken: uncommitted unit action still exists in WorkingState after reset!")
	}
	if m.TrueState.Units[unitID].HP != 5 || m.WorkingState.Units[unitID].HasMoved {
		t.Errorf("CRITICAL POINTER LEAK: Sandbox action permanently corrupted your master TrueState checkpoint!")
	}
	if len(m.PlaybackLog) > 0 {
		t.Errorf("Rollback invariant broken: playbackLog still exists after reset!")
	}
}

func TestMatch_SubmitAction_AllowResetTurn(t *testing.T) {
	unitID := NewUnitID(1, 0)
	m := Match{
		TrueState: &GameState{
			Turn: 1,
			Units: map[UnitID]*Unit{
				unitID: {Position: Coordinate{1, 0}},
			},
		},
		GameCfg: GameCfg{
			AllowResetTurn: true,
		},
	}
	m.WorkingState = m.TrueState.DeepCopy()
	m.WorkingState.Units[unitID].Position = Coordinate{10, 100}

	event := UnitMoved{UnitID: unitID, From: Coordinate{1, 0}, To: Coordinate{10, 100}}

	m.submitAction(event)

	if len(m.PlaybackLog) == 0 {
		t.Errorf("Expected PlaybackLog to capture and retain the submitted event payload successfully")
	}

	if reflect.DeepEqual(m.TrueState, m.WorkingState) {
		t.Error("Expected TrueState not to be overridden by Working state, got overridden")
	}
}

func TestMatch_SubmitAction_DisallowResetTurn(t *testing.T) {
	unitID := NewUnitID(1, 0)
	m := Match{
		TrueState: &GameState{
			Turn: 1,
			Units: map[UnitID]*Unit{
				unitID: {Position: Coordinate{1, 0}},
			},
		},
		GameCfg: GameCfg{
			AllowResetTurn: false,
		},
	}
	m.WorkingState = m.TrueState.DeepCopy()
	m.WorkingState.Units[unitID].Position = Coordinate{10, 100}

	event := UnitMoved{UnitID: unitID, From: Coordinate{1, 0}, To: Coordinate{10, 100}}

	m.submitAction(event)

	if len(m.PlaybackLog) == 0 {
		t.Errorf("Expected PlaybackLog to capture and retain the submitted event payload successfully")
	}

	if !reflect.DeepEqual(m.TrueState, m.WorkingState) {
		t.Error("Expected TrueState to be overridden by Working state, got not overridden")
	}
}

func TestMatch_Surrender(t *testing.T) {
	tests := []struct {
		name         string
		teamID       int
		winnerTeamID int
	}{
		{"Team 1 surrender", 1, 2},
		{"Team 2 surrender", 2, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestMatch(1, 2)
			m.PlaybackLog = append(m.PlaybackLog, UnitDied{UnitID: 16, Position: Coordinate{X: 0, Y: 0}})

			gameEvents := m.Surrender(tt.teamID)

			if len(gameEvents) != 1 {
				t.Errorf("Expected 1 event for surrender, got %v", len(gameEvents))
			}

			var foundEndedEvent bool
			for _, evt := range gameEvents {
				if ended, ok := evt.(MatchEnded); ok {
					foundEndedEvent = true
					if ended.WinnerTeamID != tt.winnerTeamID {
						t.Errorf("Malformed MatchEnded! Got winner %d", ended.WinnerTeamID)
					}
				}
			}
			if !foundEndedEvent {
				t.Error("Missing critical MatchEndedEvent token inside returned telemetry array stream")
			}

			if len(m.PlaybackLog) != 0 {
				t.Errorf("Expectd clean slice array from PlaybackLog, got %d items", len(m.PlaybackLog))
			}
		})
	}
}

func TestMatch_CommandMoveUnit(t *testing.T) {
	validUnitID := NewUnitID(1, 0)
	origin := Coordinate{1, 1}
	validTarget := Coordinate{1, 2}

	t.Run("Success: delegates to WorkingState.MoveUnit and logs the event", func(t *testing.T) {
		m := newActiveMatch(3, 3)
		m.WorkingState.Units[validUnitID] = &Unit{
			ID:       validUnitID,
			HP:       1,
			Team:     1,
			Position: origin,
			Speed:    3,
		}
		m.WorkingState.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantUnit, OccupantID: int64(validUnitID)}

		gameEvents, err := m.commandMoveUnit(validUnitID, validTarget)
		if err != nil {
			t.Fatalf("CommandMoveUnit() unexpected error: %v", err)
		}
		if len(gameEvents) != 1 || !eventIs[UnitMoved](gameEvents[0]) {
			t.Fatalf("expected 1 UnitMoved GameEvent returned, got %#v", gameEvents)
		}
		if len(m.PlaybackLog) != 1 || !eventIs[UnitMoved](m.PlaybackLog[0]) {
			t.Fatalf("expected event submitted to PlaybackLog, got %#v", m.PlaybackLog)
		}
	})

	t.Run("Failure: MoveUnit error propagates unchanged", func(t *testing.T) {
		m := newTestMatch(2, 2)

		gameEvents, err := m.commandMoveUnit(NewUnitID(3, 3), validTarget)
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

func TestGameState_MoveUnit(t *testing.T) {
	type testCase struct {
		name          string
		unitID        UnitID
		target        Coordinate
		setupState    func() *GameState
		wantErr       bool
		errContains   string
		verifyResults func(t *testing.T, gs *GameState, evt UnitMoved)
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
					Speed:    3,
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
			name:   "Failure: Target out of moving range",
			unitID: validUnitID,
			target: outOfRangeTarget,
			setupState: func() *GameState {
				gs := newActiveMatch(10, 10).WorkingState
				gs.Units[validUnitID] = &Unit{
					ID:       validUnitID,
					HP:       1,
					Team:     1,
					Position: origin,
					Speed:    3,
				}
				gs.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantUnit, OccupantID: int64(validUnitID)}
				return gs
			},
			wantErr:     true,
			errContains: "target out of move range",
		},
		{
			name:   "Failure: Target is reachable but not landable terrain",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *GameState {
				gs := newActiveMatch(3, 3).WorkingState
				gs.Units[validUnitID] = &Unit{
					ID:       validUnitID,
					HP:       1,
					Team:     1,
					Position: origin,
					Speed:    3,
				}
				gs.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantUnit, OccupantID: int64(validUnitID)}
				gs.Grid[validTarget.Y][validTarget.X].Type = TerrainWater
				return gs
			},
			wantErr:     true,
			errContains: "invalid landing position",
		},
		{
			name:   "Failure: Unit has moved in the same turn",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *GameState {
				gs := newActiveMatch(3, 3).WorkingState
				gs.Units[validUnitID] = &Unit{
					ID:       validUnitID,
					HP:       1,
					Team:     1,
					Position: origin,
					Speed:    3,
					HasMoved: true,
				}
				gs.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantUnit, OccupantID: int64(validUnitID)}
				return gs
			},
			wantErr:     true,
			errContains: "unit 0x10 already moved this turn",
		},
		{
			name:   "Success: Unit moves successfully",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *GameState {
				gs := newActiveMatch(3, 3).WorkingState
				gs.Units[validUnitID] = &Unit{
					ID:       validUnitID,
					HP:       1,
					Team:     1,
					Position: origin,
					Speed:    3,
				}
				gs.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantUnit, OccupantID: int64(validUnitID)}
				return gs
			},
			wantErr: false,
			verifyResults: func(t *testing.T, gs *GameState, evt UnitMoved) {
				t.Helper()
				oldCell := gs.Grid[origin.Y][origin.X]
				if oldCell.OccupantType != OccupantNone {
					t.Errorf("expected old tile to be cleared, got type %v", oldCell.OccupantType)
				}

				newCell := gs.Grid[validTarget.Y][validTarget.X]
				if newCell.OccupantID != int64(validUnitID) {
					t.Errorf("expected unit %d at target, got %d", validUnitID, newCell.OccupantID)
				}

				newPos := gs.Units[validUnitID].Position
				if newPos != validTarget {
					t.Errorf("expect unit %d pos at %#v, got %#v", validUnitID, validTarget, newPos)
				}

				if !gs.Units[validUnitID].HasMoved {
					t.Errorf("expected unit HasMoved to be set")
				}

				if want := (UnitMoved{UnitID: validUnitID, From: origin, To: validTarget}); evt != want {
					t.Errorf("malformed UnitMovedEvent returned: %+v", evt)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := tt.setupState()

			evt, err := gs.MoveUnit(tt.unitID, tt.target)

			if (err != nil) != tt.wantErr {
				t.Fatalf("MoveUnit() error = %v, wantErr %v", err, tt.wantErr)
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

func TestMatch_ApplyTurnCommand(t *testing.T) {
	unitID := NewUnitID(1, 0)

	t.Run("apply MoveCommand", func(t *testing.T) {
		m := newActiveMatch(2, 2)
		m.WorkingState.Units[unitID] = &Unit{ID: unitID, Team: 1, HP: 1, Speed: 100, Position: Coordinate{0, 0}}
		m.WorkingState.Grid[0][0] = Tile{Type: TerrainPlain, OccupantType: OccupantUnit, OccupantID: int64(unitID)}

		cmd := NewMoveCommand(unitID, Coordinate{1, 0})

		gameEvents, err := m.ApplyTurnCommand(cmd)

		if err != nil {
			t.Errorf("ApplyCommand for MoveCommand failed unexpectedly: %v", err)
		}
		if len(gameEvents) == 0 {
			t.Error("Expected gameEvents returned, got none")
		}

	})

	t.Run("apply PlaceBombCommand", func(t *testing.T) {
		m := newActiveMatch(2, 2)
		m.WorkingState.Units[unitID] = &Unit{ID: unitID, Team: 1, HP: 1, BombMaxRange: 100, MaxBombCount: 100, Position: Coordinate{0, 0}}
		m.WorkingState.Grid[0][0] = Tile{Type: TerrainPlain, OccupantType: OccupantUnit, OccupantID: int64(unitID)}

		cmd := NewPlaceBombCommand(unitID, Coordinate{1, 0})

		gameEvents, err := m.ApplyTurnCommand(cmd)

		if err != nil {
			t.Errorf("ApplyCommand for PlaceBombCommand failed unexpectedly: %v", err)
		}
		if len(gameEvents) == 0 {
			t.Error("Expected gameEvents returned, got none")
		}

	})

	t.Run("apply unsupported command type", func(t *testing.T) {
		m := newActiveMatch(2, 2)
		m.WorkingState.Units[unitID] = &Unit{ID: unitID, Team: 1, HP: 1, BombMaxRange: 100, MaxBombCount: 100, Position: Coordinate{0, 0}}
		m.WorkingState.Grid[0][0] = Tile{Type: TerrainPlain, OccupantType: OccupantUnit, OccupantID: int64(unitID)}

		gameEvents, err := m.ApplyTurnCommand(nil)

		if !errors.Is(err, ErrUnsupportedCommand) {
			t.Errorf("Expect ApplyCommand for invalid type should fail, but got %#v", err)
		}
		if len(gameEvents) > 0 {
			t.Errorf("Expected empty gameEvents returned, got %#v", gameEvents)
		}

	})
}

func TestMatch_BeginTurn_NotTriggeringSuddenDeath(t *testing.T) {
	king := mustArchetype(t, "King")

	fighter := mustArchetype(t, "Fighter")

	u1 := NewUnitID(1, 0)
	u2 := NewUnitID(2, 0)
	u3 := NewUnitID(1, 1)
	u4 := NewUnitID(2, 1)

	t.Run("match has reached a conclusion", func(t *testing.T) {
		m := newTestMatch(3, 3)
		u1 := NewUnitID(1, 0)
		u2 := NewUnitID(2, 0)
		m.WorkingState.Units[u1] = &Unit{ID: u1, Team: 1, HP: 0}
		m.WorkingState.Units[u2] = &Unit{ID: u2, Team: 2, HP: 1}

		m.beginTurn()
		gameEvents := m.WorkingState.TurnStartEvents

		if got, want := len(m.WorkingState.Bombs), 0; got != want {
			t.Errorf("Victory Evaluation failure: Bomb count = %d, want: %d", got, want)
		}
		if len(gameEvents) > 0 {
			t.Errorf("expected empty gameEvents return, got %#v", gameEvents)
		}
		if len(m.PlaybackLog) > 0 {
			t.Errorf("expected empty PlaybackLog, got %#v", m.PlaybackLog)
		}
	})

	t.Run("turn limit not exceed yet", func(t *testing.T) {
		m := newTestMatch(3, 3)
		m.TrueState.Turn = 100
		m.WorkingState.Turn = 100
		m.WorkingState.ActiveTeam = 2
		m.WorkingState.Units[u1] = &Unit{ID: u1, Team: 1, HP: 1, Position: Coordinate{0, 0}, Type: king, Role: RoleKing}
		m.WorkingState.Units[u2] = &Unit{ID: u2, Team: 2, HP: 1, Position: Coordinate{0, 1}, Type: king, Role: RoleKing}
		m.WorkingState.Units[u3] = &Unit{ID: u3, Team: 1, HP: 1, Position: Coordinate{0, 7}, Type: fighter}
		m.WorkingState.Units[u4] = &Unit{ID: u4, Team: 2, HP: 1, Position: Coordinate{0, 8}, Type: fighter}

		m.beginTurn()
		gameEvents := m.WorkingState.TurnStartEvents

		if got, want := len(m.WorkingState.Bombs), 0; got != want {
			t.Errorf("Turn limit check failure: Bomb count = %d, want: %d", got, want)
		}
		if got, want := m.WorkingState.InSuddenDeath, false; got != want {
			t.Errorf("Sudden Death check failure: InSuddenDeath = %v, want: %v", got, want)
		}
		if len(gameEvents) > 0 {
			t.Errorf("expected empty gameEvents return, got %#v", gameEvents)
		}
		if len(m.PlaybackLog) > 0 {
			t.Errorf("expected empty PlaybackLog, got %#v", m.PlaybackLog)
		}
	})
}

func TestMatch_BeginTurn_SuddenDeath(t *testing.T) {
	king := mustArchetype(t, "King")

	fighter := mustArchetype(t, "Fighter")

	u1 := NewUnitID(1, 0)
	u2 := NewUnitID(2, 0)
	u3 := NewUnitID(1, 1)
	u4 := NewUnitID(2, 1)

	t.Run("stage has many available tiles", func(t *testing.T) {
		m := newTestMatch(4, 3)
		m.TrueState.Turn = 101

		m.WorkingState.Units[u1] = &Unit{ID: u1, Team: 1, HP: 1, Position: Coordinate{0, 0}, Type: king, Role: RoleKing}
		m.WorkingState.Units[u2] = &Unit{ID: u2, Team: 2, HP: 1, Position: Coordinate{0, 1}, Type: king, Role: RoleKing}
		m.WorkingState.Units[u3] = &Unit{ID: u3, Team: 1, HP: 1, Position: Coordinate{3, 0}, Type: fighter}
		m.WorkingState.Units[u4] = &Unit{ID: u4, Team: 2, HP: 1, Position: Coordinate{3, 1}, Type: fighter}
		m.WorkingState.Grid[0][0].OccupantType = OccupantUnit
		m.WorkingState.Grid[0][0].OccupantID = int64(u1)
		m.WorkingState.Grid[1][0].OccupantType = OccupantUnit
		m.WorkingState.Grid[1][0].OccupantID = int64(u2)
		m.WorkingState.Grid[0][3].OccupantType = OccupantUnit
		m.WorkingState.Grid[0][3].OccupantID = int64(u4)
		m.WorkingState.Grid[1][3].OccupantType = OccupantUnit
		m.WorkingState.Grid[1][3].OccupantID = int64(u4)

		m.beginTurn()
		gameEvents := m.WorkingState.TurnStartEvents

		if got, want := len(m.WorkingState.Bombs), SuddenDeathBombs; got != want {
			t.Errorf("Sudden death bomb drop failure: Bomb count = %d, want: %d", got, want)
		}
		if got, want := m.WorkingState.InSuddenDeath, true; got != want {
			t.Errorf("Sudden Death check failure: InSuddenDeath = %v, want: %v", got, want)
		}
		if len(gameEvents) != 2 {
			t.Errorf("expected 2 gameEvents return, got %#v", gameEvents)
		}
		if len(m.PlaybackLog) > 0 {
			t.Errorf("expected empty PlaybackLog, got %#v", m.PlaybackLog)
		}
	})

	t.Run("stage has many 1 available tile", func(t *testing.T) {
		m := newTestMatch(1, 9)
		m.TrueState.Turn = 101
		bombID := NewBombID(100, 1, u1)
		m.WorkingState.Units[u1] = &Unit{ID: u1, Team: 1, HP: 1, Position: Coordinate{0, 0}, Type: king, Role: RoleKing}
		m.WorkingState.Units[u2] = &Unit{ID: u2, Team: 2, HP: 1, Position: Coordinate{0, 1}, Type: king, Role: RoleKing}
		m.WorkingState.Units[u3] = &Unit{ID: u3, Team: 1, HP: 1, Position: Coordinate{0, 7}, Type: fighter}
		m.WorkingState.Units[u4] = &Unit{ID: u4, Team: 2, HP: 1, Position: Coordinate{0, 8}, Type: fighter}
		m.WorkingState.Bombs[bombID] = &Bomb{ID: bombID, OwnerID: u2, Position: Coordinate{0, 3}}
		m.WorkingState.Grid[0][0].OccupantType = OccupantUnit
		m.WorkingState.Grid[0][0].OccupantID = int64(u1)
		m.WorkingState.Grid[1][0].OccupantType = OccupantUnit
		m.WorkingState.Grid[1][0].OccupantID = int64(u2)
		m.WorkingState.Grid[2][0].Type = TerrainBlock
		m.WorkingState.Grid[3][0].OccupantType = OccupantBomb
		m.WorkingState.Grid[3][0].OccupantID = int64(bombID)
		m.WorkingState.Grid[4][0].OccupantType = OccupantSoftBlock
		m.WorkingState.Grid[5][0].OccupantType = OccupantItem
		m.WorkingState.Grid[7][0].OccupantType = OccupantUnit
		m.WorkingState.Grid[7][0].OccupantID = int64(u4)
		m.WorkingState.Grid[8][0].OccupantType = OccupantUnit
		m.WorkingState.Grid[8][0].OccupantID = int64(u4)

		m.beginTurn()
		gameEvents := m.WorkingState.TurnStartEvents

		if got, want := len(m.WorkingState.Bombs), 2; got != want {
			t.Errorf("Sudden death bomb drop failure: Bomb count = %d, want: %d", got, want)
		}
		if got, want := m.WorkingState.InSuddenDeath, true; got != want {
			t.Errorf("Sudden Death check failure: InSuddenDeath = %v, want: %v", got, want)
		}
		if len(gameEvents) != 1 {
			t.Errorf("expected 1 gameEvents return, got %#v", gameEvents)
		}
		if len(m.PlaybackLog) > 0 {
			t.Errorf("expected empty PlaybackLog, got %#v", m.PlaybackLog)
		}
	})

	t.Run("stage has no available tile", func(t *testing.T) {
		m := newTestMatch(1, 5)
		m.TrueState.Turn = 101
		m.WorkingState.Units[u1] = &Unit{ID: u1, Team: 1, HP: 1, Position: Coordinate{0, 0}, Type: king, Role: RoleKing}
		m.WorkingState.Units[u2] = &Unit{ID: u2, Team: 2, HP: 1, Position: Coordinate{0, 1}, Type: king, Role: RoleKing}
		m.WorkingState.Units[u3] = &Unit{ID: u3, Team: 1, HP: 1, Position: Coordinate{0, 3}, Type: fighter}
		m.WorkingState.Units[u4] = &Unit{ID: u4, Team: 2, HP: 1, Position: Coordinate{0, 4}, Type: fighter}
		m.WorkingState.Grid[0][0].OccupantType = OccupantUnit
		m.WorkingState.Grid[0][0].OccupantID = int64(u1)
		m.WorkingState.Grid[1][0].OccupantType = OccupantUnit
		m.WorkingState.Grid[1][0].OccupantID = int64(u2)
		m.WorkingState.Grid[2][0].Type = TerrainBlock
		m.WorkingState.Grid[3][0].OccupantType = OccupantUnit
		m.WorkingState.Grid[3][0].OccupantID = int64(u4)
		m.WorkingState.Grid[4][0].OccupantType = OccupantUnit
		m.WorkingState.Grid[4][0].OccupantID = int64(u4)

		m.beginTurn()
		gameEvents := m.WorkingState.TurnStartEvents

		if got, want := len(m.WorkingState.Bombs), 0; got != want {
			t.Errorf("Sudden death bomb drop failure: Bomb count = %d, want: %d", got, want)
		}
		if got, want := m.WorkingState.InSuddenDeath, true; got != want {
			t.Errorf("Sudden Death check failure: InSuddenDeath = %v, want: %v", got, want)
		}
		if len(gameEvents) > 0 {
			t.Errorf("expected empty gameEvents return, got %#v", gameEvents)
		}
		if len(m.PlaybackLog) > 0 {
			t.Errorf("expected empty PlaybackLog, got %#v", m.PlaybackLog)
		}
	})
}

func TestMatch_ResolveTurn_SplitsPlanningFromResolution(t *testing.T) {
	king := mustArchetype(t, "King")

	m := newActiveMatch(9, 9)
	unitID := NewUnitID(1, 0)
	m.WorkingState.Units[unitID] = &Unit{
		ID: unitID, Team: 1, HP: 1, Type: king, Role: RoleKing,
		Position:     Coordinate{0, 0},
		Speed:        king.BaseSpeed,
		BombMaxRange: king.BombMaxRange,
		BombMinRange: king.BombMinRange,
		BombPower:    king.BombPower,
		MaxBombCount: king.MaxBombCount,
	}
	m.WorkingState.Grid[0][0] = Tile{Type: TerrainPlain, OccupantType: OccupantUnit, OccupantID: int64(unitID)}

	if _, err := m.CommandPlaceBomb(unitID, Coordinate{1, 0}); err != nil {
		t.Fatalf("CommandPlaceBomb() error = %v", err)
	}
	if _, err := m.commandMoveUnit(unitID, Coordinate{0, 1}); err != nil {
		t.Fatalf("CommandMoveUnit() error = %v", err)
	}
	for _, bomb := range m.WorkingState.Bombs {
		bomb.Countdown = 1
	}

	planGameEvents, resolveTurnGameEvents := m.ResolveTurn()

	if len(planGameEvents) != 2 || !eventIs[BombPlaced](planGameEvents[0]) || !eventIs[UnitMoved](planGameEvents[1]) {
		t.Fatalf("Expected planGameEvents [BombPlaced UnitMoved], got %#v", planGameEvents)
	}

	for _, evt := range resolveTurnGameEvents {
		if eventIs[BombPlaced](evt) || eventIs[UnitMoved](evt) {
			t.Errorf("Expected no planning event in resolveTurnGameEvents, got %#v", evt)
		}
	}
	if len(resolveTurnGameEvents) == 0 {
		t.Error("Expected the detonation to produce resolveTurnGameEvents, got none")
	}

	if got := len(m.PlaybackLog); got != 0 {
		t.Errorf("Expected PlaybackLog flushed, got %d entries", got)
	}
}

func TestMatch_ResolveTurn_TimelineSystemTransitions(t *testing.T) {
	king := mustArchetype(t, "King")

	fighter := mustArchetype(t, "Fighter")

	u1 := NewUnitID(1, 0)
	u2 := NewUnitID(2, 0)
	u3 := NewUnitID(1, 1)
	u4 := NewUnitID(2, 1)

	t.Run("Empty turn resolution executes smoothly with zero structural mutations", func(t *testing.T) {
		m := newTestMatch(16, 16)

		m.WorkingState.Units[u1] = &Unit{ID: u1, Team: 1, HP: 1, Type: king, Role: RoleKing, HasMoved: true}
		m.WorkingState.Units[u2] = &Unit{ID: u2, Team: 2, HP: 1, Type: king, Role: RoleKing, HasUsedSkill: true}
		m.WorkingState.Units[u3] = &Unit{ID: u3, Team: 1, HP: 1, Type: fighter, HasMoved: true, HasUsedSkill: true}
		m.WorkingState.Units[u4] = &Unit{ID: u4, Team: 2, HP: 1, Type: fighter}

		_, gameEvents := m.ResolveTurn()

		if len(gameEvents) != 0 {
			t.Errorf("Expected clean slice array from empty resolution pass, got %d items", len(gameEvents))
		}

		if m.WorkingState.Turn != 2 {
			t.Errorf("Expected Turn will increment to 2, got %d", m.WorkingState.Turn)
		}

		for unitID, unit := range m.WorkingState.Units {
			if unit.HasMoved || unit.HasUsedSkill {
				t.Errorf("Expected Unit %#X HasMoved and HasUsed are reset, got HasMoved %v, HasUsedSkill %v", unitID, unit.HasMoved, unit.HasUsedSkill)
			}
		}

		if len(m.PlaybackLog) != 0 {
			t.Errorf("Expectd clean slice array from PlaybackLog, got %d items", len(m.PlaybackLog))
		}
	})

	t.Run("Successful turn resolution advances true match clock and deep copies sandbox workspace", func(t *testing.T) {
		m := newActiveMatch(16, 16)

		m.WorkingState.Units[u1] = &Unit{ID: u1, Team: 1, HP: 2, Type: king, Role: RoleKing}
		m.WorkingState.Units[u2] = &Unit{ID: u2, Team: 2, HP: 2, Type: king, Role: RoleKing}
		m.WorkingState.Units[u3] = &Unit{ID: u3, Team: 1, HP: 2, Type: fighter}
		m.WorkingState.Units[u4] = &Unit{ID: u4, Team: 2, HP: 2, Type: fighter}

		m.WorkingState.Grid[0][0] = Tile{Type: TerrainPlain, OccupantType: OccupantUnit, OccupantID: int64(u1)}
		m.WorkingState.Grid[1][0] = Tile{Type: TerrainPlain, OccupantType: OccupantUnit, OccupantID: int64(u2)}

		m.ResolveTurn()

		if m.TrueState.Turn != 2 {
			t.Errorf("TrueState clock failed to advance! Got turn %d, want 2", m.TrueState.Turn)
		}

		promotedUnit, exists := m.TrueState.Units[u1]
		if !exists {
			t.Fatal("Master Promotion Error: Sandbox mutations missing from TrueState history after commit!")
		}
		if promotedUnit.HP != 2 {
			t.Errorf("Data Corruption: Expected promoted unit HP = 2, found %d", promotedUnit.HP)
		}

		m.ResetTurn()
		if m.WorkingState.Turn != 2 {
			t.Errorf("Sandbox rollback broke timeline integrity! WorkingState turn clock reset to %d, want 2", m.WorkingState.Turn)
		}
	})
}

func TestMatch_BeginTurn(t *testing.T) {
	std1 := teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{true, true, false, true}}
	std2 := teamFormation{King: new(true), OrdinaryUnitsAlive: []bool{true, true, false, false}}

	initGame := func(maxTurns int) func(t *testing.T) *Match {
		return func(t *testing.T) *Match {
			t.Helper()
			m, err := InitGame(GameCfg{
				StagePreset: "Plain",
				MaxTurns:    maxTurns,
				P1Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Fighter", Role: RoleNormal}},
				P2Slots:     []TeamSlot{{Archetype: "King", Role: RoleKing}, {Archetype: "Bandit", Role: RoleNormal}},
			})
			if err != nil {
				t.Fatalf("InitGame() error = %v", err)
			}

			return m
		}
	}
	teamsAt := func(maxTurns, turn int) func(t *testing.T) *Match {
		return func(t *testing.T) *Match {
			t.Helper()
			m := newTestMatchWithTeams(t, std1, std2)
			m.GameCfg.MaxTurns = maxTurns
			m.TrueState.Turn = turn
			m.WorkingState.Turn = turn

			return m
		}
	}
	resolve := func(m *Match) []GameEvent {
		_, resolveEvents := m.ResolveTurn()

		return resolveEvents
	}

	table := []struct {
		name          string
		build         func(t *testing.T) *Match
		act           func(m *Match) []GameEvent
		wantBombs     int
		wantSuddenDth bool
		wantEnded     bool
	}{
		{
			name:          "InitGame with MaxTurns 0 drops hazards",
			build:         initGame(0),
			wantBombs:     SuddenDeathBombs,
			wantSuddenDth: true,
		},
		{
			name:  "InitGame with MaxTurns 5 drops none",
			build: initGame(5),
		},
		{
			name:          "ResolveTurn from Turn = MaxTurns has the next turn's hazards placed on return",
			build:         teamsAt(3, 3),
			act:           resolve,
			wantBombs:     SuddenDeathBombs,
			wantSuddenDth: true,
		},
		{
			name:  "ResolveTurn from Turn = MaxTurns-1 drops none",
			build: teamsAt(3, 2),
			act:   resolve,
		},
		{
			name:  "match-ending ResolveTurn drops none",
			build: teamsAt(3, 4),
			act: func(m *Match) []GameEvent {
				m.WorkingState.Units[NewUnitID(2, 0)].HP = 0

				return resolve(m)
			},
			wantEnded: true,
		},
		{
			name:  "Surrender then ResolveTurn drops none",
			build: teamsAt(3, 4),
			act: func(m *Match) []GameEvent {
				m.Surrender(2)

				return resolve(m)
			},
			wantEnded: true,
		},
	}

	checkState := func(t *testing.T, label string, gs *GameState, wantBombs int) {
		t.Helper()
		if got := len(gs.Bombs); got != wantBombs {
			t.Errorf("%s bombs = %d, want %d", label, got, wantBombs)
		}
		if got := len(gs.TurnStartEvents); got != wantBombs {
			t.Fatalf("%s TurnStartEvents = %d, want %d", label, got, wantBombs)
		}
		for _, evt := range gs.TurnStartEvents {
			placed, isPlaced := evt.(BombPlaced)
			if !isPlaced {
				t.Errorf("%s event = %T, want BombPlaced", label, evt)
				continue
			}
			bomb, ok := gs.Bombs[placed.BombID]
			if !ok {
				t.Errorf("%s event bomb %v is not in the state", label, placed.BombID)
				continue
			}
			if bomb.OwnerID != SystemUnitID {
				t.Errorf("%s bomb owner = %v, want %v", label, bomb.OwnerID, SystemUnitID)
			}
		}
	}

	for _, tt := range table {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.build(t)
			if !tt.wantEnded {
				if got := m.evaluateVictoryConditions(); got != matchInProgress {
					t.Fatalf("evaluateVictoryConditions() = %d, want MatchInProgress", got)
				}
			}
			if tt.act != nil {
				resolveEvents := tt.act(m)
				for _, evt := range resolveEvents {
					if eventIs[BombPlaced](evt) {
						t.Errorf("resolve events hold a bombPlaced: %#v", evt)
					}
				}
			}
			if tt.wantEnded && m.WinnerTeamID == 0 {
				t.Fatalf("WinnerTeamID = 0, want the match ended")
			}
			if !tt.wantEnded {
				if got := m.WorkingState.InSuddenDeath; got != tt.wantSuddenDth {
					t.Errorf("InSuddenDeath = %v, want %v", got, tt.wantSuddenDth)
				}
			}
			if len(m.PlaybackLog) != 0 {
				t.Errorf("PlaybackLog = %#v, want empty", m.PlaybackLog)
			}

			checkState(t, "WorkingState", m.WorkingState, tt.wantBombs)
			checkState(t, "TrueState", m.TrueState, tt.wantBombs)

			m.ResetTurn()

			checkState(t, "WorkingState after ResetTurn", m.WorkingState, tt.wantBombs)
		})
	}
}

func TestMatch_Surrender_ReturnedEventsDoNotAliasMatch(t *testing.T) {
	m := newTestMatch(3, 3)

	gameEvents := m.Surrender(1)
	gameEvents[0] = nil

	if m.WinnerTeamID != 2 {
		t.Errorf("WinnerTeamID = %d, want 2", m.WinnerTeamID)
	}
}

func scribbleAffected(evt GameEvent) {
	if exploded, ok := evt.(BombExploded); ok {
		exploded.AffectedPositions[0].X = 99
	}
}

func scribbleMatch(m *Match) {
	m.WinnerTeamID = 1
	m.GameCfg.P1Slots[0].Archetype = "scribbled"
	scribbleAffected(m.PlaybackLog[0])
	for _, gs := range []*GameState{m.TrueState, m.WorkingState} {
		gs.Turn += 100
		gs.Grid[0][0].OccupantID += 77
		for _, unit := range gs.Units {
			unit.HasMoved = !unit.HasMoved
		}
		gs.Bombs[900].Range = 99
		gs.Bombs[901] = &Bomb{ID: 901}
		gs.SoftBlocks[900].Position.X = 99
		gs.SoftBlocks[901] = &SoftBlock{ID: 901}
		gs.TurnCommands[0] = NewMoveCommand(1, Coordinate{X: 99, Y: 2})
		scribbleAffected(gs.TurnStartEvents[0])
	}
}

func TestMatch_Clone_SharesNoMemory(t *testing.T) {
	m := newCloneFixture(t)
	if want := newCloneFixture(t); !reflect.DeepEqual(want, m) {
		t.Fatalf("fixture is not deterministic: %+v vs %+v", want, m)
	}
	if got := m.Clone(); !reflect.DeepEqual(got, m) {
		t.Fatalf("Clone() = %+v, want an equal match %+v", got, m)
	}

	t.Run("editing the clone leaves the original", func(t *testing.T) {
		got := m.Clone()
		scribbleMatch(got)
		if want := newCloneFixture(t); !reflect.DeepEqual(m, want) {
			t.Errorf("match after editing its clone = %+v, want unchanged %+v", m, want)
		}
	})

	t.Run("editing the original leaves the clone", func(t *testing.T) {
		got := m.Clone()
		scribbleMatch(m)
		if want := newCloneFixture(t); !reflect.DeepEqual(got, want) {
			t.Errorf("clone after editing the original = %+v, want unchanged %+v", got, want)
		}
	})
	t.Run("nil and empty slices stay as they are", func(t *testing.T) {
		m := newCloneFixture(t)
		m.PlaybackLog = []GameEvent{}
		m.TrueState.TurnCommands = nil
		m.TrueState.TurnStartEvents = []GameEvent{}
		m.WorkingState.TurnCommands = []TurnCommand{}
		m.WorkingState.TurnStartEvents = nil
		if got := m.Clone(); !reflect.DeepEqual(got, m) {
			t.Errorf("Clone() = %+v, want %+v", got, m)
		}
	})
}
