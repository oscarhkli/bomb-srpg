package engine

import (
	"reflect"
	"slices"
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
	}
	m.WorkingState = m.TrueState.DeepCopy()

	m.WorkingState.Units[unitID].HP = 2

	m.ResetTurn()

	if m.WorkingState.Units[unitID].HP != 5 {
		t.Errorf("Rollback invariant broken: uncommitted unit action still exists in WorkingState after reset!")
	}
	if m.TrueState.Units[unitID].HP != 5 {
		t.Errorf("CRITICAL POINTER LEAK: Sandbox action permanently corrupted your master TrueState checkpoint!")
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

	event := UnitMovedEvent{
		UnitID: unitID,
		From:   Coordinate{1, 0},
		To:     Coordinate{10, 100},
	}

	m.SubmitAction(event)

	var interfaceEvent GameEvent = event
	if len(m.PlaybackLog) == 0 || !slices.Contains(m.PlaybackLog, interfaceEvent) {
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

	event := UnitMovedEvent{
		UnitID: unitID,
		From:   Coordinate{1, 0},
		To:     Coordinate{10, 100},
	}

	m.SubmitAction(event)

	var interfaceEvent GameEvent = event
	if len(m.PlaybackLog) == 0 || !slices.Contains(m.PlaybackLog, interfaceEvent) {
		t.Errorf("Expected PlaybackLog to capture and retain the submitted event payload successfully")
	}

	if !reflect.DeepEqual(m.TrueState, m.WorkingState) {
		t.Error("Expected TrueState to be overridden by Working state, got not overridden")
	}
}

func TestCommandMoveUnit(t *testing.T) {
	// Define the schema for our table rows
	type testCase struct {
		name          string
		unitID        UnitID
		target        Coordinate
		setupState    func() *Match // Context setup helper
		wantErr       bool
		errContains   string
		verifyResults func(t *testing.T, m *Match) // Post-execution state check
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
			errContains: "security violation: unit ID 51 does not exist",
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
			errContains: "tactical restriction: unit 18 is dead",
		},
		{
			name:   "Failure: Wrong team turn",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(2, 2)
				m.WorkingState.Turn = 1 // Team 1's turn
				m.WorkingState.Units[validUnitID] = &Unit{
					ID:       validUnitID,
					HP:       1,
					Team:     2, // Team 2 Unit
					Position: origin,
				}
				return m
			},
			wantErr:     true,
			errContains: "turn restriction: unit 16 belongs to Team 2 but it's currently Team 1's turn",
		},
		// { This test is not available at the moment - not until the Skills implementation in Phase 3
		// 	name:   "Failure: Unit passes through but cannot land on HardRock",
		// },
		{
			name:   "Failure: Data corruption - unit out of bounds",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(3, 3)
				m.WorkingState.Turn = 1

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
			errContains: "data corruption: unit 16 current position",
		},
		{
			name:   "Failure: Grid data desync",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(2, 2)
				m.WorkingState.Turn = 1
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
			errContains: "data desync: grid matrix at",
		},
		{
			name:   "Failure: Target out of moving range",
			unitID: validUnitID,
			target: outOfRangeTarget,
			setupState: func() *Match {
				m := newTestMatch(10, 10)
				m.WorkingState.Turn = 1
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
			errContains: "movement restriction: target coordinate is out of moving range",
		},
		{
			name:   "Success: Unit moves successfully",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(3, 3)
				m.WorkingState.Turn = 1
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
			verifyResults: func(t *testing.T, m *Match) {
				oldCell := m.WorkingState.Grid[origin.Y][origin.X]
				if oldCell.OccupantType != OccupantNone {
					t.Errorf("expected old tile to be cleared, got type %v", oldCell.OccupantType)
				}

				newCell := m.WorkingState.Grid[validTarget.Y][validTarget.X]
				if newCell.OccupantID != int64(validUnitID) {
					t.Errorf("expected unit %d at target, got %d", validUnitID, newCell.OccupantID)
				}

				if len(m.PlaybackLog) != 1 {
					t.Errorf("expected 1 action submitted, got %d", len(m.PlaybackLog))
				}
				event, ok := m.PlaybackLog[0].(UnitMovedEvent)
				if !ok || event.UnitID != validUnitID || event.From != origin || event.To != validTarget {
					t.Errorf("malformed UnitMovedEvent logged: %+v", m.PlaybackLog[0])
				}
			},
		},
	}

	// Run loop
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := tt.setupState()

			err := match.CommandMoveUnit(tt.unitID, tt.target)

			// Error checking
			if (err != nil) != tt.wantErr {
				t.Fatalf("CommandMoveUnit() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
			}

			// State validation checking
			if tt.verifyResults != nil {
				tt.verifyResults(t, match)
			}
		})
	}
}

func TestCommandPlaceBomb(t *testing.T) {
	// Define the schema for our table rows matching the style of TestCommandMoveUnit
	type testCase struct {
		name          string
		unitID        UnitID
		target        Coordinate
		setupState    func() *Match // Context setup helper
		wantErr       bool
		errContains   string
		verifyResults func(t *testing.T, m *Match) // Post-execution state check
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
			errContains: "security violation: unit ID 51 does not exist",
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
			errContains: "tactical restriction: unit 18 is dead",
		},
		{
			name:   "Failure: Wrong team turn",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(2, 2)
				m.WorkingState.Turn = 1 // Team 1's turn
				m.WorkingState.Units[validUnitID] = &Unit{
					ID:       validUnitID,
					HP:       1,
					Team:     2, // Team 2 Unit
					Position: origin,
				}
				return m
			},
			wantErr:     true,
			errContains: "turn restriction: unit 16 belongs to Team 2 but it's currently Team 1's turn",
		},
		{
			name:   "Failure: Data corruption - unit out of bounds",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(3, 3)
				m.WorkingState.Turn = 1
				m.WorkingState.Units[validUnitID] = &Unit{
					ID:       validUnitID,
					HP:       1,
					Team:     1,
					Position: Coordinate{-5, -5}, // Out of stage bounds
				}
				return m
			},
			wantErr:     true,
			errContains: "data corruption: unit 16 current position",
		},
		{
			name:   "Failure: Grid data desync",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(2, 2)
				m.WorkingState.Turn = 1
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
			errContains: "data desync: grid matrix at",
		},
		{
			name:   "Failure: Unit has used up all bombs",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(3, 3)
				m.WorkingState.Turn = 1
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
			errContains: "unit restriction: unit 16 has used up all his bombs",
		},
		{
			name:   "Failure: Target out of placement range",
			unitID: validUnitID,
			target: outOfRangeTarget,
			setupState: func() *Match {
				m := newTestMatch(10, 10)
				m.WorkingState.Turn = 1
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
			errContains: "bomb placement restriction: target coordinate is out of placement range",
		},
		{
			name:   "Failure: Illegal target wanding",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(3, 3)
				m.WorkingState.Turn = 3
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
			errContains: "bomb placement rejected: terrain restriction",
		},
		{
			name:   "Success: Bomb placed successfully",
			unitID: validUnitID,
			target: validTarget,
			setupState: func() *Match {
				m := newTestMatch(3, 3)
				m.WorkingState.Turn = 3
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
			verifyResults: func(t *testing.T, m *Match) {
				if m.WorkingState.TurnBombCounter != 1 {
					t.Errorf("expected TurnBombCounter to be 1, got %d", m.WorkingState.TurnBombCounter)
				}

				expectedBombID := NewBombID(3, 1, validUnitID)
				bomb, exists := m.WorkingState.Bombs[expectedBombID]
				if !exists {
					t.Fatalf("expected bomb tracking map entry under ID 0x%X missing", expectedBombID)
				}

				if bomb.OwnerID != validUnitID || bomb.Position != validTarget || bomb.Range != 3 || bomb.Countdown != 5 {
					t.Errorf("registered bomb structural parameters mismatched: %+v", bomb)
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
				event, ok := m.PlaybackLog[0].(BombPlacedEvent)
				if !ok || event.UnitID != validUnitID || event.BombID != expectedBombID || event.Position != validTarget || event.Range != 3 || event.Countdown != 5 {
					t.Errorf("malformed BombPlacedEvent logged: %+v", m.PlaybackLog[0])
				}
			},
		},
	}

	// Execution loop matching TestCommandMoveUnit
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := tt.setupState()

			err := match.CommandPlaceBomb(tt.unitID, tt.target)

			// Error validation checking
			if (err != nil) != tt.wantErr {
				t.Fatalf("CommandPlaceBomb() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
			}

			// State transformation post-checks
			if tt.verifyResults != nil {
				tt.verifyResults(t, match)
			}
		})
	}
}

// newTestMatch generates a clean slate grid environment
func newTestMatch(width, height int) *Match {
	grid := make([][]Tile, height)
	for i := range grid {
		grid[i] = make([]Tile, width)
	}

	return &Match{
		WorkingState: &GameState{
			Turn:  1,
			Units: make(map[UnitID]*Unit),
			Grid:  grid,
		},
		PlaybackLog: []GameEvent{},
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
			errorContains: "boundary restriction",
		},
		{
			name:          "Failure: Within bound, non-TerrainPlain",
			pos:           Coordinate{2, 0},
			expectError:   true,
			errorContains: "terrain restriction",
		},
		{
			name:          "Failure: Within bound, TerrainPlain, on non-OccupantNone",
			pos:           Coordinate{1, 0},
			expectError:   true,
			errorContains: "occupant restriction",
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

// Helper function to build a clean, manual test board matrix with zero noise
func setupCleanTestMatch() *Match {
	m := &Match{
		PlaybackLog: []GameEvent{},
	}
	grid := make([][]Tile, 16)
	for y := range 16 {
		grid[y] = make([]Tile, 16)
		for x := range 16 {
			grid[y][x] = Tile{Type: TerrainPlain, OccupantType: OccupantNone}
		}
	}
	m.WorkingState = &GameState{
		Turn:            1,
		TurnBombCounter: 0,
		Grid:            grid,
		Units:           make(map[UnitID]*Unit),
		Bombs:           make(map[BombID]*Bomb),
		SoftBlocks:      make(map[int]*SoftBlock),
	}
	m.TrueState = m.WorkingState.DeepCopy()
	return m
}

func TestMatch_ResolveTurn_ExplosionAndCombustion(t *testing.T) {
	t.Run("Natural countdown tick reduces fuse and does not detonate early", func(t *testing.T) {
		m := setupCleanTestMatch()
		bID := NewBombID(1, 1, NewUnitID(1, 0))
		m.WorkingState.Bombs[bID] = &Bomb{ID: bID, Countdown: 3, Range: 2, Position: Coordinate{5, 5}}
		m.WorkingState.Grid[5][5] = Tile{OccupantType: OccupantBomb, OccupantID: int64(bID)}

		events := m.ResolveTurn()

		if len(events) != 0 {
			t.Errorf("Expected zero events for ticking fuse, got %d", len(events))
		}
		if m.WorkingState.Bombs[bID].Countdown != 2 {
			t.Errorf("Expected fuse to reduce to 2, got %d", m.WorkingState.Bombs[bID].Countdown)
		}
	})

	t.Run("Units caught in overlapping blast patterns receive exactly 1 flat HP damage max", func(t *testing.T) {
		m := setupCleanTestMatch()
		uID := NewUnitID(1, 0)
		m.WorkingState.Units[uID] = &Unit{ID: uID, HP: 3, Position: Coordinate{2, 2}}
		m.WorkingState.Grid[2][2] = Tile{OccupantType: OccupantUnit, OccupantID: int64(uID)}

		b1 := NewBombID(1, 1, uID)
		m.WorkingState.Bombs[b1] = &Bomb{ID: b1, Countdown: 1, Range: 3, Position: Coordinate{2, 0}}
		m.WorkingState.Grid[0][2] = Tile{OccupantType: OccupantBomb, OccupantID: int64(b1)}

		b2 := NewBombID(1, 2, uID)
		m.WorkingState.Bombs[b2] = &Bomb{ID: b2, Countdown: 1, Range: 3, Position: Coordinate{0, 2}}
		m.WorkingState.Grid[2][0] = Tile{OccupantType: OccupantBomb, OccupantID: int64(b2)}

		events := m.ResolveTurn()

		if m.WorkingState.Units[uID].HP != 2 {
			t.Errorf("Flat injury rule failed! Expected Unit HP = 2, got %d", m.WorkingState.Units[uID].HP)
		}

		damageEventsCount := 0
		for _, e := range events {
			if _, ok := e.(UnitDamagedEvent); ok {
				damageEventsCount++
			}
		}
		if damageEventsCount != 1 {
			t.Errorf("Expected exactly 1 damage log packet event, found %d", damageEventsCount)
		}
	})
}

func TestMatch_ResolveTurn_CascadingChainReactions(t *testing.T) {
	t.Run("Ticking bomb triggers secondary explosive via proximity chain reaction", func(t *testing.T) {
		m := setupCleanTestMatch()
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

		explosionPackets := 0
		for _, e := range events {
			if _, ok := e.(BombExplodedEvent); ok {
				explosionPackets++
			}
		}
		if explosionPackets != 2 {
			t.Errorf("Expected 2 separate explosion log streams inside replay, found %d", explosionPackets)
		}
	})

	t.Run("Soft blocks act as solid line of sight obstacles shielding behind tiles during turn", func(t *testing.T) {
		m := setupCleanTestMatch()
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

		_ = m.ResolveTurn()

		// Verification: Wall must be flagged for destruction, but its active shielding body
		// must prevent the blast ray from crossing over to touch Bomb 2 in this frame pass.
		if _, wallActive := m.WorkingState.SoftBlocks[sbID]; wallActive {
			t.Error("Soft block failed to be destroyed by direct ray hit")
		}
		if _, bomb2Ignited := m.WorkingState.Bombs[b2]; !bomb2Ignited {
			t.Error("Security Breach: Bomb 2 ignited through a solid shielding soft block!")
		}
	})
}

func TestMatch_ResolveTurn_TimelineSystemTransitions(t *testing.T) {
	t.Run("Empty turn resolution executes smoothly with zero structural mutations", func(t *testing.T) {
		m := setupCleanTestMatch()

		events := m.ResolveTurn()

		if len(events) != 0 {
			t.Errorf("Expected clean slice array from empty resolution pass, got %d items", len(events))
		}

		if m.WorkingState.Turn != 2 {
			t.Errorf("Expected Turn will increment to 2, go %d", m.WorkingState.Turn)
		}

		// 📝 FUTURE PLACEHOLDER EXTENSIONS:
		// Requirement 4: assert m.WinnerTeamID values to verify victory conditions.
		// Requirement 6: assert deep equality matching across m.TrueState and m.WorkingState.
	})
}
