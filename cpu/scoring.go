package cpu

import (
	"bomb-srpg/engine"
	"slices"
)

const (
	maxForecastTurn = 5 // Max number of turns to forcast the damage area for the future bomb.
)

// scoreContext identifies the units a candidate's forecast scores against.
// Its fields are fixed for the duration of one evaluate call.
type scoreContext struct {
	actorID        engine.UnitID
	allyIDs        []engine.UnitID
	opponentIDs    []engine.UnitID
	allyKingID     engine.UnitID
	opponentKingID engine.UnitID
	actorOrigin    engine.Coordinate
}

// turnResult holds one simulated turn's outcome, rebuilt fresh each forecast iteration.
type turnResult struct {
	turn                  int                            // the nth next turn, not the exact turn number
	affectedTiles         map[engine.Coordinate]struct{} // this turn's AffectedPositions overall
	allyAffectedTiles     map[engine.Coordinate]struct{} // this turn's AffectedPositions from an ally-owned bomb
	opponentAffectedTiles map[engine.Coordinate]struct{} // this turn's AffectedPositions from an opponent-owned bomb
	destroyedSoftBlocks   int                            // this turn's softBlockDestroyed count
	diedUnits             map[engine.UnitID]struct{}     // this turn's dead units
	suicides              map[engine.UnitID]struct{}     // this turn's dead allies killed by an ally-owned bomb
	distToKingBefore      int                            // unit's reachability to opponent King, before this turn resolved
	distToKingAfter       int                            // unit's reachability to opponent King, after this turn resolved
	aliveOpponentsBefore  int                            // opponent non-King units alive, before this turn resolved
	aliveOpponentsAfter   int                            // opponent non-King units alive, after this turn resolved
	aliveAlliesBefore     int                            // ally non-King units alive, before this turn resolved
	aliveAlliesAfter      int                            // ally non-King units alive, after this turn resolved
}

type scoreFactor func(gs *engine.GameState, sc scoreContext, tr turnResult) int

// aliveCount returns how many of ids are alive in gs, excluding King.
func aliveCount(gs *engine.GameState, ids []engine.UnitID, kingID engine.UnitID) int {
	count := 0
	for _, id := range ids {
		if id == kingID {
			continue
		}
		if u, ok := gs.Units[id]; ok && u.HP > 0 {
			count++
		}
	}
	return count
}

// evaluate forecasts the consequence if the Unit take certain actions.
// Returns candidate with score and tags
func evaluate(sc scoreContext, gs *engine.GameState, cmds []engine.TurnCommand) (candidate, error) {
	scratch := gs.DeepCopy()
	if err := applyCandidate(scratch, candidate{turnCommands: cmds}); err != nil {
		return candidate{}, err
	}

	factors := scoreFactorsRegistry()
	profile := defaultWeightProfile()
	total := 0
	for t := range maxForecastTurn {
		actor := scratch.Units[sc.actorID]
		opponentKing := scratch.Units[sc.opponentKingID]

		distToKingBefore := reachDistToUnit(scratch, actor, actor.Position, opponentKing)
		aliveOpponentsBefore := aliveCount(scratch, sc.opponentIDs, sc.opponentKingID)
		aliveAlliesBefore := aliveCount(scratch, sc.allyIDs, sc.allyKingID)
		gameEvents := scratch.ResolveBombExplosionAndDamage()
		distToKingAfter := reachDistToUnit(scratch, actor, actor.Position, opponentKing)
		aliveOpponentsAfter := aliveCount(scratch, sc.opponentIDs, sc.opponentKingID)
		aliveAlliesAfter := aliveCount(scratch, sc.allyIDs, sc.allyKingID)

		affectedTiles := make(map[engine.Coordinate]struct{})
		allyAffectedTiles := make(map[engine.Coordinate]struct{})
		opponentAffectedTiles := make(map[engine.Coordinate]struct{})
		destroyedSoftBlocks := 0
		diedUnits := make(map[engine.UnitID]struct{})
		suicides := make(map[engine.UnitID]struct{})
		for _, evt := range gameEvents {
			switch evt.Type {
			case engine.GameEvtBombExploded:
				_, _, ownerID := evt.BombID.Decode()
				dest := opponentAffectedTiles
				if slices.Contains(sc.allyIDs, ownerID) {
					dest = allyAffectedTiles
				}
				for _, pos := range evt.AffectedPositions {
					affectedTiles[pos] = struct{}{}
					dest[pos] = struct{}{}
				}
			case engine.GameEvtSoftBlockDestroyed:
				destroyedSoftBlocks++
			case engine.GameEvtUnitDied:
				diedUnits[evt.UnitID] = struct{}{}
				isAlly := evt.UnitID == sc.allyKingID || slices.Contains(sc.allyIDs, evt.UnitID)
				if isAlly && evt.Position != nil {
					if _, ok := allyAffectedTiles[*evt.Position]; ok {
						suicides[evt.UnitID] = struct{}{}
					}
				}
			}
		}

		tr := turnResult{
			turn:                  t,
			affectedTiles:         affectedTiles,
			allyAffectedTiles:     allyAffectedTiles,
			opponentAffectedTiles: opponentAffectedTiles,
			destroyedSoftBlocks:   destroyedSoftBlocks,
			diedUnits:             diedUnits,
			suicides:              suicides,
			distToKingBefore:      distToKingBefore,
			distToKingAfter:       distToKingAfter,
			aliveOpponentsBefore:  aliveOpponentsBefore,
			aliveOpponentsAfter:   aliveOpponentsAfter,
			aliveAlliesBefore:     aliveAlliesBefore,
			aliveAlliesAfter:      aliveAlliesAfter,
		}
		for _, entry := range factors {
			total += profile[entry.id] * entry.factor(scratch, sc, tr)
		}
	}

	return candidate{cmds, total, planTag(cmds)}, nil
}
