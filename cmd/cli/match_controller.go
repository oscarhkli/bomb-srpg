package cli

import "bomb-srpg/engine"

type MatchController struct {
	Match *engine.Match
	View  MatchView
}

func NewMatchController(m *engine.Match, v MatchView) *MatchController {
	return &MatchController{Match: m, View: v}
}

func (c *MatchController) StartInputLoop() {
	c.View.RenderBoard(c.Match.WorkingState)
	// TODO
}
