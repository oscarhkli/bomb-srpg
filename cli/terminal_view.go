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
// Not every terrain and occupant type is drawn; it is a debugging view.
func (v *TerminalView) RenderBoard(gs *engine.GameState) error {
	if gs == nil {
		return errors.New("cannot render board: GameState pointer is nil")
	}
	if len(gs.Grid) == 0 || len(gs.Grid[0]) == 0 {
		return errors.New("cannot render board: game grid matrix is empty or uninitialized")
	}

	v.renderLegend()
	v.RenderTurnHeader(gs.Turn, gs.ActiveTeam)

	fmt.Fprint(v.output, "Y\\X  ")
	v.renderColumnNumbers(len(gs.Grid))
	fmt.Fprintln(v.output)

	horizontalDivider := "   +" + strings.Repeat("---+", len(gs.Grid))

	for y, row := range gs.Grid {
		fmt.Fprintln(v.output, horizontalDivider)

		fmt.Fprintf(v.output, "%-2d |", y)

		for _, tile := range row {
			cell, err := cellText(gs, tile)
			if err != nil {
				return err
			}
			fmt.Fprintf(v.output, "%-3s|", cell)
		}

		fmt.Fprintf(v.output, " %d\n", y)
	}

	fmt.Fprintln(v.output, horizontalDivider)

	fmt.Fprint(v.output, "     ")
	v.renderColumnNumbers(len(gs.Grid))

	fmt.Fprint(v.output, "\n\n")

	return nil
}

func (v *TerminalView) renderLegend() {
	fmt.Fprintln(v.output, "============== Command Instruction ==============")
	fmt.Fprintf(v.output, "End Turn          : %s/commit%s\n", Yellow, Reset)
	fmt.Fprintf(v.output, "Reset Current Turn: %s/reset%s\n", Yellow, Reset)
	fmt.Fprintf(v.output, "Surrender         : %s/surrender%s\n", Yellow, Reset)
	fmt.Fprintln(v.output, "          ------- How to move -------")
	fmt.Fprintf(v.output, "Move      : %smove <unit-ID> <target-X> <target-Y>%s\n", Yellow, Reset)
	fmt.Fprintf(v.output, "Place Bomb: %sbomb <unit-ID> <target-X> <target-Y>%s\n", Yellow, Reset)
	fmt.Fprintf(v.output, "Example   : %smove 16 4 7%s\n", Yellow, Reset)
	fmt.Fprintln(v.output, "=================================================")
}

func (v *TerminalView) renderColumnNumbers(n int) {
	for x := range n {
		fmt.Fprintf(v.output, " %-3d", x)
	}
}

func cellText(gs *engine.GameState, tile engine.Tile) (string, error) {
	if tile.Type == engine.TerrainBlock {
		return "███", nil
	}

	switch tile.OccupantType {
	case engine.OccupantUnit:
		return fmt.Sprintf("U%d", tile.OccupantID), nil
	case engine.OccupantBomb:
		bomb, ok := gs.Bombs[tile.BombID()]
		if !ok {
			return "", fmt.Errorf("bomb %#x not found", tile.OccupantID)
		}
		return fmt.Sprintf("B-%d", bomb.Countdown), nil
	case engine.OccupantSoftBlock:
		return "SBK", nil
	case engine.OccupantNone, engine.OccupantItem:
	}

	return "   ", nil
}

// RenderGameConfig prints the match configuration options to the console.
func (v *TerminalView) RenderGameConfig(cfg *engine.GameCfg) error {
	if cfg == nil {
		return errors.New("cannot render game config: GameCfg pointer is nil")
	}

	fmt.Fprintln(v.output, "=== MATCH CONFIGURATION ===")
	fmt.Fprintf(v.output, "Stage Preset: %s\n", cfg.StagePreset)
	fmt.Fprintf(v.output, "Max Turns:    %d\n", cfg.MaxTurns)
	fmt.Fprintln(v.output, "===========================")

	return nil
}

func (v *TerminalView) RenderMessage(message string) {
	fmt.Fprintf(v.output, "%s", message)
}

func (v *TerminalView) RenderFeedback(success bool, message string) {
	if success {
		fmt.Fprintf(v.output, "%s%s%s\n", Green, message, Reset)
	} else {
		fmt.Fprintf(v.output, "%s%s%s\n", Red, message, Reset)
	}
}

func (v *TerminalView) RenderTurnHeader(turn, activeTeamID int) {
	fmt.Fprintf(v.output, "\n--- TURN %d PLAYER %d ---\n\n", turn, activeTeamID)
}

func (v *TerminalView) RenderGameEvents(gameEvents []engine.GameEvent) {
	for _, evt := range gameEvents {
		v.RenderMessage(fmt.Sprintf("%#v\n", evt))
	}
}
