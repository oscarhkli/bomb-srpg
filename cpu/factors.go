package cpu

import "bomb-srpg/engine"

// factorID identifies a scoreFactor for weight lookup, independent of func identity.
type factorID int

// King below also covers Boss - the two share the similar win/loss role.
const (
	FactorThreatOpponentKing factorID = iota
	FactorThreatOpponent
	FactorRiskAllyKing
	FactorRiskAlly
	FactorAdvanceOpponentKingReachability
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
		{FactorThreatOpponentKing, threatOpponentKing},
		{FactorThreatOpponent, threatOpponent},
		{FactorRiskAllyKing, riskAllyKing},
		{FactorRiskAlly, riskAlly},
		{FactorAdvanceOpponentKingReachability, advanceOpponentKingReachability},
	}
}

// defaultWeightProfile is today's fixed polarity:
// threats count for the actor, risks count against it.
// Future profiles (GameCfg aggressiveness, per-Archetype tuning) are additional maps of this same shape.
func defaultWeightProfile() map[factorID]int {
	return map[factorID]int{
		FactorThreatOpponentKing:              1,
		FactorThreatOpponent:                  1,
		FactorRiskAllyKing:                    -1,
		FactorRiskAlly:                        -1,
		FactorAdvanceOpponentKingReachability: 1,
	}
}

// threatOpponentKing deduces score based on the distance between Opponent King and nearest Blast tile triggered in which Turns.
func threatOpponentKing(gs *engine.GameState, sc scoreContext, tr turnResult) int {
	return 0
}

// threatOpponent deduces score based on the distance between Opponent and Blast tile triggered in which Turns.
func threatOpponent(gs *engine.GameState, sc scoreContext, tr turnResult) int {
	return 0
}

// riskAllyKing deduces score based on the distance between Ally King and Blast tile triggered in which Turns.
func riskAllyKing(gs *engine.GameState, sc scoreContext, tr turnResult) int {
	return 0
}

// riskAlly deduces score based on the distance between Ally and Blast tile triggered in which Turns.
func riskAlly(gs *engine.GameState, sc scoreContext, tr turnResult) int {
	return 0
}

// advanceOpponentKingReachability deduces score based on reachability gained toward the opponent King when a SoftBlock is cleared this Turn.
func advanceOpponentKingReachability(gs *engine.GameState, sc scoreContext, tr turnResult) int {
	if tr.destroyedSoftBlocks == 0 {
		return 0
	}
	return 0
}
