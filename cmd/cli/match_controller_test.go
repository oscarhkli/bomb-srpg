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
			verify: func(t *testing.T, spy *SpyMatchView, masterMatch *engine.Match) {
				if spy.FeedbackMessage != "" {
					t.Errorf("Expected loop to ignore spaces, but feedback registered: %q", spy.FeedbackMessage)
				}
			},
		},
		{
			name:          "/config triggers GameCfg view",
			inputCommands: "/config\n",
			verify: func(t *testing.T, spy *SpyMatchView, masterMatch *engine.Match) {
				if !spy.RenderGameConfigCalled {
					t.Error("Expected /config route to call RenderGameConfig")
				}
			},
		},
		{
			name:          "/reset executes ResetTurn",
			inputCommands: "/reset\n",
			verify: func(t *testing.T, spy *SpyMatchView, masterMatch *engine.Match) {
				if !spy.FeedbackSuccess || !strings.Contains(spy.FeedbackMessage, "reset successfully") {
					t.Errorf("Expected successful reset confirmation, got: %v - %q", spy.FeedbackSuccess, spy.FeedbackMessage)
				}
			},
		},
		{
			name:          "/commit execute ResolveTurn and prints log event entries",
			inputCommands: "/commit\n",
			verify: func(t *testing.T, spy *SpyMatchView, masterMatch *engine.Match) {
				if !spy.RenderGameEventsCalled || !spy.FeedbackSuccess {
					t.Error("Expected /commit route to pass events to View and trigger a green status code notice")
				}
			},
		},
		{
			name:          "/surrender execute Surrender and prints log event entries",
			inputCommands: "/surrender\n",
			verify: func(t *testing.T, spy *SpyMatchView, masterMatch *engine.Match) {
				if !spy.FeedbackSuccess || !strings.Contains(spy.FeedbackMessage, "Winner... PLAYER 2!") {
					t.Errorf("Expected final victory status layout sheet, got: %q", spy.FeedbackMessage)
				}
				if masterMatch.WinnerTeamID != 2 {
					t.Errorf("Expected real engine WinnerTeamID to match opponent (2) after Player 1 surrenders, got: %d", masterMatch.WinnerTeamID)
				}
			},
		},
		{
			name:          "Invalid system shortcut",
			inputCommands: "/unknown_action_command\n",
			verify: func(t *testing.T, spy *SpyMatchView, masterMatch *engine.Match) {
				if spy.FeedbackSuccess || !strings.Contains(spy.FeedbackMessage, "Unknown meta command") {
					t.Errorf("Expected unknown command failure warning, got: %v - %q", spy.FeedbackSuccess, spy.FeedbackMessage)
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

			controller := NewMatchController(match, view, strings.NewReader(tt.inputCommands))

			controller.StartInputLoop()

			tt.verify(t, view, match)
		})
	}
}
