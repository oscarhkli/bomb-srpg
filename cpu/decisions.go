package cpu

import (
	"bomb-srpg/engine"
	"fmt"
	"slices"
	"strings"
)

const (
	maxRounds       = 15 // Max attempt for candidates selection. Defensive way to prevent from infinite loop.
	maxForecastTurn = 5  // Max number of turns to forcast the damage area for the future bomb.

	IdleScore = 0 // Neutral score.
)

// candidate represents a possible action a Unit can take and what it scores.
type candidate struct {
	turnCommands []engine.TurnCommand // Multiple TurnCommands can be done in each Turn.
	score        int                  // The higher the better.
	tag          string               // Debug/log label.
}

// Decide computes the CPU's plan for the given sandbox state.
// Returns the TurnCommands to apply, in order; an empty result means no action.
func Decide(gs *engine.GameState) []engine.TurnCommand {
	sandbox := gs.DeepCopy()
	cmds := []engine.TurnCommand{}

	allies := []*engine.Unit{}
	opponents := []*engine.Unit{}
	for _, u := range gs.Units {
		if u.HP <= 0 {
			continue
		}
		if team, _ := u.ID.Decode(); team == 2 {
			allies = append(allies, u)
		} else {
			opponents = append(opponents, u)
		}
	}

	for range maxRounds {
		var best *candidate
		for _, unit := range allies {
			c, err := bestCandidateFor(unit, sandbox, allies, opponents)
			if err != nil {
				// Sandbox hasn't been mutated yet, so if unexpected error occurs,
				// skip this unit from the current round of candidate selection.
				continue
			}
			if best == nil || c.score > best.score {
				best = &c
			}
		}

		if best == nil || best.score <= IdleScore {
			break
		}

		if err := applyCandidate(sandbox, *best); err != nil {
			// Sandbox may now be partially mutated by best's own earlier commands this round.
			// Stop here and return only the plan confirmed by prior, fully-applied rounds.
			break
		}

		cmds = append(cmds, best.turnCommands...)
	}

	return cmds
}

func bestCandidateFor(unit *engine.Unit, gs *engine.GameState, allies []*engine.Unit, opponents []*engine.Unit) (candidate, error) {
	var candidates []candidate
	for _, p := range plansFor(unit, gs) {
		c, err := evaluate(unit, gs, allies, opponents, p)
		if err != nil {
			return candidate{}, err
		}
		candidates = append(candidates, c)
	}
	return slices.MaxFunc(candidates, func(a, b candidate) int {
		return b.score - a.score
	}), nil
}

// plansFor gathers all the possible actions, currently they should cover idle/move/bomb/move+bomb/bomb+move.
// Refactor to include allies and opponents when working with Skills in future.
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
		// Should never happen unless there are coding issues. Though it shouldn't affect much on the game play. Just shortcut it.
		return plans
	}

	for pos := range allowedTiles {
		plans = append(plans, []engine.TurnCommand{newCmd(unit.ID, pos)})
	}

	return plans
}

// evaluate forecasts the consequence if the Unit take certain actions.
// Returns candidate with score and tags
func evaluate(unit *engine.Unit, gs *engine.GameState, allies []*engine.Unit, opponents []*engine.Unit, cmds []engine.TurnCommand) (candidate, error) {
	scratch := gs.DeepCopy()
	if err := applyCandidate(scratch, candidate{turnCommands: cmds}); err != nil {
		return candidate{}, err
	}
	tag := planTag(cmds)
	// TODO:
	// 3. Do 5 rounds of ResolveBombExplosionAndDamage
	// 4. For each round, capture the AffectedPos, calculate various scores based on the AffectedPos
	// Calculation (with some non-linear weight for dist and turn):
	// Distance of Opponent King and nearest Blast tile triggered in how many Turns distOpponentKing(dist, turn)
	// Sum of Distance of Opponent and Blast tile triggered in how many Turns distOpponent(unit, dist, turn)
	// Distance of Ally King and Blast tile triggered in how many Turns distAllyKing(dist, turn)
	// Sum of Distance of Ally and Blast tile triggered in how many Turns distAlly(unit, dist, turn)
	// Distance between (Current?) Ally and nearest item distItem(unit, dist) - Postpone it for later Phase
	// Distance between Current Ally and Opponent King - distSoftBlockClearForOpponentKing(dist, turn)
	// distOpponentKing(dist, turn) + weighted sum(distOpponent(unit, dist, turn)) - distAllyKing(dist, turn) - weighted sum(distAlly(unit, dist, turn)) + distItem(unit, dist) + softblockClear(unit, dist)

	return candidate{cmds, IdleScore, tag}, nil
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

func applyCandidate(gs *engine.GameState, c candidate) error {
	for _, cmd := range c.turnCommands {
		var err error
		switch cmd.Type {
		case engine.TurnCmdMove:
			_, err = gs.MoveUnit(cmd.UnitID, cmd.Target)
		case engine.TurnCmdPlaceBomb:
			_, err = gs.PlaceBomb(cmd.UnitID, cmd.Target)
		default:
			err = fmt.Errorf("%w: %s", engine.ErrUnsupportedCommand, cmd.Type)
		}
		if err != nil {
			return err
		}
	}

	return nil
}
