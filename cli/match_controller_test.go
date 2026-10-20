package cli

import (
	"bytes"
	"strings"
	"testing"

	"bomb-srpg/engine"
)

func TestMatchController_SystemCommands(t *testing.T) {
	type testCase struct {
		name          string
		inputCommands string
		verify        func(t *testing.T, out string, match *engine.Match)
	}

	table := []testCase{
		{
			name:          "Blank input whitespace elements are skipped",
			inputCommands: "\n\n   \n",
			verify: func(t *testing.T, out string, match *engine.Match) {
				t.Helper()
				if strings.Contains(out, Green) || strings.Contains(out, Red) {
					t.Errorf("Expected loop to ignore spaces, but feedback registered: %q", out)
				}
			},
		},
		{
			name:          "/config triggers GameCfg view",
			inputCommands: "/config\n",
			verify: func(t *testing.T, out string, match *engine.Match) {
				t.Helper()
				if !strings.Contains(out, "=== MATCH CONFIGURATION ===") {
					t.Errorf("Expected /config route to render the game config, got: %q", out)
				}
			},
		},
		{
			name:          "/reset executes ResetTurn",
			inputCommands: "/reset\n",
			verify: func(t *testing.T, out string, match *engine.Match) {
				t.Helper()
				if !strings.Contains(out, Green+"Turn reset successfully!") {
					t.Errorf("Expected successful reset confirmation, got: %q", out)
				}
			},
		},
		{
			name:          "/commit execute ResolveTurn and prints log event entries",
			inputCommands: "/commit\n",
			verify: func(t *testing.T, out string, match *engine.Match) {
				t.Helper()
				if !strings.Contains(out, Green+"Turn committed and resolved!"+Reset+"\nengine.GameEvent{Type:\"matchEnded\"") {
					t.Errorf("Expected /commit route to trigger a green status code notice, got: %q", out)
				}
			},
		},
		{
			name:          "/surrender execute Surrender and prints log event entries",
			inputCommands: "/surrender\n",
			verify: func(t *testing.T, out string, match *engine.Match) {
				t.Helper()
				if !strings.Contains(out, Green+"PLAYER 1 surrendered!"+Reset+"\nengine.GameEvent{Type:\"matchEnded\"") || !strings.Contains(out, Green+"Winner... PLAYER 2!") {
					t.Errorf("Expected final victory status layout sheet, got: %q", out)
				}
				if match.WinnerTeamID != 2 {
					t.Errorf("Expected real engine WinnerTeamID to match opponent (2) after Player 1 surrenders, got: %d", match.WinnerTeamID)
				}
			},
		},
		{
			name:          "Invalid system shortcut",
			inputCommands: "/unknown_action_command\n",
			verify: func(t *testing.T, out string, match *engine.Match) {
				t.Helper()
				if !strings.Contains(out, Red+"Unknown meta command") {
					t.Errorf("Expected unknown command failure warning, got: %q", out)
				}
			},
		},
	}

	for _, tt := range table {
		t.Run(tt.name, func(t *testing.T) {
			match := &engine.Match{
				WorkingState: &engine.GameState{
					Turn:       1,
					ActiveTeam: 1,
					Grid:       [][]engine.Tile{{{Type: engine.TerrainPlain}}},
				},
				WinnerTeamID: 0,
			}
			match.TrueState = match.WorkingState.DeepCopy()

			var screen bytes.Buffer
			controller := NewMatchController(match, NewTerminalView(&screen), strings.NewReader(tt.inputCommands))

			controller.StartInputLoop()

			tt.verify(t, screen.String(), match)
		})
	}
}

func TestMatchController_GameplayActions(t *testing.T) {
	type testCase struct {
		name          string
		inputCommands string
		setupMatch    func() *engine.Match
		verify        func(t *testing.T, out string, match *engine.Match)
	}

	table := []testCase{
		{
			name:          "move command parses valid parameters",
			inputCommands: "move 16 1 0\n",
			setupMatch: func() *engine.Match {
				m := newTestMatch()
				unitID := engine.NewUnitID(1, 0)
				m.WorkingState.Units[unitID] = &engine.Unit{
					ID:           unitID,
					Team:         1,
					Position:     engine.Coordinate{X: 0, Y: 0},
					HP:           1,
					Speed:        100,
					BombMaxRange: 100,
				}
				m.WorkingState.Grid[0][0].OccupantType = engine.OccupantUnit
				m.WorkingState.Grid[0][0].OccupantID = int64(unitID)

				return m
			},
			verify: func(t *testing.T, out string, match *engine.Match) {
				t.Helper()
				if !strings.Contains(out, Green+"GameEvents:") {
					t.Errorf("Expected gameplay router to parse verb, but got: %q", out)
				}
			},
		},
		{
			name:          "move command parses valid parameters but invalid movement",
			inputCommands: "move 16 120 0\n",
			setupMatch: func() *engine.Match {
				m := newTestMatch()
				unitID := engine.NewUnitID(1, 0)
				m.WorkingState.Units[unitID] = &engine.Unit{
					ID:           unitID,
					Team:         1,
					Position:     engine.Coordinate{X: 0, Y: 0},
					Speed:        100,
					HP:           1,
					BombMaxRange: 100,
				}
				m.WorkingState.Grid[0][0].OccupantType = engine.OccupantUnit
				m.WorkingState.Grid[0][0].OccupantID = int64(unitID)

				return m
			},
			verify: func(t *testing.T, out string, match *engine.Match) {
				t.Helper()
				if strings.Contains(out, Green+"GameEvents:") {
					t.Error("Expected illegal move coordinates to return a failure status, but reported success")
				}
				if !strings.Contains(out, Red+"Invalid move") {
					t.Errorf("Expected feedback warning parameters to mention 'Invalid move', got: %q", out)
				}
			},
		},
		{
			name:          "move command syntax error",
			inputCommands: "move 16 1\n",
			setupMatch:    newTestMatch,
			verify: func(t *testing.T, out string, match *engine.Match) {
				t.Helper()
				if !strings.Contains(out, Red+"Syntax error in move command") || !strings.Contains(out, "Expected: move") {
					t.Errorf("Expected short arguments length to trigger syntax warning, got: %q", out)
				}
			},
		},
		{
			name:          "move command parameter syntax error",
			inputCommands: "move 16 apple 0\n",
			setupMatch:    newTestMatch,
			verify: func(t *testing.T, out string, match *engine.Match) {
				t.Helper()
				if !strings.Contains(out, Red+"Argument Error") {
					t.Errorf("Expected string parsing validation to capture character errors, got: %q", out)
				}
			},
		},
		{
			name:          "bomb command parses valid parameters",
			inputCommands: "bomb 16 1 0\n",
			setupMatch: func() *engine.Match {
				m := newTestMatch()
				unitID := engine.NewUnitID(1, 0)
				m.WorkingState.Units[unitID] = &engine.Unit{
					ID:           unitID,
					Team:         1,
					Position:     engine.Coordinate{X: 0, Y: 0},
					HP:           1,
					MaxBombCount: 100,
					BombMaxRange: 100,
				}
				m.WorkingState.Grid[0][0].OccupantType = engine.OccupantUnit
				m.WorkingState.Grid[0][0].OccupantID = int64(unitID)

				return m
			},
			verify: func(t *testing.T, out string, match *engine.Match) {
				t.Helper()
				if !strings.Contains(out, Green+"GameEvents:") {
					t.Errorf("Expected gameplay router to parse verb, but got: %q", out)
				}
			},
		},
		{
			name:          "bomb command parses valid parameters but invalid bomb placement",
			inputCommands: "bomb 16 1 0\n",
			setupMatch: func() *engine.Match {
				m := newTestMatch()
				unitID := engine.NewUnitID(1, 0)
				m.WorkingState.Units[unitID] = &engine.Unit{
					ID:           unitID,
					Team:         1,
					Position:     engine.Coordinate{X: 0, Y: 0},
					HP:           1,
					BombMaxRange: 100,
				}
				m.WorkingState.Grid[0][0].OccupantType = engine.OccupantUnit
				m.WorkingState.Grid[0][0].OccupantID = int64(unitID)

				return m
			},
			verify: func(t *testing.T, out string, match *engine.Match) {
				t.Helper()
				if strings.Contains(out, Green+"GameEvents:") {
					t.Error("Expected illegal bomb coordinates to return a failure status, but reported success")
				}
				if !strings.Contains(out, Red+"Invalid bomb placement") || !strings.Contains(out, "unit 0x10 out of bombs") {
					t.Errorf("Expected feedback warning parameters to mention 'unit 0x10 out of bombs', got: %q", out)
				}
			},
		},
		{
			name:          "bomb command syntax error",
			inputCommands: "bomb 16 1\n",
			setupMatch:    newTestMatch,
			verify: func(t *testing.T, out string, match *engine.Match) {
				t.Helper()
				if !strings.Contains(out, Red+"Syntax error in bomb command") || !strings.Contains(out, "Expected: bomb") {
					t.Errorf("Expected short arguments length to trigger syntax warning, got: %q", out)
				}
			},
		},
		{
			name:          "bomb command parameter syntax error",
			inputCommands: "bomb 16 apple 0\n",
			setupMatch:    newTestMatch,
			verify: func(t *testing.T, out string, match *engine.Match) {
				t.Helper()
				if !strings.Contains(out, Red+"Argument Error") {
					t.Errorf("Expected string parsing validation to capture character errors, got: %q", out)
				}
			},
		},
	}

	for _, tt := range table {
		t.Run(tt.name, func(t *testing.T) {
			match := tt.setupMatch()
			var screen bytes.Buffer
			controller := NewMatchController(match, NewTerminalView(&screen), strings.NewReader(tt.inputCommands))

			controller.StartInputLoop()

			tt.verify(t, screen.String(), match)
		})
	}
}

func newTestMatch() *engine.Match {
	const width, height = 2, 2

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

func TestMatchController_StartInputLoop(t *testing.T) {
	table := []struct {
		name    string
		input   string
		winner  int
		want    []string
		notWant []string
		prompts int
	}{
		{
			name:    "draw announces and returns",
			input:   "/reset\n",
			winner:  -1,
			want:    []string{"Draw Game!"},
			notWant: []string{"Enter Command", "reset successfully"},
		},
		{
			name:    "winner announces and returns",
			input:   "/reset\n",
			winner:  2,
			want:    []string{"Winner... PLAYER 2!"},
			notWant: []string{"Enter Command", "reset successfully"},
		},
		{
			name:    "EOF returns after the first prompt",
			input:   "",
			want:    []string{"Enter Command: "},
			prompts: 1,
		},
		{
			name:    "empty line prompts again",
			input:   "\n",
			want:    []string{"Enter Command: "},
			prompts: 2,
			notWant: []string{"Unknown", "Input is empty"},
		},
		{
			name:  "slash command reaches the system handler",
			input: "/reset\n",
			want:  []string{"Turn reset successfully!"},
		},
		{
			name:  "unknown meta command prints feedback",
			input: "/nope\n",
			want:  []string{"Unknown meta command: /nope. Use /config, /reset, /commit, or /surrender"},
		},
		{
			name:  "move with too few tokens",
			input: "move 16 1\n",
			want:  []string{"Expected: move <unit_idx> <x> <y>"},
		},
		{
			name:  "bomb with too few tokens",
			input: "bomb 16 1\n",
			want:  []string{"Expected: bomb <unit_idx> <x> <y>"},
		},
		{
			name:  "non-integer argument",
			input: "move 16 apple 0\n",
			want:  []string{"Argument Error: Unit index, X, and Y must be integers!"},
		},
		{
			name:  "unknown game verb",
			input: "dance\n",
			want:  []string{"Unknown meta command: dance"},
		},
	}

	for _, tt := range table {
		t.Run(tt.name, func(t *testing.T) {
			match := newTestMatch()
			match.WinnerTeamID = tt.winner

			var screen bytes.Buffer
			NewMatchController(match, NewTerminalView(&screen), strings.NewReader(tt.input)).StartInputLoop()

			got := screen.String()
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("output missing %q, got:\n%s", w, got)
				}
			}
			if n := strings.Count(got, "Enter Command: "); tt.prompts > 0 && n != tt.prompts {
				t.Errorf("prompt count = %d, want %d", n, tt.prompts)
			}
			for _, nw := range tt.notWant {
				if strings.Contains(got, nw) {
					t.Errorf("output contains %q, got:\n%s", nw, got)
				}
			}
		})
	}
}
