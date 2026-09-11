package cpu

import (
	"bomb-srpg/engine"
	"fmt"
	"slices"
	"strings"
)

// plansFor gathers all the possible actions, currently they should cover idle/move/bomb/move+bomb/bomb+move.
// Returns all possible plans (slice of slice of TurnCommand) a Unit can take.
func plansFor(unit *engine.Unit, gs *engine.GameState) [][]engine.TurnCommand {
	movePlans := movePlansFor(unit, gs)
	placeBombPlans := placeBombPlansFor(unit, gs)

	plans := [][]engine.TurnCommand{{}} // Idle is an option
	plans = append(plans, movePlans...)
	plans = append(plans, placeBombPlans...)
	plans = append(plans, combinePlans(unit, gs, movePlans, placeBombPlansFor)...)
	plans = append(plans, combinePlans(unit, gs, placeBombPlans, movePlansFor)...)

	return plans
}

// combinePlans pairs each of firstPlans with every plan second produces once firstPlans has been applied.
// Return all possible plans under the specific combinations.
func combinePlans(
	unit *engine.Unit,
	gs *engine.GameState,
	firstPlans [][]engine.TurnCommand,
	secondAction func(*engine.Unit, *engine.GameState) [][]engine.TurnCommand,
) [][]engine.TurnCommand {
	var plans [][]engine.TurnCommand
	for _, first := range firstPlans {
		scratch := gs.DeepCopy()
		if err := applyCandidate(scratch, candidate{turnCommands: first}); err != nil {
			continue
		}
		for _, next := range secondAction(unit, scratch) {
			plans = append(plans, slices.Concat(first, next))
		}
	}
	return plans
}

func movePlansFor(unit *engine.Unit, gs *engine.GameState) [][]engine.TurnCommand {
	if unit.HasMoved {
		return nil
	}
	return singleCommandPlans(unit, gs, engine.TurnCmdMove, engine.NewMoveCommand)
}

func placeBombPlansFor(unit *engine.Unit, gs *engine.GameState) [][]engine.TurnCommand {
	if unit.HasUsedSkill || unit.BombUsed >= unit.MaxBombCount {
		return nil
	}
	return singleCommandPlans(unit, gs, engine.TurnCmdPlaceBomb, engine.NewPlaceBombCommand)
}

func singleCommandPlans(unit *engine.Unit, gs *engine.GameState, turnCmdType engine.TurnCmdType, newCmd func(engine.UnitID, engine.Coordinate) engine.TurnCommand) [][]engine.TurnCommand {
	var plans [][]engine.TurnCommand

	allowedTiles, err := gs.FindAllowedTilesForCommand(unit.ID, turnCmdType)
	if err != nil {
		// Should never happen; return no plans rather than propagate.
		return plans
	}

	for pos := range allowedTiles {
		plans = append(plans, []engine.TurnCommand{newCmd(unit.ID, pos)})
	}

	return plans
}

func planTag(plan []engine.TurnCommand) string {
	if len(plan) == 0 {
		return "Idle"
	}

	parts := make([]string, len(plan))
	for i, cmd := range plan {
		parts[i] = fmt.Sprintf("%s(%d,%d)", cmd.Type, cmd.Target.X, cmd.Target.Y)
	}
	return strings.Join(parts, "+")
}
