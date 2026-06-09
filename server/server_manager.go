package server

import (
	"bomb-srpg/engine"
	"sync"
	"time"
)

// MatchRoom wraps the core engine match instance with server-layer network metadata.
type MatchRoom struct {
	ID           string
	Match        *engine.Match
	LastActivity time.Time
}

type ServerStateManager struct {
	mu    sync.RWMutex
	Rooms map[string]*MatchRoom
}

func NewServerStateManager() *ServerStateManager {
	manager := &ServerStateManager{
		Rooms: make(map[string]*MatchRoom),
	}

	return manager
}
