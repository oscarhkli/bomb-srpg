package cpu

import (
	"bomb-srpg/engine"
	"slices"
)

const (
	maxForecastTurn = 5
)

// scoreContext identifies the units a candidate's forecast scores against.
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
	turn                  int // the nth next turn, not the exact turn number
	affectedTiles         map[engine.Coordinate]struct{}
	allyAffectedTiles     map[engine.Coordinate]struct{} // tiles hit by an ally-owned bomb
	opponentAffectedTiles map[engine.Coordinate]struct{} // tiles hit by an opponent-owned bomb
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

// unitSnapshot is the actor-relative state evaluate samples before and after a turn resolves.
type unitSnapshot struct {
	distToKing     int
	aliveOpponents int
	aliveAllies    int
}

func snapshotUnits(gs *engine.GameState, sc scoreContext, actor, opponentKing *engine.Unit) unitSnapshot {
	return unitSnapshot{
		distToKing:     reachDistToUnit(gs, actor, actor.Position, opponentKing),
		aliveOpponents: aliveCount(gs, sc.opponentIDs, sc.opponentKingID),
		aliveAllies:    aliveCount(gs, sc.allyIDs, sc.allyKingID),
	}
}

// buildTurnResult folds one turn's events and its before/after snapshots into a turnResult.
func buildTurnResult(sc scoreContext, turn int, gameEvents []engine.GameEvent, before, after unitSnapshot) turnResult {
	tr := turnResult{
		turn:                  turn,
		affectedTiles:         make(map[engine.Coordinate]struct{}),
		allyAffectedTiles:     make(map[engine.Coordinate]struct{}),
		opponentAffectedTiles: make(map[engine.Coordinate]struct{}),
		diedUnits:             make(map[engine.UnitID]struct{}),
		suicides:              make(map[engine.UnitID]struct{}),
		distToKingBefore:      before.distToKing,
		distToKingAfter:       after.distToKing,
		aliveOpponentsBefore:  before.aliveOpponents,
		aliveOpponentsAfter:   after.aliveOpponents,
		aliveAlliesBefore:     before.aliveAllies,
		aliveAlliesAfter:      after.aliveAllies,
	}
	for _, evt := range gameEvents {
		tr.fold(sc, evt)
	}
	return tr
}

func (tr *turnResult) fold(sc scoreContext, evt engine.GameEvent) {
	switch evt.Type {
	case engine.GameEvtBombExploded:
		_, _, ownerID := evt.BombID.Decode()
		dest := tr.opponentAffectedTiles
		if slices.Contains(sc.allyIDs, ownerID) {
			dest = tr.allyAffectedTiles
		}
		for _, pos := range evt.AffectedPositions {
			tr.affectedTiles[pos] = struct{}{}
			dest[pos] = struct{}{}
		}
	case engine.GameEvtSoftBlockDestroyed:
		tr.destroyedSoftBlocks++
	case engine.GameEvtUnitDied:
		tr.diedUnits[evt.UnitID] = struct{}{}
		isAlly := evt.UnitID == sc.allyKingID || slices.Contains(sc.allyIDs, evt.UnitID)
		if isAlly && evt.Position != nil {
			if _, ok := tr.allyAffectedTiles[*evt.Position]; ok {
				tr.suicides[evt.UnitID] = struct{}{}
			}
		}
	case engine.GameEvtUnitMoved, engine.GameEvtUnitDamaged, engine.GameEvtBombPlaced,
		engine.GameEvtBombCountdownUpdated, engine.GameEvtMatchEnded:
	}
}

// evaluate scores the plan by forecasting maxForecastTurn turns of bomb resolution.
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

		before := snapshotUnits(scratch, sc, actor, opponentKing)
		gameEvents := scratch.ResolveBombExplosionAndDamage()
		after := snapshotUnits(scratch, sc, actor, opponentKing)

		tr := buildTurnResult(sc, t, gameEvents, before, after)
		for _, entry := range factors {
			total += profile[entry.id] * entry.factor(scratch, sc, tr)
		}
	}

	return candidate{cmds, total, planTag(cmds)}, nil
}
