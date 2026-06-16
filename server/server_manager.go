package server

import (
	"bomb-srpg/engine"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"
)

const (
	roomIDLength      = 5
	crockfordAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
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
		return "", fmt.Errorf("room unavailable: failed to generate a MatchRoom ID after %d times of retry", maxRetry)
	}

	s.Rooms[id] = &MatchRoom{
		ID:           id,
		Match:        nil,
		LastActivity: time.Now(),
	}

	return id, nil
}

func generateRoomID(length int) string {
	code := make([]byte, length)
	for i := range length {
		code[i] = crockfordAlphabet[rand.IntN(len(crockfordAlphabet))]
	}

	return string(code)
}
