package cpu

import (
	"bomb-srpg/engine"
	"math"
)

type factorID int

// King also covers Boss: both play the win/loss role.
const (
	factorAdvanceOpponentKingReachability factorID = iota
	factorAdvanceOpponentKingDistance
	factorKillOpponentKing
	factorKillAllyKing
	factorKillOpponents
	factorKillAllies
	factorThreatOpponentKing
	factorRiskAllyKing
	factorThreatOpponents
	factorRiskAllies
	factorSuicideAllyKing
	factorSuicideAllies
)

type scoreFactorEntry struct {
	id     factorID
	factor scoreFactor
}

func scoreFactorsRegistry() []scoreFactorEntry {
	return []scoreFactorEntry{
		{factorAdvanceOpponentKingReachability, advanceOpponentKingReachability},
		{factorAdvanceOpponentKingDistance, advanceOpponentKingDistance},
		{factorKillOpponentKing, killOpponentKing},
		{factorKillAllyKing, killAllyKing},
		{factorKillOpponents, killOpponents},
		{factorKillAllies, killAllies},
		{factorThreatOpponentKing, threatOpponentKing},
		{factorRiskAllyKing, riskAllyKing},
		{factorThreatOpponents, threatOpponents},
		{factorRiskAllies, riskAllies},
		{factorSuicideAllyKing, suicideAllyKing},
		{factorSuicideAllies, suicideAllies},
	}
}

func defaultWeightProfile() map[factorID]int {
	return map[factorID]int{
		factorAdvanceOpponentKingReachability: 1,
		factorAdvanceOpponentKingDistance:     1,
		factorKillOpponentKing:                1,
		factorKillAllyKing:                    -1,
		factorKillOpponents:                   1,
		factorKillAllies:                      -1,
		factorThreatOpponentKing:              1,
		factorRiskAllyKing:                    -1,
		factorThreatOpponents:                 1,
		factorRiskAllies:                      -1,
		factorSuicideAllyKing:                 -1,
		factorSuicideAllies:                   -1,
	}
}

// Score magnitudes, tiered so a certain outcome always outranks a predicted one.
const (
	killKingScore         = 100000 // Very high score to make a result guaranteed.
	killUnitScore         = 10000  // Lower than killKingScore: a predicted, not guaranteed, outcome.
	riskOpponentKingScore = 5000
	riskOpponentScore     = 1000
	riskAllyKingScore     = 3000
	riskAllyScore         = 600
)

func distanceDeltaScore(before, after, val int) int {
	if before == after {
		return 0
	}
	if after == -1 {
		return -val
	}
	if before == -1 {
		return val
	}
	if before > after {
		return val
	}
	return -val
}

func advanceOpponentKingReachability(gs *engine.GameState, sc scoreContext, tr turnResult) int {
	if tr.destroyedSoftBlocks == 0 {
		return 0
	}
	actor := gs.Units[sc.actorID]
	if actor.HP <= 0 {
		return 0
	}
	king := gs.Units[sc.opponentKingID]
	if king.HP <= 0 {
		return 0
	}

	return distanceDeltaScore(tr.distToKingBefore, tr.distToKingAfter, maxForecastTurn-tr.turn)
}

// advanceOpponentKingDistance counts only T+0, the only turn a move can occur before the forecast loop starts.
func advanceOpponentKingDistance(gs *engine.GameState, sc scoreContext, tr turnResult) int {
	if tr.turn > 0 {
		return 0
	}
	actor := gs.Units[sc.actorID]
	if actor.HP <= 0 || actor.Position == sc.actorOrigin {
		return 0
	}
	king := gs.Units[sc.opponentKingID]
	if king.HP <= 0 {
		return 0
	}

	before := reachDistToUnit(gs, actor, sc.actorOrigin, king)
	after := reachDistToUnit(gs, actor, actor.Position, king)
	return distanceDeltaScore(before, after, maxForecastTurn-tr.turn)
}

func killOpponentKing(_ *engine.GameState, sc scoreContext, tr turnResult) int {
	if tr.turn > 0 {
		return 0
	}
	if _, ok := tr.diedUnits[sc.opponentKingID]; ok {
		return killKingScore
	}
	return 0
}

// killAllyKing counts T+0 and T+1 because the ally team cannot act at T+1.
func killAllyKing(_ *engine.GameState, sc scoreContext, tr turnResult) int {
	if tr.turn > 1 {
		return 0
	}
	if _, ok := tr.diedUnits[sc.allyKingID]; ok {
		return killKingScore
	}
	return 0
}

func killOpponents(_ *engine.GameState, _ scoreContext, tr turnResult) int {
	if tr.turn > 0 || tr.aliveOpponentsBefore == 0 {
		return 0
	}
	killed := tr.aliveOpponentsBefore - tr.aliveOpponentsAfter
	return killUnitScore * killed / tr.aliveOpponentsBefore
}

// killAllies counts T+0 and T+1 because the ally team cannot act at T+1.
func killAllies(_ *engine.GameState, _ scoreContext, tr turnResult) int {
	if tr.turn > 1 || tr.aliveAlliesBefore == 0 {
		return 0
	}
	killed := tr.aliveAlliesBefore - tr.aliveAlliesAfter
	return killUnitScore * killed / tr.aliveAlliesBefore
}

// suicideAllyKing fires only at tr.turn>1: killAllyKing already covers earlier turns for any cause of death.
// This must not double-count the same death.
func suicideAllyKing(_ *engine.GameState, sc scoreContext, tr turnResult) int {
	if tr.turn <= 1 {
		return 0
	}
	if _, ok := tr.suicides[sc.allyKingID]; ok {
		return killKingScore
	}
	return 0
}

// suicideAllies fires only at tr.turn>1, for the same reason as suicideAllyKing.
func suicideAllies(_ *engine.GameState, sc scoreContext, tr turnResult) int {
	total := nonKingCount(sc.allyIDs, sc.allyKingID)
	if tr.turn <= 1 || total == 0 {
		return 0
	}
	killed := 0
	for _, id := range sc.allyIDs {
		if id == sc.allyKingID {
			continue
		}
		if _, ok := tr.suicides[id]; ok {
			killed++
		}
	}
	return killUnitScore * killed / total
}

func escapable(gs *engine.GameState, unit *engine.Unit, affectedTiles map[engine.Coordinate]struct{}) bool {
	tiles, err := gs.FindAllowedTilesForCommand(unit.ID, engine.TurnCmdMove)
	if err != nil {
		// exposureIndex guarantees a live unit, so treat an unexpected error as escapable.
		return true
	}

	for tile := range tiles {
		if _, ok := affectedTiles[tile]; !ok {
			return true
		}
	}

	return false
}

// decayRatio returns 1 at x = 0, decaying to 0 at x = t. k > 0 shapes the plateau-then-cliff curve.
func decayRatio(x, t int, k float64) float64 {
	if x >= t {
		return 0
	}
	kt := k * float64(t)
	return (math.Exp(kt) - math.Exp(k*float64(x))) / (math.Exp(kt) - 1)
}

const (
	riskFreeDist = 6 // distance beyond which threat is considered negligible
	kDistThreat  = 4.0 / riskFreeDist
	kDistRisk    = 1.0 / riskFreeDist
	riskFreeTurn = 5 // turns beyond which a bomb's fuse is considered escapable
	kTurn        = 1.0 / riskFreeTurn
)

// exposureIndex returns a unit's exposure to the nearest tile in affectedTiles, as a 0..1 ratio.
// turnOffset is how many turns of this unit's certain-kill window have already elapsed.
// k is the distance-axis decayRatio constant; offense and defense use different curves.
func exposureIndex(gs *engine.GameState, unitID engine.UnitID, affectedTiles map[engine.Coordinate]struct{}, tr turnResult, turnOffset int, k float64) float64 {
	if len(affectedTiles) == 0 {
		return 0
	}

	unit := gs.Units[unitID]

	dist := nearestAffectedDist(gs, unit, affectedTiles)
	if dist == -1 {
		return 0
	}

	escapeRatio := 1.0
	if dist > 0 || (dist == 0 && escapable(gs, unit, affectedTiles)) {
		escapeRatio *= 0.5
	}

	distRisk := decayRatio(dist, riskFreeDist, k)
	turnRisk := decayRatio(max(tr.turn-turnOffset, 0), riskFreeTurn, kTurn)

	return distRisk * turnRisk * escapeRatio
}

func threatOpponentKing(gs *engine.GameState, sc scoreContext, tr turnResult) int {
	return int(float64(riskOpponentKingScore) * exposureIndex(gs, sc.opponentKingID, tr.affectedTiles, tr, 0, kDistThreat))
}

// riskAllyKing scores only opponent-owned bombs: the ally team controls whether its units stay near its own.
func riskAllyKing(gs *engine.GameState, sc scoreContext, tr turnResult) int {
	return int(float64(riskAllyKingScore) * exposureIndex(gs, sc.allyKingID, tr.opponentAffectedTiles, tr, 1, kDistRisk))
}

// nonKingCount returns how many of ids aren't kingID, alive or not.
// It is the fixed group size to average over, so a wipeout still scores instead of dividing by zero.
func nonKingCount(ids []engine.UnitID, kingID engine.UnitID) int {
	count := 0
	for _, id := range ids {
		if id != kingID {
			count++
		}
	}
	return count
}

func threatOpponents(gs *engine.GameState, sc scoreContext, tr turnResult) int {
	turnOffset := 0
	total := nonKingCount(sc.opponentIDs, sc.opponentKingID)
	if len(tr.affectedTiles) == 0 || total == 0 {
		return 0
	}

	sum := 0.0
	for _, unitID := range sc.opponentIDs {
		if unitID == sc.opponentKingID {
			continue
		}
		sum += exposureIndex(gs, unitID, tr.affectedTiles, tr, turnOffset, kDistThreat)
	}

	return int(float64(riskOpponentScore) * sum / float64(total))
}

// riskAllies scores only opponent-owned bombs: the ally team controls whether its units stay near its own.
func riskAllies(gs *engine.GameState, sc scoreContext, tr turnResult) int {
	turnOffset := 1
	total := nonKingCount(sc.allyIDs, sc.allyKingID)
	if len(tr.opponentAffectedTiles) == 0 || total == 0 {
		return 0
	}

	sum := 0.0
	for _, unitID := range sc.allyIDs {
		if unitID == sc.allyKingID {
			continue
		}
		sum += exposureIndex(gs, unitID, tr.opponentAffectedTiles, tr, turnOffset, kDistRisk)
	}

	return int(float64(riskAllyScore) * sum / float64(total))
}
