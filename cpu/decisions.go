package cpu

import (
	"bomb-srpg/engine"
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

// roster splits the living units into the CPU's allies (team 2) and everyone else.
type roster struct {
	allies       []*engine.Unit
	allyIDs      []engine.UnitID
	opponentIDs  []engine.UnitID
	allyKing     *engine.Unit
	opponentKing *engine.Unit
}

// buildRoster reads positions from gs, so pass the original state, not a sandbox that plans move units in.
func buildRoster(gs *engine.GameState) (roster, bool) {
	var r roster
	for _, u := range gs.Units {
		if u.HP <= 0 {
			continue
		}
		if team, _ := u.ID.Decode(); team == engine.Team2 {
			r.allies = append(r.allies, u)
			r.allyIDs = append(r.allyIDs, u.ID)
			if isKingRole(u) {
				r.allyKing = u
			}
		} else {
			r.opponentIDs = append(r.opponentIDs, u.ID)
			if isKingRole(u) {
				r.opponentKing = u
			}
		}
	}
	return r, r.allyKing != nil && r.opponentKing != nil
}

// bestRoundCandidate returns the highest-scoring non-empty plan across allies, or nil when none has one.
func bestRoundCandidate(r roster, sandbox *engine.GameState) *candidate {
	var best *candidate
	for _, unit := range r.allies {
		sc := scoreContext{
			actorID:        unit.ID,
			allyIDs:        r.allyIDs,
			opponentIDs:    r.opponentIDs,
			allyKingID:     r.allyKing.ID,
			opponentKingID: r.opponentKing.ID,
			actorOrigin:    unit.Position,
		}
		c, err := bestCandidateFor(sc, sandbox)
		if err != nil || len(c.turnCommands) == 0 {
			continue
		}
		if best == nil || c.score > best.score {
			best = &c
		}
	}
	return best
}

// Decide returns the TurnCommands the CPU plays this turn, in order; empty means no action.
func Decide(gs *engine.GameState) []engine.TurnCommand {
	r, ok := buildRoster(gs)
	if !ok {
		return nil
	}

	sandbox := gs.DeepCopy()
	var cmds []engine.TurnCommand
	for range maxAttempts {
		best := bestRoundCandidate(r, sandbox)
		if best == nil {
			break
		}
		if err := applyCandidate(sandbox, *best); err != nil {
			break
		}
		cmds = append(cmds, best.turnCommands...)
	}
	return cmds
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
