package cpu

import "bomb-srpg/engine"

// Decide computes the CPU's plan for the given sandbox state.
// Returns the TurnCommands to apply, in order; an empty result means no action.
func Decide(gs *engine.GameState) []engine.TurnCommand {
      // TODO: implement decision logic - currently always no-op.
      return []engine.TurnCommand{}
}
