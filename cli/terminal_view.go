package cli

import (
	"bomb-srpg/engine"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
)

type TerminalView struct {
	output io.Writer
}

func NewTerminalView(o io.Writer) *TerminalView {
	return &TerminalView{output: o}
}

// RenderBoard prints a 2D ASCII grid representation of the map.
// Since this func is for Phase 1 testing only, some terrains/occupants are skipped.
// Test coverage may suffer.
func (v *TerminalView) RenderBoard(gs *engine.GameState) error {
	if gs == nil {
		return errors.New("cannot render board: GameState pointer is nil")
	}
	if len(gs.Grid) == 0 || len(gs.Grid[0]) == 0 {
		return errors.New("cannot render board: game grid matrix is empty or uninitialized")
	}

	fmt.Fprintln(v.output, "============== Command Instruction ==============")
	fmt.Fprintf(v.output, "End Turn          : %s/commit%s\n", Yellow, Reset)
	fmt.Fprintf(v.output, "Reset Current Turn: %s/reset%s\n", Yellow, Reset)
	fmt.Fprintf(v.output, "Surrender         : %s/surrender%s\n", Yellow, Reset)
	fmt.Fprintln(v.output, "          ------- How to move -------")
	fmt.Fprintf(v.output, "Move      : %smove <unit-ID> <target-X> <target-Y>%s\n", Yellow, Reset)
	fmt.Fprintf(v.output, "Place Bomb: %sbomb <unit-ID> <target-X> <target-Y>%s\n", Yellow, Reset)
	fmt.Fprintf(v.output, "Example   : %smove 16 4 7%s\n", Yellow, Reset)
	fmt.Fprintln(v.output, "=================================================")

	if err := v.RenderTurnHeader(gs.Turn, gs.ActiveTeam); err != nil {
		return err
	}

	// 1. Print top X coordinate header
	fmt.Fprint(v.output, "Y\\X  ") // Space for the left Y column padding
	for x := range gs.Grid {
		fmt.Fprintf(v.output, " %-3d", x)
	}
	fmt.Fprintln(v.output)

	// Create the horizontal divider line (e.g., +---+---+---+)
	horizontalDivider := "   +" + strings.Repeat("---+", len(gs.Grid))

	// 2. Loop through rows (Y-axis)
	for y, row := range gs.Grid {
		// Print the divider line above each row
		fmt.Fprintln(v.output, horizontalDivider)

		// Print left Y coordinate header and start the row border
		fmt.Fprintf(v.output, "%-2d |", y)

		// Loop through columns (X-axis)
		for _, tile := range row {
			cellStr := "   " // Default blank for TerrainPlain / empty

			if tile.Type == engine.TerrainBlock {
				cellStr = "███" // Solid block representation
			} else {
				switch tile.OccupantType {
				case engine.OccupantUnit:
					cellStr = fmt.Sprintf("U%d", tile.OccupantID)
				case engine.OccupantBomb:
					bomb, ok := gs.Bombs[engine.BombID(tile.OccupantID)]
					if !ok {
						return fmt.Errorf("cannot render board due to data corruption: Bomb %#X not found", tile.OccupantID)
					}
					cellStr = fmt.Sprintf("B-%d", bomb.Countdown)
				case engine.OccupantSoftBlock:
					cellStr = "SBK"
				}
			}

			// Format to ensure exactly 3 characters wide inside the cell borders
			fmt.Fprintf(v.output, "%-3s|", cellStr)
		}

		// Print right Y coordinate header
		fmt.Fprintf(v.output, " %d\n", y)
	}

	// Print the very bottom border line
	fmt.Fprintln(v.output, horizontalDivider)

	// 3. Print bottom X coordinate header
	fmt.Fprint(v.output, "     ")
	for x := range gs.Grid {
		fmt.Fprintf(v.output, " %-3d", x)
	}

	fmt.Fprint(v.output, "\n\n")

	return nil
}

// RenderGameConfig prints the match configuration options to the console.
func (v *TerminalView) RenderGameConfig(cfg *engine.GameCfg) error {
	if cfg == nil {
		return errors.New("cannot render game config: GameCfg pointer is nil")
	}

	fmt.Fprintln(v.output, "=== MATCH CONFIGURATION ===")
	fmt.Fprintf(v.output, "Stage Preset: %s\n", cfg.StagePreset)
	fmt.Fprintf(v.output, "Max Turns:    %d\n", cfg.MaxTurns)
	fmt.Fprintf(v.output, "Sudden Death: %t\n", cfg.SuddenDeath)
	fmt.Fprintln(v.output, "===========================")

	return nil
}

func (v *TerminalView) RenderMessage(message string) error {
	fmt.Fprintf(v.output, "%s", message)
	return nil
}

func (v *TerminalView) RenderFeedback(success bool, message string) error {
	if success {
		fmt.Fprintf(v.output, "%s%s%s\n", Green, message, Reset)
	} else {
		fmt.Fprintf(v.output, "%s%s%s\n", Red, message, Reset)
	}
	return nil
}

func (v *TerminalView) RenderTurnHeader(turn, activeTeamID int) error {
	fmt.Fprintf(v.output, "\n--- TURN %d PLAYER %d ---\n\n", turn, activeTeamID)
	return nil
}

func (v *TerminalView) RenderGameEvents(events []engine.GameEvent) error {
	for _, event := range events {
		v.RenderMessage(fmt.Sprintf("%#v\n", event))
	}
	return nil
}
