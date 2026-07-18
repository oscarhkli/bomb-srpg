package engine

import (
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
		PlaybackLog: []GameEvent{NewUnitMovedEvent(16, Coordinate{X: 1, Y: 2}, Coordinate{X: 1, Y: 3})},
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

	event := NewUnitMovedEvent(unitID, Coordinate{1, 0}, Coordinate{10, 100})

	m.SubmitAction(event)

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

	event := NewUnitMovedEvent(unitID, Coordinate{1, 0}, Coordinate{10, 100})

	m.SubmitAction(event)

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
			m.PlaybackLog = append(m.PlaybackLog, NewUnitDiedEvent(16))

			events := m.Surrender(tt.teamID)

			if len(events) != 1 {
				t.Errorf("Expected 1 event for surrender, got %v", len(events))
			}

			var foundEndedEvent bool
			for _, event := range events {
				if event.Type == GameEvtMatchEnded {
					foundEndedEvent = true
					if event.WinnerTeamID != tt.winnerTeamID {
						t.Errorf("Malformed MatchEndedEvent! Got winner %d", event.WinnerTeamID)
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
	// Define the schema for our table rows
	type testCase struct {
		name          string
		unitID        UnitID
		target        Coordinate
		setupState    func() *Match // Context setup helper
		wantErr       bool
		errContains   string
		verifyResults func(t *testing.T, m *Match, evts []GameEvent) // Post-execution state check
	}

	validUnitID := NewUnitID(1, 0)
	deadUnitID := NewUnitID(1, 2)
	origin := Coordinate{1, 1}
	validTarget := Coordinate{1, 2}
	outOfRangeTarget := Coordinate{5, 5}

	tests := []testCase{
		{
			name:   "Failure: Unit does not exist",
			unitID: NewUnitID(3, 3), // Missing ID
			target: validTarget,
			setupState: func() *Match {
				return newTestMatch(2, 2) // Empty map, no units
			},
			wantErr:     true,
			errContains: "unit 0x33 does not exist",
		},
		{
			name:   "Failure: Unit is dead",
			unitID: deadUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(2, 2)
				m.WorkingState.Units[deadUnitID] = &Unit{ID: deadUnitID, HP: 0, Team: 1, Position: origin}
				return m
			},
			wantErr:     true,
			errContains: "unit 0x12 is dead",
		},
		{
			name:   "Failure: Wrong team turn",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(2, 2)
				m.WorkingState.Turn = 1 // Team 1's turn
				m.WorkingState.ActiveTeam = 1
				m.WorkingState.Units[validUnitID] = &Unit{
					ID:       validUnitID,
					HP:       1,
					Team:     2, // Team 2 Unit
					Position: origin,
				}
				return m
			},
			wantErr:     true,
			errContains: "unit 0x10 not active team",
		},
		// { This test is not available at the moment - not until the Skills implementation in Phase 4
		// 	name:   "Failure: Unit passes through but cannot land on HardRock",
		// },
		{
			name:   "Failure: Data corruption - unit out of bounds",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(3, 3)
				m.WorkingState.Turn = 1
				m.WorkingState.ActiveTeam = 1

				// Intentionally corrupt the position data
				m.WorkingState.Units[validUnitID] = &Unit{
					ID:       validUnitID,
					HP:       1,
					Team:     1,
					Position: Coordinate{-5, -5}, // Out of stage bounds
					Speed:    3,
				}
				// Do not add it to the Grid matrix since the coordinate is invalid
				return m
			},
			wantErr:     true,
			errContains: "unit 0x10 out of bounds",
		},
		{
			name:   "Failure: Grid data desync",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(2, 2)
				m.WorkingState.Turn = 1
				m.WorkingState.ActiveTeam = 1
				m.WorkingState.Units[validUnitID] = &Unit{
					ID:       validUnitID,
					HP:       1,
					Team:     1,
					Position: origin,
				}
				m.WorkingState.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantNone}
				// Intentionally do NOT put the unit into m.WorkingState.Grid matrix to trigger desync
				return m
			},
			wantErr:     true,
			errContains: "unit 0x10 desynced at",
		},
		{
			name:   "Failure: Target out of moving range",
			unitID: validUnitID,
			target: outOfRangeTarget,
			setupState: func() *Match {
				m := newTestMatch(10, 10)
				m.WorkingState.Turn = 1
				m.WorkingState.ActiveTeam = 1
				m.WorkingState.Units[validUnitID] = &Unit{
					ID:       validUnitID,
					HP:       1,
					Team:     1,
					Position: origin,
					Speed:    3,
				}
				m.WorkingState.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantUnit, OccupantID: int64(validUnitID)}
				// Mock rule where target won't be found in reachable tiles
				return m
			},
			wantErr:     true,
			errContains: "target out of move range",
		},
		{
			name:   "Failure: Unit has moved in the same turn",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(3, 3)
				m.WorkingState.Turn = 1
				m.WorkingState.ActiveTeam = 1
				m.WorkingState.Units[validUnitID] = &Unit{
					ID:       validUnitID,
					HP:       1,
					Team:     1,
					Position: origin,
					Speed:    3,
					HasMoved: true,
				}
				m.WorkingState.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantUnit, OccupantID: int64(validUnitID)}
				return m
			},
			wantErr:     true,
			errContains: "unit 0x10 already moved this turn",
		},
		{
			name:   "Success: Unit moves successfully",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(3, 3)
				m.WorkingState.Turn = 1
				m.WorkingState.ActiveTeam = 1
				m.WorkingState.Units[validUnitID] = &Unit{
					ID:       validUnitID,
					HP:       1,
					Team:     1,
					Position: origin,
					Speed:    3,
				}
				m.WorkingState.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantUnit, OccupantID: int64(validUnitID)}
				// FindReachableTiles/IsLandingLegal pass this target
				return m
			},
			wantErr: false,
			verifyResults: func(t *testing.T, m *Match, evts []GameEvent) {
				oldCell := m.WorkingState.Grid[origin.Y][origin.X]
				if oldCell.OccupantType != OccupantNone {
					t.Errorf("expected old tile to be cleared, got type %v", oldCell.OccupantType)
				}

				newCell := m.WorkingState.Grid[validTarget.Y][validTarget.X]
				if newCell.OccupantID != int64(validUnitID) {
					t.Errorf("expected unit %d at target, got %d", validUnitID, newCell.OccupantID)
				}

				newPos := m.WorkingState.Units[validUnitID].Position
				if newPos != validTarget {
					t.Errorf("expect unit %d pos at %#v, got %#v", validUnitID, validTarget, newPos)
				}

				if len(m.PlaybackLog) != 1 {
					t.Errorf("expected 1 action submitted, got %d", len(m.PlaybackLog))
				}
				event := m.PlaybackLog[0]
				if event.Type != GameEvtUnitMoved || event.UnitID != validUnitID || event.From == nil || event.To == nil || *event.From != origin || *event.To != validTarget {
					t.Errorf("malformed UnitMovedEvent logged: %+v", event)
				}
				if len(evts) != 1 {
					t.Errorf("expected 1 GameEvent returned, got %d", len(evts))
				}
				resEvt := evts[0]
				if resEvt.Type != GameEvtUnitMoved || resEvt.UnitID != validUnitID || resEvt.From == nil || resEvt.To == nil || *resEvt.From != origin || *resEvt.To != validTarget {
					t.Errorf("malformed UnitMovedEvent returned: %+v", resEvt)
				}
			},
		},
	}

	// Run loop
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := tt.setupState()

			gameEvents, err := match.CommandMoveUnit(tt.unitID, tt.target)

			// Error checking
			if (err != nil) != tt.wantErr {
				t.Fatalf("CommandMoveUnit() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
			}

			// State validation checking
			if tt.verifyResults != nil {
				tt.verifyResults(t, match, gameEvents)
			} else {
				if len(gameEvents) > 0 {
					t.Errorf("expected empty gameEvents return, got %#v", gameEvents)
				}
			}
		})
	}
}

func TestMatch_CommandPlaceBomb(t *testing.T) {
	// Define the schema for our table rows matching the style of TestCommandMoveUnit
	type testCase struct {
		name          string
		unitID        UnitID
		target        Coordinate
		setupState    func() *Match // Context setup helper
		wantErr       bool
		errContains   string
		verifyResults func(t *testing.T, m *Match, evts []GameEvent) // Post-execution state check
	}

	// Constants for easy setup
	validUnitID := NewUnitID(1, 0)
	deadUnitID := NewUnitID(1, 2)
	origin := Coordinate{1, 1}
	validTarget := Coordinate{1, 2}
	outOfRangeTarget := Coordinate{5, 5}

	tests := []testCase{
		{
			name:   "Failure: Unit does not exist",
			unitID: NewUnitID(3, 3), // Missing ID
			target: validTarget,
			setupState: func() *Match {
				return newTestMatch(2, 2) // Empty map, no units
			},
			wantErr:     true,
			errContains: "unit 0x33 does not exist",
		},
		{
			name:   "Failure: Unit is dead",
			unitID: deadUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(2, 2)
				m.WorkingState.Units[deadUnitID] = &Unit{ID: deadUnitID, HP: 0, Team: 1, Position: origin}
				return m
			},
			wantErr:     true,
			errContains: "unit 0x12 is dead",
		},
		{
			name:   "Failure: Wrong team turn",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(2, 2)
				m.WorkingState.Turn = 1 // Team 1's turn
				m.WorkingState.ActiveTeam = 1
				m.WorkingState.Units[validUnitID] = &Unit{
					ID:       validUnitID,
					HP:       1,
					Team:     2, // Team 2 Unit
					Position: origin,
				}
				return m
			},
			wantErr:     true,
			errContains: "unit 0x10 not active team",
		},
		{
			name:   "Failure: Data corruption - unit out of bounds",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(3, 3)
				m.WorkingState.Turn = 1
				m.WorkingState.ActiveTeam = 1
				m.WorkingState.Units[validUnitID] = &Unit{
					ID:       validUnitID,
					HP:       1,
					Team:     1,
					Position: Coordinate{-5, -5}, // Out of stage bounds
				}
				return m
			},
			wantErr:     true,
			errContains: "unit 0x10 out of bounds",
		},
		{
			name:   "Failure: Grid data desync",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(2, 2)
				m.WorkingState.Turn = 1
				m.WorkingState.ActiveTeam = 1
				m.WorkingState.Units[validUnitID] = &Unit{
					ID:       validUnitID,
					HP:       1,
					Team:     1,
					Position: origin,
				}
				m.WorkingState.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantNone}
				return m
			},
			wantErr:     true,
			errContains: "unit 0x10 desynced at",
		},
		{
			name:   "Failure: Unit has used up all bombs",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(3, 3)
				m.WorkingState.Turn = 1
				m.WorkingState.ActiveTeam = 1
				m.WorkingState.Units[validUnitID] = &Unit{
					ID:           validUnitID,
					HP:           1,
					Team:         1,
					Position:     origin,
					MaxBombCount: 2,
					BombUsed:     2, // All bombs deployed
				}
				m.WorkingState.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantUnit, OccupantID: int64(validUnitID)}
				return m
			},
			wantErr:     true,
			errContains: "unit 0x10 out of bombs",
		},
		{
			name:   "Failure: Target out of placement range",
			unitID: validUnitID,
			target: outOfRangeTarget,
			setupState: func() *Match {
				m := newTestMatch(10, 10)
				m.WorkingState.Turn = 1
				m.WorkingState.ActiveTeam = 1
				m.WorkingState.Units[validUnitID] = &Unit{
					ID:           validUnitID,
					HP:           1,
					Team:         1,
					Position:     origin,
					BombMaxRange: 1,
					MaxBombCount: 1,
					BombUsed:     0,
				}
				m.WorkingState.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantUnit, OccupantID: int64(validUnitID)}
				return m
			},
			wantErr:     true,
			errContains: "target out of bomb range",
		},
		{
			name:   "Failure: Illegal target",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(3, 3)
				m.WorkingState.Turn = 3
				m.WorkingState.ActiveTeam = 1
				m.WorkingState.TurnBombCounter = 0
				m.WorkingState.Bombs = make(map[BombID]*Bomb)
				m.WorkingState.Units[validUnitID] = &Unit{
					ID:           validUnitID,
					HP:           1,
					Team:         1,
					Position:     origin,
					BombPower:    3,
					BombMaxRange: 3,
					MaxBombCount: 2,
					BombUsed:     0,
				}
				m.WorkingState.Grid[validTarget.Y][validTarget.X].Type = TerrainBlock // make the target tile illegal to place a bomb
				m.WorkingState.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantUnit, OccupantID: int64(validUnitID)}
				return m
			},
			wantErr:     true,
			errContains: "can only place on plain terrain",
		},
		{
			name:   "Failure: Unit has used skill in the turn",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(3, 3)
				m.WorkingState.Turn = 3
				m.WorkingState.ActiveTeam = 1
				m.WorkingState.TurnBombCounter = 0
				m.WorkingState.Bombs = make(map[BombID]*Bomb)
				m.WorkingState.Units[validUnitID] = &Unit{
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
				m.WorkingState.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantUnit, OccupantID: int64(validUnitID)}
				return m
			},
			wantErr:     true,
			errContains: "unit 0x10 already used skill this turn",
		},
		{
			name:   "Success: Bomb placed successfully",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(3, 3)
				m.WorkingState.Turn = 3
				m.WorkingState.ActiveTeam = 1
				m.WorkingState.TurnBombCounter = 0
				m.WorkingState.Bombs = make(map[BombID]*Bomb)
				m.WorkingState.Units[validUnitID] = &Unit{
					ID:           validUnitID,
					HP:           1,
					Team:         1,
					Position:     origin,
					BombPower:    3,
					BombMaxRange: 3,
					MaxBombCount: 2,
					BombUsed:     0,
				}
				m.WorkingState.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantUnit, OccupantID: int64(validUnitID)}
				return m
			},
			wantErr: false,
			verifyResults: func(t *testing.T, m *Match, evts []GameEvent) {
				if m.WorkingState.TurnBombCounter != 1 {
					t.Errorf("expected TurnBombCounter to be 1, got %d", m.WorkingState.TurnBombCounter)
				}

				expectedBombID := NewBombID(3, 1, validUnitID)
				bomb, exists := m.WorkingState.Bombs[expectedBombID]
				if !exists {
					t.Fatalf("expected bomb tracking map entry under ID %#X missing", expectedBombID)
				}
				if bomb.OwnerID != validUnitID || bomb.Position != validTarget || bomb.Range != 3 || bomb.Countdown != 5 {
					t.Errorf("registered bomb structural parameters mismatched: %+v", bomb)
				}

				unit := m.WorkingState.Units[validUnitID]
				if got, want := unit.BombUsed, 1; got != want {
					t.Errorf("expected unit bombUsed reduced to %v, got: %v", want, got)
				}

				targetCell := m.WorkingState.Grid[validTarget.Y][validTarget.X]
				if targetCell.OccupantType != OccupantBomb || targetCell.OccupantID != int64(expectedBombID) {
					t.Errorf("expected target grid tile to hold bomb entity, got type %v, id %d", targetCell.OccupantType, targetCell.OccupantID)
				}

				originCell := m.WorkingState.Grid[origin.Y][origin.X]
				if originCell.OccupantType != OccupantUnit || originCell.OccupantID != int64(validUnitID) {
					t.Errorf("expected origin unit tile to remain intact, got type %v, id %d", originCell.OccupantType, originCell.OccupantID)
				}

				if len(m.PlaybackLog) != 1 {
					t.Fatalf("expected 1 action submitted, got %d", len(m.PlaybackLog))
				}
				event := m.PlaybackLog[0]
				if event.Type != GameEvtBombPlaced || event.UnitID != validUnitID || event.BombID != expectedBombID || event.Position == nil || *event.Position != validTarget || event.Range != 3 || event.Countdown != 5 {
					t.Errorf("malformed BombPlacedEvent logged: %+v", event)
				}
				if len(evts) != 1 {
					t.Errorf("expected 1 GameEvent returned, got %d", len(evts))
				}
				resEvt := evts[0]
				if resEvt.Type != GameEvtBombPlaced || resEvt.UnitID != validUnitID || resEvt.BombID != expectedBombID || resEvt.Position == nil || *resEvt.Position != validTarget || resEvt.Range != 3 || resEvt.Countdown != 5 {
					t.Errorf("malformed BombPlacedEvent returned: %+v", resEvt)
				}
			},
		},
	}

	// Execution loop matching TestCommandMoveUnit
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := tt.setupState()

			gameEvents, err := match.CommandPlaceBomb(tt.unitID, tt.target)

			// Error validation checking
			if (err != nil) != tt.wantErr {
				t.Fatalf("CommandPlaceBomb() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
			}

			// State transformation post-checks
			if tt.verifyResults != nil {
				tt.verifyResults(t, match, gameEvents)
			} else {
				if len(gameEvents) > 0 {
					t.Errorf("expected empty gameEvents return, got %#v", gameEvents)
				}
			}
		})
	}
}

// newTestMatch generates a clean slate grid environment
func newTestMatch(width, height int) *Match {
	grid := make([][]Tile, height)
	for y, row := range grid {
		grid[y] = make([]Tile, width)
		for x := range row {
			grid[y][x] = Tile{Type: TerrainPlain, OccupantType: OccupantNone}
		}
	}

	m := &Match{
		GameCfg: GameCfg{MaxTurns: 100},
		WorkingState: &GameState{
			Turn:            1,
			TurnBombCounter: 0,
			Grid:            grid,
			Units:           make(map[UnitID]*Unit),
			Bombs:           make(map[BombID]*Bomb),
			SoftBlocks:      make(map[int]*SoftBlock),
		},
		PlaybackLog: []GameEvent{},
	}
	m.TrueState = m.WorkingState.DeepCopy()

	return m
}

func TestMatch_ApplyTurnCommand(t *testing.T) {
	uID := NewUnitID(1, 0)

	t.Run("apply MoveCommand", func(t *testing.T) {
		m := newTestMatch(2, 2)
		m.WorkingState.Turn = 1
		m.WorkingState.ActiveTeam = 1
		m.WorkingState.Units[uID] = &Unit{ID: uID, Team: 1, HP: 1, Speed: 100, Position: Coordinate{0, 0}}
		m.WorkingState.Grid[0][0] = Tile{Type: TerrainPlain, OccupantType: OccupantUnit, OccupantID: int64(uID)}

		cmd := NewMoveCommand(uID, Coordinate{1, 0})

		gameEvents, err := m.ApplyTurnCommand(cmd)

		if err != nil {
			t.Errorf("ApplyCommand for MoveCommand failed unexpectedly: %v", err)
		}
		if len(gameEvents) == 0 {
			t.Error("Expected gameEvents returned, got none")
		}

	})

	t.Run("apply PlaceBombCommand", func(t *testing.T) {
		m := newTestMatch(2, 2)
		m.WorkingState.Turn = 1
		m.WorkingState.ActiveTeam = 1
		m.WorkingState.Units[uID] = &Unit{ID: uID, Team: 1, HP: 1, BombMaxRange: 100, MaxBombCount: 100, Position: Coordinate{0, 0}}
		m.WorkingState.Grid[0][0] = Tile{Type: TerrainPlain, OccupantType: OccupantUnit, OccupantID: int64(uID)}

		cmd := NewPlaceBombCommand(uID, Coordinate{1, 0})

		gameEvents, err := m.ApplyTurnCommand(cmd)

		if err != nil {
			t.Errorf("ApplyCommand for PlaceBombCommand failed unexpectedly: %v", err)
		}
		if len(gameEvents) == 0 {
			t.Error("Expected gameEvents returned, got none")
		}

	})

	t.Run("apply unsupported command type", func(t *testing.T) {
		m := newTestMatch(2, 2)
		m.WorkingState.Turn = 1
		m.WorkingState.ActiveTeam = 1
		m.WorkingState.Units[uID] = &Unit{ID: uID, Team: 1, HP: 1, BombMaxRange: 100, MaxBombCount: 100, Position: Coordinate{0, 0}}
		m.WorkingState.Grid[0][0] = Tile{Type: TerrainPlain, OccupantType: OccupantUnit, OccupantID: int64(uID)}

		cmd := TurnCommand{Type: "invalid", UnitID: uID, Target: Coordinate{1, 0}}

		gameEvents, err := m.ApplyTurnCommand(cmd)

		if err == nil || !strings.Contains(err.Error(), "unsupported command type") {
			t.Errorf("Expect ApplyCommand for invalid type should fail, but got %#v", err)
		}
		if len(gameEvents) > 0 {
			t.Errorf("Expected empty gameEvents returned, got %#v", gameEvents)
		}

	})
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
				return // No need to check further if we expected an error
			}

			if err != nil {
				t.Errorf("Expected no error, got '%s'", err.Error())
			}
		})
	}
}

func TestMatch_StartTurn_NotTriggeringSuddenDeath(t *testing.T) {
	king, ok := GetArchetype("King")
	if !ok {
		t.Fatalf("Archetype King does not exist")
	}

	fighter, ok := GetArchetype("Fighter")
	if !ok {
		t.Fatalf("Archetype Fighter does not exist")
	}

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

		gameEvents := m.StartTurn()

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
		m.WorkingState.Units[u1] = &Unit{ID: u1, Team: 1, HP: 1, Position: Coordinate{0, 0}, Type: king}
		m.WorkingState.Units[u2] = &Unit{ID: u2, Team: 2, HP: 1, Position: Coordinate{0, 1}, Type: king}
		m.WorkingState.Units[u3] = &Unit{ID: u3, Team: 1, HP: 1, Position: Coordinate{0, 7}, Type: fighter}
		m.WorkingState.Units[u4] = &Unit{ID: u4, Team: 2, HP: 1, Position: Coordinate{0, 8}, Type: fighter}

		gameEvents := m.StartTurn()

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

func TestMatch_StartTurn_SuddenDeath(t *testing.T) {
	king, ok := GetArchetype("King")
	if !ok {
		t.Fatalf("Archetype King does not exist")
	}

	fighter, ok := GetArchetype("Fighter")
	if !ok {
		t.Fatalf("Archetype Fighter does not exist")
	}

	u1 := NewUnitID(1, 0)
	u2 := NewUnitID(2, 0)
	u3 := NewUnitID(1, 1)
	u4 := NewUnitID(2, 1)

	t.Run("stage has many available tiles", func(t *testing.T) {
		m := newTestMatch(4, 3)
		m.TrueState.Turn = 101

		m.WorkingState.Units[u1] = &Unit{ID: u1, Team: 1, HP: 1, Position: Coordinate{0, 0}, Type: king}
		m.WorkingState.Units[u2] = &Unit{ID: u2, Team: 2, HP: 1, Position: Coordinate{0, 1}, Type: king}
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

		gameEvents := m.StartTurn()

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
		bID := NewBombID(100, 1, u1)
		m.WorkingState.Units[u1] = &Unit{ID: u1, Team: 1, HP: 1, Position: Coordinate{0, 0}, Type: king}
		m.WorkingState.Units[u2] = &Unit{ID: u2, Team: 2, HP: 1, Position: Coordinate{0, 1}, Type: king}
		m.WorkingState.Units[u3] = &Unit{ID: u3, Team: 1, HP: 1, Position: Coordinate{0, 7}, Type: fighter}
		m.WorkingState.Units[u4] = &Unit{ID: u4, Team: 2, HP: 1, Position: Coordinate{0, 8}, Type: fighter}
		m.WorkingState.Bombs[bID] = &Bomb{ID: bID, OwnerID: u2, Position: Coordinate{0, 3}}
		m.WorkingState.Grid[0][0].OccupantType = OccupantUnit
		m.WorkingState.Grid[0][0].OccupantID = int64(u1)
		m.WorkingState.Grid[1][0].OccupantType = OccupantUnit
		m.WorkingState.Grid[1][0].OccupantID = int64(u2)
		m.WorkingState.Grid[2][0].Type = TerrainBlock
		m.WorkingState.Grid[3][0].OccupantType = OccupantBomb
		m.WorkingState.Grid[3][0].OccupantID = int64(bID)
		m.WorkingState.Grid[4][0].OccupantType = OccupantSoftBlock
		m.WorkingState.Grid[5][0].OccupantType = OccupantItem
		m.WorkingState.Grid[7][0].OccupantType = OccupantUnit
		m.WorkingState.Grid[7][0].OccupantID = int64(u4)
		m.WorkingState.Grid[8][0].OccupantType = OccupantUnit
		m.WorkingState.Grid[8][0].OccupantID = int64(u4)

		gameEvents := m.StartTurn()

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
		m.WorkingState.Units[u1] = &Unit{ID: u1, Team: 1, HP: 1, Position: Coordinate{0, 0}, Type: king}
		m.WorkingState.Units[u2] = &Unit{ID: u2, Team: 2, HP: 1, Position: Coordinate{0, 1}, Type: king}
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

		gameEvents := m.StartTurn()

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

func TestMatch_ResolveTurn_ExplosionAndBlast(t *testing.T) {
	king, ok := GetArchetype("King")
	if !ok {
		t.Fatalf("Archetype King does not exist")
	}

	fighter, ok := GetArchetype("Fighter")
	if !ok {
		t.Fatalf("Archetype Fighter does not exist")
	}

	t.Run("Natural countdown tick reduces for countdown bombs and does not detonate early", func(t *testing.T) {
		m := newTestMatch(16, 16)
		u1 := NewUnitID(1, 0)
		u2 := NewUnitID(2, 0)
		u3 := NewUnitID(1, 1)
		u4 := NewUnitID(2, 1)
		b1 := NewBombID(1, 1, u1)
		b2 := NewBombID(1, 2, u2)
		m.WorkingState.Units[u1] = &Unit{ID: u1, Team: 1, HP: 1, Type: king}
		m.WorkingState.Units[u2] = &Unit{ID: u2, Team: 2, HP: 1, Type: king}
		m.WorkingState.Units[u3] = &Unit{ID: u3, Team: 1, HP: 1, Type: fighter}
		m.WorkingState.Units[u4] = &Unit{ID: u4, Team: 2, HP: 1, Type: fighter}
		m.WorkingState.Bombs[b1] = &Bomb{ID: b1, Countdown: 3, Range: 2, Position: Coordinate{5, 5}}
		m.WorkingState.Bombs[b2] = &Bomb{ID: b2, Countdown: -1, Range: 2, Position: Coordinate{15, 15}}
		m.WorkingState.Grid[5][5] = Tile{OccupantType: OccupantBomb, OccupantID: int64(b1)}
		m.WorkingState.Grid[15][15] = Tile{OccupantType: OccupantBomb, OccupantID: int64(b2)}

		evts := m.ResolveTurn()

		if len(evts) != 1 {
			t.Errorf("expected 1 GameEvent returned, got %d", len(evts))
		}
		resEvt := evts[0]
		if resEvt.Type != GameEvtBombCountdownUpdated || resEvt.BombID != b1 || resEvt.Countdown != 2 {
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

		events := m.ResolveTurn()

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
		for _, e := range events {
			if e.Type == GameEvtUnitDamaged {
				damageEventsCount++
				continue
			}
			if e.Type == GameEvtUnitDied {
				unitDiedEventsCount++
			}
			if e.Type == GameEvtSoftBlockDestroyed {
				softBlockDestroyedEventsCount++
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
		uID := NewUnitID(1, 0)

		b1 := NewBombID(1, 1, uID)
		m.WorkingState.Bombs[b1] = &Bomb{ID: b1, Countdown: 1, Range: 2, Position: Coordinate{1, 1}}
		m.WorkingState.Grid[1][1] = Tile{OccupantType: OccupantBomb, OccupantID: int64(b1)}

		b2 := NewBombID(1, 2, uID)
		m.WorkingState.Bombs[b2] = &Bomb{ID: b2, Countdown: 3, Range: 2, Position: Coordinate{1, 2}}
		m.WorkingState.Grid[2][1] = Tile{OccupantType: OccupantBomb, OccupantID: int64(b2)}

		events := m.ResolveTurn()

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

		explosionPackets := 0
		for _, e := range events {
			if e.Type == GameEvtBombExploded {
				explosionPackets++
			}
		}
		if explosionPackets != 2 {
			t.Errorf("Expected 2 separate explosion log streams inside replay, found %d", explosionPackets)
		}
	})

	t.Run("Soft blocks act as solid line of sight obstacles shielding behind tiles during turn", func(t *testing.T) {
		m := newTestMatch(16, 16)
		uID := NewUnitID(1, 0)

		bID := NewBombID(1, 1, uID)
		m.WorkingState.Bombs[bID] = &Bomb{ID: bID, Countdown: 1, Range: 5, Position: Coordinate{0, 0}}
		m.WorkingState.Grid[0][0] = Tile{OccupantType: OccupantBomb, OccupantID: int64(bID)}

		sbID := 55
		m.WorkingState.SoftBlocks[sbID] = &SoftBlock{ID: sbID, Position: Coordinate{1, 0}}
		m.WorkingState.Grid[0][1] = Tile{OccupantType: OccupantSoftBlock, OccupantID: int64(sbID)}

		b2 := NewBombID(1, 2, uID)
		m.WorkingState.Bombs[b2] = &Bomb{ID: b2, Countdown: 3, Range: 2, Position: Coordinate{2, 0}}
		m.WorkingState.Grid[0][2] = Tile{OccupantType: OccupantBomb, OccupantID: int64(b2)}

		events := m.ResolveTurn()

		// Verification: SoftBlock must be flagged for destruction, but its active shielding body
		// must prevent the blast ray from crossing over to touch Bomb 2 in this frame pass.
		if _, ok := m.WorkingState.SoftBlocks[sbID]; ok {
			t.Error("Soft block failed to be destroyed by direct ray hit")
		}
		if _, ok := m.WorkingState.Bombs[b2]; !ok {
			t.Error("Security Breach: Bomb 2 ignited through a solid shielding soft block!")
		}
		if m.WorkingState.Grid[0][1].OccupantType != OccupantNone {
			t.Errorf("Grid Clearance Bug: Exploded SoftBlock positions %#v failed to revert to OccupantNone, got %v", Coordinate{1, 0}, m.WorkingState.Grid[0][1].OccupantType)
		}

		softBlockDestroyedPackets := 0
		for _, e := range events {
			if e.Type == GameEvtSoftBlockDestroyed {
				softBlockDestroyedPackets++
			}
		}
		if softBlockDestroyedPackets != 1 {
			t.Errorf("Expected 1 soft block destroyed log streams inside replay, found %d", softBlockDestroyedPackets)
		}
	})
}

func TestMatch_ResolveTurn_TimelineSystemTransitions(t *testing.T) {
	king, ok := GetArchetype("King")
	if !ok {
		t.Fatalf("Archetype King does not exist")
	}

	fighter, ok := GetArchetype("Fighter")
	if !ok {
		t.Fatalf("Archetype Fighter does not exist")
	}

	u1 := NewUnitID(1, 0)
	u2 := NewUnitID(2, 0)
	u3 := NewUnitID(1, 1)
	u4 := NewUnitID(2, 1)

	t.Run("Empty turn resolution executes smoothly with zero structural mutations", func(t *testing.T) {
		m := newTestMatch(16, 16)

		m.WorkingState.Units[u1] = &Unit{ID: u1, Team: 1, HP: 1, Type: king, HasMoved: true}
		m.WorkingState.Units[u2] = &Unit{ID: u2, Team: 2, HP: 1, Type: king, HasUsedSkill: true}
		m.WorkingState.Units[u3] = &Unit{ID: u3, Team: 1, HP: 1, Type: fighter, HasMoved: true, HasUsedSkill: true}
		m.WorkingState.Units[u4] = &Unit{ID: u4, Team: 2, HP: 1, Type: fighter}

		events := m.ResolveTurn()

		if len(events) != 0 {
			t.Errorf("Expected clean slice array from empty resolution pass, got %d items", len(events))
		}

		if m.WorkingState.Turn != 2 {
			t.Errorf("Expected Turn will increment to 2, got %d", m.WorkingState.Turn)
		}

		for uID, unit := range m.WorkingState.Units {
			if unit.HasMoved || unit.HasUsedSkill {
				t.Errorf("Expected Unit %#X HasMoved and HasUsed are reset, got HasMoved %v, HasUsedSkill %v", uID, unit.HasMoved, unit.HasUsedSkill)
			}
		}

		if len(m.PlaybackLog) != 0 {
			t.Errorf("Expectd clean slice array from PlaybackLog, got %d items", len(m.PlaybackLog))
		}
	})

	t.Run("Successful turn resolution advances true match clock and deep copies sandbox workspace", func(t *testing.T) {
		m := newTestMatch(16, 16)
		m.TrueState.Turn = 1
		m.WorkingState.Turn = 1
		m.WorkingState.ActiveTeam = 1

		m.WorkingState.Units[u1] = &Unit{ID: u1, Team: 1, HP: 2, Type: king}
		m.WorkingState.Units[u2] = &Unit{ID: u2, Team: 2, HP: 2, Type: king}
		m.WorkingState.Units[u3] = &Unit{ID: u3, Team: 1, HP: 2, Type: fighter}
		m.WorkingState.Units[u4] = &Unit{ID: u4, Team: 2, HP: 2, Type: fighter}

		m.WorkingState.Grid[0][0] = Tile{Type: TerrainPlain, OccupantType: OccupantUnit, OccupantID: int64(u1)}
		m.WorkingState.Grid[1][0] = Tile{Type: TerrainPlain, OccupantType: OccupantUnit, OccupantID: int64(u2)}

		_ = m.ResolveTurn()

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

// AddUnit adds a unit to the match's WorkingState and syncs the grid.
// Test-only helper.
func (m *Match) AddUnit(team, idx int, arch Archetype, pos Coordinate, hp int) *Unit {
	id := NewUnitID(team, idx)
	u := &Unit{
		ID:           id,
		Type:         arch,
		Position:     pos,
		Speed:        arch.BaseSpeed,
		BombMaxRange: arch.BombMaxRange,
		BombMinRange: arch.BombMinRange,
		BombPower:    arch.BombPower,
		MaxBombCount: arch.MaxBombCount,
		BombUsed:     0,
		Team:         team,
		HP:           hp,
		Skills:       arch.PresetSkills,
	}
	m.WorkingState.Units[id] = u
	m.WorkingState.Grid[pos.Y][pos.X] = Tile{
		Type:         TerrainPlain,
		OccupantType: OccupantUnit,
		OccupantID:   int64(id),
	}
	return u
}

// AddBomb adds a bomb to the match's WorkingState and syncs the grid.
// Test-only helper.
func (m *Match) AddBomb(turn, counter int, owner UnitID, pos Coordinate, range_, cd int) *Bomb {
	b := &Bomb{
		ID:        NewBombID(turn, counter, owner),
		OwnerID:   owner,
		Position:  pos,
		Range:     range_,
		Countdown: cd,
	}
	m.WorkingState.Bombs[b.ID] = b
	m.WorkingState.Grid[pos.Y][pos.X] = Tile{
		Type:         TerrainPlain,
		OccupantType: OccupantBomb,
		OccupantID:   int64(b.ID),
	}
	return b
}

// AddSoftBlock adds a soft block to the match's WorkingState and syncs the grid.
// Test-only helper.
func (m *Match) AddSoftBlock(id int, pos Coordinate) *SoftBlock {
	sb := &SoftBlock{ID: id, Position: pos}
	m.WorkingState.SoftBlocks[id] = sb
	m.WorkingState.Grid[pos.Y][pos.X] = Tile{
		Type:         TerrainPlain,
		OccupantType: OccupantSoftBlock,
		OccupantID:   int64(id),
	}
	return sb
}

func newTestMatchWithFullTeam(t *testing.T, p1KingAlive, p2KingAlive bool, p1Alive, p2Alive [4]bool) *Match {
	t.Helper()

	m := newTestMatch(5, 2)
	m.WorkingState.Turn = 1
	m.WorkingState.ActiveTeam = 1

	king, ok := GetArchetype("King")
	if !ok {
		t.Fatalf("Archetype King does not exist")
	}

	fighter, ok := GetArchetype("Fighter")
	if !ok {
		t.Fatalf("Archetype Fighter does not exist")
	}

	p1KingHP := 1
	if !p1KingAlive {
		p1KingHP = 0
	}
	m.AddUnit(1, 0, king, Coordinate{0, 0}, p1KingHP)

	p2KingHP := 1
	if !p2KingAlive {
		p2KingHP = 0
	}
	m.AddUnit(2, 0, king, Coordinate{0, 1}, p2KingHP)

	for i, alive := range p1Alive {
		hp := 1
		if !alive {
			hp = 0
		}
		m.AddUnit(1, i+1, fighter, Coordinate{i + 1, 0}, hp)
	}

	for i, alive := range p2Alive {
		hp := 1
		if !alive {
			hp = 0
		}
		m.AddUnit(2, i+1, fighter, Coordinate{i + 1, 1}, hp)
	}

	m.TrueState = m.WorkingState.DeepCopy()

	return m
}

func TestMatch_Resolve_VictoryCondition_Suite(t *testing.T) {
	tests := []struct {
		name                string
		p1King              bool
		p1NonKings          [4]bool
		p2King              bool
		p2NonKings          [4]bool
		expectedWinningTeam int
	}{
		{
			name:                "Standard gameplay state",
			p1King:              true,
			p1NonKings:          [4]bool{true, true, false, true},
			p2King:              true,
			p2NonKings:          [4]bool{true, true, false, false},
			expectedWinningTeam: MatchInProgress,
		},
		// P1 Wins
		{
			name:                "P1 Wins: P2 misses non-king",
			p1King:              true,
			p1NonKings:          [4]bool{true, true, false, true},
			p2King:              true,
			p2NonKings:          [4]bool{false, false, false, false},
			expectedWinningTeam: 1,
		},
		{
			name:                "P1 Wins: P2 King dead",
			p1King:              true,
			p1NonKings:          [4]bool{true, true, false, true},
			p2King:              false,
			p2NonKings:          [4]bool{true, false, false, false},
			expectedWinningTeam: 1,
		},
		{
			name:                "P1 Wins: P2 wiped out",
			p1King:              true,
			p1NonKings:          [4]bool{true, true, false, true},
			p2King:              false,
			p2NonKings:          [4]bool{false, false, false, false},
			expectedWinningTeam: 1,
		},
		{
			name:                "P1 Wins: P1 lone king",
			p1King:              true,
			p1NonKings:          [4]bool{false, false, false, false},
			p2King:              false,
			p2NonKings:          [4]bool{false, false, false, false},
			expectedWinningTeam: 1,
		},
		// P2 Wins
		{
			name:                "P2 Wins: P1 misses non-king",
			p1King:              true,
			p1NonKings:          [4]bool{false, false, false, false},
			p2King:              true,
			p2NonKings:          [4]bool{true, true, false, true},
			expectedWinningTeam: 2,
		},
		{
			name:                "P2 Wins: P1 King dead",
			p1King:              false,
			p1NonKings:          [4]bool{true, true, false, true},
			p2King:              true,
			p2NonKings:          [4]bool{true, false, false, false},
			expectedWinningTeam: 2,
		},
		{
			name:                "P2 Wins: P1 wiped out",
			p1King:              false,
			p1NonKings:          [4]bool{false, false, false, false},
			p2King:              true,
			p2NonKings:          [4]bool{true, true, false, true},
			expectedWinningTeam: 2,
		},
		{
			name:                "P2 Wins: P2 lone king",
			p1King:              false,
			p1NonKings:          [4]bool{false, false, false, false},
			p2King:              true,
			p2NonKings:          [4]bool{false, false, false, false},
			expectedWinningTeam: 2,
		},
		// Draw Conditions
		{
			name:                "Draw: Both Kings dead",
			p1King:              false,
			p1NonKings:          [4]bool{false, true, false, false},
			p2King:              false,
			p2NonKings:          [4]bool{false, false, true, false},
			expectedWinningTeam: MatchDrawn,
		},
		{
			name:                "Draw: Both Kings dead, P2 wiped out",
			p1King:              false,
			p1NonKings:          [4]bool{false, true, false, false},
			p2King:              false,
			p2NonKings:          [4]bool{false, false, false, false},
			expectedWinningTeam: MatchDrawn,
		},
		{
			name:                "Draw: Both Kings dead, P1 wiped out",
			p1King:              false,
			p1NonKings:          [4]bool{false, false, false, false},
			p2King:              false,
			p2NonKings:          [4]bool{false, false, true, false},
			expectedWinningTeam: MatchDrawn,
		},
		{
			name:                "Draw: Everyone dead",
			p1King:              false,
			p1NonKings:          [4]bool{false, false, false, false},
			p2King:              false,
			p2NonKings:          [4]bool{false, false, false, false},
			expectedWinningTeam: MatchDrawn,
		},
		{
			name:                "Draw: Both Lone Kings dead",
			p1King:              true,
			p1NonKings:          [4]bool{false, false, false, false},
			p2King:              true,
			p2NonKings:          [4]bool{false, false, false, false},
			expectedWinningTeam: MatchDrawn,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestMatchWithFullTeam(t, tt.p1King, tt.p2King, tt.p1NonKings, tt.p2NonKings)

			events := m.ResolveTurn()

			if m.WinnerTeamID != tt.expectedWinningTeam {
				t.Errorf("Victory Guard Failed! Expected WinnerTeamID = %d, got %d", tt.expectedWinningTeam, m.WinnerTeamID)
			}

			var foundEndedEvent bool
			for _, event := range events {
				if event.Type == GameEvtMatchEnded {
					foundEndedEvent = true
					if event.WinnerTeamID != tt.expectedWinningTeam {
						t.Errorf("Malformed MatchEndedEvent.WinnerTeam, got winner %d", event.WinnerTeamID)
					}
				}
			}

			if tt.expectedWinningTeam == MatchInProgress {
				if foundEndedEvent {
					t.Errorf("MatchEndedEvent should not be captured, but got one")
				}
				return // all the subsequent verifications aren't related to MatchInProgress
			}
			if !foundEndedEvent {
				t.Error("Missing critical MatchEndedEvent token inside returned telemetry array stream")
			}
		})
	}
}
