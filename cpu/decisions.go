package cpu

import "bomb-srpg/engine"

// Decide computes the CPU's plan for the given sandbox state.
// Returns the TurnCommands to apply, in order; an empty result means no action.
func Decide(gs *engine.GameState) []engine.TurnCommand {
	// TODO: implement decision logic - currently always walk if possible (untested)
	cmds := []engine.TurnCommand{}

	livingUnits := []*engine.Unit{}
	livingEnemyUnits := []*engine.Unit{}

	for _, u := range gs.Units {
		if u.HP == 0 {
			continue
		}
		if team, _ := u.ID.Decode(); team == 2 {
			livingUnits = append(livingUnits, u)
		} else {
			livingEnemyUnits = append(livingEnemyUnits, u)
		}

		for _, u := range livingUnits {
			rts := gs.FindReachableTiles(u.Position, u.NewMovementRule())
			if len(rts) == 0 {
				continue
			}
			for k, _ := range rts {
				cmds = append(cmds, engine.NewMoveCommand(u.ID, k))
				break
			}
		}
	}

	return cmds
}
