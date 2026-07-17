package server

import (
	"bomb-srpg/engine"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{
		Level: slog.LevelError + 1, // Discard all logs
	})))
	m.Run()
}

func TestServerStateManager_CreateMatchRoom(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *ServerStateManager
		wantErr  bool
		validate func(t *testing.T, s *ServerStateManager, id string, err error)
	}{
		{
			name: "Success",
			setup: func() *ServerStateManager {
				return NewServerStateManager()
			},
			wantErr: false,
			validate: func(t *testing.T, s *ServerStateManager, id string, err error) {
				if id == "" {
					t.Fatal("Expected non-empty room ID")
				}
				if len(id) != 5 {
					t.Errorf("Expected ID length 5, got %d: %s", len(id), id)
				}
				if !isValidCrockfordCode(id) {
					t.Errorf("ID contains invalid characters: %s", id)
				}
				roomVal, ok := s.Rooms.Load(id)
				if !ok {
					t.Fatal("Room not added to manager.Rooms")
				}
				room := roomVal.(*MatchRoom)
				if room.Match != nil {
					t.Errorf("Expected Match=nil, got %v", room.Match)
				}
				if room.ID != id {
					t.Errorf("Room.ID mismatch: got %s want %s", room.ID, id)
				}
			},
		},
		{
			name: "Room ID collision retry",
			setup: func() *ServerStateManager {
				s := NewServerStateManager()
				existingID := "ABCDE"
				s.Rooms.Store(existingID, &MatchRoom{ID: existingID})
				return s
			},
			wantErr: false,
			validate: func(t *testing.T, s *ServerStateManager, id string, err error) {
				if id == "ABCDE" {
					t.Fatal("Returned ID should not match pre-seeded ID")
				}
				if len(id) != 5 || !isValidCrockfordCode(id) {
					t.Errorf("Invalid generated ID: %s", id)
				}
				if _, ok := s.Rooms.Load("ABCDE"); !ok {
					t.Error("Predefined room missing")
				}
				if _, ok := s.Rooms.Load(id); !ok {
					t.Error("New room not added")
				}
			},
		},
		{
			name: "Max retries exhausted",
			setup: func() *ServerStateManager {
				s := NewServerStateManager()
				callCount := 0
				roomIDs := []string{"ID001", "ID002", "ID003", "ID004", "ID005"}
				for _, id := range roomIDs {
					s.Rooms.Store(id, &MatchRoom{ID: id})
				}
				s.generateRoomID = func(int) string {
					if callCount < len(roomIDs) {
						id := roomIDs[callCount]
						callCount++
						return id
					}
					return "SHOULD_NOT_REACH"
				}
				return s
			},
			wantErr: true,
			validate: func(t *testing.T, s *ServerStateManager, id string, err error) {
				if id != "" {
					t.Errorf("Expected empty ID on error, got: %s", id)
				}
				for _, existing := range []string{"ID001", "ID002", "ID003", "ID004", "ID005"} {
					if _, ok := s.Rooms.Load(existing); !ok {
						t.Errorf("Predefined room %s missing", existing)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.setup()
			id, err := s.CreateMatchRoom()
			if (err != nil) != tt.wantErr {
				t.Fatalf("CreateMatchRoom() error = %v, wantErr %v", err, tt.wantErr)
			}
			tt.validate(t, s, id, err)
		})
	}
}

func isValidCrockfordCode(s string) bool {
	const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	for _, c := range s {
		if !strings.ContainsRune(alphabet, c) {
			return false
		}
	}
	return true
}

func TestServerStateManager_LastActivityUpdated(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (string, *ServerStateManager, [2]string)
		action   func(t *testing.T, s *ServerStateManager, roomID string, tokens [2]string)
		validate func(t *testing.T, s *ServerStateManager, roomID string)
	}{
		{
			name: "CreateMatch updates LastActivity",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				return roomID, s, [2]string{}
			},
			action: func(t *testing.T, s *ServerStateManager, roomID string, _ [2]string) {
				_, _ = s.CreateMatch(roomID, validGameCfg())
			},
		},
		{
			name: "SubmitTurnCommand updates LastActivity",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				roomID, tokens, s := createTestRoom(t)
				return roomID, s, tokens
			},
			action: func(t *testing.T, s *ServerStateManager, roomID string, tokens [2]string) {
				uID := engine.NewUnitID(1, 0)
				s.SubmitTurnCommand(roomID, engine.NewMoveCommand(uID, engine.Coordinate{X: 4, Y: 7}), tokens[0])
			},
		},
		{
			name: "StartTurn updates LastActivity",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				roomID, tokens, s := createTestRoom(t)
				return roomID, s, tokens
			},
			action: func(t *testing.T, s *ServerStateManager, roomID string, tokens [2]string) {
				s.StartTurn(roomID, tokens[0])
			},
		},
		{
			name: "ResetTurn updates LastActivity",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				roomID, tokens, s := createTestRoom(t)
				return roomID, s, tokens
			},
			action: func(t *testing.T, s *ServerStateManager, roomID string, tokens [2]string) {
				s.ResetTurn(roomID, tokens[0])
			},
		},
		{
			name: "ResolveTurn updates LastActivity",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				roomID, tokens, s := createTestRoom(t)
				s.SubmitTurnCommand(roomID, engine.NewPlaceBombCommand(16, engine.Coordinate{X: 4, Y: 7}), tokens[0])
				return roomID, s, tokens
			},
			action: func(t *testing.T, s *ServerStateManager, roomID string, tokens [2]string) {
				s.ResolveTurn(roomID, tokens[0])
			},
		},
		{
			name: "Surrender updates LastActivity",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				roomID, tokens, s := createTestRoom(t)
				return roomID, s, tokens
			},
			action: func(t *testing.T, s *ServerStateManager, roomID string, tokens [2]string) {
				s.Surrender(roomID, 1, tokens[0])
			},
			validate: func(t *testing.T, s *ServerStateManager, roomID string) {
				// Surrender doesn't delete the room, verify it's still here
				if _, ok := s.Rooms.Load(roomID); !ok {
					t.Error("Expected room not to be deleted after surrender")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s, tokens := tt.setup(t)
			before := time.Now()
			time.Sleep(10 * time.Millisecond)
			tt.action(t, s, roomID, tokens)

			if tt.validate != nil {
				tt.validate(t, s, roomID)
				return
			}

			roomVal, ok := s.Rooms.Load(roomID)
			if !ok {
				t.Fatal("Room not found")
			}
			room := roomVal.(*MatchRoom)
			room.mu.RLock()
			defer room.mu.RUnlock()
			if !room.LastActivity.After(before) {
				t.Errorf("LastActivity not updated: before=%v, after=%v", before, room.LastActivity)
			}
		})
	}
}

func TestServerStateManager_ReadOnlyMethodsDoNotUpdateLastActivity(t *testing.T) {
	roomID, _, s := createTestRoom(t)
	before := time.Now()
	time.Sleep(10 * time.Millisecond)

	s.GetMatchState(roomID)
	s.GetMatchConfig(roomID)
	s.GetAllowedTiles(roomID, engine.NewUnitID(1, 0), engine.TurnCmdPlaceBomb)

	roomVal, ok := s.Rooms.Load(roomID)
	if !ok {
		t.Fatal("Room not found")
	}
	room := roomVal.(*MatchRoom)
	room.mu.RLock()
	defer room.mu.RUnlock()
	if !room.LastActivity.Equal(before) && !room.LastActivity.Before(before) {
		t.Errorf("LastActivity should not be updated by read-only methods: before=%v, after=%v", before, room.LastActivity)
	}
}

func validGameCfg() engine.GameCfg {
	return engine.GameCfg{
		StagePreset: "MAP01",
		P1Teams:     []string{"King", "Fighter"},
		P2Teams:     []string{"King", "Witch"},
		MaxTurns:    10,
	}
}

func assertGameCfgSynced(t *testing.T, room *MatchRoom) {
	t.Helper()
	if room.GameCfg == nil {
		t.Fatal("Expected room.GameCfg to be set")
	}
	if !reflect.DeepEqual(*room.GameCfg, room.Match.GameCfg) {
		t.Errorf("Expected MatchRoom and Match to have equal GameCfg values, MatchRoom %+v vs Match %+v", *room.GameCfg, room.Match.GameCfg)
	}
}

func createTestRoom(t *testing.T) (string, [2]string, *ServerStateManager) {
	t.Helper()
	s := NewServerStateManager()
	roomID, _ := s.CreateMatchRoom()
	tokens, _ := s.CreateMatch(roomID, validGameCfg())
	return roomID, tokens, s
}

func TestMapError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
		wantMsg  string
	}{
		// Server errors
		{"room not found", ErrRoomNotFound, 404, "room not found"},
		{"match not found", ErrMatchNotFound, 404, "match not found"},
		{"match exists", ErrMatchExists, 409, "match already exists"},
		{"invalid config", ErrInvalidConfig, 400, "invalid game config"},
		{"invalid player token", ErrInvalidToken, 401, "invalid player token"},

		// Engine config errors (InitGame): 400
		{"invalid stage preset", engine.ErrInvalidStagePreset, 400, "invalid stage preset"},
		{"invalid team size", engine.ErrInvalidTeamSize, 400, "invalid team size"},
		{"missing king", engine.ErrMissingKing, 400, "missing king"},
		{"invalid stage layout", engine.ErrInvalidStageLayout, 400, "invalid stage layout"},
		{"invalid terrain", engine.ErrInvalidTerrain, 400, "invalid terrain"},
		{"unknown archetype", engine.ErrUnknownArchetype, 400, "unknown archetype"},

		// Engine gameplay errors: 409
		{"unit not found", engine.ErrUnitNotFound, 409, "unit not found"},
		{"unit dead", engine.ErrUnitDead, 409, "unit is dead"},
		{"not active team", engine.ErrNotActiveTeam, 409, "not active team"},
		{"already moved", engine.ErrAlreadyMoved, 409, "unit already moved this turn"},
		{"already used skill", engine.ErrAlreadyUsedSkill, 409, "unit already used skill this turn"},
		{"out of move range", engine.ErrOutOfMoveRange, 409, "target out of move range"},
		{"out of bomb range", engine.ErrOutOfBombRange, 409, "target out of bomb range"},
		{"cell occupied", engine.ErrCellOccupied, 409, "cell occupied"},
		{"out of bombs", engine.ErrOutOfBombs, 409, "out of bombs"},
		{"unsupported command", engine.ErrUnsupportedCommand, 409, "unsupported command type"},
		{"invalid landing", engine.ErrInvalidLanding, 409, "invalid landing position"},

		// Internal bugs: 409 (game rule violations)
		{"desynced", fmt.Errorf("%w: unit %#x desynced at %v", engine.ErrDesynced, 1, engine.Coordinate{}), 409, "desynced: unit 0x1 desynced at {0 0}"},
		{"out of bounds", fmt.Errorf("%w: unit %#x out of bounds", engine.ErrOutOfBounds, 1), 409, "out of bounds: unit 0x1 out of bounds"},

		// Unknown: 500
		{"unknown", fmt.Errorf("something else"), 500, "internal error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, msg := mapError(tt.err)
			if code != tt.wantCode || msg != tt.wantMsg {
				t.Errorf("got (%d, %q) want (%d, %q)", code, msg, tt.wantCode, tt.wantMsg)
			}
		})
	}
}

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
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				return roomID, s
			},
			gameCfg: validGameCfg(),
			wantErr: nil,
			validate: func(t *testing.T, s *ServerStateManager, roomID string) {
				roomVal, ok := s.Rooms.Load(roomID)
				if !ok {
					t.Fatal("Room not found")
				}
				room := roomVal.(*MatchRoom)
				if room.Match == nil {
					t.Fatal("Expected Match to be created, got nil")
				}
				if room.Match.GameCfg.StagePreset != "MAP01" {
					t.Errorf("Expected StagePreset 'MAP01', got '%s'", room.Match.GameCfg.StagePreset)
				}
				if room.Match.GameCfg.MaxTurns != 10 {
					t.Errorf("Expected MaxTurns 10, got %d", room.Match.GameCfg.MaxTurns)
				}
			},
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				s := NewServerStateManager()
				return "NONEXISTENT", s
			},
			gameCfg: validGameCfg(),
			wantErr: ErrRoomNotFound,
		},
		{
			name: "Match Already Exists",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				s.CreateMatch(roomID, validGameCfg())
				return roomID, s
			},
			gameCfg: validGameCfg(),
			wantErr: ErrMatchExists,
		},
		{
			name: "Invalid Config",
			setup: func(t *testing.T) (string, *ServerStateManager) {
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
				roomVal, _ := s.Rooms.Load(roomID)
				room := roomVal.(*MatchRoom)
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
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				tokens, _ := s.CreateMatch(roomID, validGameCfg())
				roomVal, _ := s.Rooms.Load(roomID)
				room := roomVal.(*MatchRoom)
				room.Match = nil // kill the match
				gameCfg := validGameCfg()
				room.GameCfg = &gameCfg
				return roomID, s, tokens
			},
			wantErr: nil,
		},
		{
			name: "Success - Wipe existing Match",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				tokens, _ := s.CreateMatch(roomID, validGameCfg())
				roomVal, _ := s.Rooms.Load(roomID)
				room := roomVal.(*MatchRoom)
				gameCfg := validGameCfg()
				room.GameCfg = &gameCfg
				return roomID, s, tokens
			},
			wantErr: nil,
		},
		{
			name: "Invalid token",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				s.CreateMatch(roomID, validGameCfg())
				roomVal, _ := s.Rooms.Load(roomID)
				room := roomVal.(*MatchRoom)
				gameCfg := validGameCfg()
				room.GameCfg = &gameCfg
				return roomID, s, [2]string{"INVALID_TOKEN", ""}
			},
			wantErr: ErrInvalidToken,
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				s := NewServerStateManager()
				return "NONEXISTENT", s, [2]string{}
			},
			wantErr: ErrRoomNotFound,
		},
		{
			name: "Failure - No previous match",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
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
				roomVal, ok := s.Rooms.Load(roomID)
				if !ok {
					t.Fatal("Room not found")
				}
				room := roomVal.(*MatchRoom)
				if room.Match == nil {
					t.Fatal("Expected Match to be created, got nil")
				}
				if room.Match.GameCfg.StagePreset != "MAP01" {
					t.Errorf("Expected StagePreset 'MAP01', got '%s'", room.Match.GameCfg.StagePreset)
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
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				tokens, _ := s.CreateMatch(roomID, validGameCfg())
				roomVal, _ := s.Rooms.Load(roomID)
				room := roomVal.(*MatchRoom)
				room.Match = nil // kill the match
				return roomID, s, tokens
			},
			wantErr: nil,
		},
		{
			name: "Success - Existing Match",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				tokens, _ := s.CreateMatch(roomID, validGameCfg())
				roomVal, _ := s.Rooms.Load(roomID)
				room := roomVal.(*MatchRoom)
				room.Match.WinnerTeamID = 1 // conclude the match
				return roomID, s, tokens
			},
			wantErr: nil,
		},
		{
			name: "Failure - Match still in progress",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
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
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				s.CreateMatch(roomID, validGameCfg())
				roomVal, _ := s.Rooms.Load(roomID)
				room := roomVal.(*MatchRoom)
				gameCfg := validGameCfg()
				room.GameCfg = &gameCfg
				return roomID, s, [2]string{"INVALID_TOKEN", ""}
			},
			wantErr: ErrInvalidToken,
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
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
				roomVal, ok := s.Rooms.Load(roomID)
				if !ok {
					t.Fatal("Room not found")
				}
				room := roomVal.(*MatchRoom)
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
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				s.CreateMatch(roomID, validGameCfg())
				return roomID, s
			},
			wantErr: nil,
			validate: func(t *testing.T, gs *engine.GameState, s *ServerStateManager, roomID string) {
				roomVal, ok := s.Rooms.Load(roomID)
				if !ok {
					t.Fatal("Room not found")
				}
				room := roomVal.(*MatchRoom)
				if gs != room.Match.WorkingState {
					t.Errorf("Expected matchState pointer %p, got %p", room.Match.WorkingState, gs)
				}
			},
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				return "NONEXISTENT", NewServerStateManager()
			},
			wantErr: ErrRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
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
				roomID, tokens, s := createTestRoom(t)
				uID := engine.NewUnitID(1, 0)
				newPos := engine.Coordinate{X: 4, Y: 7}
				cmd := engine.NewMoveCommand(uID, newPos)
				return roomID, s, cmd, tokens[0]
			},
			wantErr: nil,
			validate: func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string, cmd engine.TurnCommand) {
				roomVal, ok := s.Rooms.Load(roomID)
				if !ok {
					t.Fatal("Room not found")
				}
				room := roomVal.(*MatchRoom)
				uID := engine.NewUnitID(1, 0)
				newPos := engine.Coordinate{X: 4, Y: 7}
				if gotPos := room.Match.WorkingState.Units[uID].Position; gotPos != newPos {
					t.Errorf("Expected Unit %#X new position %#v, got %#v", uID, newPos, gotPos)
				}
				if len(gameEvents) != 1 {
					t.Errorf("expected 1 GameEvent returned, got %d", len(gameEvents))
				}
				resEvt := gameEvents[0]
				validFrom := engine.Coordinate{X: 4, Y: 8}
				if resEvt.Type != engine.GameEvtUnitMoved || resEvt.UnitID != uID || *resEvt.From != validFrom || *resEvt.To != newPos {
					t.Errorf("malformed UnitMoveEvent returned: %+v", resEvt)
				}
			},
		},
		{
			name: "Invalid TurnCommand (out of range)",
			setup: func(t *testing.T) (string, *ServerStateManager, engine.TurnCommand, string) {
				roomID, tokens, s := createTestRoom(t)
				uID := engine.NewUnitID(1, 0)
				newPos := engine.Coordinate{X: 4, Y: 7777}
				cmd := engine.NewMoveCommand(uID, newPos)
				return roomID, s, cmd, tokens[0]
			},
			wantErr: ErrInvalidTurnCmd,
			validate: func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string, cmd engine.TurnCommand) {
				roomVal, ok := s.Rooms.Load(roomID)
				if !ok {
					t.Fatal("Room not found")
				}
				room := roomVal.(*MatchRoom)
				uID := engine.NewUnitID(1, 0)
				if gotPos := room.Match.WorkingState.Units[uID].Position; gotPos.X == 4 && gotPos.Y == 7777 {
					t.Errorf("Expected Unit %#X didn't move", uID)
				}
				if len(gameEvents) > 0 {
					t.Errorf("Expected gameEvents to be empty, got %p", gameEvents)
				}
			},
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, engine.TurnCommand, string) {
				s := NewServerStateManager()
				uID := engine.NewUnitID(1, 0)
				newPos := engine.Coordinate{X: 4, Y: 7777}
				cmd := engine.NewMoveCommand(uID, newPos)
				return "NONEXISTENT", s, cmd, "dummy-token"
			},
			wantErr: ErrRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, engine.TurnCommand, string) {
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				uID := engine.NewUnitID(1, 0)
				newPos := engine.Coordinate{X: 4, Y: 7777}
				cmd := engine.NewMoveCommand(uID, newPos)
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
			} else {
				if len(gameEvents) > 0 {
					t.Errorf("Expected gameEvents to be empty, got %p", gameEvents)
				}
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
				roomID, tokens, s := createTestRoom(t)
				roomVal, _ := s.Rooms.Load(roomID)
				room := roomVal.(*MatchRoom)
				room.Match.TrueState.Turn = 1000
				room.Match.WorkingState.Turn = 1000
				return roomID, s, tokens[0]
			},
			wantErr: nil,
			validate: func(t *testing.T, inSuddenDeath bool, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string) {
				roomVal, ok := s.Rooms.Load(roomID)
				if !ok {
					t.Fatal("Room not found")
				}
				room := roomVal.(*MatchRoom)

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
				roomID, tokens, s := createTestRoom(t)
				roomVal, _ := s.Rooms.Load(roomID)
				room := roomVal.(*MatchRoom)
				room.Match.WinnerTeamID = 1
				return roomID, s, tokens[0]
			},
			wantErr: ErrMatchEnded,
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				s := NewServerStateManager()
				return "NONEXISTENT", s, "dummy-token"
			},
			wantErr: ErrRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
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
				roomID, tokens, s := createTestRoom(t)
				roomVal, _ := s.Rooms.Load(roomID)
				room := roomVal.(*MatchRoom)
				room.Match.WorkingState.Units[16].HasMoved = true
				return roomID, s, tokens[0]
			},
			wantErr: nil,
			validate: func(t *testing.T, s *ServerStateManager, roomID string) {
				roomVal, ok := s.Rooms.Load(roomID)
				if !ok {
					t.Fatal("Room not found")
				}
				room := roomVal.(*MatchRoom)

				if got, want := room.Match.WorkingState.Units[16].HasMoved, false; got != want {
					t.Errorf("Expected Unit %#X HasMoved reset to %v, got %v", 16, want, got)
				}
			},
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				s := NewServerStateManager()
				return "NONEXISTENT", s, "dummy-token"
			},
			wantErr: ErrRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
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
		validate func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string)
	}{
		{
			name: "Success",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				roomID, tokens, s := createTestRoom(t)
				s.SubmitTurnCommand(roomID, engine.NewPlaceBombCommand(16, engine.Coordinate{X: 4, Y: 7}), tokens[0])
				roomVal, _ := s.Rooms.Load(roomID)
				room := roomVal.(*MatchRoom)
				room.Match.WorkingState.Bombs[engine.NewBombID(1, 1, 16)].Countdown = 1
				return roomID, s, tokens[0]
			},
			wantErr: nil,
			validate: func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string) {
				roomVal, ok := s.Rooms.Load(roomID)
				if !ok {
					t.Fatal("Room not found")
				}
				room := roomVal.(*MatchRoom)

				if got, want := gameEvents, 6; len(got) != want {
					t.Errorf("Expected %d gameEvents returned, got %#v", want, got)
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
				s := NewServerStateManager()
				return "NONEXISTENT", s, "dummy-token"
			},
			wantErr: ErrRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
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
			gameEvents, err := s.ResolveTurn(roomID, token)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ResolveTurn() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, gameEvents, s, roomID)
			} else {
				if len(gameEvents) != 0 {
					t.Errorf("Expected gameEvents to be empty, got %#v", gameEvents)
				}
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
				roomID, tokens, s := createTestRoom(t)
				return roomID, s, tokens
			},
			req:     SurrenderRequest{TeamID: 1},
			wantErr: nil,
			validate: func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string) {
				if got, want := gameEvents, 1; len(got) != want {
					t.Errorf("Expected %d gameEvents returned, got %#v", want, got)
				}
				if got, want := gameEvents[0].WinnerTeamID, 2; got != want {
					t.Errorf("Expected gameEvent WinnerTeamID = %v, got %v", want, got)
				}
				// Room should not be deleted after surrender
				if _, ok := s.Rooms.Load(roomID); !ok {
					t.Error("Expected room not to be deleted after surrender")
				}
			},
		},
		{
			name: "Invalid SurrenderRequest",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				roomID, tokens, s := createTestRoom(t)
				return roomID, s, tokens
			},
			req:     SurrenderRequest{TeamID: 3},
			wantErr: ErrInvalidConfig,
			validate: func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string) {
				roomVal, ok := s.Rooms.Load(roomID)
				if !ok {
					t.Fatal("Room not found")
				}
				room := roomVal.(*MatchRoom)

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
				s := NewServerStateManager()
				return "NONEXISTENT", s, [2]string{"dummy", "dummy"}
			},
			req:     SurrenderRequest{TeamID: 1},
			wantErr: ErrRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
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
			} else {
				if len(gameEvents) != 0 {
					t.Errorf("Expected gameEvents to be empty, got %#v", gameEvents)
				}
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
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				s.CreateMatch(roomID, validGameCfg())
				return roomID, s
			},
			wantErr: nil,
			validate: func(t *testing.T, gameCfg *engine.GameCfg, s *ServerStateManager, roomID string) {
				roomVal, ok := s.Rooms.Load(roomID)
				if !ok {
					t.Fatal("Room not found")
				}
				room := roomVal.(*MatchRoom)
				if gameCfg != &room.Match.GameCfg {
					t.Errorf("Expected matchState pointer %p, got %p", &room.Match.GameCfg, gameCfg)
				}
			},
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				return "NONEXISTENT", NewServerStateManager()
			},
			wantErr: ErrRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
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
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				s.CreateMatch(roomID, validGameCfg())
				return roomID, s
			},
			unitID:      engine.NewUnitID(1, 0),
			turnCmdType: engine.TurnCmdPlaceBomb,
			wantErr:     nil,
			validate: func(t *testing.T, allowed []engine.Coordinate, s *ServerStateManager, roomID string) {
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
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				s.CreateMatch(roomID, validGameCfg())
				return roomID, s
			},
			unitID:      engine.NewUnitID(1, 0),
			turnCmdType: "invalid",
			wantErr:     engine.ErrUnsupportedCommand,
			validate:    func(t *testing.T, allowed []engine.Coordinate, s *ServerStateManager, roomID string) {},
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				return "NONEXISTENT", NewServerStateManager()
			},
			wantErr: ErrRoomNotFound,
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
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

func TestServerStateManager_cleanupInactiveRooms(t *testing.T) {
	s := NewServerStateManager()

	// Room 1: active (recent activity)
	roomID1, _ := s.CreateMatchRoom()
	s.CreateMatch(roomID1, validGameCfg())

	// Room 2: inactive (old LastActivity)
	roomID2, _ := s.CreateMatchRoom()
	s.CreateMatch(roomID2, validGameCfg())
	roomVal, _ := s.Rooms.Load(roomID2)
	room := roomVal.(*MatchRoom)
	room.mu.Lock()
	room.LastActivity = time.Now().Add(-12 * time.Minute)
	room.mu.Unlock()

	// Room 3: ended match
	roomID3, _ := s.CreateMatchRoom()
	s.CreateMatch(roomID3, validGameCfg())
	roomVal, _ = s.Rooms.Load(roomID3)
	room = roomVal.(*MatchRoom)
	room.mu.Lock()
	room.Match.WinnerTeamID = 1
	room.mu.Unlock()

	// Run cleanup
	s.cleanupInactiveRooms()

	// Verify
	_, ok1 := s.Rooms.Load(roomID1)
	_, ok2 := s.Rooms.Load(roomID2)
	_, ok3 := s.Rooms.Load(roomID3)

	if !ok1 {
		t.Error("active room should not be cleaned")
	}
	if ok2 {
		t.Error("inactive room should be cleaned")
	}
	if !ok3 {
		t.Error("ended but still active match room should not be cleaned")
	}
}

func TestServerStateManager_StartCleanupLoop_Cancellation(t *testing.T) {
	s := NewServerStateManager()
	ctx, cancel := context.WithCancel(context.Background())

	s.StartCleanupLoop(ctx, 10*time.Millisecond)
	time.Sleep(25 * time.Millisecond) // let it tick a couple times
	cancel()
	time.Sleep(10 * time.Millisecond) // let goroutine exit

	// No panic/leak = success
}

func TestServerStateManager_WithLoggerOption(t *testing.T) {
	customLogger := slog.New(slog.NewTextHandler(&testLogWriter{t: t}, nil))
	s := NewServerStateManager(WithLogger(customLogger))
	if s.Logger != customLogger {
		t.Errorf("WithLogger option not applied")
	}
}

func TestHandler_WithLoggerOption(t *testing.T) {
	customLogger := slog.New(slog.NewTextHandler(&testLogWriter{t: t}, nil))
	s := NewServerStateManager()
	h := NewHandler(s, WithHandlerLogger(customLogger))
	if h.Logger != customLogger {
		t.Errorf("WithHandlerLogger option not applied")
	}
}

// testLogger returns a logger that writes to t.Log (only visible on failure or -v).
func testLogger(t *testing.T) *slog.Logger {
	return slog.New(slog.NewTextHandler(&testLogWriter{t: t}, nil))
}

type testLogWriter struct {
	t *testing.T
}

func (w *testLogWriter) Write(p []byte) (int, error) {
	w.t.Log(string(bytes.TrimSpace(p)))
	return len(p), nil
}
