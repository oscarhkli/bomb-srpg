package server

import (
	"bomb-srpg/cpu"
	"bomb-srpg/engine"
	"bomb-srpg/room"
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

const CleanupInterval = 60 * time.Minute

var errRoomNotFound = errors.New("room not found")

func mapError(err error) (int, string) {
	switch {
	case errors.Is(err, errRoomNotFound), errors.Is(err, room.ErrMatchNotFound):
		return http.StatusNotFound, err.Error()
	case errors.Is(err, room.ErrMatchExists):
		return http.StatusConflict, err.Error()
	case errors.Is(err, room.ErrInvalidConfig):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, room.ErrInvalidTurnCmd), errors.Is(err, room.ErrCPUTurnInProgress):
		return http.StatusConflict, err.Error()
	case errors.Is(err, room.ErrInvalidToken):
		return http.StatusUnauthorized, err.Error()
	case errors.Is(err, engine.ErrInvalidStagePreset),
		errors.Is(err, engine.ErrInvalidTeamSize),
		errors.Is(err, engine.ErrMissingKing),
		errors.Is(err, engine.ErrInvalidStageLayout),
		errors.Is(err, engine.ErrInvalidTerrain),
		errors.Is(err, engine.ErrUnknownArchetype):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, engine.ErrUnitNotFound),
		errors.Is(err, engine.ErrUnitDead),
		errors.Is(err, engine.ErrNotActiveTeam),
		errors.Is(err, engine.ErrAlreadyMoved),
		errors.Is(err, engine.ErrAlreadyUsedSkill),
		errors.Is(err, engine.ErrOutOfMoveRange),
		errors.Is(err, engine.ErrOutOfBombRange),
		errors.Is(err, engine.ErrCellOccupied),
		errors.Is(err, engine.ErrOutOfBombs),
		errors.Is(err, engine.ErrUnsupportedCommand),
		errors.Is(err, engine.ErrInvalidLanding),
		errors.Is(err, engine.ErrDesynced),
		errors.Is(err, engine.ErrOutOfBounds),
		errors.Is(err, room.ErrMatchInProgress):
		return http.StatusConflict, err.Error()
	default:
		return http.StatusInternalServerError, "internal error"
	}
}

type roomRegistry struct {
	rooms sync.Map
}

func (r *roomRegistry) Load(id string) (*room.Room, bool) {
	val, ok := r.rooms.Load(id)
	if !ok {
		return nil, false
	}
	rm, ok := val.(*room.Room)
	return rm, ok
}

func (r *roomRegistry) Store(id string, rm *room.Room) { r.rooms.Store(id, rm) }

func (r *roomRegistry) LoadOrStore(id string, rm *room.Room) (loaded bool) {
	_, loaded = r.rooms.LoadOrStore(id, rm)
	return loaded
}

func (r *roomRegistry) Delete(id string) { r.rooms.Delete(id) }

func (r *roomRegistry) Range(f func(id string, rm *room.Room) bool) {
	r.rooms.Range(func(key, val any) bool {
		id, _ := key.(string)
		rm, ok := val.(*room.Room)
		if !ok {
			return true
		}
		return f(id, rm)
	})
}

type ServerStateManager struct {
	Rooms          roomRegistry
	generateRoomID func(int) (string, error)
	decide         func(*engine.GameState) ([]engine.TurnCommand, error)
	spawn          func(func())
	newMatch       func(engine.GameCfg) (*engine.Match, error)
	Logger         *slog.Logger
}

type Option func(*ServerStateManager)

func WithLogger(logger *slog.Logger) Option {
	return func(s *ServerStateManager) {
		s.Logger = logger
	}
}

func WithDecider(decide func(*engine.GameState) ([]engine.TurnCommand, error)) Option {
	return func(s *ServerStateManager) {
		s.decide = decide
	}
}

func WithSpawner(spawn func(func())) Option {
	return func(s *ServerStateManager) {
		s.spawn = spawn
	}
}

func WithMatchFactory(newMatch func(engine.GameCfg) (*engine.Match, error)) Option {
	return func(s *ServerStateManager) {
		s.newMatch = newMatch
	}
}

func WithRoomIDGenerator(generate func(int) (string, error)) Option {
	return func(s *ServerStateManager) {
		s.generateRoomID = generate
	}
}

func NewServerStateManager(opts ...Option) *ServerStateManager {
	manager := &ServerStateManager{
		generateRoomID: randomHex,
		decide:         cpu.Decide,
		spawn:          func(f func()) { go f() },
		newMatch:       engine.InitGame,
		Logger:         slog.Default(),
	}
	for _, opt := range opts {
		opt(manager)
	}

	return manager
}

const roomIDBytes = 5

func (s *ServerStateManager) createMatchRoom() (string, error) {
	maxRetry := 5

	for range maxRetry {
		id, err := s.generateRoomID(roomIDBytes)
		if err != nil {
			s.Logger.Warn("failed to generate room ID", "error", err)
			return "", fmt.Errorf("failed to generate room ID: %w", err)
		}
		rm := room.New(id, room.Deps{
			Logger:   s.Logger,
			Decide:   s.decide,
			Spawn:    s.spawn,
			NewMatch: s.newMatch,
		})
		if !s.Rooms.LoadOrStore(id, rm) {
			return id, nil
		}
	}

	s.Logger.Warn("failed to generate room ID", "retries", maxRetry)
	return "", fmt.Errorf("room unavailable: failed to generate a MatchRoom ID after %d times of retry", maxRetry)
}

func (s *ServerStateManager) Room(roomID string) (*room.Room, error) {
	rm, ok := s.Rooms.Load(roomID)
	if !ok {
		s.Logger.Warn("match room not found", "roomID", roomID)
		return nil, fmt.Errorf("%w: roomID=%s", errRoomNotFound, roomID)
	}
	return rm, nil
}

func randomHex(byteLength int) (string, error) {
	b := make([]byte, byteLength)
	if _, err := cryptorand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *ServerStateManager) createMatch(roomID string, gameCfg engine.GameCfg) ([2]string, error) {
	rm, err := s.Room(roomID)
	if err != nil {
		return [2]string{}, err
	}
	return rm.Start(gameCfg)
}

func (s *ServerStateManager) rematch(roomID, token string) ([2]string, error) {
	rm, err := s.Room(roomID)
	if err != nil {
		return [2]string{}, err
	}
	return rm.Rematch(token)
}

func (s *ServerStateManager) deleteMatch(roomID, token string) error {
	rm, err := s.Room(roomID)
	if err != nil {
		return err
	}
	return rm.Delete(token)
}

func (s *ServerStateManager) getMatchState(roomID string) (*engine.GameState, error) {
	rm, err := s.Room(roomID)
	if err != nil {
		return nil, err
	}
	return rm.State()
}

func (s *ServerStateManager) submitTurnCommand(roomID string, cmd engine.TurnCommand, token string) ([]engine.GameEvent, error) {
	rm, err := s.Room(roomID)
	if err != nil {
		return nil, err
	}
	return rm.Submit(cmd, token)
}

func (s *ServerStateManager) consumeCPUStatus(roomID, token string) (engine.CPUTurnPhase, []engine.GameEvent, []engine.GameEvent, error) {
	rm, err := s.Room(roomID)
	if err != nil {
		return 0, nil, nil, err
	}
	status, err := rm.ConsumeCPU(token)
	return status.Phase, status.Plan, status.Resolve, err
}

func (s *ServerStateManager) resetTurn(roomID, token string) error {
	rm, err := s.Room(roomID)
	if err != nil {
		return err
	}
	return rm.Reset(token)
}

func (s *ServerStateManager) resolveTurn(roomID, token string) ([]engine.GameEvent, []engine.GameEvent, error) {
	rm, err := s.Room(roomID)
	if err != nil {
		return nil, nil, err
	}
	return rm.Resolve(token)
}

func (s *ServerStateManager) surrender(roomID string, teamID int, token string) ([]engine.GameEvent, error) {
	if teamID != 1 && teamID != 2 {
		return nil, fmt.Errorf("%w: team must be 1 or 2", room.ErrInvalidConfig)
	}

	rm, err := s.Room(roomID)
	if err != nil {
		return nil, err
	}
	return rm.Surrender(teamID, token)
}

func (s *ServerStateManager) getMatchConfig(roomID string) (*engine.GameCfg, error) {
	rm, err := s.Room(roomID)
	if err != nil {
		return nil, err
	}
	gameCfg, err := rm.Config()
	if err != nil {
		return nil, err
	}
	return &gameCfg, nil
}

func (s *ServerStateManager) getAllowedTiles(roomID string, unitID engine.UnitID, turnCmdType engine.TurnCmdType) ([]engine.Coordinate, error) {
	rm, err := s.Room(roomID)
	if err != nil {
		return nil, err
	}
	return rm.AllowedTiles(unitID, turnCmdType)
}

// RunCleanupLoop blocks until ctx is cancelled.
func (s *ServerStateManager) RunCleanupLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanupInactiveRooms(time.Now())
		}
	}
}

const roomInactivityTimeout = 60 * time.Minute

func (s *ServerStateManager) cleanupInactiveRooms(now time.Time) {
	s.Rooms.Range(func(id string, rm *room.Room) bool {
		if rm.Expired(now, roomInactivityTimeout) {
			s.Rooms.Delete(id)
			s.Logger.Info("removed room", "roomID", id, "inactive", true)
		}
		return true
	})
}
