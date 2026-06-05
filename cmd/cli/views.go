package cli

import "bomb-srpg/engine"

// MatchView defines the contract for rendering game information.
type MatchView interface {
	RenderBoard(gs *engine.GameState) error
	RenderGameConfig(cfg *engine.GameCfg) error
}
