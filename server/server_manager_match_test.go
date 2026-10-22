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
				roomID, _ := s.CreateMatchRoom()
				return roomID, s
			},
			gameCfg: validGameCfg(),
			wantErr: nil,
			validate: func(t *testing.T, s *ServerStateManager, roomID string) {
				t.Helper()
				room := mustRoom(t, s, roomID)
				if room.Match == nil {
					t.Fatal("Expected Match to be created, got nil")
				}
				if room.Match.GameCfg.StagePreset != "Plain" {
					t.Errorf("Expected StagePreset 'Plain', got '%s'", room.Match.GameCfg.StagePreset)
				}
				if room.Match.GameCfg.MaxTurns != 10 {
					t.Errorf("Expected MaxTurns 10, got %d", room.Match.GameCfg.MaxTurns)
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
			wantErr: ErrRoomNotFound,
		},
		{
			name: "Match Already Exists",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				mustCreateMatch(t, s, roomID, validGameCfg())
				return roomID, s
			},
			gameCfg: validGameCfg(),
			wantErr: ErrMatchExists,
		},
		{
			name: "Invalid Config",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				return roomID, s
			},
			gameCfg: engine.GameCfg{
				StagePreset: "INVALID_STAGE",
				MaxTurns:    10,
			},
			wantErr: ErrInvalidConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s := tt.setup(t)
			playerTokens, err := s.CreateMatch(roomID, tt.gameCfg)
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
				assertGameCfgSynced(t, room)
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
				roomID, _ := s.CreateMatchRoom()
				tokens, _ := s.CreateMatch(roomID, validGameCfg())
				room := mustRoom(t, s, roomID)
				room.Match = nil
				gameCfg := validGameCfg()
				room.GameCfg = &gameCfg
				return roomID, s, tokens
			},
			wantErr: nil,
		},
		{
			name: "Success - Wipe existing Match",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				tokens, _ := s.CreateMatch(roomID, validGameCfg())
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
				roomID, _ := s.CreateMatchRoom()
				mustCreateMatch(t, s, roomID, validGameCfg())
				room := mustRoom(t, s, roomID)
				gameCfg := validGameCfg()
				room.GameCfg = &gameCfg
				return roomID, s, [2]string{"INVALID_TOKEN", ""}
			},
			wantErr: ErrInvalidToken,
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				s := NewServerStateManager()
				return "NONEXISTENT", s, [2]string{}
			},
			wantErr: ErrRoomNotFound,
		},
		{
			name: "Failure - No previous match",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				return roomID, s, [2]string{}
			},
			wantErr: ErrMatchNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s, tokens := tt.setup(t)
			playerTokens, err := s.Rematch(roomID, tokens[0])
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Rematch() error = %v, want %v", err, tt.wantErr)
			}
			if err == nil {
				room := mustRoom(t, s, roomID)
				if room.Match == nil {
					t.Fatal("Expected Match to be created, got nil")
				}
				if room.Match.GameCfg.StagePreset != "Plain" {
					t.Errorf("Expected StagePreset 'Plain', got '%s'", room.Match.GameCfg.StagePreset)
				}
				if len(playerTokens) != 2 || playerTokens[0] == "" || playerTokens[1] == "" || playerTokens[0] == playerTokens[1] {
					t.Errorf("Expected 2 unique non-empty PlayerToken, got %v", playerTokens)
				}
				assertGameCfgSynced(t, room)
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
				roomID, _ := s.CreateMatchRoom()
				tokens, _ := s.CreateMatch(roomID, validGameCfg())
				room := mustRoom(t, s, roomID)
				room.Match = nil
				return roomID, s, tokens
			},
			wantErr: nil,
		},
		{
			name: "Success - Existing Match",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				tokens, _ := s.CreateMatch(roomID, validGameCfg())
				room := mustRoom(t, s, roomID)
				room.Match.WinnerTeamID = 1
				return roomID, s, tokens
			},
			wantErr: nil,
		},
		{
			name: "Failure - Match still in progress",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				tokens, _ := s.CreateMatch(roomID, validGameCfg())
				return roomID, s, tokens
			},
			wantErr: ErrMatchInProgress,
		},
		{
			name: "Invalid token",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				mustCreateMatch(t, s, roomID, validGameCfg())
				room := mustRoom(t, s, roomID)
				gameCfg := validGameCfg()
				room.GameCfg = &gameCfg
				return roomID, s, [2]string{"INVALID_TOKEN", ""}
			},
			wantErr: ErrInvalidToken,
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				s := NewServerStateManager()
				return "NONEXISTENT", s, [2]string{}
			},
			wantErr: ErrRoomNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s, tokens := tt.setup(t)
			err := s.DeleteMatch(roomID, tokens[0])
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Rematch() error = %v, want %v", err, tt.wantErr)
			}
			if err == nil {
				room := mustRoom(t, s, roomID)
				if room.Match != nil {
					t.Fatalf("Expected Match to be deleted, got %p", room.Match)
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
				roomID, _ := s.CreateMatchRoom()
				mustCreateMatch(t, s, roomID, validGameCfg())
				return roomID, s
			},
			wantErr: nil,
			validate: func(t *testing.T, gs *engine.GameState, s *ServerStateManager, roomID string) {
				t.Helper()
				room := mustRoom(t, s, roomID)
				live := room.Match.WorkingState
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
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				mustCreateMatch(t, s, roomID, validGameCfg())
				room := mustRoom(t, s, roomID)
				room.Match.WorkingState.InSuddenDeath = true
				for _, u := range room.Match.WorkingState.Units {
					u.HasMoved = true
					u.HasUsedSkill = true
				}
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
			wantErr: ErrRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				return roomID, s
			},
			wantErr: ErrMatchNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s := tt.setup(t)
			gs, err := s.GetMatchState(roomID)
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
				room := mustRoom(t, s, roomID)
				unitID := engine.NewUnitID(1, 0)
				newPos := engine.Coordinate{X: 4, Y: 7}
				if gotPos := room.Match.WorkingState.Units[unitID].Position; gotPos != newPos {
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
			wantErr: ErrInvalidTurnCmd,
			validate: func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string, cmd engine.TurnCommand) {
				t.Helper()
				room := mustRoom(t, s, roomID)
				unitID := engine.NewUnitID(1, 0)
				if gotPos := room.Match.WorkingState.Units[unitID].Position; gotPos.X == 4 && gotPos.Y == 7777 {
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
			wantErr: ErrInvalidConfig,
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
			wantErr: ErrRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, engine.TurnCommand, string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				unitID := engine.NewUnitID(1, 0)
				newPos := engine.Coordinate{X: 4, Y: 7777}
				cmd := engine.NewMoveCommand(unitID, newPos)
				return roomID, s, cmd, "dummy-token"
			},
			wantErr: ErrMatchNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s, cmd, token := tt.setup(t)
			gameEvents, err := s.SubmitTurnCommand(roomID, cmd, token)
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
				roomID, _ := s.CreateMatchRoom()
				_, err := s.CreateMatch(roomID, gameCfg)
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
				_, err := s.SubmitTurnCommand(roomID, cmd, tokens[0])
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
		{"team-2 unit with team 2's token", 2, 2, ErrInvalidTurnCmd},
		{"team-2 unit with team 1's token", 2, 1, ErrInvalidToken},
		{"team-1 unit with team 2's token", 1, 2, ErrInvalidToken},
		{"team nibble 0", 0, 1, ErrInvalidConfig},
		{"team nibble 3", 3, 1, ErrInvalidConfig},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, tokens, s := createTestRoom(t)
			cmd := engine.NewMoveCommand(engine.NewUnitID(tt.team, 0), engine.Coordinate{X: 4, Y: 7})
			_, err := s.SubmitTurnCommand(roomID, cmd, tokens[tt.tokenTeam-1])
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("SubmitTurnCommand() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerStateManager_StartTurn(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (string, *ServerStateManager, string)
		wantErr  error
		validate func(t *testing.T, inSuddenDeath bool, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string)
	}{
		{
			name: "Success",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				t.Helper()
				roomID, tokens, s := createTestRoom(t)
				room := mustRoom(t, s, roomID)
				room.Match.TrueState.Turn = 1000
				room.Match.WorkingState.Turn = 1000
				return roomID, s, tokens[0]
			},
			wantErr: nil,
			validate: func(t *testing.T, inSuddenDeath bool, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string) {
				t.Helper()
				room := mustRoom(t, s, roomID)

				if got, want := len(room.Match.WorkingState.Bombs), 2; got != want {
					t.Errorf("Expected SuddenDeath triggered and drop %d bombs, got %d", want, got)
				}
				if len(gameEvents) != 2 {
					t.Errorf("expected 2 GameEvent returned, got %d", len(gameEvents))
				}
				for _, evt := range gameEvents {
					if evt.Type != engine.GameEvtBombPlaced {
						t.Errorf("malformed EvtBombPlaced returned: %+v", evt)
					}
				}
				if inSuddenDeath == false {
					t.Errorf("Expected suddenDeath to be true, got %v", inSuddenDeath)
				}
			},
		},
		{
			name: "Match already ended",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				t.Helper()
				roomID, tokens, s := createTestRoom(t)
				room := mustRoom(t, s, roomID)
				room.Match.WinnerTeamID = 1
				return roomID, s, tokens[0]
			},
			wantErr: ErrMatchEnded,
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				t.Helper()
				s := NewServerStateManager()
				return "NONEXISTENT", s, "dummy-token"
			},
			wantErr: ErrRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				return roomID, s, "dummy-token"
			},
			wantErr: ErrMatchNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s, token := tt.setup(t)
			inSuddenDeath, gameEvents, err := s.StartTurn(roomID, token)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("StartTurn() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, inSuddenDeath, gameEvents, s, roomID)
			} else {
				if len(gameEvents) > 0 {
					t.Errorf("Expected gameEvents to be empty, got %p", gameEvents)
				}
				if inSuddenDeath {
					t.Errorf("Expected suddenDeath to be false, got %v", inSuddenDeath)
				}
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
				roomID, tokens, s := createTestRoom(t)
				room := mustRoom(t, s, roomID)
				room.Match.WorkingState.Units[16].HasMoved = true
				return roomID, s, tokens[0]
			},
			wantErr: nil,
			validate: func(t *testing.T, s *ServerStateManager, roomID string) {
				t.Helper()
				room := mustRoom(t, s, roomID)

				if got, want := room.Match.WorkingState.Units[16].HasMoved, false; got != want {
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
			wantErr: ErrRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				return roomID, s, "dummy-token"
			},
			wantErr: ErrMatchNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s, token := tt.setup(t)
			err := s.ResetTurn(roomID, token)
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
				roomID, tokens, s := createTestRoom(t)
				if _, err := s.SubmitTurnCommand(roomID, engine.NewPlaceBombCommand(16, engine.Coordinate{X: 4, Y: 7}), tokens[0]); err != nil {
					t.Fatalf("SubmitTurnCommand() error = %v", err)
				}
				room := mustRoom(t, s, roomID)
				room.Match.WorkingState.Bombs[engine.NewBombID(1, 1, 16)].Countdown = 1
				return roomID, s, tokens[0]
			},
			wantErr: nil,
			validate: func(t *testing.T, planGameEvents, resolveTurnGameEvents []engine.GameEvent, s *ServerStateManager, roomID string) {
				t.Helper()
				room := mustRoom(t, s, roomID)

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
				if got, want := room.Match.WorkingState.Units[16].HP, 0; got != want {
					t.Errorf("Expected Unit %#X HP %v, got %v", 16, want, got)
				}
				if got, want := room.Match.WinnerTeamID, 2; got != want {
					t.Errorf("Expected match winner = %v, got %v", want, got)
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
			wantErr: ErrRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				return roomID, s, "dummy-token"
			},
			wantErr: ErrMatchNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s, token := tt.setup(t)
			planGameEvents, resolveTurnGameEvents, err := s.ResolveTurn(roomID, token)
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
		req      SurrenderRequest
		wantErr  error
		validate func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string)
	}{
		{
			name: "Success",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				roomID, tokens, s := createTestRoom(t)
				return roomID, s, tokens
			},
			req:     SurrenderRequest{TeamID: 1},
			wantErr: nil,
			validate: func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string) {
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
			req:     SurrenderRequest{TeamID: 3},
			wantErr: ErrInvalidConfig,
			validate: func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string) {
				t.Helper()
				room := mustRoom(t, s, roomID)

				if len(gameEvents) != 0 {
					t.Errorf("Expected gameEvents to be empty, got %#v", gameEvents)
				}
				if got, want := room.Match.WinnerTeamID, 0; got != want {
					t.Errorf("Expect match WinnerTeamID %v, got %v", want, got)
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
			req:     SurrenderRequest{TeamID: 1},
			wantErr: ErrRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				return roomID, s, [2]string{"dummy", "dummy"}
			},
			req:     SurrenderRequest{TeamID: 1},
			wantErr: ErrMatchNotFound,
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
			gameEvents, err := s.Surrender(roomID, tt.req.TeamID, token)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Surrender() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, gameEvents, s, roomID)
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
				roomID, _ := s.CreateMatchRoom()
				mustCreateMatch(t, s, roomID, validGameCfg())
				return roomID, s
			},
			wantErr: nil,
			validate: func(t *testing.T, gameCfg *engine.GameCfg, s *ServerStateManager, roomID string) {
				t.Helper()
				room := mustRoom(t, s, roomID)
				if gameCfg == &room.Match.GameCfg {
					t.Error("Expected a copy of the GameCfg, got the live pointer")
				}
				if got, want := *gameCfg, room.Match.GameCfg; !reflect.DeepEqual(got, want) {
					t.Errorf("Expected GameCfg %+v, got %+v", want, got)
				}
			},
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				t.Helper()
				return "NONEXISTENT", NewServerStateManager()
			},
			wantErr: ErrRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				return roomID, s
			},
			wantErr: ErrMatchNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s := tt.setup(t)
			gameCfg, err := s.GetMatchConfig(roomID)
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
				roomID, _ := s.CreateMatchRoom()
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
				roomID, _ := s.CreateMatchRoom()
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
			wantErr: ErrRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				return roomID, s
			},
			wantErr: ErrMatchNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s := tt.setup(t)
			allowed, err := s.GetAllowedTiles(roomID, tt.unitID, tt.turnCmdType)
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
