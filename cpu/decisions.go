package cpu

import (
	"bomb-srpg/engine"
	"fmt"
	"slices"
)

const (
	maxAttempts = 15 // Max attempt for candidates selection. Defensive way to prevent from infinite loop.

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

	for range maxAttempts {
		var best *candidate
		for _, unit := range allies {
			sc := scoreContext{unit: unit, allies: allies, opponents: opponents}
			c, err := bestCandidateFor(sc, sandbox)
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

func bestCandidateFor(sc scoreContext, gs *engine.GameState) (candidate, error) {
	var candidates []candidate
	for _, p := range plansFor(sc.unit, gs) {
		c, err := evaluate(sc, gs, p)
		if err != nil {
			return candidate{}, err
		}
		candidates = append(candidates, c)
	}
	return slices.MaxFunc(candidates, func(a, b candidate) int {
		return b.score - a.score
	}), nil
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
