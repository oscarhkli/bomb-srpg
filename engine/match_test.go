package engine

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestMatch_ResetTurn(t *testing.T) {
	unitID := 8
	m := Match{
		TrueState: &GameState{
			Turn: 1,
			Units: map[int]*Unit{
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
	unitID := 8
	m := Match{
		TrueState: &GameState{
			Turn: 1,
			Units: map[int]*Unit{
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
	unitID := 8
	m := Match{
		TrueState: &GameState{
			Turn: 1,
			Units: map[int]*Unit{
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
		unitID        int
		target        Coordinate
		setupState    func() *Match // Context setup helper
		wantErr       bool
		errContains   string
		verifyResults func(t *testing.T, m *Match) // Post-execution state check
	}

	// Constants for easy setup
	const (
		validUnitID = 42
		deadUnitID  = 99
	)
	origin := Coordinate{X: 1, Y: 1}
	validTarget := Coordinate{X: 1, Y: 2}
	outOfRangeTarget := Coordinate{X: 5, Y: 5}

	tests := []testCase{
		{
			name:   "Failure: Unit does not exist",
			unitID: 999, // Missing ID
			target: validTarget,
			setupState: func() *Match {
				return newTestMatch(2, 2) // Empty map, no units
			},
			wantErr:     true,
			errContains: "security violation: unit ID 999 does not exist",
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
			errContains: "tactical restriction: unit 99 is dead",
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
			errContains: "turn restriction: unit 42 belongs to Team 2 but it's currently Team 1's turn",
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
					Position: Coordinate{X: -5, Y: -5}, // Out of stage bounds
					Speed:    3,
				}
				// Do not add it to the Grid matrix since the coordinate is invalid
				return m
			},
			wantErr:     true,
			errContains: "data corruption: unit 42 current position",
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
				m.WorkingState.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantUnit, OccupantID: validUnitID}
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
				m.WorkingState.Grid[origin.Y][origin.X] = Tile{OccupantType: OccupantUnit, OccupantID: validUnitID}
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
				if newCell.OccupantID != validUnitID {
					t.Errorf("expected unit %d at target, got %d", validUnitID, newCell.OccupantID)
				}

				if len(m.PlaybackLog) != 1 {
					t.Errorf("expected 1 action submitted, got %d", len(m.PlaybackLog))
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

// newTestMatch generates a clean slate grid environment
func newTestMatch(width, height int) *Match {
	grid := make([][]Tile, height)
	for i := range grid {
		grid[i] = make([]Tile, width)
	}

	return &Match{
		WorkingState: &GameState{
			Turn:  1,
			Units: make(map[int]*Unit),
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
