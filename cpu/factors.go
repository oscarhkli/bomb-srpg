package cpu

import "bomb-srpg/engine"

// factorID identifies a scoreFactor for weight lookup, independent of func identity.
type factorID int

// King below also covers Boss - the two share the similar win/loss role.
const (
	FactorAdvanceOpponentKingReachability factorID = iota
	FactorKillOpponentKing
	FactorKillAllyKing
	FactorKillOpponents
	FactorKillAllies
	FactorRiskAllyKing
	FactorThreatOpponentKing
	FactorRiskAllies
	FactorThreatOpponents
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
		{FactorKillOpponentKing, killOpponentKing},
		{FactorKillAllyKing, killAllyKing},
		{FactorKillOpponents, killOpponents},
		{FactorKillAllies, killAllies},
		{FactorRiskAllyKing, riskAllyKing},
		{FactorThreatOpponentKing, threatOpponentKing},
		{FactorRiskAllies, riskAllies},
		{FactorThreatOpponents, threatOpponents},
	}
}

// defaultWeightProfile is today's fixed polarity:
// threats count for the actor, risks count against it.
// Future profiles (GameCfg aggressiveness, per-Archetype tuning) are additional maps of this same shape.
func defaultWeightProfile() map[factorID]int {
	return map[factorID]int{
		FactorAdvanceOpponentKingReachability: 1,
		FactorKillOpponentKing:                1,
		FactorKillAllyKing:                    -1,
		FactorKillOpponents:                   1,
		FactorKillAllies:                      -1,
		FactorRiskAllyKing:                    -1,
		FactorThreatOpponentKing:              1,
		FactorRiskAllies:                      -1,
		FactorThreatOpponents:                 1,
	}
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

	score := maxForecastTurn - tr.turn

	if tr.distToKingBefore == tr.distToKingAfter {
		return 0
	}
	if tr.distToKingAfter == -1 {
		return -score
	}
	if tr.distToKingBefore == -1 {
		return score
	}
	if tr.distToKingBefore > tr.distToKingAfter {
		return score
	}
	return -score
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

// riskAllyKing deduces score based on the distance between Ally King and bomb affected tile triggered in which Turns.
// Also consider the escapability.
func riskAllyKing(gs *engine.GameState, sc scoreContext, tr turnResult) int {
	return 0
}

// threatOpponentKing deduces score based on the distance between Opponent King and nearest bomb affected tile triggered in which Turns.
// Also consider the escapability.
func threatOpponentKing(gs *engine.GameState, sc scoreContext, tr turnResult) int {
	return 0
}

// riskAllies deduces score based on the distance between Ally and bomb affected tile triggered in which Turns.
// Also consider the escapability.
func riskAllies(gs *engine.GameState, sc scoreContext, tr turnResult) int {
	return 0
}

// threatOpponents deduces score based on the distance between Opponent and bomb affected tile triggered in which Turns.
// Also consider the escapability.
func threatOpponents(gs *engine.GameState, sc scoreContext, tr turnResult) int {
	return 0
}
