package server

import (
	"bomb-srpg/engine"
	"errors"
	"net/http"
	"reflect"
	"slices"
	"testing"
)

func TestServerStateManager_CreateMatch(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (string, *ServerStateManager)
		gameCfg  engine.GameCfg
		wantErr  error
		validate func(t *testing.T, s *ServerStateManager, roomID string)
	}{
		{
			name: "Success",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				return roomID, s
			},
			gameCfg: validGameCfg(),
			wantErr: nil,
			validate: func(t *testing.T, s *ServerStateManager, roomID string) {
				t.Helper()
				gameCfg := mustMatchConfig(t, s, roomID)
				if gameCfg.StagePreset != "Plain" {
					t.Errorf("Expected StagePreset 'Plain', got '%s'", gameCfg.StagePreset)
				}
				if gameCfg.MaxTurns != 10 {
					t.Errorf("Expected MaxTurns 10, got %d", gameCfg.MaxTurns)
				}
			},
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				t.Helper()
				s := NewServerStateManager()
				return "NONEXISTENT", s
			},
			gameCfg: validGameCfg(),
			wantErr: errRoomNotFound,
		},
		{
			name: "Match Already Exists",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				mustCreateMatch(t, s, roomID, validGameCfg())
				return roomID, s
			},
			gameCfg: validGameCfg(),
			wantErr: errMatchExists,
		},
		{
			name: "Invalid Config",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				return roomID, s
			},
			gameCfg: engine.GameCfg{
				StagePreset: "INVALID_STAGE",
				MaxTurns:    10,
			},
			wantErr: errInvalidConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s := tt.setup(t)
			playerTokens, err := s.createMatch(roomID, tt.gameCfg)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CreateMatch() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, s, roomID)
			}
			if err == nil {
				if len(playerTokens) != 2 || playerTokens[0] == "" || playerTokens[1] == "" || playerTokens[0] == playerTokens[1] {
					t.Errorf("Expected 2 unique non-empty PlayerToken, got %v", playerTokens)
				}
				room := mustRoom(t, s, roomID)
				assertGameCfgSynced(t, s, roomID)
				if playerTokens != room.PlayerTokens {
					t.Errorf("Expected response and MatchRoom share the same PlayerTokens, response %v vs MatchRoom %v", playerTokens, room.PlayerTokens)
				}
			}
		})
	}
}

func TestServerStateManager_Rematch(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (string, *ServerStateManager, [2]string)
		wantErr  error
		validate func(t *testing.T, s *ServerStateManager, roomID string)
	}{
		{
			name: "Success - Match not exist but with previous gameCfg",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				tokens, _ := s.createMatch(roomID, validGameCfg())
				mustEndAndDeleteMatch(t, s, roomID, tokens[0])
				return roomID, s, tokens
			},
			wantErr: nil,
		},
		{
			name: "Success - Wipe existing Match",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				tokens, _ := s.createMatch(roomID, validGameCfg())
				room := mustRoom(t, s, roomID)
				gameCfg := validGameCfg()
				room.GameCfg = &gameCfg
				return roomID, s, tokens
			},
			wantErr: nil,
		},
		{
			name: "Invalid token",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				mustCreateMatch(t, s, roomID, validGameCfg())
				room := mustRoom(t, s, roomID)
				gameCfg := validGameCfg()
				room.GameCfg = &gameCfg
				return roomID, s, [2]string{"INVALID_TOKEN", ""}
			},
			wantErr: errInvalidToken,
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				s := NewServerStateManager()
				return "NONEXISTENT", s, [2]string{}
			},
			wantErr: errRoomNotFound,
		},
		{
			name: "Failure - No previous match",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				return roomID, s, [2]string{}
			},
			wantErr: errMatchNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s, tokens := tt.setup(t)
			playerTokens, err := s.rematch(roomID, tokens[0])
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Rematch() error = %v, want %v", err, tt.wantErr)
			}
			if err == nil {
				room := mustRoom(t, s, roomID)
				gameCfg := mustMatchConfig(t, s, roomID)
				if gameCfg.StagePreset != "Plain" {
					t.Errorf("Expected StagePreset 'Plain', got '%s'", gameCfg.StagePreset)
				}
				if len(playerTokens) != 2 || playerTokens[0] == "" || playerTokens[1] == "" || playerTokens[0] == playerTokens[1] {
					t.Errorf("Expected 2 unique non-empty PlayerToken, got %v", playerTokens)
				}
				assertGameCfgSynced(t, s, roomID)
				if playerTokens != room.PlayerTokens {
					t.Errorf("Expected response and MatchRoom share the same PlayerTokens, response %v vs MatchRoom %v", playerTokens, room.PlayerTokens)
				}
			}
		})
	}
}

func TestServerStateManager_DeleteMatch(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (string, *ServerStateManager, [2]string)
		wantErr  error
		validate func(t *testing.T, s *ServerStateManager, roomID string)
	}{
		{
			name: "Success - Without existing Match",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				tokens, _ := s.createMatch(roomID, validGameCfg())
				mustEndAndDeleteMatch(t, s, roomID, tokens[0])
				return roomID, s, tokens
			},
			wantErr: nil,
		},
		{
			name: "Success - Existing Match",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				tokens, _ := s.createMatch(roomID, validGameCfg())
				if _, err := s.surrender(roomID, 1, tokens[0]); err != nil {
					t.Fatalf("surrender() error = %v", err)
				}
				return roomID, s, tokens
			},
			wantErr: nil,
		},
		{
			name: "Failure - Match still in progress",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				tokens, _ := s.createMatch(roomID, validGameCfg())
				return roomID, s, tokens
			},
			wantErr: errMatchInProgress,
		},
		{
			name: "Invalid token",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				mustCreateMatch(t, s, roomID, validGameCfg())
				room := mustRoom(t, s, roomID)
				gameCfg := validGameCfg()
				room.GameCfg = &gameCfg
				return roomID, s, [2]string{"INVALID_TOKEN", ""}
			},
			wantErr: errInvalidToken,
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				s := NewServerStateManager()
				return "NONEXISTENT", s, [2]string{}
			},
			wantErr: errRoomNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s, tokens := tt.setup(t)
			err := s.deleteMatch(roomID, tokens[0])
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Rematch() error = %v, want %v", err, tt.wantErr)
			}
			if err == nil {
				if _, err := s.getMatchState(roomID); !errors.Is(err, errMatchNotFound) {
					t.Fatalf("getMatchState() error = %v, want %v", err, errMatchNotFound)
				}
			}
		})
	}
}

func TestServerStateManager_GetMatchState(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (string, *ServerStateManager)
		wantErr  error
		validate func(t *testing.T, gs *engine.GameState, s *ServerStateManager, roomID string)
	}{
		{
			name: "Success",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				mustCreateMatch(t, s, roomID, validGameCfg())
				return roomID, s
			},
			wantErr: nil,
			validate: func(t *testing.T, gs *engine.GameState, s *ServerStateManager, roomID string) {
				t.Helper()
				live := mustRoom(t, s, roomID).Match.WorkingState
				if gs == live {
					t.Error("Expected a copy of the WorkingState, got the live pointer")
				}
				if got, want := gs.Turn, live.Turn; got != want {
					t.Errorf("Expected Turn %d, got %d", want, got)
				}
				if got, want := gs.ActiveTeam, live.ActiveTeam; got != want {
					t.Errorf("Expected ActiveTeam %d, got %d", want, got)
				}
				if got, want := len(gs.Units), len(live.Units); got != want {
					t.Errorf("Expected %d units, got %d", want, got)
				}
			},
		},
		{
			name: "Mid-Turn Progress Survives The Copy",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				t.Helper()
				s := NewServerStateManager(arrangedMatch(suddenDeathMidTurn))
				roomID, _ := s.createMatchRoom()
				mustCreateMatch(t, s, roomID, validGameCfg())
				return roomID, s
			},
			wantErr: nil,
			validate: func(t *testing.T, gs *engine.GameState, s *ServerStateManager, roomID string) {
				t.Helper()
				if !gs.InSuddenDeath {
					t.Error("Expected InSuddenDeath to survive the copy, got false")
				}
				for unitID, u := range gs.Units {
					if !u.HasMoved || !u.HasUsedSkill {
						t.Errorf("Expected unit %#x action economy to survive the copy, got HasMoved %v, HasUsedSkill %v", unitID, u.HasMoved, u.HasUsedSkill)
					}
				}
			},
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				t.Helper()
				return "NONEXISTENT", NewServerStateManager()
			},
			wantErr: errRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				return roomID, s
			},
			wantErr: errMatchNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s := tt.setup(t)
			gs, err := s.getMatchState(roomID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("GetMatchState() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, gs, s, roomID)
			}
		})
	}
}

func TestServerStateManager_SubmitTurnCommand(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (string, *ServerStateManager, engine.TurnCommand, string)
		wantErr  error
		validate func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string, cmd engine.TurnCommand)
	}{
		{
			name: "Success",
			setup: func(t *testing.T) (string, *ServerStateManager, engine.TurnCommand, string) {
				t.Helper()
				roomID, tokens, s := createTestRoom(t)
				unitID := engine.NewUnitID(1, 0)
				newPos := engine.Coordinate{X: 4, Y: 7}
				cmd := engine.NewMoveCommand(unitID, newPos)
				return roomID, s, cmd, tokens[0]
			},
			wantErr: nil,
			validate: func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string, cmd engine.TurnCommand) {
				t.Helper()
				unitID := engine.NewUnitID(1, 0)
				newPos := engine.Coordinate{X: 4, Y: 7}
				if gotPos := mustMatchState(t, s, roomID).Units[unitID].Position; gotPos != newPos {
					t.Errorf("Expected Unit %#X new position %#v, got %#v", unitID, newPos, gotPos)
				}
				if len(gameEvents) != 1 {
					t.Errorf("expected 1 GameEvent returned, got %d", len(gameEvents))
				}
				resEvt := gameEvents[0]
				validFrom := engine.Coordinate{X: 4, Y: 8}
				if resEvt.Type != engine.GameEvtUnitMoved || resEvt.UnitID != unitID || *resEvt.From != validFrom || *resEvt.To != newPos {
					t.Errorf("malformed UnitMoveEvent returned: %+v", resEvt)
				}
			},
		},
		{
			name: "Invalid TurnCommand (out of range)",
			setup: func(t *testing.T) (string, *ServerStateManager, engine.TurnCommand, string) {
				t.Helper()
				roomID, tokens, s := createTestRoom(t)
				unitID := engine.NewUnitID(1, 0)
				newPos := engine.Coordinate{X: 4, Y: 7777}
				cmd := engine.NewMoveCommand(unitID, newPos)
				return roomID, s, cmd, tokens[0]
			},
			wantErr: errInvalidTurnCmd,
			validate: func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string, cmd engine.TurnCommand) {
				t.Helper()
				unitID := engine.NewUnitID(1, 0)
				if gotPos := mustMatchState(t, s, roomID).Units[unitID].Position; gotPos.X == 4 && gotPos.Y == 7777 {
					t.Errorf("Expected Unit %#X didn't move", unitID)
				}
				if len(gameEvents) > 0 {
					t.Errorf("Expected gameEvents to be empty, got %p", gameEvents)
				}
			},
		},
		{
			name: "Invalid TeamID (out of range)",
			setup: func(t *testing.T) (string, *ServerStateManager, engine.TurnCommand, string) {
				t.Helper()
				roomID, _, s := createTestRoom(t)
				unitID := engine.NewUnitID(3, 0)
				newPos := engine.Coordinate{X: 4, Y: 7}
				cmd := engine.NewMoveCommand(unitID, newPos)
				return roomID, s, cmd, "dummy-token"
			},
			wantErr: errInvalidConfig,
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, engine.TurnCommand, string) {
				t.Helper()
				s := NewServerStateManager()
				unitID := engine.NewUnitID(1, 0)
				newPos := engine.Coordinate{X: 4, Y: 7777}
				cmd := engine.NewMoveCommand(unitID, newPos)
				return "NONEXISTENT", s, cmd, "dummy-token"
			},
			wantErr: errRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, engine.TurnCommand, string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				unitID := engine.NewUnitID(1, 0)
				newPos := engine.Coordinate{X: 4, Y: 7777}
				cmd := engine.NewMoveCommand(unitID, newPos)
				return roomID, s, cmd, "dummy-token"
			},
			wantErr: errMatchNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s, cmd, token := tt.setup(t)
			gameEvents, err := s.submitTurnCommand(roomID, cmd, token)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("SubmitTurnCommand() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, gameEvents, s, roomID, cmd)
			} else if len(gameEvents) > 0 {
				t.Errorf("Expected gameEvents to be empty, got %p", gameEvents)
			}
		})
	}
}

func TestServerStateManager_EngineRejectionStatus(t *testing.T) {
	tests := []struct {
		name     string
		run      func(t *testing.T) error
		wantCode int
		wantMsg  string
	}{
		{
			name: "config rejected by the engine",
			run: func(t *testing.T) error {
				t.Helper()
				gameCfg := validGameCfg()
				gameCfg.StagePreset = "NoSuchStage"
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				_, err := s.createMatch(roomID, gameCfg)
				return err
			},
			wantCode: http.StatusBadRequest,
			wantMsg:  "invalid game config: invalid stage preset: stage preset 'NoSuchStage' not found",
		},
		{
			name: "turn command rejected by the engine",
			run: func(t *testing.T) error {
				t.Helper()
				roomID, tokens, s := createTestRoom(t)
				cmd := engine.NewMoveCommand(engine.NewUnitID(1, 0), engine.Coordinate{X: 4, Y: 7777})
				_, err := s.submitTurnCommand(roomID, cmd, tokens[0])
				return err
			},
			wantCode: http.StatusConflict,
			wantMsg:  "invalid turn command: target out of move range",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, msg := mapError(tt.run(t))
			if code != tt.wantCode || msg != tt.wantMsg {
				t.Errorf("got (%d, %q) want (%d, %q)", code, msg, tt.wantCode, tt.wantMsg)
			}
		})
	}
}

func TestServerStateManager_SubmitTurnCommand_TeamFromUnitID(t *testing.T) {
	tests := []struct {
		name      string
		team      int
		tokenTeam int
		wantErr   error
	}{
		{"team-1 unit with team 1's token", 1, 1, nil},
		{"team-2 unit with team 2's token", 2, 2, errInvalidTurnCmd},
		{"team-2 unit with team 1's token", 2, 1, errInvalidToken},
		{"team-1 unit with team 2's token", 1, 2, errInvalidToken},
		{"team nibble 0", 0, 1, errInvalidConfig},
		{"team nibble 3", 3, 1, errInvalidConfig},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, tokens, s := createTestRoom(t)
			cmd := engine.NewMoveCommand(engine.NewUnitID(tt.team, 0), engine.Coordinate{X: 4, Y: 7})
			_, err := s.submitTurnCommand(roomID, cmd, tokens[tt.tokenTeam-1])
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("SubmitTurnCommand() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerStateManager_ResetTurn(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (string, *ServerStateManager, string)
		wantErr  error
		validate func(t *testing.T, s *ServerStateManager, roomID string)
	}{
		{
			name: "Success",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				t.Helper()
				roomID, tokens, s := createTestRoomWithCfg(t, validGameCfg(), arrangedMatch(unitAlreadyMoved(16)))
				if !mustMatchState(t, s, roomID).Units[16].HasMoved {
					t.Fatal("arranged unit has not moved")
				}
				return roomID, s, tokens[0]
			},
			wantErr: nil,
			validate: func(t *testing.T, s *ServerStateManager, roomID string) {
				t.Helper()
				if got, want := mustMatchState(t, s, roomID).Units[16].HasMoved, false; got != want {
					t.Errorf("Expected Unit %#X HasMoved reset to %v, got %v", 16, want, got)
				}
			},
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				t.Helper()
				s := NewServerStateManager()
				return "NONEXISTENT", s, "dummy-token"
			},
			wantErr: errRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				return roomID, s, "dummy-token"
			},
			wantErr: errMatchNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s, token := tt.setup(t)
			err := s.resetTurn(roomID, token)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ResetTurn() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, s, roomID)
			}
		})
	}
}

func TestServerStateManager_ResolveTurn(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (string, *ServerStateManager, string)
		wantErr  error
		validate func(t *testing.T, planGameEvents, resolveTurnGameEvents []engine.GameEvent, s *ServerStateManager, roomID string)
	}{
		{
			name: "Success",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				t.Helper()
				roomID, tokens, s := createTestRoomWithCfg(t, validGameCfg(), arrangedMatch(bombAboutToExplode(16)))
				return roomID, s, tokens[0]
			},
			wantErr: nil,
			validate: func(t *testing.T, planGameEvents, resolveTurnGameEvents []engine.GameEvent, s *ServerStateManager, roomID string) {
				t.Helper()
				if got, want := planGameEvents, 1; len(got) != want {
					t.Errorf("Expected %d planGameEvents returned, got %#v", want, got)
				}
				if !hasGameEvent(planGameEvents, engine.GameEvtBombPlaced, 16) {
					t.Errorf("Expected the placed bomb in planGameEvents, got %#v", planGameEvents)
				}
				if got, want := resolveTurnGameEvents, 5; len(got) != want {
					t.Errorf("Expected %d resolveTurnGameEvents returned, got %#v", want, got)
				}
				if hasGameEvent(resolveTurnGameEvents, engine.GameEvtBombPlaced, 16) {
					t.Errorf("Expected no planning event in resolveTurnGameEvents, got %#v", resolveTurnGameEvents)
				}
				if got, want := mustMatchState(t, s, roomID).Units[16].HP, 0; got != want {
					t.Errorf("Expected Unit %#X HP %v, got %v", 16, want, got)
				}
				if !slices.ContainsFunc(resolveTurnGameEvents, func(evt engine.GameEvent) bool {
					return evt.Type == engine.GameEvtMatchEnded && evt.WinnerTeamID == 2
				}) {
					t.Errorf("Expected a matchEnded event with winner 2, got %+v", resolveTurnGameEvents)
				}
			},
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				t.Helper()
				s := NewServerStateManager()
				return "NONEXISTENT", s, "dummy-token"
			},
			wantErr: errRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				return roomID, s, "dummy-token"
			},
			wantErr: errMatchNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s, token := tt.setup(t)
			planGameEvents, resolveTurnGameEvents, err := s.resolveTurn(roomID, token)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ResolveTurn() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, planGameEvents, resolveTurnGameEvents, s, roomID)
			} else if len(planGameEvents) != 0 || len(resolveTurnGameEvents) != 0 {
				t.Errorf("Expected gameEvents to be empty, got %#v and %#v", planGameEvents, resolveTurnGameEvents)
			}
		})
	}
}

func TestServerStateManager_Surrender(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (string, *ServerStateManager, [2]string)
		req      surrenderRequest
		wantErr  error
		validate func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string, tokens [2]string)
	}{
		{
			name: "Success",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				roomID, tokens, s := createTestRoom(t)
				return roomID, s, tokens
			},
			req:     surrenderRequest{TeamID: 1},
			wantErr: nil,
			validate: func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string, tokens [2]string) {
				t.Helper()
				if got, want := gameEvents, 1; len(got) != want {
					t.Errorf("Expected %d gameEvents returned, got %#v", want, got)
				}
				if got, want := gameEvents[0].WinnerTeamID, 2; got != want {
					t.Errorf("Expected gameEvent WinnerTeamID = %v, got %v", want, got)
				}
				if _, ok := s.Rooms.Load(roomID); !ok {
					t.Error("Expected room not to be deleted after surrender")
				}
			},
		},
		{
			name: "Invalid SurrenderRequest",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				roomID, tokens, s := createTestRoom(t)
				return roomID, s, tokens
			},
			req:     surrenderRequest{TeamID: 3},
			wantErr: errInvalidConfig,
			validate: func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string, tokens [2]string) {
				t.Helper()
				if len(gameEvents) != 0 {
					t.Errorf("Expected gameEvents to be empty, got %#v", gameEvents)
				}
				if err := s.deleteMatch(roomID, tokens[0]); !errors.Is(err, errMatchInProgress) {
					t.Errorf("deleteMatch() error = %v, want %v", err, errMatchInProgress)
				}
			},
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				s := NewServerStateManager()
				return "NONEXISTENT", s, [2]string{"dummy", "dummy"}
			},
			req:     surrenderRequest{TeamID: 1},
			wantErr: errRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				return roomID, s, [2]string{"dummy", "dummy"}
			},
			req:     surrenderRequest{TeamID: 1},
			wantErr: errMatchNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s, tokens := tt.setup(t)
			var token string
			if tt.req.TeamID >= 1 && tt.req.TeamID <= 2 {
				token = tokens[tt.req.TeamID-1]
			} else {
				token = "dummy-token"
			}
			gameEvents, err := s.surrender(roomID, tt.req.TeamID, token)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Surrender() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, gameEvents, s, roomID, tokens)
			} else if len(gameEvents) != 0 {
				t.Errorf("Expected gameEvents to be empty, got %#v", gameEvents)
			}
		})
	}
}

func TestServerStateManager_GetMatchConfig(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (string, *ServerStateManager)
		wantErr  error
		validate func(t *testing.T, gameCfg *engine.GameCfg, s *ServerStateManager, roomID string)
	}{
		{
			name: "Success",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				mustCreateMatch(t, s, roomID, validGameCfg())
				return roomID, s
			},
			wantErr: nil,
			validate: func(t *testing.T, gameCfg *engine.GameCfg, s *ServerStateManager, roomID string) {
				t.Helper()
				if got, want := *gameCfg, validGameCfg(); !reflect.DeepEqual(got, want) {
					t.Errorf("Expected GameCfg %+v, got %+v", want, got)
				}
				gameCfg.MaxTurns = 99
				gameCfg.P1Slots[0].Archetype = "Mutated"
				again, err := s.getMatchConfig(roomID)
				if err != nil {
					t.Fatalf("getMatchConfig() error = %v", err)
				}
				if got, want := *again, validGameCfg(); !reflect.DeepEqual(got, want) {
					t.Errorf("Expected a write to the returned GameCfg not to reach the match, got %+v, want %+v", got, want)
				}
			},
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				t.Helper()
				return "NONEXISTENT", NewServerStateManager()
			},
			wantErr: errRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				return roomID, s
			},
			wantErr: errMatchNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s := tt.setup(t)
			gameCfg, err := s.getMatchConfig(roomID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("GetMatchConfig() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, gameCfg, s, roomID)
			}
		})
	}
}

func TestServerStateManager_GetAllowedTiles(t *testing.T) {
	cmpFunc := func(a, b engine.Coordinate) int {
		if a.X != b.X {
			return a.X - b.X
		}
		return a.Y - b.Y
	}

	tests := []struct {
		name        string
		setup       func(t *testing.T) (string, *ServerStateManager)
		unitID      engine.UnitID
		turnCmdType engine.TurnCmdType
		wantErr     error
		validate    func(t *testing.T, allowed []engine.Coordinate, s *ServerStateManager, roomID string)
	}{
		{
			name: "Success",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				mustCreateMatch(t, s, roomID, validGameCfg())
				return roomID, s
			},
			unitID:      engine.NewUnitID(1, 0),
			turnCmdType: engine.TurnCmdPlaceBomb,
			wantErr:     nil,
			validate: func(t *testing.T, allowed []engine.Coordinate, s *ServerStateManager, roomID string) {
				t.Helper()
				want := []engine.Coordinate{
					{X: 2, Y: 8}, {X: 5, Y: 8}, {X: 6, Y: 8}, {X: 4, Y: 7}, {X: 4, Y: 6},
				}
				slices.SortFunc(want, cmpFunc)
				if !slices.Equal(want, allowed) {
					t.Errorf("Expected coordinates %#v, got %#v", want, allowed)
				}
			},
		},
		{
			name: "Unsupported command",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				mustCreateMatch(t, s, roomID, validGameCfg())
				return roomID, s
			},
			unitID:      engine.NewUnitID(1, 0),
			turnCmdType: "invalid",
			wantErr:     engine.ErrUnsupportedCommand,
			validate: func(t *testing.T, allowed []engine.Coordinate, s *ServerStateManager, roomID string) {
				t.Helper()
			},
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				t.Helper()
				return "NONEXISTENT", NewServerStateManager()
			},
			wantErr: errRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.createMatchRoom()
				return roomID, s
			},
			wantErr: errMatchNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s := tt.setup(t)
			allowed, err := s.getAllowedTiles(roomID, tt.unitID, tt.turnCmdType)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("GetAllowedTiles() error = %v, want %v", err, tt.wantErr)
			}
			slices.SortFunc(allowed, cmpFunc)
			if tt.validate != nil {
				tt.validate(t, allowed, s, roomID)
			}
		})
	}
}

func TestServerStateManager_GetMatchConfig_DoesNotAliasMatch(t *testing.T) {
	roomID, _, s := createTestRoom(t)
	cfg, err := s.getMatchConfig(roomID)
	if err != nil {
		t.Fatalf("getMatchConfig() error = %v", err)
	}
	wantP1, wantP2 := cfg.P1Slots[0].Archetype, cfg.P2Slots[0].Archetype

	cfg.P1Slots[0].Archetype = "MUTATED"
	cfg.P2Slots[0].Archetype = "MUTATED"

	got, err := s.getMatchConfig(roomID)
	if err != nil {
		t.Fatalf("getMatchConfig() error = %v", err)
	}
	if got.P1Slots[0].Archetype != wantP1 || got.P2Slots[0].Archetype != wantP2 {
		t.Errorf("slots = %q, %q, want %q, %q", got.P1Slots[0].Archetype, got.P2Slots[0].Archetype, wantP1, wantP2)
	}
}

func TestServerStateManager_CreateMatch_DoesNotAliasInput(t *testing.T) {
	cfg := validGameCfg()
	want := cfg.P1Slots[0].Archetype
	roomID, _, s := createTestRoomWithCfg(t, cfg)

	cfg.P1Slots[0].Archetype = "MUTATED"

	got, err := s.getMatchConfig(roomID)
	if err != nil {
		t.Fatalf("getMatchConfig() error = %v", err)
	}
	if got.P1Slots[0].Archetype != want {
		t.Errorf("P1Slots[0].Archetype = %q, want %q", got.P1Slots[0].Archetype, want)
	}
}
