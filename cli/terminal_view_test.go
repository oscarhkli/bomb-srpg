package cli

import (
	"bomb-srpg/engine"
	"bytes"
	"strings"
	"testing"
)

func TestTerminalView_RenderBoard(t *testing.T) {
	t.Run("Successfully renders terrain an unit", func(t *testing.T) {
		var fakeScreen bytes.Buffer

		view := NewTerminalView(&fakeScreen)

		gs := newTestGameState(3, 3)
		gs.Turn = 101
		gs.ActiveTeam = 1
		u1 := engine.NewUnitID(1, 0)
		u2 := engine.NewUnitID(2, 0)
		bID := engine.NewBombID(100, 1, u1)
		gs.Units[u1] = &engine.Unit{ID: u1, Team: 1, HP: 1, Position: engine.Coordinate{X: 0, Y: 0}}
		gs.Units[u2] = &engine.Unit{ID: u2, Team: 2, HP: 1, Position: engine.Coordinate{X: 1, Y: 0}}
		gs.Bombs[bID] = &engine.Bomb{ID: bID, OwnerID: u1, Countdown: 5, Position: engine.Coordinate{X: 2, Y: 1}}
		gs.Grid[0][0].OccupantType = engine.OccupantUnit
		gs.Grid[0][0].OccupantID = int64(u1)
		gs.Grid[0][1].OccupantType = engine.OccupantUnit
		gs.Grid[0][1].OccupantID = int64(u2)
		gs.Grid[1][1].Type = engine.TerrainBlock
		gs.Grid[1][2].OccupantType = engine.OccupantBomb
		gs.Grid[1][2].OccupantID = int64(bID)
		gs.Grid[2][0].OccupantType = engine.OccupantSoftBlock
		gs.Grid[2][2].OccupantType = engine.OccupantItem

		err := view.RenderBoard(gs)

		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}

		outputString := fakeScreen.String()

		if !strings.Contains(outputString, "TURN 101") {
			t.Errorf("Expected output to contain header 'TURN 101', got:\n%s", outputString)
		}

		if !strings.Contains(outputString, "PLAYER 1") {
			t.Errorf("Expected output to contain header 'PLAYER 1', got:\n%s", outputString)
		}

		if !strings.Contains(outputString, "U16") {
			t.Errorf("Expected output to contain Unit glyph 'U16', got:\n%s", outputString)
		}

		if !strings.Contains(outputString, "U32") {
			t.Errorf("Expected output to contain Unit glyph 'U32', got:\n%s", outputString)
		}

		if !strings.Contains(outputString, "B-5") {
			t.Errorf("Expected output to contain Bomb glyph 'B-5', got:\n%s", outputString)
		}

		if !strings.Contains(outputString, "SBK") {
			t.Errorf("Expected output to contain Soft Block glyph 'SBK', got:\n%s", outputString)
		}

		if !strings.Contains(outputString, "█") {
			t.Errorf("Expected output to contain Block glyph '█', got:\n%s", outputString)
		}
	})
}

func TestTerminalView_RenderBoard_Errors(t *testing.T) {
	t.Run("returns error on nil state matrix data", func(t *testing.T) {
		var fakeScreen bytes.Buffer
		view := NewTerminalView(&fakeScreen)

		err := view.RenderBoard(nil)
		if err == nil {
			t.Error("Expected an error when passing a nil GameState pointer, but got nil")
		}
	})

	t.Run("returns error on empty uninitialized grid rows", func(t *testing.T) {
		var fakeScreen bytes.Buffer
		view := NewTerminalView(&fakeScreen)

		gs := &engine.GameState{
			Grid: [][]engine.Tile{}, // Completely empty matrix grid array
		}

		err := view.RenderBoard(gs)
		if err == nil {
			t.Error("Expected an error when passing an empty map grid structure, but got nil")
		}
	})
}

// newTestGameState generates a clean slate grid environment
func newTestGameState(width, height int) *engine.GameState {
	grid := make([][]engine.Tile, height)
	for y, row := range grid {
		grid[y] = make([]engine.Tile, width)
		for x := range row {
			grid[y][x] = engine.Tile{Type: engine.TerrainPlain, OccupantType: engine.OccupantNone}
		}
	}

	gs := &engine.GameState{
		Turn:            1,
		TurnBombCounter: 0,
		Grid:            grid,
		Units:           make(map[engine.UnitID]*engine.Unit),
		Bombs:           make(map[engine.BombID]*engine.Bomb),
		SoftBlocks:      make(map[int]*engine.SoftBlock),
	}

	return gs
}

func TestTerminalView_RenderGameConfig(t *testing.T) {
	t.Run("renders game configurations cleanly", func(t *testing.T) {
		var fakeScreen bytes.Buffer

		view := NewTerminalView(&fakeScreen)

		cfg := &engine.GameCfg{
			StagePreset: "Plain",
			MaxTurns:    50,
			SuddenDeath: true,
		}

		err := view.RenderGameConfig(cfg)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		result := fakeScreen.String()

		if !strings.Contains(result, "Stage Preset: Plain") {
			t.Errorf("Expected output to mention stage preset, got:\n%s", result)
		}
		if !strings.Contains(result, "Max Turns:    50") {
			t.Errorf("Expected output to mention turn limit, got:\n%s", result)
		}
		if !strings.Contains(result, "Sudden Death: true") {
			t.Errorf("Expected output to show sudden death flag, got:\n%s", result)
		}
	})

	t.Run("returns error on nil config data", func(t *testing.T) {
		var fakeScreen bytes.Buffer
		view := NewTerminalView(&fakeScreen)

		err := view.RenderGameConfig(nil)
		if err == nil {
			t.Error("Expected an error when passing a nil configuration pointer, but got nil")
		}
	})
}

func TestRenderMessage(t *testing.T) {
	t.Run("renders success message", func(t *testing.T) {
		var fakeScreen bytes.Buffer
		view := NewTerminalView(&fakeScreen)

		view.RenderMessage("msg")

		result := fakeScreen.String()
		expected := "msg"

		if result != expected {
			t.Errorf("Expected exact output %q, got %q", expected, result)
		}
	})
}

func TestRenderFeedback(t *testing.T) {
	t.Run("renders success message in green with ANSI escape codes", func(t *testing.T) {
		var fakeScreen bytes.Buffer
		view := NewTerminalView(&fakeScreen)

		view.RenderFeedback(true, "success msg")

		result := fakeScreen.String()
		expected := Green + "success msg" + Reset + "\n"

		if result != expected {
			t.Errorf("Expected exact ANSI output %q, got %q", expected, result)
		}
	})

	t.Run("renders failure message in red with ANSI escape codes", func(t *testing.T) {
		var fakeScreen bytes.Buffer
		view := NewTerminalView(&fakeScreen)

		view.RenderFeedback(false, "failed msg")

		result := fakeScreen.String()
		expected := Red + "failed msg" + Reset + "\n"

		if result != expected {
			t.Errorf("Expected exact ANSI output %q, got %q", expected, result)
		}
	})
}

func TestTerminalView_RenderTurnHeader(t *testing.T) {
	t.Run("renders turn clock and active team text accurately", func(t *testing.T) {
		var fakeScreen bytes.Buffer
		view := NewTerminalView(&fakeScreen)

		err := view.RenderTurnHeader(101, 2)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		result := fakeScreen.String()

		if !strings.Contains(result, "TURN 101") {
			t.Errorf("Expected output to contain turn count 'TURN 101', got:\n%s", result)
		}
		if !strings.Contains(result, "PLAYER 2") {
			t.Errorf("Expected output to announce active player 'PLAYER 2', got:\n%s", result)
		}
	})
}

func TestTerminalView_RenderGameEvents(t *testing.T) {
	t.Run("renders various GameEvents", func(t *testing.T) {
		var fakeScreen bytes.Buffer
		view := NewTerminalView(&fakeScreen)

		events := []engine.GameEvent{
			engine.NewUnitMovedEvent(16, engine.Coordinate{X: 1, Y: 2}, engine.Coordinate{X: 2, Y: 2}),
			engine.NewMatchEndedEvent(1),
		}

		err := view.RenderGameEvents(events)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		result := fakeScreen.String()

		if !strings.Contains(result, "unitMoved") {
			t.Errorf("Expected output to contain unitMoved, got:\n%s", result)
		}
		if !strings.Contains(result, "matchEnded") {
			t.Errorf("Expected output to contain matchEnded, got:\n%s", result)
		}
	})
}
