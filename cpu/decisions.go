package cpu

import (
	"bomb-srpg/engine"
	"fmt"
)

const (
	maxRounds       = 15 // Max attempt for candidates selection. Defensive way to prevent from infinite loop.
	maxForecastTurn = 5  // Max number of turns to forcast the damage area for the future bomb.

	IdleScore = 0 // Neutral score.
)

// Candidate represent a possible action a Unit can take and what it scores.
type Candidate struct {
	TurnCommands []engine.TurnCommand // Multiple TurnCommands can be done in each Turn.
	Score        int                  // The higher the better.
	Tag          string               // Debug/log label.
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
		var best *Candidate
		for _, unit := range allies {
			candidate, err := bestCandidateFor(unit, sandbox, allies, opponents)
			if err != nil {
				// Sandbox hasn't been mutated yet, so if unexpected error occurs,
				// skip this unit from the current round of Candidate selection.
				continue
			}
			if best == nil || candidate.Score > best.Score {
				best = &candidate
			}
		}

		if best == nil || best.Score <= IdleScore {
			break
		}

		err := applyCandidate(sandbox, *best)
		if err != nil {
			// Sandbox may now be partially mutated by best's own earlier commands this round.
			// Stop here and return only the plan confirmed by prior, fully-applied rounds.
			break
		}

		cmds = append(cmds, best.TurnCommands...)
	}

	return cmds
}

func bestCandidateFor(unit *engine.Unit, gs *engine.GameState, allies []*engine.Unit, opponents []*engine.Unit) (Candidate, error) {
	// TODO:
	// 1. Gather all the possible actions, idle/move/bomb/move+bomb/bomb+move
	// 2. Evaluate all the result and get the score
	// 3. Return the Candidate with the best score
	return Candidate{[]engine.TurnCommand{}, IdleScore, "Idle"}, nil
}

func evaluate(unit *engine.Unit, gs *engine.GameState, allies []*engine.Unit, opponents []*engine.Unit, cmds []engine.TurnCommand) (Candidate, error) {
	scratch := gs.DeepCopy()
	err := applyCandidate(scratch, Candidate{TurnCommands: cmds})
	if err != nil {
		return Candidate{}, err
	}
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

	return Candidate{[]engine.TurnCommand{}, IdleScore, "Idle"}, nil
}

func applyCandidate(gs *engine.GameState, candidate Candidate) error {
	for _, cmd := range candidate.TurnCommands {
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
