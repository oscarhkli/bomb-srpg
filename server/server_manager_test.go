package server

import (
	"bomb-srpg/engine"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)

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
				room, ok := s.Rooms[id]
				if !ok {
					t.Fatal("Room not added to manager.Rooms")
				}
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
				s.Rooms[existingID] = &MatchRoom{ID: existingID}
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
				if _, ok := s.Rooms["ABCDE"]; !ok {
					t.Error("Predefined room missing")
				}
				if _, ok := s.Rooms[id]; !ok {
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
					s.Rooms[id] = &MatchRoom{ID: id}
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
					if _, ok := s.Rooms[existing]; !ok {
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

func validGameCfg() engine.GameCfg {
	return engine.GameCfg{
		StagePreset: "MAP01",
		P1Teams:     []string{"King", "Fighter"},
		P2Teams:     []string{"King", "Witch"},
		MaxTurns:    10,
		SuddenDeath: true,
	}
}

func createTestRoom(t *testing.T) (string, *ServerStateManager) {
	t.Helper()
	s := NewServerStateManager()
	roomID, _ := s.CreateMatchRoom()
	s.CreateMatch(roomID, validGameCfg())
	return roomID, s
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
				room, ok := s.Rooms[roomID]
				if !ok {
					t.Fatal("Room not found")
				}
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
			gameCfg:  validGameCfg(),
			wantErr:  ErrRoomNotFound,
			validate: func(t *testing.T, s *ServerStateManager, roomID string) {},
		},
		{
			name: "Match Already Exists",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				s.CreateMatch(roomID, validGameCfg())
				return roomID, s
			},
			gameCfg:  validGameCfg(),
			wantErr:  ErrMatchExists,
			validate: func(t *testing.T, s *ServerStateManager, roomID string) {},
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
			wantErr:  ErrInvalidConfig,
			validate: func(t *testing.T, s *ServerStateManager, roomID string) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s := tt.setup(t)
			err := s.CreateMatch(roomID, tt.gameCfg)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CreateMatch() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, s, roomID)
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
				room, ok := s.Rooms[roomID]
				if !ok {
					t.Fatal("Room not found")
				}
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
			wantErr:  ErrRoomNotFound,
			validate: func(t *testing.T, gs *engine.GameState, s *ServerStateManager, roomID string) {},
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				return roomID, s
			},
			wantErr:  ErrMatchNotFound,
			validate: func(t *testing.T, gs *engine.GameState, s *ServerStateManager, roomID string) {},
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
		setup    func(t *testing.T) (string, *ServerStateManager, engine.TurnCommand)
		wantErr  error
		validate func(t *testing.T, gs *engine.GameState, s *ServerStateManager, roomID string, cmd engine.TurnCommand)
	}{
		{
			name: "Success",
			setup: func(t *testing.T) (string, *ServerStateManager, engine.TurnCommand) {
				roomID, s := createTestRoom(t)
				uID := engine.NewUnitID(1, 0)
				newPos := engine.Coordinate{X: 4, Y: 7}
				cmd := engine.NewMoveCommand(uID, newPos)
				return roomID, s, cmd
			},
			wantErr: nil,
			validate: func(t *testing.T, gs *engine.GameState, s *ServerStateManager, roomID string, cmd engine.TurnCommand) {
				room, ok := s.Rooms[roomID]
				if !ok {
					t.Fatal("Room not found")
				}
				uID := engine.NewUnitID(1, 0)
				newPos := engine.Coordinate{X: 4, Y: 7}
				if gotPos := room.Match.WorkingState.Units[uID].Position; gotPos != newPos {
					t.Errorf("Expected Unit %#X new position %#v, got %#v", uID, newPos, gotPos)
				}
				if gs != room.Match.WorkingState {
					t.Errorf("Expected matchState pointer %p, got %p", room.Match.WorkingState, gs)
				}
			},
		},
		{
			name: "Invalid TurnCommand (out of range)",
			setup: func(t *testing.T) (string, *ServerStateManager, engine.TurnCommand) {
				roomID, s := createTestRoom(t)
				uID := engine.NewUnitID(1, 0)
				newPos := engine.Coordinate{X: 4, Y: 7777}
				cmd := engine.NewMoveCommand(uID, newPos)
				return roomID, s, cmd
			},
			wantErr: ErrInvalidTurnCmd,
			validate: func(t *testing.T, gs *engine.GameState, s *ServerStateManager, roomID string, cmd engine.TurnCommand) {
				room, ok := s.Rooms[roomID]
				if !ok {
					t.Fatal("Room not found")
				}
				uID := engine.NewUnitID(1, 0)
				if gotPos := room.Match.WorkingState.Units[uID].Position; gotPos.X == 4 && gotPos.Y == 7777 {
					t.Errorf("Expected Unit %#X didn't move", uID)
				}
				if gs != nil {
					t.Errorf("Expected matchState to be nil, got %p", gs)
				}
			},
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, engine.TurnCommand) {
				s := NewServerStateManager()
				uID := engine.NewUnitID(1, 0)
				newPos := engine.Coordinate{X: 4, Y: 7777}
				cmd := engine.NewMoveCommand(uID, newPos)
				return "NONEXISTENT", s, cmd
			},
			wantErr: ErrRoomNotFound,
			validate: func(t *testing.T, gs *engine.GameState, s *ServerStateManager, roomID string, cmd engine.TurnCommand) {
				if gs != nil {
					t.Errorf("Expected matchState to be nil, got %p", gs)
				}
			},
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager, engine.TurnCommand) {
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				uID := engine.NewUnitID(1, 0)
				newPos := engine.Coordinate{X: 4, Y: 7777}
				cmd := engine.NewMoveCommand(uID, newPos)
				return roomID, s, cmd
			},
			wantErr: ErrMatchNotFound,
			validate: func(t *testing.T, gs *engine.GameState, s *ServerStateManager, roomID string, cmd engine.TurnCommand) {
				if gs != nil {
					t.Errorf("Expected matchState to be nil, got %p", gs)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s, cmd := tt.setup(t)
			gs, err := s.SubmitTurnCommand(roomID, cmd)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("SubmitTurnCommand() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, gs, s, roomID, cmd)
			}
		})
	}
}

func TestServerStateManager_StartTurn(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (string, *ServerStateManager)
		wantErr  error
		validate func(t *testing.T, gs *engine.GameState, s *ServerStateManager, roomID string)
	}{
		{
			name: "Success",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				roomID, s := createTestRoom(t)
				s.Rooms[roomID].Match.TrueState.Turn = 1000
				s.Rooms[roomID].Match.WorkingState.Turn = 1000
				return roomID, s
			},
			wantErr: nil,
			validate: func(t *testing.T, gs *engine.GameState, s *ServerStateManager, roomID string) {
				room, ok := s.Rooms[roomID]
				if !ok {
					t.Fatal("Room not found")
				}

				if got, want := len(room.Match.WorkingState.Bombs), 2; got != want {
					t.Errorf("Expected SuddenDeath triggered and drop %d bombs, got %d", want, got)
				}
				if gs != room.Match.WorkingState {
					t.Errorf("Expected matchState pointer %p, got %p", room.Match.WorkingState, gs)
				}
			},
		},
		{
			name: "Match already ended",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				roomID, s := createTestRoom(t)
				s.Rooms[roomID].Match.WinnerTeamID = 1
				return roomID, s
			},
			wantErr: ErrMatchEnded,
			validate: func(t *testing.T, gs *engine.GameState, s *ServerStateManager, roomID string) {
				if gs != nil {
					t.Errorf("Expected matchState to be nil, got %p", gs)
				}
			},
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				s := NewServerStateManager()
				return "NONEXISTENT", s
			},
			wantErr: ErrRoomNotFound,
			validate: func(t *testing.T, gs *engine.GameState, s *ServerStateManager, roomID string) {
				if gs != nil {
					t.Errorf("Expected matchState to be nil, got %p", gs)
				}
			},
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				return roomID, s
			},
			wantErr: ErrMatchNotFound,
			validate: func(t *testing.T, gs *engine.GameState, s *ServerStateManager, roomID string) {
				if gs != nil {
					t.Errorf("Expected matchState to be nil, got %p", gs)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s := tt.setup(t)
			gs, err := s.StartTurn(roomID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("StartTurn() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, gs, s, roomID)
			}
		})
	}
}

func TestServerStateManager_ResetTurn(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (string, *ServerStateManager)
		wantErr  error
		validate func(t *testing.T, gs *engine.GameState, s *ServerStateManager, roomID string)
	}{
		{
			name: "Success",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				roomID, s := createTestRoom(t)
				s.Rooms[roomID].Match.WorkingState.Units[16].HasMoved = true
				return roomID, s
			},
			wantErr: nil,
			validate: func(t *testing.T, gs *engine.GameState, s *ServerStateManager, roomID string) {
				room, ok := s.Rooms[roomID]
				if !ok {
					t.Fatal("Room not found")
				}

				if got, want := room.Match.WorkingState.Units[16].HasMoved, false; got != want {
					t.Errorf("Expected Unit %#X HasMoved reset to %v, got %v", 16, want, got)
				}
				if gs != room.Match.WorkingState {
					t.Errorf("Expected matchState pointer %p, got %p", room.Match.WorkingState, gs)
				}
			},
		},
		{
			name: "Room Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				s := NewServerStateManager()
				return "NONEXISTENT", s
			},
			wantErr: ErrRoomNotFound,
			validate: func(t *testing.T, gs *engine.GameState, s *ServerStateManager, roomID string) {
				if gs != nil {
					t.Errorf("Expected matchState to be nil, got %p", gs)
				}
			},
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				return roomID, s
			},
			wantErr: ErrMatchNotFound,
			validate: func(t *testing.T, gs *engine.GameState, s *ServerStateManager, roomID string) {
				if gs != nil {
					t.Errorf("Expected matchState to be nil, got %p", gs)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s := tt.setup(t)
			gs, err := s.ResetTurn(roomID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ResetTurn() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, gs, s, roomID)
			}
		})
	}
}

func TestServerStateManager_ResolveTurn(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (string, *ServerStateManager)
		wantErr  error
		validate func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string)
	}{
		{
			name: "Success",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				roomID, s := createTestRoom(t)
				s.SubmitTurnCommand(roomID, engine.NewPlaceBombCommand(16, engine.Coordinate{X: 4, Y: 7}))
				s.Rooms[roomID].Match.WorkingState.Bombs[engine.NewBombID(1, 1, 16)].Countdown = 1 // force the bomb to explode in the next ResolveTurn
				return roomID, s
			},
			wantErr: nil,
			validate: func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string) {
				room, ok := s.Rooms[roomID]
				if !ok {
					t.Fatal("Room not found")
				}

				if got, want := gameEvents, 5; len(got) != want {
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
			setup: func(t *testing.T) (string, *ServerStateManager) {
				s := NewServerStateManager()
				return "NONEXISTENT", s
			},
			wantErr: ErrRoomNotFound,
			validate: func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string) {
				if len(gameEvents) != 0 {
					t.Errorf("Expected gameEvents to be empty, got %#v", gameEvents)
				}
			},
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				return roomID, s
			},
			wantErr: ErrMatchNotFound,
			validate: func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string) {
				if len(gameEvents) != 0 {
					t.Errorf("Expected gameEvents to be empty, got %#v", gameEvents)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s := tt.setup(t)
			gameEvents, err := s.ResolveTurn(roomID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ResolveTurn() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, gameEvents, s, roomID)
			}
		})
	}
}

func TestServerStateManager_Surrender(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (string, *ServerStateManager)
		req      SurrenderRequest
		wantErr  error
		validate func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string)
	}{
		{
			name: "Success",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				roomID, s := createTestRoom(t)
				return roomID, s
			},
			req:     SurrenderRequest{TeamID: 1},
			wantErr: nil,
			validate: func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string) {
				room, ok := s.Rooms[roomID]
				if !ok {
					t.Fatal("Room not found")
				}

				if got, want := gameEvents, 1; len(got) != want {
					t.Errorf("Expected %d gameEvents returned, got %#v", want, got)
				}
				if got, want := gameEvents[0].WinnerTeamID, 2; got != want {
					t.Errorf("Expected gameEvent WinnerTeamID = %v, got %v", want, got)
				}
				if got, want := room.Match.WinnerTeamID, 2; got != want {
					t.Errorf("Expected match winner = %v, got %v", want, got)
				}
			},
		},
		{
			name: "Invalid SurrenderRequest",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				roomID, s := createTestRoom(t)
				return roomID, s
			},
			req:     SurrenderRequest{TeamID: 3},
			wantErr: ErrInvalidConfig,
			validate: func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string) {
				room, ok := s.Rooms[roomID]
				if !ok {
					t.Fatal("Room not found")
				}

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
			setup: func(t *testing.T) (string, *ServerStateManager) {
				s := NewServerStateManager()
				return "NONEXISTENT", s
			},
			req:     SurrenderRequest{TeamID: 1},
			wantErr: ErrRoomNotFound,
			validate: func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string) {
				if len(gameEvents) != 0 {
					t.Errorf("Expected gameEvents to be empty, got %#v", gameEvents)
				}
			},
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				return roomID, s
			},
			req:     SurrenderRequest{TeamID: 1},
			wantErr: ErrMatchNotFound,
			validate: func(t *testing.T, gameEvents []engine.GameEvent, s *ServerStateManager, roomID string) {
				if len(gameEvents) != 0 {
					t.Errorf("Expected gameEvents to be empty, got %#v", gameEvents)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s := tt.setup(t)
			gameEvents, err := s.Surrender(roomID, tt.req.TeamID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Surrender() error = %v, want %v", err, tt.wantErr)
			}
			if tt.validate != nil {
				tt.validate(t, gameEvents, s, roomID)
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
				room, ok := s.Rooms[roomID]
				if !ok {
					t.Fatal("Room not found")
				}
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
			wantErr:  ErrRoomNotFound,
			validate: func(t *testing.T, gameCfg *engine.GameCfg, s *ServerStateManager, roomID string) {},
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				return roomID, s
			},
			wantErr:  ErrMatchNotFound,
			validate: func(t *testing.T, gameCfg *engine.GameCfg, s *ServerStateManager, roomID string) {},
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
			wantErr:  ErrRoomNotFound,
			validate: func(t *testing.T, allowed []engine.Coordinate, s *ServerStateManager, roomID string) {},
		},
		{
			name: "Match Not Found",
			setup: func(t *testing.T) (string, *ServerStateManager) {
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				return roomID, s
			},
			wantErr:  ErrMatchNotFound,
			validate: func(t *testing.T, allowed []engine.Coordinate, s *ServerStateManager, roomID string) {},
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
