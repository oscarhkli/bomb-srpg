package cpu

import (
	"bomb-srpg/engine"
	"cmp"
	"errors"
	"fmt"
	"slices"
)

// maxAttempts caps candidate-selection rounds, defending against an infinite loop.
const maxAttempts = 15

// candidate represents a possible action a Unit can take and what it scores.
type candidate struct {
	turnCommands []engine.TurnCommand
	score        int    // The higher the better.
	tag          string // Debug/log label.
}

// priority breaks a score tie; move-then-bomb ranks lowest.
func (c candidate) priority() int {
	if len(c.turnCommands) == 2 && c.turnCommands[0].Type == engine.TurnCmdMove {
		return 0
	}
	return 1
}

// buildRoster reads positions from gs, so pass the original state, not a sandbox that plans move units in.
func buildRoster(gs *engine.GameState) (allies []*engine.Unit, base scoreContext, ok bool) {
	var allyKing, opponentKing *engine.Unit
	for _, u := range gs.Units {
		if u.HP <= 0 {
			continue
		}
		if team, _ := u.ID.Decode(); team == engine.Team2 {
			allies = append(allies, u)
			base.allyIDs = append(base.allyIDs, u.ID)
			if isKingRole(u) {
				allyKing = u
			}
		} else {
			base.opponentIDs = append(base.opponentIDs, u.ID)
			if isKingRole(u) {
				opponentKing = u
			}
		}
	}
	if allyKing == nil || opponentKing == nil {
		return nil, scoreContext{}, false
	}
	base.allyKingID = allyKing.ID
	base.opponentKingID = opponentKing.ID
	return allies, base, true
}

// Decide returns the TurnCommands the CPU plays this turn, in order; empty means no action.
// The commands are valid to apply even when the error is non-nil: it joins the failures of units
// that were skipped, and the plans of the remaining units are still returned.
func Decide(gs *engine.GameState) ([]engine.TurnCommand, error) {
	allies, base, ok := buildRoster(gs)
	if !ok {
		return nil, nil
	}

	sandbox := gs.DeepCopy()
	var cmds []engine.TurnCommand
	var errs []error
	for range maxAttempts {
		var round []candidate
		for _, unit := range allies {
			sc := base
			sc.actorID = unit.ID
			sc.actorOrigin = unit.Position
			c, err := bestCandidateFor(sc, sandbox)
			if err != nil {
				errs = append(errs, fmt.Errorf("unit %#x: %w", unit.ID, err))
				continue
			}
			if len(c.turnCommands) == 0 {
				continue
			}
			round = append(round, c)
		}
		if len(round) == 0 {
			break
		}
		best := slices.MaxFunc(round, func(a, b candidate) int {
			return cmp.Compare(a.score, b.score)
		})
		if err := applyCandidate(sandbox, best); err != nil {
			errs = append(errs, fmt.Errorf("apply best candidate for unit %#x: %w", best.turnCommands[0].UnitID, err))
			break
		}
		cmds = append(cmds, best.turnCommands...)
	}
	return cmds, errors.Join(errs...)
}

func isKingRole(u *engine.Unit) bool {
	return u.Role == engine.RoleKing || u.Role == engine.RoleBoss
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
