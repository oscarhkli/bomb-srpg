package server

import (
	"bomb-srpg/engine"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"
)

const (
	roomIDLength      = 5
	crockfordAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
)

var (
	ErrRoomNotFound  = errors.New("room not found")
	ErrMatchExists   = errors.New("match already exists")
	ErrMatchNotFound = errors.New("active match not exist")
	ErrInvalidConfig = errors.New("invalid game config")
)

// MatchRoom wraps the core engine match instance with server-layer network metadata.
type MatchRoom struct {
	ID           string
	Match        *engine.Match
	LastActivity time.Time
}

type ServerStateManager struct {
	mu             sync.RWMutex
	Rooms          map[string]*MatchRoom
	generateRoomID func(int) string
}

// NewServerStateManager constructs a new ServerStateManager with an empty room map.
// It uses the Crockford32 alphabet to generate collision-resistant room IDs.
func NewServerStateManager() *ServerStateManager {
	manager := &ServerStateManager{
		Rooms:          make(map[string]*MatchRoom),
		generateRoomID: generateRoomID,
	}

	return manager
}

// CreateMatchRoom generates a unique room ID and registers an empty MatchRoom.
// It retries up to 5 times on ID collision. Returns the room ID or an error if exhausted.
func (s *ServerStateManager) CreateMatchRoom() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	maxRetry := 5

	var id string
	found := false
	for range maxRetry {
		id = s.generateRoomID(roomIDLength)
		if _, ok := s.Rooms[id]; !ok {
			found = true
			break
		}
	}

	if !found {
		slog.Warn("failed to generate room ID", "retries", maxRetry)
		return "", fmt.Errorf("room unavailable: failed to generate a MatchRoom ID after %d times of retry", maxRetry)
	}

	s.Rooms[id] = &MatchRoom{
		ID:           id,
		Match:        nil,
		LastActivity: time.Now(),
	}

	slog.Info("match room created", "roomID", id)
	return id, nil
}

func generateRoomID(length int) string {
	code := make([]byte, length)
	for i := range length {
		code[i] = crockfordAlphabet[rand.IntN(len(crockfordAlphabet))]
	}

	return string(code)
}

func (s *ServerStateManager) CreateMatch(roomID string, gameCfg engine.GameCfg) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, ok := s.Rooms[roomID]
	if !ok {
		slog.Warn("match room not found", "roomID", roomID)
		return fmt.Errorf("%w: roomID=%s", ErrRoomNotFound, roomID)
	}

	if room.Match != nil {
		slog.Warn("match already exists", "roomID", roomID)
		return fmt.Errorf("%w: roomID=%s", ErrMatchExists, roomID)
	}

	match, err := engine.InitGame(gameCfg)

	if err != nil {
		slog.Error("invalid game config", "roomID", roomID, "error", err)
		return fmt.Errorf("%w: gameCfg=%+v: %v", ErrInvalidConfig, gameCfg, err)
	}

	room.Match = match

	return nil
}

func (s *ServerStateManager) GetMatchState(roomID string) (*engine.GameState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	room, ok := s.Rooms[roomID]
	if !ok {
		slog.Warn("match room not found", "roomID", roomID)
		return nil, fmt.Errorf("%w: roomID=%s", ErrRoomNotFound, roomID)
	}

	if room.Match == nil {
		slog.Warn("match not found", "roomID", roomID)
		return nil, fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, roomID)
	}

	return room.Match.WorkingState, nil
}
