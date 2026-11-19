package server

import (
	"bomb-srpg/room"
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
				mustRoom(t, s, id)
				if _, err := s.getMatchState(id); !errors.Is(err, room.ErrMatchNotFound) {
					t.Errorf("getMatchState() error = %v, want %v", err, room.ErrMatchNotFound)
				}
			},
		},
		{
			name: "Room ID collision retry",
			setup: func() *ServerStateManager {
				s := NewServerStateManager()
				existingID := "ABCDE"
				s.rooms.Store(existingID, room.New(existingID, room.Deps{Logger: discardLogger()}))
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
				if _, ok := s.rooms.Load("ABCDE"); !ok {
					t.Error("Predefined room missing")
				}
				if _, ok := s.rooms.Load(id); !ok {
					t.Error("New room not added")
				}
			},
		},
		{
			name: "Max retries exhausted",
			setup: func() *ServerStateManager {
				callCount := 0
				roomIDs := []string{"ID001", "ID002", "ID003", "ID004", "ID005"}
				gen := func(int) (string, error) {
					if callCount < len(roomIDs) {
						id := roomIDs[callCount]
						callCount++
						return id, nil
					}
					return "SHOULD_NOT_REACH", nil
				}
				s := NewServerStateManager(WithRoomIDGenerator(gen))
				for _, id := range roomIDs {
					s.rooms.Store(id, room.New(id, room.Deps{Logger: discardLogger()}))
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
					if _, ok := s.rooms.Load(existing); !ok {
						t.Errorf("Predefined room %s missing", existing)
					}
				}
			},
		},
		{
			name: "Error thrown during generateRoomID",
			setup: func() *ServerStateManager {
				return NewServerStateManager(WithRoomIDGenerator(func(int) (string, error) {
					return "", errors.New("bad things happened")
				}))
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
			id, err := s.createMatchRoom()
			if (err != nil) != tt.wantErr {
				t.Fatalf("CreateMatchRoom() error = %v, wantErr %v", err, tt.wantErr)
			}
			tt.validate(t, s, id, err)
		})
	}
}

func TestServerStateManager_cleanupInactiveRooms(t *testing.T) {
	s := NewServerStateManager()

	activeID, _ := s.createMatchRoom()
	mustCreateMatch(t, s, activeID, validGameCfg())

	endedID, _ := s.createMatchRoom()
	endedTokens := mustCreateMatch(t, s, endedID, validGameCfg())
	if _, err := s.surrender(endedID, 1, endedTokens[0]); err != nil {
		t.Fatalf("surrender() error = %v", err)
	}

	tests := []struct {
		name       string
		advance    time.Duration
		wantActive bool
		wantEnded  bool
	}{
		{"now", 0, true, true},
		{"half an hour later, the ended match still counts as active", 30 * time.Minute, true, true},
		{"two hours later", 2 * time.Hour, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s.cleanupInactiveRooms(time.Now().Add(tt.advance))

			_, gotActive := s.rooms.Load(activeID)
			_, gotEnded := s.rooms.Load(endedID)
			if gotActive != tt.wantActive {
				t.Errorf("active room kept = %v, want %v", gotActive, tt.wantActive)
			}
			if gotEnded != tt.wantEnded {
				t.Errorf("ended room kept = %v, want %v", gotEnded, tt.wantEnded)
			}
		})
	}
}

func TestServerStateManager_RunCleanupLoop(t *testing.T) {
	s := NewServerStateManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		s.RunCleanupLoop(ctx, time.Millisecond)
		close(done)
	}()

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
