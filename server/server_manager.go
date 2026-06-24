package server

import (
	"bomb-srpg/engine"
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"math/rand/v2"
	"net/http"
	"slices"
	"sync"
	"time"
)

const (
	roomIDLength          = 5
	crockfordAlphabet     = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	RoomInactivityTimeout = 5 * time.Minute
	CleanupInterval       = 1 * time.Minute
)

var (
	ErrRoomNotFound   = errors.New("room not found")
	ErrMatchEnded     = errors.New("match already ended")
	ErrMatchExists    = errors.New("match already exists")
	ErrMatchNotFound  = errors.New("match not found")
	ErrInvalidConfig  = errors.New("invalid game config")
	ErrInvalidTurnCmd = errors.New("invalid turn command")
	ErrInvalidToken   = errors.New("invalid player token")
)

// mapError converts an error to an HTTP status code and message.
func mapError(err error) (int, string) {
	switch {
	case errors.Is(err, ErrRoomNotFound), errors.Is(err, ErrMatchNotFound):
		return http.StatusNotFound, err.Error()
	case errors.Is(err, ErrMatchExists):
		return http.StatusConflict, err.Error()
	case errors.Is(err, ErrInvalidConfig):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, ErrInvalidTurnCmd):
		return http.StatusConflict, err.Error()
	case errors.Is(err, ErrInvalidToken):
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
		errors.Is(err, ErrMatchEnded):
		return http.StatusConflict, err.Error()
	default:
		return http.StatusInternalServerError, "internal error"
	}
}

// MatchRoom wraps the core engine match instance with server-layer network metadata.
type MatchRoom struct {
	ID           string
	Match        *engine.Match
	LastActivity time.Time
	PlayerTokens [2]string // [0]=Team1, [1]=Team2
	Logger       *slog.Logger
}

type ServerStateManager struct {
	mu             sync.RWMutex
	Rooms          map[string]*MatchRoom
	generateRoomID func(int) string
	Logger         *slog.Logger
}

// Option configures a ServerStateManager.
type Option func(*ServerStateManager)

// WithLogger sets the logger for the ServerStateManager.
func WithLogger(logger *slog.Logger) Option {
	return func(s *ServerStateManager) {
		s.Logger = logger
	}
}

// NewServerStateManager constructs a new ServerStateManager with an empty room map.
// It uses the Crockford32 alphabet to generate collision-resistant room IDs.
func NewServerStateManager(opts ...Option) *ServerStateManager {
	manager := &ServerStateManager{
		Rooms:          make(map[string]*MatchRoom),
		generateRoomID: generateRoomID,
		Logger:         slog.Default(),
	}
	for _, opt := range opts {
		opt(manager)
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
		s.Logger.Warn("failed to generate room ID", "retries", maxRetry)
		return "", fmt.Errorf("room unavailable: failed to generate a MatchRoom ID after %d times of retry", maxRetry)
	}

	s.Rooms[id] = &MatchRoom{
		ID:           id,
		Match:        nil,
		LastActivity: time.Now(),
		Logger:       s.Logger.With("roomID", id),
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

func generatePlayerToken() (string, error) {
	b := make([]byte, 16)
	if _, err := cryptorand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CreateMatch initialize the game in a given MatchRoom.
// Returns an error if any setup rule is violated.
func (s *ServerStateManager) CreateMatch(roomID string, gameCfg engine.GameCfg) ([2]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, ok := s.Rooms[roomID]
	if !ok {
		s.Logger.Warn("match room not found", "roomID", roomID)
		return [2]string{}, fmt.Errorf("%w: roomID=%s", ErrRoomNotFound, roomID)
	}

	if room.Match != nil {
		s.Logger.Warn("match already exists", "roomID", roomID)
		return [2]string{}, fmt.Errorf("%w: roomID=%s", ErrMatchExists, roomID)
	}

	match, err := engine.InitGame(gameCfg)

	if err != nil {
		s.Logger.Error("invalid game config", "roomID", roomID, "error", err)
		return [2]string{}, fmt.Errorf("%w: gameCfg=%+v: %v", ErrInvalidConfig, gameCfg, err)
	}

	var tokens [2]string
	for i := range 2 {
		token, err := generatePlayerToken()
		if err != nil {
			s.Logger.Warn("failed to generate player token", "roomID", roomID, "player", i)
			return [2]string{}, fmt.Errorf("failed to generate playerToken for Player %d in MatchRoom %v", i, roomID)
		}

		tokens[i] = token
	}

	room.Match = match
	room.PlayerTokens = tokens
	room.LastActivity = time.Now()

	return tokens, nil
}

func (mr *MatchRoom) validatePlayerToken(teamID int, token string) error {
	idx := teamID - 1
	if idx < 0 || idx > 1 {
		return ErrInvalidConfig
	}

	if mr.PlayerTokens[idx] != token {
		return ErrInvalidToken
	}
	return nil
}

func (s *ServerStateManager) roomReadyForMatch(roomID string) (*MatchRoom, error) {
	room, ok := s.Rooms[roomID]
	if !ok {
		s.Logger.Warn("match room not found", "roomID", roomID)
		return nil, fmt.Errorf("%w: roomID=%s", ErrRoomNotFound, roomID)
	}

	if room.Match == nil {
		s.Logger.Warn("match not found", "roomID", roomID)
		return nil, fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, roomID)
	}

	return room, nil
}

// GetMatchState gets the WorkingState of the Match in a given MatchRoom.
// Returns the WorkingState or an error if any pre-check is violated.
func (s *ServerStateManager) GetMatchState(roomID string) (*engine.GameState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	room, err := s.roomReadyForMatch(roomID)
	if err != nil {
		return nil, err
	}

	return room.Match.WorkingState, nil
}

// SubmitTurnCommand delivers TurnCommand to engine to move a Unit or place a bomb in a given MatchRoom.
// Returns the latest WorkingState or an error if any pre-check is violated
func (s *ServerStateManager) SubmitTurnCommand(roomID string, cmd engine.TurnCommand, token string) (*engine.GameState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, err := s.roomReadyForMatch(roomID)
	if err != nil {
		return nil, err
	}

	teamID := int(cmd.UnitID >> 4)
	if err := room.validatePlayerToken(teamID, token); err != nil {
		return nil, err
	}

	err = room.Match.ApplyTurnCommand(cmd)
	if err != nil {
		s.Logger.Error("invalid turn command", "roomID", roomID, "turnCmdType", cmd.Type, "error", err)
		return nil, fmt.Errorf("%w: turnCommand=%+v: %v", ErrInvalidTurnCmd, cmd, err)
	}

	room.LastActivity = time.Now()
	return room.Match.WorkingState, nil
}

// StartTurn sends StartTurn signal engine to start a new turn in a given MatchRoom.
// Returns the latest WorkingState or an error if any pre-check is violated
func (s *ServerStateManager) StartTurn(roomID string, token string) (*engine.GameState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, err := s.roomReadyForMatch(roomID)
	if err != nil {
		return nil, err
	}

	teamID := room.Match.WorkingState.ActiveTeam
	if err := room.validatePlayerToken(teamID, token); err != nil {
		return nil, err
	}

	room.Match.StartTurn()

	if room.Match.WinnerTeamID != 0 {
		return nil, fmt.Errorf("%w: match already ended", ErrMatchEnded)
	}

	room.LastActivity = time.Now()
	return room.Match.WorkingState, nil
}

// ResetTurn sends ResetTurn signal to engine to drop the current WorkingState and reset to TrueState in a given MatchRoom.
// Returns the latest WorkingState or an error if any pre-check is violated
func (s *ServerStateManager) ResetTurn(roomID string, token string) (*engine.GameState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, err := s.roomReadyForMatch(roomID)
	if err != nil {
		return nil, err
	}

	teamID := room.Match.WorkingState.ActiveTeam
	if err := room.validatePlayerToken(teamID, token); err != nil {
		return nil, err
	}

	room.Match.ResetTurn()

	room.LastActivity = time.Now()
	return room.Match.WorkingState, nil
}

// ResetTurn sends ResolveTurn signal to engine to calculate the impacts of the Player's action in a given MatchRoom.
// Returns the gameEvents or an error if any pre-check is violated
func (s *ServerStateManager) ResolveTurn(roomID string, token string) ([]engine.GameEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, err := s.roomReadyForMatch(roomID)
	if err != nil {
		return nil, err
	}

	teamID := room.Match.WorkingState.ActiveTeam
	if err := room.validatePlayerToken(teamID, token); err != nil {
		return nil, err
	}

	events := room.Match.ResolveTurn()
	room.LastActivity = time.Now()
	return events, nil
}

// ResetTurn sends Surrender signal to engine to end the current Match in a given MatchRoom.
// Returns the gameEvents or an error if any pre-check is violated
func (s *ServerStateManager) Surrender(roomID string, teamID int, token string) ([]engine.GameEvent, error) {
	if teamID != 1 && teamID != 2 {
		return nil, fmt.Errorf("%w: team must be 1 or 2", ErrInvalidConfig)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	room, err := s.roomReadyForMatch(roomID)
	if err != nil {
		return nil, err
	}

	if err := room.validatePlayerToken(teamID, token); err != nil {
		return nil, err
	}

	events := room.Match.Surrender(teamID)
	room.Match = nil
	room.LastActivity = time.Now()
	return events, nil
}

// GetMatchConfig gets the GameConfig of the current Match in a given MatchRoom.
func (s *ServerStateManager) GetMatchConfig(roomID string) (*engine.GameCfg, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	room, err := s.roomReadyForMatch(roomID)
	if err != nil {
		return nil, err
	}

	return &room.Match.GameCfg, nil
}

// GetAllowedTiles gets the hints for Player to identify which tiles are available according to the TurnCmdAction
// Returns the coordinates of the allowed tiles or an error if any pre-check is violated
func (s *ServerStateManager) GetAllowedTiles(roomID string, unitID engine.UnitID, turnCmdType engine.TurnCmdType) ([]engine.Coordinate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	room, err := s.roomReadyForMatch(roomID)
	if err != nil {
		return nil, err
	}

	allowedTiles, err := room.Match.WorkingState.FindAllowedTilesForCommand(unitID, turnCmdType)

	if err != nil {
		return nil, err
	}

	return slices.Collect(maps.Keys(allowedTiles)), nil
}

// StartCleanupLoop runs background cleanup until ctx is cancelled.
func (s *ServerStateManager) StartCleanupLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.cleanupInactiveRooms()
			}
		}
	}()
}

// cleanupInactiveRooms removes rooms inactive > RoomInactivityTimeout OR already ended.
func (s *ServerStateManager) cleanupInactiveRooms() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, room := range s.Rooms {
		inactive := now.Sub(room.LastActivity) > RoomInactivityTimeout
		ended := room.Match != nil && room.Match.WinnerTeamID != 0
		if inactive || ended {
			delete(s.Rooms, id)
			s.Logger.Info("removed room", "roomID", id, "inactive", inactive, "ended", ended)
		}
	}
}
