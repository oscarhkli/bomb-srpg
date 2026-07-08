package cli

import (
	"bomb-srpg/engine"
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
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
		c.Match.StartTurn()

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
			continue
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
		_ = c.View.RenderFeedback(true, fmt.Sprintf("PLAYER %d surrendered!", c.Match.WorkingState.ActiveTeam))
		_ = c.View.RenderGameEvents(events)

	default:
		_ = c.View.RenderFeedback(false, fmt.Sprintf("Unknown meta command: %s. Use /config, /reset, /commit, or /surrender\n", cmd))
	}
}

// routeGameAction handles movement and bomb placement command
// CLI version doesn't and won't have view stats / reachable info - will work directly in Web version instead
func (c *MatchController) routeGameAction(cmd string) {
	token := strings.Fields(strings.ToLower(cmd))

	if len(token) == 0 {
		_ = c.View.RenderFeedback(false, "Input is empty")
		return
	}

	action := token[0]

	switch action {
	case "move":
		if len(token) < 4 {
			_ = c.View.RenderFeedback(false, fmt.Sprintf("Syntax error in move command: %s. Syntax Error! Expected: move <unit_idx> <x> <y>", cmd))
			return
		}

		unitID, target, err := c.parseActionCommand(token[1], token[2], token[3])
		if err != nil {
			_ = c.View.RenderFeedback(false, "Argument Error: Unit index, X, and Y must be integers!")
			return
		}

		gameEvents, err := c.Match.ApplyTurnCommand(engine.NewMoveCommand(unitID, target))
		if err != nil {
			_ = c.View.RenderFeedback(false, fmt.Sprintf("Invalid move: %v", err))
			return
		}
		_ = c.View.RenderFeedback(true, fmt.Sprintf("GameEvents: %#v", gameEvents))

	case "bomb":
		if len(token) < 4 {
			_ = c.View.RenderFeedback(false, fmt.Sprintf("Syntax error in bomb command: %s. Syntax Error! Expected: bomb <unit_idx> <x> <y>", cmd))
			return
		}

		unitID, target, err := c.parseActionCommand(token[1], token[2], token[3])
		if err != nil {
			_ = c.View.RenderFeedback(false, "Argument Error: Unit index, X, and Y must be integers!")
			return
		}

		gameEvents, err := c.Match.ApplyTurnCommand(engine.NewPlaceBombCommand(unitID, target))
		if err != nil {
			_ = c.View.RenderFeedback(false, fmt.Sprintf("Invalid bomb placement: %v", err))
			return
		}
		_ = c.View.RenderFeedback(true, fmt.Sprintf("GameEvents: %#v", gameEvents))

	default:
		_ = c.View.RenderFeedback(false, fmt.Sprintf("Unknown meta command: %s\n", cmd))
		return
	}
}

func (c *MatchController) parseActionCommand(s1, s2, s3 string) (engine.UnitID, engine.Coordinate, error) {
	id, err1 := strconv.ParseUint(s1, 10, 8)
	x, err2 := strconv.Atoi(s2)
	y, err3 := strconv.Atoi(s3)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, engine.Coordinate{}, errors.New("invalid integer syntax")
	}
	return engine.UnitID(id), engine.Coordinate{X: x, Y: y}, nil
}
