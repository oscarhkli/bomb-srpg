package cpu

import (
	"bomb-srpg/engine"
	"fmt"
	"slices"
)

const (
	maxAttempts = 15 // Max attempt for candidates selection. Defensive way to prevent from infinite loop.
)

// candidate represents a possible action a Unit can take and what it scores.
type candidate struct {
	turnCommands []engine.TurnCommand // Multiple TurnCommands can be done in each Turn.
	score        int                  // The higher the better.
	tag          string               // Debug/log label.
}

// priority breaks a score tie: move+bomb is worse than other action.
func (c candidate) priority() int {
	if len(c.turnCommands) == 2 && c.turnCommands[0].Type == engine.TurnCmdMove {
		return 0
	}
	return 1
}

// Decide computes the CPU's plan for the given sandbox state.
// Returns the TurnCommands to apply, in order; an empty result means no action.
func Decide(gs *engine.GameState) []engine.TurnCommand {
	sandbox := gs.DeepCopy()
	var cmds []engine.TurnCommand

	var allies []*engine.Unit
	var opponents []*engine.Unit
	var allyKing *engine.Unit
	var opponentKing *engine.Unit
	for _, u := range gs.Units {
		if u.HP <= 0 {
			continue
		}
		if team, _ := u.ID.Decode(); team == 2 {
			allies = append(allies, u)
			if u.Role == engine.RoleKing || u.Role == engine.RoleBoss {
				allyKing = u
			}
		} else {
			opponents = append(opponents, u)
			if u.Role == engine.RoleKing || u.Role == engine.RoleBoss {
				opponentKing = u
			}
		}

	}

	if allyKing == nil || opponentKing == nil {
		return cmds
	}

	allyIDs := make([]engine.UnitID, len(allies))
	for i, u := range allies {
		allyIDs[i] = u.ID
	}
	opponentIDs := make([]engine.UnitID, len(opponents))
	for i, u := range opponents {
		opponentIDs[i] = u.ID
	}

	for range maxAttempts {
		var best *candidate
		for _, unit := range allies {
			sc := scoreContext{
				actorID:        unit.ID,
				allyIDs:        allyIDs,
				opponentIDs:    opponentIDs,
				allyKingID:     allyKing.ID,
				opponentKingID: opponentKing.ID,
				actorOrigin:    unit.Position,
			}
			c, err := bestCandidateFor(sc, sandbox)
			if err != nil {
				// Sandbox is untouched here; skip this unit for the round.
				continue
			}
			if len(c.turnCommands) == 0 {
				continue
			}
			if best == nil || c.score > best.score {
				best = &c
			}
		}

		if best == nil {
			break
		}

		if err := applyCandidate(sandbox, *best); err != nil {
			// Sandbox may be partially mutated by best's own earlier commands; stop here.
			break
		}

		cmds = append(cmds, best.turnCommands...)
	}

	return cmds
}

func bestCandidateFor(sc scoreContext, gs *engine.GameState) (candidate, error) {
	var candidates []candidate
	for _, p := range plansFor(gs.Units[sc.actorID], gs) {
		c, err := evaluate(sc, gs, p)
		if err != nil {
			return candidate{}, err
		}
		candidates = append(candidates, c)
	}
	return slices.MaxFunc(candidates, func(a, b candidate) int {
		if d := a.score - b.score; d != 0 {
			return d
		}
		return a.priority() - b.priority()
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
