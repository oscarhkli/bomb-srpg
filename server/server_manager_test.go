package server

import (
	"bomb-srpg/engine"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"sync"
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
				if len(id) != 10 {
					t.Errorf("Expected ID length 10, got %d: %s", len(id), id)
				}
				room := mustRoom(t, s, id)
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
				if len(id) != 10 {
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
				s.generateRoomID = func(int) (string, error) {
					if callCount < len(roomIDs) {
						id := roomIDs[callCount]
						callCount++
						return id, nil
					}
					return "SHOULD_NOT_REACH", nil
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
		{
			name: "Error thrown during generateRoomID",
			setup: func() *ServerStateManager {
				s := NewServerStateManager()
				s.generateRoomID = func(int) (string, error) {
					return "", errors.New("bad things happened")
				}
				return s
			},
			wantErr: true,
			validate: func(t *testing.T, s *ServerStateManager, id string, err error) {
				if id != "" {
					t.Errorf("Expected empty ID on error, got: %s", id)
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

// dropped in RoomID but will be used in RoomKey in online mtulti-player mode phase
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
				unitID := engine.NewUnitID(1, 0)
				s.SubmitTurnCommand(roomID, engine.NewMoveCommand(unitID, engine.Coordinate{X: 4, Y: 7}), tokens[0])
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
			name: "runCPUTurn updates LastActivity",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				roomID, tokens, s := createTestRoomWithCfg(t, vsCpuGameCfg())
				return roomID, s, tokens
			},
			action: func(t *testing.T, s *ServerStateManager, roomID string, tokens [2]string) {
				room := mustRoom(t, s, roomID)
				s.runCPUTurn(room, room.Match)
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

			room := mustRoom(t, s, roomID)
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

	room := mustRoom(t, s, roomID)
	room.mu.RLock()
	defer room.mu.RUnlock()
	if !room.LastActivity.Equal(before) && !room.LastActivity.Before(before) {
		t.Errorf("LastActivity should not be updated by read-only methods: before=%v, after=%v", before, room.LastActivity)
	}
}

func validGameCfg() engine.GameCfg {
	return engine.GameCfg{
		StagePreset: "Plain",
		P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
		P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
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

func createTestRoomWithCfg(t *testing.T, gameCfg engine.GameCfg) (string, [2]string, *ServerStateManager) {
	t.Helper()
	s := NewServerStateManager()
	roomID, _ := s.CreateMatchRoom()
	tokens, _ := s.CreateMatch(roomID, gameCfg)
	return roomID, tokens, s
}

func createTestRoom(t *testing.T) (string, [2]string, *ServerStateManager) {
	t.Helper()
	return createTestRoomWithCfg(t, validGameCfg())
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
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				tokens, _ := s.CreateMatch(roomID, validGameCfg())
				room := mustRoom(t, s, roomID)
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
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				s.CreateMatch(roomID, validGameCfg())
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
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				tokens, _ := s.CreateMatch(roomID, validGameCfg())
				room := mustRoom(t, s, roomID)
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
				room := mustRoom(t, s, roomID)
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
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				s.CreateMatch(roomID, validGameCfg())
				return roomID, s
			},
			wantErr: nil,
			validate: func(t *testing.T, gs *engine.GameState, s *ServerStateManager, roomID string) {
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
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				s.CreateMatch(roomID, validGameCfg())
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
				unitID := engine.NewUnitID(1, 0)
				newPos := engine.Coordinate{X: 4, Y: 7}
				cmd := engine.NewMoveCommand(unitID, newPos)
				return roomID, s, cmd, tokens[0]
			},
			wantErr: nil,
			validate: func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string, cmd engine.TurnCommand) {
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
				roomID, tokens, s := createTestRoom(t)
				unitID := engine.NewUnitID(1, 0)
				newPos := engine.Coordinate{X: 4, Y: 7777}
				cmd := engine.NewMoveCommand(unitID, newPos)
				return roomID, s, cmd, tokens[0]
			},
			wantErr: ErrInvalidTurnCmd,
			validate: func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string, cmd engine.TurnCommand) {
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
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, engine.TurnCommand, string) {
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
				room := mustRoom(t, s, roomID)
				room.Match.TrueState.Turn = 1000
				room.Match.WorkingState.Turn = 1000
				return roomID, s, tokens[0]
			},
			wantErr: nil,
			validate: func(t *testing.T, inSuddenDeath bool, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string) {
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

func TestApplyPlan(t *testing.T) {
	moverUnitID := engine.NewUnitID(1, 0)
	bomberUnitID := engine.NewUnitID(1, 1)

	tests := []struct {
		name     string
		plan     func(t *testing.T, m *engine.Match) []engine.TurnCommand
		wantErr  error
		validate func(t *testing.T, m *engine.Match)
	}{
		{
			name: "Empty Plan",
			plan: func(t *testing.T, m *engine.Match) []engine.TurnCommand { return nil },
			validate: func(t *testing.T, m *engine.Match) {
				if got := len(m.PlaybackLog); got != 0 {
					t.Errorf("Expected no events logged, got %d", got)
				}
			},
		},
		{
			name: "Full Plan Applied",
			plan: func(t *testing.T, m *engine.Match) []engine.TurnCommand {
				return []engine.TurnCommand{
					engine.NewMoveCommand(moverUnitID, engine.Coordinate{X: 4, Y: 7}),
					engine.NewPlaceBombCommand(bomberUnitID, engine.Coordinate{X: 3, Y: 7}),
				}
			},
			validate: func(t *testing.T, m *engine.Match) {
				if !hasGameEvent(m.PlaybackLog, engine.GameEvtUnitMoved, moverUnitID) {
					t.Errorf("Expected unit %#x to have moved, got %#v", moverUnitID, m.PlaybackLog)
				}
				if !hasGameEvent(m.PlaybackLog, engine.GameEvtBombPlaced, bomberUnitID) {
					t.Errorf("Expected unit %#x to have placed a bomb, got %#v", bomberUnitID, m.PlaybackLog)
				}
			},
		},
		{
			name: "Stops At First Rejection",
			plan: func(t *testing.T, m *engine.Match) []engine.TurnCommand {
				return []engine.TurnCommand{
					engine.NewMoveCommand(moverUnitID, engine.Coordinate{X: 99, Y: 99}),
					engine.NewPlaceBombCommand(bomberUnitID, engine.Coordinate{X: 3, Y: 7}),
				}
			},
			wantErr: engine.ErrOutOfMoveRange,
			validate: func(t *testing.T, m *engine.Match) {
				if hasGameEvent(m.PlaybackLog, engine.GameEvtBombPlaced, bomberUnitID) {
					t.Errorf("Expected command after the rejected one to be skipped, got %#v", m.PlaybackLog)
				}
			},
		},
		{
			name: "Unsupported Command",
			plan: func(t *testing.T, m *engine.Match) []engine.TurnCommand {
				return []engine.TurnCommand{{UnitID: moverUnitID, Target: engine.Coordinate{X: 1, Y: 1}}}
			},
			wantErr: engine.ErrUnsupportedCommand,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, _, s := createTestRoom(t)
			match := mustRoom(t, s, roomID).Match

			err := applyPlan(match, tt.plan(t, match))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("applyPlan() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, match)
			}
		})
	}
}

// recordingDecider is an injectable cpu.Decide stand-in that counts invocations and
// samples CPU.Phase from inside the CPU goroutine, where the room write lock is held.
type recordingDecider struct {
	match         *engine.Match
	plan          func(gs *engine.GameState, call int) []engine.TurnCommand
	calls         int
	observedPhase engine.CPUTurnPhase
	observedBombs int
	called        chan struct{}
}

func (d *recordingDecider) decide(gs *engine.GameState) []engine.TurnCommand {
	call := d.calls
	if call == 0 {
		d.observedPhase = d.match.CPU.Phase
		d.observedBombs = len(gs.Bombs)
	}
	d.calls++

	select {
	case d.called <- struct{}{}:
	default:
	}

	if d.plan == nil {
		return nil
	}
	return d.plan(gs, call)
}

func TestServerStateManager_StartTurn_CPUTurn(t *testing.T) {
	humanUnitID := engine.NewUnitID(1, 0)
	cpuUnitID := engine.NewUnitID(2, 0)

	legalCPUMove := []engine.TurnCommand{engine.NewMoveCommand(cpuUnitID, engine.Coordinate{X: 4, Y: 1})}
	rejectedPlan := []engine.TurnCommand{engine.NewMoveCommand(humanUnitID, engine.Coordinate{X: 1, Y: 1})}

	// Opens the CPU's turn: hands the board to team 2 the way a resolved human turn would.
	cpuTurnPending := func(t *testing.T, room *MatchRoom) int {
		t.Helper()
		room.Match.WorkingState.Turn = 2
		room.Match.WorkingState.ActiveTeam = 2
		room.Match.TrueState = room.Match.WorkingState.DeepCopy()
		return 1
	}

	tests := []struct {
		name           string
		vsCpu          bool
		maxTurns       int
		setup          func(t *testing.T, room *MatchRoom) int
		plan           func(gs *engine.GameState, call int) []engine.TurnCommand
		wantCPUTurn    bool
		wantCalls      int
		wantActiveTeam int
		validate       func(t *testing.T, room *MatchRoom, d *recordingDecider)
	}{
		{
			name:  "Plan Applied",
			vsCpu: true,
			setup: cpuTurnPending,
			plan: func(gs *engine.GameState, call int) []engine.TurnCommand {
				return legalCPUMove
			},
			wantCPUTurn:    true,
			wantCalls:      1,
			wantActiveTeam: 1,
			validate: func(t *testing.T, room *MatchRoom, d *recordingDecider) {
				if got, want := d.observedPhase, engine.TurnPhasePlanning; got != want {
					t.Errorf("Expected phase %v while planning, got %v", want, got)
				}
				if !hasGameEvent(room.Match.CPU.PlanGameEvents, engine.GameEvtUnitMoved, cpuUnitID) {
					t.Errorf("Expected unit %#x to have moved, got %#v", cpuUnitID, room.Match.CPU.PlanGameEvents)
				}
			},
		},
		{
			name:  "Empty Plan",
			vsCpu: true,
			setup: cpuTurnPending,
			plan: func(gs *engine.GameState, call int) []engine.TurnCommand {
				return nil
			},
			wantCPUTurn:    true,
			wantCalls:      1,
			wantActiveTeam: 1,
			validate: func(t *testing.T, room *MatchRoom, d *recordingDecider) {
				if got := len(room.Match.CPU.PlanGameEvents); got != 0 {
					t.Errorf("Expected no pending events, got %d", got)
				}
			},
		},
		{
			name:  "Replan On Rejection",
			vsCpu: true,
			setup: cpuTurnPending,
			plan: func(gs *engine.GameState, call int) []engine.TurnCommand {
				if call == 0 {
					return rejectedPlan
				}
				return legalCPUMove
			},
			wantCPUTurn:    true,
			wantCalls:      2,
			wantActiveTeam: 1,
			validate: func(t *testing.T, room *MatchRoom, d *recordingDecider) {
				if !hasGameEvent(room.Match.CPU.PlanGameEvents, engine.GameEvtUnitMoved, cpuUnitID) {
					t.Errorf("Expected replanned move to apply, got %#v", room.Match.CPU.PlanGameEvents)
				}
			},
		},
		{
			name:  "Replan Cap Exhausted",
			vsCpu: true,
			setup: cpuTurnPending,
			plan: func(gs *engine.GameState, call int) []engine.TurnCommand {
				return rejectedPlan
			},
			wantCPUTurn:    true,
			wantCalls:      maxCPUReplanAttempts,
			wantActiveTeam: 1,
			validate: func(t *testing.T, room *MatchRoom, d *recordingDecider) {
				if got := len(room.Match.CPU.PlanGameEvents); got != 0 {
					t.Errorf("Expected no command to apply, got %#v", room.Match.CPU.PlanGameEvents)
				}
			},
		},
		{
			name:     "Sudden Death Injected Before Planning",
			vsCpu:    true,
			maxTurns: 1,
			setup:    cpuTurnPending,
			plan: func(gs *engine.GameState, call int) []engine.TurnCommand {
				return nil
			},
			wantCPUTurn:    true,
			wantCalls:      1,
			wantActiveTeam: 1,
			validate: func(t *testing.T, room *MatchRoom, d *recordingDecider) {
				if got, want := d.observedBombs, engine.SuddenDeathBombs; got < want {
					t.Errorf("Expected at least %d hazard bombs on the board when planning, got %d", want, got)
				}
			},
		},
		{
			name:  "Unconsumed Ready Phase Relaunches",
			vsCpu: true,
			setup: func(t *testing.T, room *MatchRoom) int {
				tokenIdx := cpuTurnPending(t, room)
				room.Match.CPU.Phase = engine.TurnPhaseReady
				room.Match.CPU.ResolveTurnGameEvents = []engine.GameEvent{engine.NewMatchEndedEvent(1)}
				room.Match.CPU.PlanGameEvents = []engine.GameEvent{engine.NewUnitMovedEvent(cpuUnitID, engine.Coordinate{X: 0, Y: 0}, engine.Coordinate{X: 1, Y: 0})}
				return tokenIdx
			},
			plan: func(gs *engine.GameState, call int) []engine.TurnCommand {
				return legalCPUMove
			},
			wantCPUTurn:    true,
			wantCalls:      1,
			wantActiveTeam: 1,
			validate: func(t *testing.T, room *MatchRoom, d *recordingDecider) {
				if hasGameEvent(room.Match.CPU.ResolveTurnGameEvents, engine.GameEvtMatchEnded, 0) {
					t.Errorf("Expected the stale event dropped, got %#v", room.Match.CPU.ResolveTurnGameEvents)
				}
				if !hasGameEvent(room.Match.CPU.PlanGameEvents, engine.GameEvtUnitMoved, cpuUnitID) {
					t.Errorf("Expected unit %#x to have moved, got %#v", cpuUnitID, room.Match.CPU.PlanGameEvents)
				}
			},
		},
		{
			name:           "Not VS CPU",
			vsCpu:          false,
			setup:          cpuTurnPending,
			wantCPUTurn:    false,
			wantCalls:      0,
			wantActiveTeam: 2,
		},
		{
			name:           "Human Turn",
			vsCpu:          true,
			wantCPUTurn:    false,
			wantCalls:      0,
			wantActiveTeam: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gameCfg := validGameCfg()
			gameCfg.VSCpu = tt.vsCpu
			if tt.maxTurns != 0 {
				gameCfg.MaxTurns = tt.maxTurns
			}
			roomID, tokens, s := createTestRoomWithCfg(t, gameCfg)
			room := mustRoom(t, s, roomID)

			tokenIdx := 0
			if tt.setup != nil {
				tokenIdx = tt.setup(t, room)
			}

			decider := &recordingDecider{match: room.Match, plan: tt.plan, called: make(chan struct{}, 1)}
			s.decide = decider.decide

			if _, _, err := s.StartTurn(roomID, tokens[tokenIdx]); err != nil {
				t.Fatalf("StartTurn() error = %v", err)
			}

			if tt.wantCPUTurn {
				select {
				case <-decider.called:
				case <-time.After(5 * time.Second):
					t.Fatal("Timed out waiting for the CPU goroutine to invoke decide")
				}
			}

			// Blocks until runCPUTurn releases the write lock, ordering every assertion below after it.
			room.mu.RLock()
			defer room.mu.RUnlock()

			if got, want := decider.calls, tt.wantCalls; got != want {
				t.Errorf("Expected %d decide calls, got %d", want, got)
			}

			wantPhase := engine.TurnPhaseIdle
			if tt.wantCPUTurn {
				wantPhase = engine.TurnPhaseReady
			}
			if got := room.Match.CPU.Phase; got != wantPhase {
				t.Errorf("Expected CPU phase %v, got %v", wantPhase, got)
			}
			if got, want := room.Match.TrueState.ActiveTeam, tt.wantActiveTeam; got != want {
				t.Errorf("Expected active team %d, got %d", want, got)
			}
			if tt.validate != nil {
				tt.validate(t, room, decider)
			}
		})
	}
}

func TestServerStateManager_runCPUTurn_Abandons(t *testing.T) {
	tests := []struct {
		name      string
		swap      func(t *testing.T, room *MatchRoom)
		wantPhase engine.CPUTurnPhase
	}{
		{
			name: "Match Replaced",
			swap: func(t *testing.T, room *MatchRoom) {
				replacement, err := engine.InitGame(vsCpuGameCfg())
				if err != nil {
					t.Fatalf("InitGame() error = %v", err)
				}
				room.Match = replacement
			},
			wantPhase: engine.TurnPhasePlanning,
		},
		{
			name:      "Match Deleted",
			swap:      func(t *testing.T, room *MatchRoom) { room.Match = nil },
			wantPhase: engine.TurnPhasePlanning,
		},
		{
			name:      "Match Already Ended",
			swap:      func(t *testing.T, room *MatchRoom) { room.Match.WinnerTeamID = 1 },
			wantPhase: engine.TurnPhaseReady,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, _, s := createTestRoomWithCfg(t, vsCpuGameCfg())
			room := mustRoom(t, s, roomID)

			captured := room.Match
			captured.CPU.Phase = engine.TurnPhasePlanning
			decider := &recordingDecider{match: captured, called: make(chan struct{}, 1)}
			s.decide = decider.decide

			tt.swap(t, room)
			s.runCPUTurn(room, captured)

			if got := decider.calls; got != 0 {
				t.Errorf("Expected the abandoned turn to skip planning, got %d decide calls", got)
			}
			if got, want := captured.CPU.Phase, tt.wantPhase; got != want {
				t.Errorf("Expected phase %v on the abandoned match, got %v", want, got)
			}
			if got, want := captured.TrueState.Turn, 1; got != want {
				t.Errorf("Expected turn %d on the abandoned match, got %d", want, got)
			}
			if room.Match != nil && room.Match != captured && room.Match.CPU.Phase != engine.TurnPhaseIdle {
				t.Errorf("Expected the replacement match untouched, got phase %v", room.Match.CPU.Phase)
			}
		})
	}
}

// Readers must not touch the match while the CPU goroutine mutates it. Meaningful under -race.
func TestServerStateManager_ReadsDuringCPUTurn(t *testing.T) {
	cpuUnitID := engine.NewUnitID(2, 0)

	roomID, _, s := createTestRoomWithCfg(t, vsCpuGameCfg())
	room := mustRoom(t, s, roomID)
	room.Match.WorkingState.ActiveTeam = 2
	room.Match.CPU.Phase = engine.TurnPhasePlanning
	match := room.Match

	planning, release := make(chan struct{}), make(chan struct{})
	s.decide = func(*engine.GameState) []engine.TurnCommand {
		close(planning)
		<-release
		return nil
	}

	cpuDone := make(chan struct{})
	go func() {
		defer close(cpuDone)
		s.runCPUTurn(room, match)
	}()

	<-planning

	stop := make(chan struct{})
	var readers sync.WaitGroup
	for range 4 {
		readers.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
				}
				gs, err := s.GetMatchState(roomID)
				if err != nil {
					t.Errorf("GetMatchState() error = %v", err)
					return
				}
				// Consume it the way HandleGetMatchState does, or nothing racy is ever touched.
				if _, err := json.Marshal(gs); err != nil {
					t.Errorf("Marshal(gameState) error = %v", err)
					return
				}
				s.GetAllowedTiles(roomID, cpuUnitID, engine.TurnCmdMove)
				s.GetMatchConfig(roomID)
			}
		})
	}

	close(release)
	<-cpuDone
	close(stop)
	readers.Wait()

	if got, want := match.CPU.Phase, engine.TurnPhaseReady; got != want {
		t.Errorf("Expected phase %v after the CPU turn, got %v", want, got)
	}
}

func TestServerStateManager_runCPUTurn_Panic(t *testing.T) {
	roomID, tokens, s := createTestRoomWithCfg(t, vsCpuGameCfg())
	room := mustRoom(t, s, roomID)
	room.Match.WorkingState.ActiveTeam = 2
	room.Match.CPU.Phase = engine.TurnPhasePlanning
	match := room.Match

	s.decide = func(*engine.GameState) []engine.TurnCommand { panic("decide exploded") }

	s.runCPUTurn(room, match)

	if got, want := match.CPU.Phase, engine.TurnPhaseReady; got != want {
		t.Errorf("Expected phase %v after the panic, got %v", want, got)
	}
	if got, want := match.TrueState.Turn, 2; got != want {
		t.Errorf("Expected the forfeited turn to still advance to %d, got %d", want, got)
	}

	// Proves the room mutex was released rather than left held by the unwinding goroutine.
	turnPhase, _, _, err := s.ConsumeCPUStatus(roomID, tokens[0])
	if err != nil {
		t.Fatalf("ConsumeCPUStatus() error = %v", err)
	}
	if got, want := turnPhase, engine.TurnPhaseReady; got != want {
		t.Errorf("Expected %v turnPhase return, got %v", want, got)
	}
}

func TestServerStateManager_ConsumeCPUStatus(t *testing.T) {
	unitID := engine.NewUnitID(2, 0)
	tests := []struct {
		name     string
		setup    func(t *testing.T) (string, *ServerStateManager, string)
		wantErr  error
		validate func(t *testing.T, turnPhase engine.CPUTurnPhase, planGameEvents, resolveTurnGameEvents []engine.GameEvent, s *ServerStateManager, roomID string)
	}{
		{
			name: "Success - TurnPhaseReady and clear status",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				roomID, tokens, s := createTestRoom(t)
				room := mustRoom(t, s, roomID)
				room.Match.CPU.Phase = engine.TurnPhaseReady
				room.Match.CPU.PlanGameEvents = []engine.GameEvent{
					engine.NewUnitMovedEvent(unitID, engine.Coordinate{X: 1, Y: 2}, engine.Coordinate{X: 2, Y: 2}),
				}
				room.Match.CPU.ResolveTurnGameEvents = []engine.GameEvent{
					engine.NewUnitDiedEvent(unitID),
				}
				return roomID, s, tokens[0]
			},
			wantErr: nil,
			validate: func(t *testing.T, turnPhase engine.CPUTurnPhase, planGameEvents, resolveTurnGameEvents []engine.GameEvent, s *ServerStateManager, roomID string) {
				room := mustRoom(t, s, roomID)

				if got, want := turnPhase, engine.TurnPhaseReady; got != want {
					t.Errorf("Expected %v turnPhase return, got %v", want, got)
				}
				if !hasGameEvent(planGameEvents, engine.GameEvtUnitMoved, unitID) {
					t.Errorf("Expected the move in planGameEvents, got %#v", planGameEvents)
				}
				if !hasGameEvent(resolveTurnGameEvents, engine.GameEvtUnitDied, unitID) {
					t.Errorf("Expected the death in resolveTurnGameEvents, got %#v", resolveTurnGameEvents)
				}

				if got, want := room.Match.CPU.Phase, engine.TurnPhaseIdle; got != want {
					t.Errorf("Expected CPU.Phase.TurnPhase reset to %v, got %v", want, got)
				}
				if got, want := len(room.Match.CPU.PlanGameEvents), 0; got != want {
					t.Errorf("Expected CPU.PlanGameEvents cleared, got %#v", room.Match.CPU.PlanGameEvents)
				}
				if got, want := len(room.Match.CPU.ResolveTurnGameEvents), 0; got != want {
					t.Errorf("Expected CPU.ResolveTurnGameEvents cleared, got %#v", room.Match.CPU.ResolveTurnGameEvents)
				}
			},
		},
		{
			name: "Success - non-TurnPhaseReady and do nothing",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				roomID, tokens, s := createTestRoom(t)
				room := mustRoom(t, s, roomID)
				room.Match.CPU.Phase = engine.TurnPhasePlanning
				room.Match.CPU.ResolveTurnGameEvents = []engine.GameEvent{
					engine.NewUnitMovedEvent(unitID, engine.Coordinate{X: 1, Y: 2}, engine.Coordinate{X: 2, Y: 2}),
				}
				return roomID, s, tokens[0]
			},
			wantErr: nil,
			validate: func(t *testing.T, turnPhase engine.CPUTurnPhase, planGameEvents, resolveTurnGameEvents []engine.GameEvent, s *ServerStateManager, roomID string) {
				room := mustRoom(t, s, roomID)

				if got, want := turnPhase, engine.TurnPhasePlanning; got != want {
					t.Errorf("Expected %v turnPhase return, got %v", want, got)
				}
				if got, want := len(resolveTurnGameEvents), 1; got != want {
					t.Errorf("Expected %d resolveTurnGameEvents returned, got %#v", want, resolveTurnGameEvents)
				}

				if got, want := room.Match.CPU.Phase, engine.TurnPhasePlanning; got != want {
					t.Errorf("Expected CPU.Phase.TurnPhase stays at %v, got %v", want, got)
				}
				if got, want := len(room.Match.CPU.ResolveTurnGameEvents), 1; got != want {
					t.Errorf("Expected CPU.ResolveTurnGameEvents remain unchanged as %d, got %#v", want, room.Match.CPU.ResolveTurnGameEvents)
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
			turnPhase, planGameEvents, resolveTurnGameEvents, err := s.ConsumeCPUStatus(roomID, token)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ConsumeCPUStatus() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, turnPhase, planGameEvents, resolveTurnGameEvents, s, roomID)
			} else {
				if len(planGameEvents) != 0 || len(resolveTurnGameEvents) != 0 {
					t.Errorf("Expected gameEvents to be empty, got %#v and %#v", planGameEvents, resolveTurnGameEvents)
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
				room := mustRoom(t, s, roomID)
				room.Match.WorkingState.Units[16].HasMoved = true
				return roomID, s, tokens[0]
			},
			wantErr: nil,
			validate: func(t *testing.T, s *ServerStateManager, roomID string) {
				room := mustRoom(t, s, roomID)

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
		validate func(t *testing.T, planGameEvents, resolveTurnGameEvents []engine.GameEvent, s *ServerStateManager, roomID string)
	}{
		{
			name: "Success",
			setup: func(t *testing.T) (string, *ServerStateManager, string) {
				roomID, tokens, s := createTestRoom(t)
				s.SubmitTurnCommand(roomID, engine.NewPlaceBombCommand(16, engine.Coordinate{X: 4, Y: 7}), tokens[0])
				room := mustRoom(t, s, roomID)
				room.Match.WorkingState.Bombs[engine.NewBombID(1, 1, 16)].Countdown = 1
				return roomID, s, tokens[0]
			},
			wantErr: nil,
			validate: func(t *testing.T, planGameEvents, resolveTurnGameEvents []engine.GameEvent, s *ServerStateManager, roomID string) {
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
			planGameEvents, resolveTurnGameEvents, err := s.ResolveTurn(roomID, token)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ResolveTurn() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, planGameEvents, resolveTurnGameEvents, s, roomID)
			} else {
				if len(planGameEvents) != 0 || len(resolveTurnGameEvents) != 0 {
					t.Errorf("Expected gameEvents to be empty, got %#v and %#v", planGameEvents, resolveTurnGameEvents)
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
	room := mustRoom(t, s, roomID2)
	room.mu.Lock()
	room.LastActivity = time.Now().Add(-120 * time.Minute)
	room.mu.Unlock()

	// Room 3: ended match
	roomID3, _ := s.CreateMatchRoom()
	s.CreateMatch(roomID3, validGameCfg())
	room = mustRoom(t, s, roomID3)
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

func vsCpuGameCfg() engine.GameCfg {
	gameCfg := validGameCfg()
	gameCfg.VSCpu = true
	return gameCfg
}

func mustRoom(t *testing.T, s *ServerStateManager, roomID string) *MatchRoom {
	t.Helper()
	roomVal, ok := s.Rooms.Load(roomID)
	if !ok {
		t.Fatalf("Room %s not found", roomID)
	}
	return roomVal.(*MatchRoom)
}

func hasGameEvent(gameEvents []engine.GameEvent, evtType engine.GameEvtType, unitID engine.UnitID) bool {
	return slices.ContainsFunc(gameEvents, func(evt engine.GameEvent) bool {
		return evt.Type == evtType && evt.UnitID == unitID
	})
}
