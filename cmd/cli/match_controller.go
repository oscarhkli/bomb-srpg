package cli

import (
	"bomb-srpg/engine"
	"bufio"
	"fmt"
	"io"
	"log"
	"strings"
)

type MatchController struct {
	Match *engine.Match
	View  MatchView
	input io.Reader
}

func NewMatchController(m *engine.Match, v MatchView, in io.Reader) *MatchController {
	return &MatchController{Match: m, View: v, input: in}
}

func (c *MatchController) StartInputLoop() {
	scanner := bufio.NewScanner(c.input)

	for {
		// Always render the latest situation
		if err := c.View.RenderBoard(c.Match.WorkingState); err != nil {
			log.Fatalf("Critical Interface Failure: %v", err)
		}

		// State check: Victory / Surrender
		if c.Match.WinnerTeamID != 0 {
			var message string
			if c.Match.WinnerTeamID == -1 {
				message = "Draw Game!"
			} else {
				message = fmt.Sprintf("Winner... PLAYER %d!", c.Match.WinnerTeamID)
			}
			c.View.RenderFeedback(true, message)

			return
		}

		c.View.RenderMessage("\nEnter Command: ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue // Skip processing if user typed whitespace
		}

		// 4. Intercept system shortcuts
		if strings.HasPrefix(line, "/") {
			c.handleSystemCommand(line)
		}

		c.routeGameAction(line)
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Critical Input Failure: %v", err)
	}
}

func (c *MatchController) handleSystemCommand(cmd string) {
	switch strings.ToLower(cmd) {
	case "/config":
		_ = c.View.RenderGameConfig(&c.Match.GameCfg)

	case "/reset":
		c.Match.ResetTurn()
		_ = c.View.RenderFeedback(true, "Turn reset successfully!")

	case "/commit":
		events := c.Match.ResolveTurn()
		_ = c.View.RenderFeedback(true, "Turn committed and resolved!")
		_ = c.View.RenderGameEvents(events)

	case "/surrender":
		events := c.Match.Surrender(c.Match.WorkingState.ActiveTeam)
		_ = c.View.RenderFeedback(true, fmt.Sprintf("PLAYER %d surrendered", c.Match.WorkingState.ActiveTeam))
		_ = c.View.RenderGameEvents(events)

	default:
		_ = c.View.RenderFeedback(false, fmt.Sprintf("Unknown meta command: %s. Use /config, /reset, /commit, or /surrender\n", cmd))
	}
}

// routeGameAction
func (c *MatchController) routeGameAction(line string) {
	//
}
