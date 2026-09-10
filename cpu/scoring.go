package cpu

import (
	"bomb-srpg/engine"
	"maps"
)

const (
	maxForecastTurn = 5 // Max number of turns to forcast the damage area for the future bomb.
)

// scoreContext holds the parameters stable across a candidate's whole 5-turn forecast.
// Fields that vary per simulated turn live in turnResult instead.
type scoreContext struct {
	unit      *engine.Unit
	allies    []*engine.Unit
	opponents []*engine.Unit
}

// turnResult holds one simulated turn's outcome, rebuilt fresh each forecast iteration.
type turnResult struct {
	turn               int
	blastTiles         map[engine.Coordinate]bool // this turn's AffectedPositions
	cumulatedDeadUnits map[engine.UnitID]bool     // dead Units so far
	distToKingBefore   int                        // unit's reachability to opponent King, before this turn resolved
	distToKingAfter    int                        // unit's reachability to opponent King, after this turn resolved
}

type scoreFactor func(sc scoreContext, tr turnResult) int

// evaluate forecasts the consequence if the Unit take certain actions.
// Returns candidate with score and tags
func evaluate(sc scoreContext, gs *engine.GameState, cmds []engine.TurnCommand) (candidate, error) {
	scratch := gs.DeepCopy()
	if err := applyCandidate(scratch, candidate{turnCommands: cmds}); err != nil {
		return candidate{}, err
	}

	deadUnits := make(map[engine.UnitID]bool)
	factors := scoreFactorsRegistry()
	total := 0
	for t := range maxForecastTurn {
		distBefore := 0
		gameEvents := scratch.ResolveBombExplosionAndDamage()
		distAfter := 0

		blastTiles := make(map[engine.Coordinate]bool)
		for _, evt := range gameEvents {
			switch evt.Type {
			case engine.GameEvtBombExploded:
				for _, pos := range evt.AffectedPositions {
					blastTiles[pos] = true
				}
			case engine.GameEvtUnitDied:
				deadUnits[evt.UnitID] = true
			}
		}

		tr := turnResult{
			turn:               t,
			blastTiles:         blastTiles,
			cumulatedDeadUnits: maps.Clone(deadUnits),
			distToKingBefore:   distBefore,
			distToKingAfter:    distAfter,
		}
		for _, factor := range factors {
			total += factor(sc, tr)
		}
	}

	return candidate{cmds, total, planTag(cmds)}, nil
}

func dist(gs *engine.GameState, unit *engine.Unit, target engine.Coordinate) int {
	return 0
}