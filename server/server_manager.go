package server

import (
	"bomb-srpg/cpu"
	"bomb-srpg/engine"
	"context"
	cryptorand "crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"sync"
	"time"
)

const (
	roomIDBytes           = 5
	RoomInactivityTimeout = 60 * time.Minute
	CleanupInterval       = 60 * time.Minute

	maxCPUReplanAttempts = 5
)

var (
	ErrRoomNotFound    = errors.New("room not found")
	ErrMatchEnded      = errors.New("match already ended")
	ErrMatchInProgress = errors.New("match still in progress")
	ErrMatchExists     = errors.New("match already exists")
	ErrMatchNotFound   = errors.New("match not found")
	ErrInvalidConfig   = errors.New("invalid game config")
	ErrInvalidTurnCmd  = errors.New("invalid turn command")
	ErrInvalidToken    = errors.New("invalid player token")
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
		errors.Is(err, ErrMatchEnded),
		errors.Is(err, ErrMatchInProgress):
		return http.StatusConflict, err.Error()
	default:
		return http.StatusInternalServerError, "internal error"
	}
}

// MatchRoom wraps the core engine match instance with server-layer network metadata.
type MatchRoom struct {
	mu           sync.RWMutex
	ID           string
	Match        *engine.Match
	GameCfg      *engine.GameCfg // record for rematch use
	LastActivity time.Time
	PlayerTokens [2]string // [0]=Team1, [1]=Team2
	Logger       *slog.Logger
}

type ServerStateManager struct {
	Rooms          sync.Map
	generateRoomID func(int) (string, error)
	decide         func(*engine.GameState) []engine.TurnCommand
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
// It generates collision-resistant room IDs.
func NewServerStateManager(opts ...Option) *ServerStateManager {
	manager := &ServerStateManager{
		generateRoomID: randomHex,
		decide:         cpu.Decide,
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
	maxRetry := 5

	for range maxRetry {
		id, err := s.generateRoomID(roomIDBytes)
		if err != nil {
			s.Logger.Warn("failed to generate room ID", "error", err)
			return "", fmt.Errorf("failed to generate room ID: %w", err)
		}
		room := &MatchRoom{
			ID:           id,
			Match:        nil,
			LastActivity: time.Now(),
			Logger:       s.Logger.With("roomID", id),
		}
		if _, loaded := s.Rooms.LoadOrStore(id, room); !loaded {
			return id, nil
		}
	}

	s.Logger.Warn("failed to generate room ID", "retries", maxRetry)
	return "", fmt.Errorf("room unavailable: failed to generate a MatchRoom ID after %d times of retry", maxRetry)
}

// loadRoom looks up a MatchRoom without locking it.
// Callers must hold room.mu before touching room.Match, which a CPU goroutine may be mutating.
func (s *ServerStateManager) loadRoom(roomID string) (*MatchRoom, error) {
	roomVal, ok := s.Rooms.Load(roomID)
	if !ok {
		s.Logger.Warn("match room not found", "roomID", roomID)
		return nil, fmt.Errorf("%w: roomID=%s", ErrRoomNotFound, roomID)
	}
	return roomVal.(*MatchRoom), nil
}

func randomHex(byteLength int) (string, error) {
	b := make([]byte, byteLength)
	if _, err := cryptorand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func generatePlayerTokens() ([2]string, error) {
	var tokens [2]string
	for i := range 2 {
		token, err := randomHex(16)
		if err != nil {
			return [2]string{}, fmt.Errorf("failed to generate playerToken for Player %d: %w", i, err)
		}
		tokens[i] = token
	}
	return tokens, nil
}

// CreateMatch initialize the game in a given MatchRoom.
// Returns an error if any setup rule is violated.
func (s *ServerStateManager) CreateMatch(roomID string, gameCfg engine.GameCfg) ([2]string, error) {
	room, err := s.loadRoom(roomID)
	if err != nil {
		return [2]string{}, err
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	match, err := s.createMatchLocked(room, gameCfg)
	if err != nil {
		return [2]string{}, err
	}

	tokens, err := generatePlayerTokens()
	if err != nil {
		room.Logger.Warn("failed to generate player tokens", "roomID", roomID, "error", err)
		return [2]string{}, err
	}

	room.Match = match
	room.GameCfg = &gameCfg
	room.PlayerTokens = tokens
	room.LastActivity = time.Now()

	return tokens, nil
}

func (mr *MatchRoom) validatePlayerToken(teamID int, token string) error {
	idx := teamID - 1
	if idx < 0 || idx > 1 {
		return ErrInvalidConfig
	}

	if !matchToken(mr.PlayerTokens[idx], token) {
		return ErrInvalidToken
	}
	return nil
}

func (mr *MatchRoom) validateAnyMatchPlayerToken(token string) error {
	if !matchToken(mr.PlayerTokens[0], token) && !matchToken(mr.PlayerTokens[1], token) {
		return ErrInvalidToken
	}
	return nil
}

func matchToken(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func (s *ServerStateManager) createMatchLocked(room *MatchRoom, gameCfg engine.GameCfg) (*engine.Match, error) {
	roomID := room.ID
	if room.Match != nil {
		room.Logger.Warn("match already exists")
		return nil, fmt.Errorf("%w: roomID=%s", ErrMatchExists, roomID)
	}

	match, err := engine.InitGame(gameCfg)
	if err != nil {
		room.Logger.Error("invalid game config", "gameCfg", gameCfg, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	return match, nil
}

// Rematch wipes the existing Match in a given MatchRoom and recreate one using GameCfg.
func (s *ServerStateManager) Rematch(roomID, token string) ([2]string, error) {
	room, err := s.loadRoom(roomID)
	if err != nil {
		return [2]string{}, err
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if err := room.validateAnyMatchPlayerToken(token); err != nil {
		return [2]string{}, err
	}

	if room.GameCfg == nil {
		room.Logger.Warn("previous match not found")
		return [2]string{}, fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, roomID)
	}

	room.Match = nil

	match, err := s.createMatchLocked(room, *room.GameCfg)
	if err != nil {
		return [2]string{}, err
	}

	room.Match = match
	room.LastActivity = time.Now()

	return room.PlayerTokens, nil
}

// DeleteMatch removes the existing concluded Match in a given MatchRoom.
// Returns an error if any pre-check is violated.
func (s *ServerStateManager) DeleteMatch(roomID, token string) error {
	room, err := s.loadRoom(roomID)
	if err != nil {
		return err
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if err := room.validateAnyMatchPlayerToken(token); err != nil {
		return err
	}

	if room.Match == nil {
		room.Logger.Info("match not found, no-op")
		return nil
	}

	if room.Match.WinnerTeamID == 0 {
		room.Logger.Warn("match still in progress")
		return fmt.Errorf("%w: roomID=%s", ErrMatchInProgress, roomID)
	}

	room.Match = nil

	return nil
}

// GetMatchState gets the WorkingState of the Match in a given MatchRoom.
// Returns a copy of the WorkingState or an error if any pre-check is violated.
func (s *ServerStateManager) GetMatchState(roomID string) (*engine.GameState, error) {
	room, err := s.loadRoom(roomID)
	if err != nil {
		return nil, err
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	if room.Match == nil {
		room.Logger.Warn("match not found")
		return nil, fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, roomID)
	}

	// A copy, so the caller can marshal it after the lock is released.
	return room.Match.WorkingState.DeepCopy(), nil
}

// SubmitTurnCommand delivers TurnCommand to engine to move a Unit or place a bomb in a given MatchRoom.
// Returns the GameEvents or an error if any pre-check is violated
func (s *ServerStateManager) SubmitTurnCommand(roomID string, cmd engine.TurnCommand, token string) ([]engine.GameEvent, error) {
	room, err := s.loadRoom(roomID)
	if err != nil {
		return nil, err
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.Match == nil {
		s.Logger.Warn("match not found", "roomID", roomID)
		return nil, fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, roomID)
	}

	teamID := int(cmd.UnitID >> 4)
	if err := room.validatePlayerToken(teamID, token); err != nil {
		return nil, err
	}

	gameEvents, err := room.Match.ApplyTurnCommand(cmd)
	if err != nil {
		room.Logger.Error("invalid turn command", "turnCommand", cmd, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrInvalidTurnCmd, err)
	}

	room.LastActivity = time.Now()
	return gameEvents, nil
}

// StartTurn sends StartTurn signal engine to start a new turn in a given MatchRoom.
// Returns the GameEvents or an error if any pre-check is violated
func (s *ServerStateManager) StartTurn(roomID, token string) (bool, []engine.GameEvent, error) {
	room, err := s.loadRoom(roomID)
	if err != nil {
		return false, nil, err
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.Match == nil {
		room.Logger.Warn("match not found")
		return false, nil, fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, roomID)
	}

	teamID := room.Match.WorkingState.ActiveTeam
	if err := room.validatePlayerToken(teamID, token); err != nil {
		return false, nil, err
	}

	gameEvents := room.Match.StartTurn()

	if room.Match.WinnerTeamID != 0 {
		return false, nil, fmt.Errorf("%w: match already ended", ErrMatchEnded)
	}

	// Launch CPU planning against the post-hazard board.
	// A Ready phase here means the last turn's events were never consumed; they are stale now.
	if room.Match.GameCfg.VSCpu && teamID == 2 && room.Match.CPU.Phase != engine.TurnPhasePlanning {
		room.Match.CPU.Phase = engine.TurnPhasePlanning
		room.Match.CPU.PlanGameEvents = nil
		room.Match.CPU.ResolveTurnGameEvents = nil
		go s.runCPUTurn(room, room.Match)
	}

	room.LastActivity = time.Now()
	return room.Match.WorkingState.InSuddenDeath, gameEvents, nil
}

// runCPUTurn plays the CPU's turn to completion, holding the room lock throughout.
// It abandons if the match was replaced, deleted, or already won while the goroutine was queued.
func (s *ServerStateManager) runCPUTurn(room *MatchRoom, match *engine.Match) {
	room.mu.Lock()
	defer room.mu.Unlock()

	if room.Match != match {
		room.Logger.Info("CPU turn abandoned: match replaced or deleted")
		return
	}
	defer func() {
		match.CPU.Phase = engine.TurnPhaseReady
		room.LastActivity = time.Now()
	}()

	if match.WinnerTeamID != 0 {
		room.Logger.Info("CPU turn abandoned: match already ended")
		return
	}

	// A panic here would kill the process: the HTTP RecoverPanic middleware does not span goroutines.
	// The turn is forfeited from a clean sandbox so the match can still advance; a panic from that
	// forfeit stays fatal, since re-resolving the same board would panic identically.
	defer func() {
		if rec := recover(); rec != nil {
			room.Logger.Error("CPU turn panicked, forfeiting", "panic", rec)
			match.ResetTurn()
			match.CPU.PlanGameEvents, match.CPU.ResolveTurnGameEvents = match.ResolveTurn()
		}
	}()

	var err error
	for attempt := range maxCPUReplanAttempts {
		if err = applyPlan(match, s.decide(match.WorkingState)); err == nil {
			break
		}
		room.Logger.Warn("CPU plan rejected, replanning", "attempt", attempt, "error", err)
	}
	if err != nil {
		room.Logger.Error("CPU replan attempts exhausted, committing partial turn", "attempts", maxCPUReplanAttempts, "error", err)
	}

	match.CPU.PlanGameEvents, match.CPU.ResolveTurnGameEvents = match.ResolveTurn()
}

// applyPlan applies each TurnCommand in order, stopping at the first rejection.
// Returns the rejecting command's error, or nil if the whole plan applied.
func applyPlan(match *engine.Match, plan []engine.TurnCommand) error {
	for _, cmd := range plan {
		if _, err := match.ApplyTurnCommand(cmd); err != nil {
			return fmt.Errorf("command %+v rejected: %w", cmd, err)
		}
	}
	return nil
}

// ConsumeCPUStatus consumes current CPU Turn States in given MatchRoom and reset the state if CPU TurnPhase is in Ready - planning is done.
// Returns the cpuTurnPhase, the CPU's planning and resolution gameEvents, or an error if any pre-check is violated
func (s *ServerStateManager) ConsumeCPUStatus(roomID, token string) (engine.CPUTurnPhase, []engine.GameEvent, []engine.GameEvent, error) {
	room, err := s.loadRoom(roomID)
	if err != nil {
		return 0, nil, nil, err
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.Match == nil {
		room.Logger.Warn("match not found")
		return 0, nil, nil, fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, roomID)
	}

	if err := room.validateAnyMatchPlayerToken(token); err != nil {
		return 0, nil, nil, err
	}

	turnPhase := room.Match.CPU.Phase
	planGameEvents, resolveTurnGameEvents := room.Match.CPU.PlanGameEvents, room.Match.CPU.ResolveTurnGameEvents
	if turnPhase == engine.TurnPhaseReady {
		room.Match.CPU.PlanGameEvents = nil
		room.Match.CPU.ResolveTurnGameEvents = nil
		room.Match.CPU.Phase = engine.TurnPhaseIdle
	}
	if planGameEvents == nil {
		planGameEvents = []engine.GameEvent{}
	}
	if resolveTurnGameEvents == nil {
		resolveTurnGameEvents = []engine.GameEvent{}
	}

	return turnPhase, planGameEvents, resolveTurnGameEvents, nil
}

// ResetTurn sends ResetTurn signal to engine to drop the current WorkingState and reset to TrueState in a given MatchRoom.
// Returns an error if any pre-check is violated
func (s *ServerStateManager) ResetTurn(roomID, token string) error {
	room, err := s.loadRoom(roomID)
	if err != nil {
		return err
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.Match == nil {
		room.Logger.Warn("match not found")
		return fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, roomID)
	}

	teamID := room.Match.WorkingState.ActiveTeam
	if err := room.validatePlayerToken(teamID, token); err != nil {
		return err
	}

	room.Match.ResetTurn()

	room.LastActivity = time.Now()
	return nil
}

// ResolveTurn sends ResolveTurn signal to engine to calculate the impacts of the Player's action in a given MatchRoom.
// Returns the planning and resolution gameEvents or an error if any pre-check is violated
func (s *ServerStateManager) ResolveTurn(roomID, token string) ([]engine.GameEvent, []engine.GameEvent, error) {
	room, err := s.loadRoom(roomID)
	if err != nil {
		return nil, nil, err
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.Match == nil {
		room.Logger.Warn("match not found")
		return nil, nil, fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, roomID)
	}

	teamID := room.Match.WorkingState.ActiveTeam
	if err := room.validatePlayerToken(teamID, token); err != nil {
		return nil, nil, err
	}

	planGameEvents, resolveTurnGameEvents := room.Match.ResolveTurn()
	room.LastActivity = time.Now()

	return planGameEvents, resolveTurnGameEvents, nil
}

// ResetTurn sends Surrender signal to engine to end the current Match in a given MatchRoom.
// Returns the gameEvents or an error if any pre-check is violated
func (s *ServerStateManager) Surrender(roomID string, teamID int, token string) ([]engine.GameEvent, error) {
	if teamID != 1 && teamID != 2 {
		return nil, fmt.Errorf("%w: team must be 1 or 2", ErrInvalidConfig)
	}

	room, err := s.loadRoom(roomID)
	if err != nil {
		return nil, err
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.Match == nil {
		room.Logger.Warn("match not found")
		return nil, fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, roomID)
	}

	if err := room.validatePlayerToken(teamID, token); err != nil {
		return nil, err
	}

	gameEvents := room.Match.Surrender(teamID)
	room.LastActivity = time.Now()

	return gameEvents, nil
}

// GetMatchConfig gets the GameConfig of the current Match in a given MatchRoom.
func (s *ServerStateManager) GetMatchConfig(roomID string) (*engine.GameCfg, error) {
	room, err := s.loadRoom(roomID)
	if err != nil {
		return nil, err
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	if room.Match == nil {
		room.Logger.Warn("match not found")
		return nil, fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, roomID)
	}

	gameCfg := room.Match.GameCfg
	return &gameCfg, nil
}

// GetAllowedTiles gets the hints for Player to identify which tiles are available according to the TurnCmdAction
// Returns the coordinates of the allowed tiles or an error if any pre-check is violated
func (s *ServerStateManager) GetAllowedTiles(roomID string, unitID engine.UnitID, turnCmdType engine.TurnCmdType) ([]engine.Coordinate, error) {
	room, err := s.loadRoom(roomID)
	if err != nil {
		return nil, err
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	if room.Match == nil {
		room.Logger.Warn("match not found")
		return nil, fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, roomID)
	}

	allowedTiles, err := room.Match.WorkingState.FindAllowedTilesForCommand(unitID, turnCmdType)

	if err != nil {
		return nil, err
	}

	// slices.Collect yields nil for an empty map; the client types this as a non-nullable array.
	tiles := make([]engine.Coordinate, 0, len(allowedTiles))
	return slices.AppendSeq(tiles, maps.Keys(allowedTiles)), nil
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

// cleanupInactiveRooms removes rooms inactive > RoomInactivityTimeout.
func (s *ServerStateManager) cleanupInactiveRooms() {
	now := time.Now()
	s.Rooms.Range(func(key, value any) bool {
		room := value.(*MatchRoom)
		room.mu.Lock()
		inactive := now.Sub(room.LastActivity) > RoomInactivityTimeout
		room.mu.Unlock()

		if inactive {
			s.Rooms.Delete(key)
			s.Logger.Info("removed room", "roomID", key, "inactive", inactive)
		}
		return true
	})
}
