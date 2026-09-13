package cpu

import (
	"bomb-srpg/engine"
	"math"
)

// factorID identifies a scoreFactor for weight lookup, independent of func identity.
type factorID int

// King below also covers Boss - the two share the similar win/loss role.
const (
	FactorAdvanceOpponentKingReachability factorID = iota
	FactorAdvanceOpponentKingDistance
	FactorKillOpponentKing
	FactorKillAllyKing
	FactorKillOpponents
	FactorKillAllies
	FactorThreatOpponentKing
	FactorRiskAllyKing
	FactorThreatOpponents
	FactorRiskAllies
	FactorSuicideAllyKing
	FactorSuicideAllies
)

// scoreFactorEntry pairs a scoreFactor with the identity used to look up its weight.
type scoreFactorEntry struct {
	id     factorID
	factor scoreFactor
}

// scoreFactorsRegistry stores the base of scoreFactors.
// This initializer func protects the slice from mutation.
func scoreFactorsRegistry() []scoreFactorEntry {
	return []scoreFactorEntry{
		{FactorAdvanceOpponentKingReachability, advanceOpponentKingReachability},
		{FactorAdvanceOpponentKingDistance, advanceOpponentKingDistance},
		{FactorKillOpponentKing, killOpponentKing},
		{FactorKillAllyKing, killAllyKing},
		{FactorKillOpponents, killOpponents},
		{FactorKillAllies, killAllies},
		{FactorThreatOpponentKing, threatOpponentKing},
		{FactorRiskAllyKing, riskAllyKing},
		{FactorThreatOpponents, threatOpponents},
		{FactorRiskAllies, riskAllies},
		{FactorSuicideAllyKing, suicideAllyKing},
		{FactorSuicideAllies, suicideAllies},
	}
}

// defaultWeightProfile is today's fixed polarity:
// threats count for the actor, risks count against it.
// Future profiles (GameCfg aggressiveness, per-Archetype tuning) are additional maps of this same shape.
func defaultWeightProfile() map[factorID]int {
	return map[factorID]int{
		FactorAdvanceOpponentKingReachability: 1,
		FactorAdvanceOpponentKingDistance:     1,
		FactorKillOpponentKing:                1,
		FactorKillAllyKing:                    -1,
		FactorKillOpponents:                   1,
		FactorKillAllies:                      -1,
		FactorThreatOpponentKing:              1,
		FactorRiskAllyKing:                    -1,
		FactorThreatOpponents:                 1,
		FactorRiskAllies:                      -1,
		FactorSuicideAllyKing:                 -1,
		FactorSuicideAllies:                   -1,
	}
}

// Score magnitudes, tiered so a certain outcome always outranks a predicted one
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

// advanceOpponentKingReachability deduces score based on reachability gained toward the opponent King when a SoftBlock is cleared this Turn.
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

// advanceOpponentKingDistance deduces score based on distance changed toward the opponent King this Turn.
// Only cares T+0: it's the only turn a move can occur before the forecast loop starts.
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

// killOpponentKing deduces score based on whether the opponent King died this Turn.
// Only cares T+0.
func killOpponentKing(gs *engine.GameState, sc scoreContext, tr turnResult) int {
	if tr.turn > 0 {
		return 0
	}
	if _, ok := tr.diedUnits[sc.opponentKingID]; ok {
		return killKingScore
	}
	return 0
}

// killAllyKing deduces score based on whether the ally King died this Turn.
// Cares T+0 & T+1 as Ally Team can't act at T+1.
func killAllyKing(gs *engine.GameState, sc scoreContext, tr turnResult) int {
	if tr.turn > 1 {
		return 0
	}
	if _, ok := tr.diedUnits[sc.allyKingID]; ok {
		return killKingScore
	}
	return 0
}

// killOpponents deduces score based on the opponent non-King survive rate this Turn.
// Only cares T+0.
func killOpponents(gs *engine.GameState, sc scoreContext, tr turnResult) int {
	if tr.turn > 0 || tr.aliveOpponentsBefore == 0 {
		return 0
	}
	killed := tr.aliveOpponentsBefore - tr.aliveOpponentsAfter
	return killUnitScore * killed / tr.aliveOpponentsBefore
}

// killAllies deduces score based on the ally non-King survive rate this Turn.
// Cares T+0 & T+1 as Ally Team can't act at T+1.
func killAllies(gs *engine.GameState, sc scoreContext, tr turnResult) int {
	if tr.turn > 1 || tr.aliveAlliesBefore == 0 {
		return 0
	}
	killed := tr.aliveAlliesBefore - tr.aliveAlliesAfter
	return killUnitScore * killed / tr.aliveAlliesBefore
}

// suicideAllyKing deduces score based on whether the ally King died from an ally-owned bomb.
// Only fires at tr.turn>1: killAllyKing already covers tr.turn<=1 for any cause of death,
// self-inflicted included, and this must not double-count that same death.
func suicideAllyKing(gs *engine.GameState, sc scoreContext, tr turnResult) int {
	if tr.turn <= 1 {
		return 0
	}
	if _, ok := tr.suicides[sc.allyKingID]; ok {
		return killKingScore
	}
	return 0
}

// suicideAllies deduces score based on the ally non-King survive rate lost to an ally-owned bomb.
// Only fires at tr.turn>1, for the same reason as suicideAllyKing.
func suicideAllies(gs *engine.GameState, sc scoreContext, tr turnResult) int {
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
		// exposureIndex already guarantees a live, existing unit. Treat unexpected error as escapable rather than propagating it.
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
	riskFreeDist = 6                  // distance beyond which threat is considered negligible
	kDistThreat  = 4.0 / riskFreeDist // decayRatio's distance-axis k for offense
	kDistRisk    = 1.0 / riskFreeDist // decayRatio's distance-axis k for defense
	riskFreeTurn = 5                  // turns beyond which a bomb's fuse is considered escapable
	kTurn        = 1.0 / riskFreeTurn // decayRatio's k for the turn axis
)

// exposureIndex deduces a unit's exposure to the nearest tile in affectedTiles, as a 0..1 ratio.
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

// threatOpponentKing deduces score based on the distance between Opponent King and nearest bomb affected tile triggered in which Turns.
func threatOpponentKing(gs *engine.GameState, sc scoreContext, tr turnResult) int {
	return int(float64(riskOpponentKingScore) * exposureIndex(gs, sc.opponentKingID, tr.affectedTiles, tr, 0, kDistThreat))
}

// riskAllyKing deduces score based on the distance between Ally King and an opponent-owned bomb's affected tile.
// A bomb the ally team itself placed isn't scored here: the team controls whether it keeps units near it.
func riskAllyKing(gs *engine.GameState, sc scoreContext, tr turnResult) int {
	return int(float64(riskAllyKingScore) * exposureIndex(gs, sc.allyKingID, tr.opponentAffectedTiles, tr, 1, kDistRisk))
}

// nonKingCount returns how many of ids aren't kingID, alive or not — the fixed group size
// to average risk/threat exposure over, so a wipeout still scores instead of dividing by zero.
func nonKingCount(ids []engine.UnitID, kingID engine.UnitID) int {
	count := 0
	for _, id := range ids {
		if id != kingID {
			count++
		}
	}
	return count
}

// threatOpponents deduces the average score based on the distance between Opponents and bomb affected tile triggered in which Turns.
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

// riskAllies deduces the average score based on the distance between Allies and an opponent-owned bomb's affected tile.
// A bomb the ally team itself placed isn't scored here: the team controls whether it keeps units near it.
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
