package cli

import (
	"strings"
	"testing"

	"bomb-srpg/engine"
)

// SpyMatchView traps view interaction calls for behavioral assertions.
type SpyMatchView struct {
	RenderBoardCalled      bool
	RenderGameConfigCalled bool
	FeedbackMessage        string
	FeedbackSuccess        bool
	RenderTurnHeaderCalled bool
	RenderGameEventsCalled bool
}

func (s *SpyMatchView) RenderBoard(gs *engine.GameState) error {
	s.RenderBoardCalled = true
	return nil
}

func (s *SpyMatchView) RenderGameConfig(cfg *engine.GameCfg) error {
	s.RenderGameConfigCalled = true
	return nil
}

func (s *SpyMatchView) RenderFeedback(success bool, msg string) error {
	s.FeedbackSuccess = success
	s.FeedbackMessage = msg
	return nil
}
func (s *SpyMatchView) RenderMessage(msg string) error {
	return nil
}

func (s *SpyMatchView) RenderTurnHeader(turn, activeTeamID int) error {
	s.RenderTurnHeaderCalled = true
	return nil
}

func (s *SpyMatchView) RenderGameEvents(events []engine.GameEvent) error {
	s.RenderGameEventsCalled = true
	return nil
}

func TestMatchController_SystemCommands(t *testing.T) {
	type testCase struct {
		name          string
		inputCommands string
		verify        func(t *testing.T, view *SpyMatchView, match *engine.Match)
	}

	table := []testCase{
		{
			name:          "Blank input whitespace elements are skipped",
			inputCommands: "\n\n   \n",
			verify: func(t *testing.T, view *SpyMatchView, match *engine.Match) {
				if view.FeedbackMessage != "" {
					t.Errorf("Expected loop to ignore spaces, but feedback registered: %q", view.FeedbackMessage)
				}
			},
		},
		{
			name:          "/config triggers GameCfg view",
			inputCommands: "/config\n",
			verify: func(t *testing.T, view *SpyMatchView, match *engine.Match) {
				if !view.RenderGameConfigCalled {
					t.Error("Expected /config route to call RenderGameConfig")
				}
			},
		},
		{
			name:          "/reset executes ResetTurn",
			inputCommands: "/reset\n",
			verify: func(t *testing.T, view *SpyMatchView, match *engine.Match) {
				if !view.FeedbackSuccess || !strings.Contains(view.FeedbackMessage, "reset successfully") {
					t.Errorf("Expected successful reset confirmation, got: %v - %q", view.FeedbackSuccess, view.FeedbackMessage)
				}
			},
		},
		{
			name:          "/commit execute ResolveTurn and prints log event entries",
			inputCommands: "/commit\n",
			verify: func(t *testing.T, view *SpyMatchView, match *engine.Match) {
				if !view.RenderGameEventsCalled || !view.FeedbackSuccess {
					t.Error("Expected /commit route to pass events to View and trigger a green status code notice")
				}
			},
		},
		{
			name:          "/surrender execute Surrender and prints log event entries",
			inputCommands: "/surrender\n",
			verify: func(t *testing.T, view *SpyMatchView, match *engine.Match) {
				if !view.FeedbackSuccess || !strings.Contains(view.FeedbackMessage, "Winner... PLAYER 2!") {
					t.Errorf("Expected final victory status layout sheet, got: %q", view.FeedbackMessage)
				}
				if match.WinnerTeamID != 2 {
					t.Errorf("Expected real engine WinnerTeamID to match opponent (2) after Player 1 surrenders, got: %d", match.WinnerTeamID)
				}
			},
		},
		{
			name:          "Invalid system shortcut",
			inputCommands: "/unknown_action_command\n",
			verify: func(t *testing.T, view *SpyMatchView, match *engine.Match) {
				if view.FeedbackSuccess || !strings.Contains(view.FeedbackMessage, "Unknown meta command") {
					t.Errorf("Expected unknown command failure warning, got: %v - %q", view.FeedbackSuccess, view.FeedbackMessage)
				}
			},
		},
	}

	for _, tt := range table {
		t.Run(tt.name, func(t *testing.T) {
			view := &SpyMatchView{}

			match := &engine.Match{
				WorkingState: &engine.GameState{
					Turn:       1,
					ActiveTeam: 1,
					Grid:       [][]engine.Tile{{{Type: engine.TerrainPlain}}},
				},
				WinnerTeamID: 0,
			}
			match.TrueState = match.WorkingState.DeepCopy()

			controller := NewMatchController(match, view, strings.NewReader(tt.inputCommands))

			controller.StartInputLoop()

			tt.verify(t, view, match)
		})
	}
}

func TestMatchController_GameplayActions(t *testing.T) {
	type testCase struct {
		name          string
		inputCommands string
		setupState    func(gs *engine.GameState)
		setupMatch    func() *engine.Match
		verify        func(t *testing.T, view *SpyMatchView, match *engine.Match)
	}

	table := []testCase{
		{
			name:          "move command parses valid parameters",
			inputCommands: "move 16 1 0\n",
			setupMatch: func() *engine.Match {
				m := newTestMatch(2, 2)
				uID := engine.NewUnitID(1, 0)
				m.WorkingState.Units[uID] = &engine.Unit{
					ID:           uID,
					Team:         1,
					Position:     engine.Coordinate{X: 0, Y: 0},
					HP:           1,
					Speed:        100,
					BombMaxRange: 100,
				}
				m.WorkingState.Grid[0][0].OccupantType = engine.OccupantUnit
				m.WorkingState.Grid[0][0].OccupantID = int64(uID)

				return m
			},
			verify: func(t *testing.T, view *SpyMatchView, match *engine.Match) {
				if !view.FeedbackSuccess {
					t.Errorf("Expected gameplay router to parse verb, but got error: %q", view.FeedbackMessage)
				}
			},
		},
		{
			name:          "move command parses valid parameters but invalid movement",
			inputCommands: "move 16 120 0\n",
			setupMatch: func() *engine.Match {
				m := newTestMatch(2, 2)
				uID := engine.NewUnitID(1, 0)
				m.WorkingState.Units[uID] = &engine.Unit{
					ID:           uID,
					Team:         1,
					Position:     engine.Coordinate{X: 0, Y: 0},
					Speed:        100,
					HP:           1,
					BombMaxRange: 100,
				}
				m.WorkingState.Grid[0][0].OccupantType = engine.OccupantUnit
				m.WorkingState.Grid[0][0].OccupantID = int64(uID)

				return m
			},
			verify: func(t *testing.T, view *SpyMatchView, match *engine.Match) {
				if view.FeedbackSuccess {
					t.Error("Expected illegal move coordinates to return a failure status, but reported success")
				}
				if !strings.Contains(view.FeedbackMessage, "Invalid move") {
					t.Errorf("Expected feedback warning parameters to mention 'Invalid move', got: %q", view.FeedbackMessage)
				}
			},
		},
		{
			name:          "move command syntax error",
			inputCommands: "move 16 1\n",
			setupMatch: func() *engine.Match {
				return newTestMatch(2, 2)
			},
			verify: func(t *testing.T, view *SpyMatchView, match *engine.Match) {
				if view.FeedbackSuccess || !strings.Contains(view.FeedbackMessage, "Expected: move") {
					t.Errorf("Expected short arguments length to trigger syntax warning, got: %q", view.FeedbackMessage)
				}
			},
		},
		{
			name:          "move command parameter syntax error",
			inputCommands: "move 16 apple 0\n",
			setupMatch: func() *engine.Match {
				return newTestMatch(2, 2)
			},
			verify: func(t *testing.T, view *SpyMatchView, match *engine.Match) {
				if view.FeedbackSuccess || !strings.Contains(view.FeedbackMessage, "Argument Error") {
					t.Errorf("Expected string parsing validation to capture character errors, got: %q", view.FeedbackMessage)
				}
			},
		},
		{
			name:          "bomb command parses valid parameters",
			inputCommands: "bomb 16 1 0\n",
			setupMatch: func() *engine.Match {
				m := newTestMatch(2, 2)
				uID := engine.NewUnitID(1, 0)
				m.WorkingState.Units[uID] = &engine.Unit{
					ID:           uID,
					Team:         1,
					Position:     engine.Coordinate{X: 0, Y: 0},
					HP:           1,
					MaxBombCount: 100,
					BombMaxRange: 100,
				}
				m.WorkingState.Grid[0][0].OccupantType = engine.OccupantUnit
				m.WorkingState.Grid[0][0].OccupantID = int64(uID)

				return m
			},
			verify: func(t *testing.T, view *SpyMatchView, match *engine.Match) {
				if !view.FeedbackSuccess {
					t.Errorf("Expected gameplay router to parse verb, but got error: %q", view.FeedbackMessage)
				}
			},
		},
		{
			name:          "bomb command parses valid parameters but invalid bomb placement",
			inputCommands: "bomb 16 1 0\n",
			setupMatch: func() *engine.Match {
				m := newTestMatch(2, 2)
				uID := engine.NewUnitID(1, 0)
				m.WorkingState.Units[uID] = &engine.Unit{
					ID:           uID,
					Team:         1,
					Position:     engine.Coordinate{X: 0, Y: 0},
					HP:           1,
					BombMaxRange: 100,
				}
				m.WorkingState.Grid[0][0].OccupantType = engine.OccupantUnit
				m.WorkingState.Grid[0][0].OccupantID = int64(uID)

				return m
			},
			verify: func(t *testing.T, view *SpyMatchView, match *engine.Match) {
				if view.FeedbackSuccess {
					t.Error("Expected illegal bomb coordinates to return a failure status, but reported success")
				}
				if !strings.Contains(view.FeedbackMessage, "unit 0x10 out of bombs") {
					t.Errorf("Expected feedback warning parameters to mention 'unit 0x10 out of bombs', got: %q", view.FeedbackMessage)
				}
			},
		},
		{
			name:          "bomb command syntax error",
			inputCommands: "bomb 16 1\n",
			setupMatch: func() *engine.Match {
				return newTestMatch(2, 2)
			},
			verify: func(t *testing.T, view *SpyMatchView, match *engine.Match) {
				if view.FeedbackSuccess || !strings.Contains(view.FeedbackMessage, "Expected: bomb") {
					t.Errorf("Expected short arguments length to trigger syntax warning, got: %q", view.FeedbackMessage)
				}
			},
		},
		{
			name:          "bomb command parameter syntax error",
			inputCommands: "bomb 16 apple 0\n",
			setupMatch: func() *engine.Match {
				return newTestMatch(2, 2)
			},
			verify: func(t *testing.T, view *SpyMatchView, match *engine.Match) {
				if view.FeedbackSuccess || !strings.Contains(view.FeedbackMessage, "Argument Error") {
					t.Errorf("Expected string parsing validation to capture character errors, got: %q", view.FeedbackMessage)
				}
			},
		},
	}

	for _, tt := range table {
		t.Run(tt.name, func(t *testing.T) {
			view := &SpyMatchView{}
			match := tt.setupMatch()
			controller := NewMatchController(match, view, strings.NewReader(tt.inputCommands))

			controller.StartInputLoop()

			tt.verify(t, view, match)
		})
	}
}

// newTestMatch generates a clean slate grid environment
func newTestMatch(width, height int) *engine.Match {
	grid := make([][]engine.Tile, height)
	for y, row := range grid {
		grid[y] = make([]engine.Tile, width)
		for x := range row {
			grid[y][x] = engine.Tile{Type: engine.TerrainPlain, OccupantType: engine.OccupantNone}
		}
	}

	m := &engine.Match{
		GameCfg: engine.GameCfg{MaxTurns: 100},
		WorkingState: &engine.GameState{
			Turn:            1,
			ActiveTeam:      1,
			TurnBombCounter: 0,
			Grid:            grid,
			Units:           make(map[engine.UnitID]*engine.Unit),
			Bombs:           make(map[engine.BombID]*engine.Bomb),
			SoftBlocks:      make(map[int]*engine.SoftBlock),
		},
		PlaybackLog: []engine.GameEvent{},
	}
	m.TrueState = m.WorkingState.DeepCopy()

	return m
}
