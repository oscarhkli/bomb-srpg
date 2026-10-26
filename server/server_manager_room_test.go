package server

import (
	"bomb-srpg/engine"
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
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
				t.Helper()
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
				t.Helper()
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
				t.Helper()
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
				t.Helper()
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
				t.Helper()
				s := NewServerStateManager()
				roomID, _ := s.CreateMatchRoom()
				return roomID, s, [2]string{}
			},
			action: func(t *testing.T, s *ServerStateManager, roomID string, _ [2]string) {
				t.Helper()
				_, _ = s.CreateMatch(roomID, validGameCfg())
			},
		},
		{
			name: "SubmitTurnCommand updates LastActivity",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				roomID, tokens, s := createTestRoom(t)
				return roomID, s, tokens
			},
			action: func(t *testing.T, s *ServerStateManager, roomID string, tokens [2]string) {
				t.Helper()
				unitID := engine.NewUnitID(1, 0)
				if _, err := s.SubmitTurnCommand(roomID, engine.NewMoveCommand(unitID, engine.Coordinate{X: 4, Y: 7}), tokens[0]); err != nil {
					t.Fatalf("SubmitTurnCommand() error = %v", err)
				}
			},
		},
		{
			name: "StartTurn updates LastActivity",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				roomID, tokens, s := createTestRoom(t)
				return roomID, s, tokens
			},
			action: func(t *testing.T, s *ServerStateManager, roomID string, tokens [2]string) {
				t.Helper()
				if _, _, err := s.StartTurn(roomID, tokens[0]); err != nil {
					t.Fatalf("StartTurn() error = %v", err)
				}
			},
		},
		{
			name: "ResetTurn updates LastActivity",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				roomID, tokens, s := createTestRoom(t)
				return roomID, s, tokens
			},
			action: func(t *testing.T, s *ServerStateManager, roomID string, tokens [2]string) {
				t.Helper()
				if err := s.ResetTurn(roomID, tokens[0]); err != nil {
					t.Fatalf("ResetTurn() error = %v", err)
				}
			},
		},
		{
			name: "ResolveTurn updates LastActivity",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				roomID, tokens, s := createTestRoom(t)
				if _, err := s.SubmitTurnCommand(roomID, engine.NewPlaceBombCommand(16, engine.Coordinate{X: 4, Y: 7}), tokens[0]); err != nil {
					t.Fatalf("SubmitTurnCommand() error = %v", err)
				}
				return roomID, s, tokens
			},
			action: func(t *testing.T, s *ServerStateManager, roomID string, tokens [2]string) {
				t.Helper()
				if _, _, err := s.ResolveTurn(roomID, tokens[0]); err != nil {
					t.Fatalf("ResolveTurn() error = %v", err)
				}
			},
		},
		{
			name: "runCPUTurn updates LastActivity",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				roomID, tokens, s := createTestRoomWithCfg(t, vsCpuGameCfg())
				return roomID, s, tokens
			},
			action: func(t *testing.T, s *ServerStateManager, roomID string, tokens [2]string) {
				t.Helper()
				room := mustRoom(t, s, roomID)
				room.Match.WorkingState.ActiveTeam = 2
				s.runCPUTurn(room, room.Match)
			},
		},
		{
			name: "Surrender updates LastActivity",
			setup: func(t *testing.T) (string, *ServerStateManager, [2]string) {
				t.Helper()
				roomID, tokens, s := createTestRoom(t)
				return roomID, s, tokens
			},
			action: func(t *testing.T, s *ServerStateManager, roomID string, tokens [2]string) {
				t.Helper()
				if _, err := s.Surrender(roomID, 1, tokens[0]); err != nil {
					t.Fatalf("Surrender() error = %v", err)
				}
			},
			validate: func(t *testing.T, s *ServerStateManager, roomID string) {
				t.Helper()
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

	if _, err := s.GetMatchState(roomID); err != nil {
		t.Fatalf("GetMatchState() error = %v", err)
	}
	if _, err := s.GetMatchConfig(roomID); err != nil {
		t.Fatalf("GetMatchConfig() error = %v", err)
	}
	if _, err := s.GetAllowedTiles(roomID, engine.NewUnitID(1, 0), engine.TurnCmdPlaceBomb); err != nil {
		t.Fatalf("GetAllowedTiles() error = %v", err)
	}

	room := mustRoom(t, s, roomID)
	room.mu.RLock()
	defer room.mu.RUnlock()
	if !room.LastActivity.Equal(before) && !room.LastActivity.Before(before) {
		t.Errorf("LastActivity should not be updated by read-only methods: before=%v, after=%v", before, room.LastActivity)
	}
}

func TestServerStateManager_cleanupInactiveRooms(t *testing.T) {
	s := NewServerStateManager()

	roomID1, _ := s.CreateMatchRoom()
	mustCreateMatch(t, s, roomID1, validGameCfg())

	roomID2, _ := s.CreateMatchRoom()
	mustCreateMatch(t, s, roomID2, validGameCfg())
	room := mustRoom(t, s, roomID2)
	room.mu.Lock()
	room.LastActivity = time.Now().Add(-120 * time.Minute)
	room.mu.Unlock()

	roomID3, _ := s.CreateMatchRoom()
	mustCreateMatch(t, s, roomID3, validGameCfg())
	room = mustRoom(t, s, roomID3)
	room.mu.Lock()
	room.Match.WinnerTeamID = 1
	room.mu.Unlock()

	s.cleanupInactiveRooms()

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

func TestServerStateManager_loadRoom_WrongType(t *testing.T) {
	s := NewServerStateManager()
	s.Rooms.Store("bad", "not a room")
	room, err := s.loadRoom("bad")
	if err == nil || room != nil {
		t.Errorf("loadRoom() = (%v, %v), want nil room and an error", room, err)
	}
}

func TestServerStateManager_cleanupInactiveRooms_WrongType(t *testing.T) {
	s := NewServerStateManager()
	s.Rooms.Store("bad", "not a room")
	s.cleanupInactiveRooms()
	if _, ok := s.Rooms.Load("bad"); ok {
		t.Error("wrong-typed entry should be removed")
	}
}

func TestServerStateManager_RunCleanupLoop(t *testing.T) {
	s := NewServerStateManager()
	roomID, _ := s.CreateMatchRoom()
	mustCreateMatch(t, s, roomID, validGameCfg())
	room := mustRoom(t, s, roomID)
	room.mu.Lock()
	room.LastActivity = time.Now().Add(-2 * roomInactivityTimeout)
	room.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		s.RunCleanupLoop(ctx, time.Millisecond)
		close(done)
	}()

	deadline := time.After(2 * time.Second)
	poll := time.NewTicker(time.Millisecond)
	defer poll.Stop()
	for {
		if _, ok := s.Rooms.Load(roomID); !ok {
			break
		}
		select {
		case <-deadline:
			t.Fatal("inactive room was not removed by the loop")
		case <-poll.C:
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("loop did not stop after cancel")
	}
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

type testLogWriter struct {
	t *testing.T
}

func (w *testLogWriter) Write(p []byte) (int, error) {
	w.t.Log(string(bytes.TrimSpace(p)))
	return len(p), nil
}
