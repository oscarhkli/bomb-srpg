package server

import (
	"errors"
	"testing"

	"bomb-srpg/engine"
)

type matchArrangement func(*engine.Match) error

func arrangedMatch(arrangements ...matchArrangement) Option {
	return WithMatchFactory(func(gameCfg engine.GameCfg) (*engine.Match, error) {
		match, err := engine.InitGame(gameCfg)
		if err != nil {
			return nil, err
		}
		for _, arrange := range arrangements {
			if err := arrange(match); err != nil {
				return nil, err
			}
		}
		return match, nil
	})
}

func cpuTurnPending(match *engine.Match) error {
	match.WorkingState.Turn = 2
	match.WorkingState.ActiveTeam = 2
	match.TrueState = match.WorkingState.DeepCopy()
	return nil
}

func matchEnded(match *engine.Match) error {
	match.WinnerTeamID = 1
	return nil
}

func unitAlreadyMoved(unitID engine.UnitID) matchArrangement {
	return func(match *engine.Match) error {
		unit, ok := match.WorkingState.Units[unitID]
		if !ok {
			return errors.New("unit not found")
		}
		unit.HasMoved = true
		return nil
	}
}

func firstTurnResolved(match *engine.Match) error {
	match.ResolveTurn()
	return nil
}

func bombPlacedBy(unitID engine.UnitID) matchArrangement {
	return func(match *engine.Match) error {
		unit, ok := match.WorkingState.Units[unitID]
		if !ok {
			return errors.New("unit not found")
		}
		target, ok := findFreeNeighbour(match.WorkingState, unit.Position)
		if !ok {
			return errors.New("no free neighbour")
		}
		_, err := match.CommandPlaceBomb(unitID, target)
		return err
	}
}

func bombAboutToExplode(unitID engine.UnitID) matchArrangement {
	return func(match *engine.Match) error {
		if err := bombPlacedBy(unitID)(match); err != nil {
			return err
		}
		for _, bomb := range match.WorkingState.Bombs {
			if bomb.OwnerID == unitID {
				bomb.Countdown = 1
			}
		}
		return nil
	}
}

func suddenDeathMidTurn(match *engine.Match) error {
	match.WorkingState.InSuddenDeath = true
	for _, unit := range match.WorkingState.Units {
		unit.HasMoved = true
		unit.HasUsedSkill = true
	}
	return nil
}

func findFreeNeighbour(gs *engine.GameState, pos engine.Coordinate) (engine.Coordinate, bool) {
	for _, step := range []engine.Coordinate{{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0}} {
		target := engine.Coordinate{X: pos.X + step.X, Y: pos.Y + step.Y}
		if gs.IsLandingLegal(target, engine.OccupantBomb) == nil {
			return target, true
		}
	}
	return engine.Coordinate{}, false
}

func mustMatchState(t *testing.T, s *ServerStateManager, roomID string) *engine.GameState {
	t.Helper()
	gs, err := s.getMatchState(roomID)
	if err != nil {
		t.Fatalf("getMatchState() error = %v", err)
	}
	return gs
}

func softBlocksAt(positions ...engine.Coordinate) matchArrangement {
	return func(match *engine.Match) error {
		for i, pos := range positions {
			match.WorkingState.UpdateStageOccupant(pos, engine.OccupantSoftBlock, int64(i+1))
		}
		return nil
	}
}

func turnCommandsQueued(cmds ...engine.TurnCommand) matchArrangement {
	return func(match *engine.Match) error {
		match.WorkingState.TurnCommands = cmds
		return nil
	}
}

func mustMatchConfig(t *testing.T, s *ServerStateManager, roomID string) *engine.GameCfg {
	t.Helper()
	mustMatchState(t, s, roomID)
	gameCfg, err := s.getMatchConfig(roomID)
	if err != nil {
		t.Fatalf("getMatchConfig() error = %v", err)
	}
	return gameCfg
}

func mustEndAndDeleteMatch(t *testing.T, s *ServerStateManager, roomID, token string) {
	t.Helper()
	if _, err := s.surrender(roomID, 1, token); err != nil {
		t.Fatalf("surrender() error = %v", err)
	}
	if err := s.deleteMatch(roomID, token); err != nil {
		t.Fatalf("deleteMatch() error = %v", err)
	}
}
