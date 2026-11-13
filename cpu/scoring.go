package cpu

import (
	"bomb-srpg/engine"
	"slices"
)

const (
	maxForecastTurn = 5
)

type scoreContext struct {
	actorID        engine.UnitID
	allyIDs        []engine.UnitID
	opponentIDs    []engine.UnitID
	allyKingID     engine.UnitID
	opponentKingID engine.UnitID
	actorOrigin    engine.Coordinate
}

// turnResult holds one simulated turn's outcome.
type turnResult struct {
	turn                  int
	affectedTiles         map[engine.Coordinate]struct{}
	allyAffectedTiles     map[engine.Coordinate]struct{}
	opponentAffectedTiles map[engine.Coordinate]struct{}
	destroyedSoftBlocks   int
	diedUnits             map[engine.UnitID]struct{}
	// this turn's dead allies killed by an ally-owned bomb.
	suicides             map[engine.UnitID]struct{}
	distToKingBefore     int
	distToKingAfter      int
	aliveOpponentsBefore int
	aliveOpponentsAfter  int
	aliveAlliesBefore    int
	aliveAlliesAfter     int
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

		tr := turnResult{
			turn:                  t,
			affectedTiles:         make(map[engine.Coordinate]struct{}),
			allyAffectedTiles:     make(map[engine.Coordinate]struct{}),
			opponentAffectedTiles: make(map[engine.Coordinate]struct{}),
			diedUnits:             make(map[engine.UnitID]struct{}),
			suicides:              make(map[engine.UnitID]struct{}),
			distToKingBefore:      reachDistToUnit(scratch, actor, actor.Position, opponentKing),
			aliveOpponentsBefore:  aliveCount(scratch, sc.opponentIDs, sc.opponentKingID),
			aliveAlliesBefore:     aliveCount(scratch, sc.allyIDs, sc.allyKingID),
		}
		for _, evt := range scratch.ResolveBombExplosionAndDamage() {
			tr.fold(sc, evt)
		}
		tr.distToKingAfter = reachDistToUnit(scratch, actor, actor.Position, opponentKing)
		tr.aliveOpponentsAfter = aliveCount(scratch, sc.opponentIDs, sc.opponentKingID)
		tr.aliveAlliesAfter = aliveCount(scratch, sc.allyIDs, sc.allyKingID)
		for _, entry := range factors {
			total += profile[entry.id] * entry.factor(scratch, sc, tr)
		}
	}

	return candidate{cmds, total, planTag(cmds)}, nil
}
