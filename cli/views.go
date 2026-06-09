package cli

import "bomb-srpg/engine"

// MatchView defines the contract for rendering game information.
type MatchView interface {
	RenderBoard(gs *engine.GameState) error
	RenderGameConfig(cfg *engine.GameCfg) error
	RenderMessage(message string) error
	RenderFeedback(success bool, message string) error
	RenderTurnHeader(turn, activeTeamID int) error
	RenderGameEvents(events []engine.GameEvent) error
}
