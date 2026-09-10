package cpu

// distOpponentKing deduces score based on the distance between Opponent King and nearest Blast tile triggered in which Turns.
func distOpponentKing(sc scoreContext, tr turnResult) int {
	return 0
}

// distOpponent deduces score based on the distance between Opponent and Blast tile triggered in which Turns.
func distOpponent(sc scoreContext, tr turnResult) int {
	return 0
}

// distAllyKing deduces score based on the distance between Ally King and Blast tile triggered in which Turns.
func distAllyKing(sc scoreContext, tr turnResult) int {
	return 0
}

// distAlly deduces score based on the distance between Ally and Blast tile triggered in which Turns.
func distAlly(sc scoreContext, tr turnResult) int {
	return 0
}

// distSoftBlockClearForOpponentKing deduces score based on reachability gained toward the
// opponent King/Boss when a SoftBlock is cleared this Turn.
func distSoftBlockClearForOpponentKing(sc scoreContext, tr turnResult) int {
	return 0
}

// scoreFactorsRegistry stores the base of scoreFactors.
// This initializer func protects the slice from mutation.
func scoreFactorsRegistry() []scoreFactor {
	return []scoreFactor{
		distOpponentKing,
		distOpponent,
		distAllyKing,
		distAlly,
		distSoftBlockClearForOpponentKing,
	}
}
