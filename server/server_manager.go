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

const CleanupInterval = 60 * time.Minute

var (
	errRoomNotFound      = errors.New("room not found")
	errMatchInProgress   = errors.New("match still in progress")
	errMatchExists       = errors.New("match already exists")
	errMatchNotFound     = errors.New("match not found")
	errInvalidConfig     = errors.New("invalid game config")
	errInvalidTurnCmd    = errors.New("invalid turn command")
	errInvalidToken      = errors.New("invalid player token")
	errCPUTurnInProgress = errors.New("cpu turn in progress")
)

func mapError(err error) (int, string) {
	switch {
	case errors.Is(err, errRoomNotFound), errors.Is(err, errMatchNotFound):
		return http.StatusNotFound, err.Error()
	case errors.Is(err, errMatchExists):
		return http.StatusConflict, err.Error()
	case errors.Is(err, errInvalidConfig):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, errInvalidTurnCmd), errors.Is(err, errCPUTurnInProgress):
		return http.StatusConflict, err.Error()
	case errors.Is(err, errInvalidToken):
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
		errors.Is(err, errMatchInProgress):
		return http.StatusConflict, err.Error()
	default:
		return http.StatusInternalServerError, "internal error"
	}
}

type matchRoom struct {
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
	decide         func(*engine.GameState) ([]engine.TurnCommand, error)
	Logger         *slog.Logger
}

type Option func(*ServerStateManager)

func WithLogger(logger *slog.Logger) Option {
	return func(s *ServerStateManager) {
		s.Logger = logger
	}
}

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

const roomIDBytes = 5

func (s *ServerStateManager) createMatchRoom() (string, error) {
	maxRetry := 5

	for range maxRetry {
		id, err := s.generateRoomID(roomIDBytes)
		if err != nil {
			s.Logger.Warn("failed to generate room ID", "error", err)
			return "", fmt.Errorf("failed to generate room ID: %w", err)
		}
		room := &matchRoom{
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

// loadRoom does not lock. Callers must hold room.mu before touching room.Match, which a CPU goroutine may be mutating.
func (s *ServerStateManager) loadRoom(roomID string) (*matchRoom, error) {
	roomVal, ok := s.Rooms.Load(roomID)
	if !ok {
		s.Logger.Warn("match room not found", "roomID", roomID)
		return nil, fmt.Errorf("%w: roomID=%s", errRoomNotFound, roomID)
	}
	room, ok := roomVal.(*matchRoom)
	if !ok {
		s.Logger.Error("room has unexpected type", "roomID", roomID, "type", fmt.Sprintf("%T", roomVal))
		return nil, fmt.Errorf("room %s holds unexpected type %T", roomID, roomVal)
	}
	return room, nil
}

func randomHex(byteLength int) (string, error) {
	b := make([]byte, byteLength)
	if _, err := cryptorand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

const tokenBytes = 16

func generatePlayerTokens() ([2]string, error) {
	var tokens [2]string
	for i := range 2 {
		token, err := randomHex(tokenBytes)
		if err != nil {
			return [2]string{}, fmt.Errorf("failed to generate playerToken for Player %d: %w", i, err)
		}
		tokens[i] = token
	}
	return tokens, nil
}

func (s *ServerStateManager) createMatch(roomID string, gameCfg engine.GameCfg) ([2]string, error) {
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

func (mr *matchRoom) validatePlayerToken(teamID int, token string) error {
	idx := teamID - 1
	if idx < 0 || idx > 1 {
		return errInvalidConfig
	}

	if !matchToken(mr.PlayerTokens[idx], token) {
		return errInvalidToken
	}
	return nil
}

func (mr *matchRoom) rejectDuringCPUTurn() error {
	if mr.Match.CPU.Phase == engine.TurnPhasePlanning {
		return errCPUTurnInProgress
	}
	return nil
}

func (mr *matchRoom) validateAnyMatchPlayerToken(token string) error {
	if !matchToken(mr.PlayerTokens[0], token) && !matchToken(mr.PlayerTokens[1], token) {
		return errInvalidToken
	}
	return nil
}

func matchToken(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func (s *ServerStateManager) createMatchLocked(room *matchRoom, gameCfg engine.GameCfg) (*engine.Match, error) {
	roomID := room.ID
	if room.Match != nil {
		room.Logger.Warn("match already exists")
		return nil, fmt.Errorf("%w: roomID=%s", errMatchExists, roomID)
	}

	match, err := engine.InitGame(gameCfg)
	if err != nil {
		room.Logger.Error("invalid game config", "gameCfg", gameCfg, "error", err)
		return nil, fmt.Errorf("%w: %w", errInvalidConfig, err)
	}

	return match, nil
}

func (s *ServerStateManager) rematch(roomID, token string) ([2]string, error) {
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
		return [2]string{}, fmt.Errorf("%w: roomID=%s", errMatchNotFound, roomID)
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

func (s *ServerStateManager) deleteMatch(roomID, token string) error {
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
		return fmt.Errorf("%w: roomID=%s", errMatchInProgress, roomID)
	}

	room.Match = nil

	return nil
}

// getMatchState returns a copy of the WorkingState, safe to marshal after the lock is released.
func (s *ServerStateManager) getMatchState(roomID string) (*engine.GameState, error) {
	room, err := s.loadRoom(roomID)
	if err != nil {
		return nil, err
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	if room.Match == nil {
		room.Logger.Warn("match not found")
		return nil, fmt.Errorf("%w: roomID=%s", errMatchNotFound, roomID)
	}

	return room.Match.WorkingState.DeepCopy(), nil
}

func (s *ServerStateManager) submitTurnCommand(roomID string, cmd engine.TurnCommand, token string) ([]engine.GameEvent, error) {
	room, err := s.loadRoom(roomID)
	if err != nil {
		return nil, err
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.Match == nil {
		s.Logger.Warn("match not found", "roomID", roomID)
		return nil, fmt.Errorf("%w: roomID=%s", errMatchNotFound, roomID)
	}

	teamID, _ := cmd.UnitID.Decode()
	if err := room.validatePlayerToken(teamID, token); err != nil {
		return nil, err
	}
	if err := room.rejectDuringCPUTurn(); err != nil {
		return nil, err
	}

	gameEvents, err := room.Match.ApplyTurnCommand(cmd)
	if err != nil {
		room.Logger.Error("invalid turn command", "turnCommand", cmd, "error", err)
		return nil, fmt.Errorf("%w: %w", errInvalidTurnCmd, err)
	}

	room.LastActivity = time.Now()
	return gameEvents, nil
}

const maxCPUReplanAttempts = 5

// Phase Ready on team 2's turn is stale: the finished turn would have moved play to team 1 or set a winner.
func (s *ServerStateManager) launchCPUTurnIfDueLocked(room *matchRoom) {
	match := room.Match
	if !match.GameCfg.VSCpu || match.WinnerTeamID != 0 || match.WorkingState.ActiveTeam != engine.Team2 ||
		match.CPU.Phase == engine.TurnPhasePlanning {
		return
	}
	match.CPU.Phase = engine.TurnPhasePlanning
	match.CPU.PlanGameEvents = nil
	match.CPU.ResolveTurnGameEvents = nil
	go s.runCPUTurn(room, match)
}

// runCPUTurn abandons if the match was replaced, deleted, or already won while the goroutine was queued.
func (s *ServerStateManager) runCPUTurn(room *matchRoom, match *engine.Match) {
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
	if match.WorkingState.ActiveTeam != engine.Team2 {
		room.Logger.Info("CPU turn abandoned: not the CPU's turn")
		return
	}

	// RecoverPanic does not span goroutines. Forfeit the turn from a clean sandbox;
	// a panic from the forfeit itself stays fatal.
	defer func() {
		if rec := recover(); rec != nil {
			room.Logger.Error("CPU turn panicked, forfeiting", "panic", rec)
			match.ResetTurn()
			match.CPU.PlanGameEvents, match.CPU.ResolveTurnGameEvents = match.ResolveTurn()
		}
	}()

	var err error
	for attempt := range maxCPUReplanAttempts {
		cmds, decideErr := s.decide(match.WorkingState)
		if decideErr != nil {
			room.Logger.Warn("CPU decide reported errors", "attempt", attempt, "error", decideErr)
		}
		if err = applyPlan(match, cmds); err == nil {
			break
		}
		room.Logger.Warn("CPU plan rejected, replanning", "attempt", attempt, "error", err)
	}
	if err != nil {
		room.Logger.Error("CPU replan attempts exhausted, committing partial turn", "attempts", maxCPUReplanAttempts, "error", err)
	}

	match.CPU.PlanGameEvents, match.CPU.ResolveTurnGameEvents = match.ResolveTurn()
}

// applyPlan applies the plan in order and returns the first rejection, if any.
func applyPlan(match *engine.Match, plan []engine.TurnCommand) error {
	for _, cmd := range plan {
		if _, err := match.ApplyTurnCommand(cmd); err != nil {
			return fmt.Errorf("command %+v rejected: %w", cmd, err)
		}
	}
	return nil
}

// consumeCPUStatus returns the CPU turn phase and events, and resets a Ready phase to Idle.
// It launches the CPU turn first if one is due.
func (s *ServerStateManager) consumeCPUStatus(roomID, token string) (engine.CPUTurnPhase, []engine.GameEvent, []engine.GameEvent, error) {
	room, err := s.loadRoom(roomID)
	if err != nil {
		return 0, nil, nil, err
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.Match == nil {
		room.Logger.Warn("match not found")
		return 0, nil, nil, fmt.Errorf("%w: roomID=%s", errMatchNotFound, roomID)
	}

	if err := room.validateAnyMatchPlayerToken(token); err != nil {
		return 0, nil, nil, err
	}

	s.launchCPUTurnIfDueLocked(room)

	turnPhase := room.Match.CPU.Phase
	planGameEvents, resolveTurnGameEvents := room.Match.CPU.PlanGameEvents, room.Match.CPU.ResolveTurnGameEvents
	if turnPhase == engine.TurnPhaseReady {
		room.Match.CPU.PlanGameEvents = nil
		room.Match.CPU.ResolveTurnGameEvents = nil
		room.Match.CPU.Phase = engine.TurnPhaseIdle
	}

	return turnPhase, planGameEvents, resolveTurnGameEvents, nil
}

func (s *ServerStateManager) resetTurn(roomID, token string) error {
	room, err := s.loadRoom(roomID)
	if err != nil {
		return err
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.Match == nil {
		room.Logger.Warn("match not found")
		return fmt.Errorf("%w: roomID=%s", errMatchNotFound, roomID)
	}

	teamID := room.Match.WorkingState.ActiveTeam
	if err := room.validatePlayerToken(teamID, token); err != nil {
		return err
	}
	if err := room.rejectDuringCPUTurn(); err != nil {
		return err
	}

	room.Match.ResetTurn()

	room.LastActivity = time.Now()
	return nil
}

func (s *ServerStateManager) resolveTurn(roomID, token string) ([]engine.GameEvent, []engine.GameEvent, error) {
	room, err := s.loadRoom(roomID)
	if err != nil {
		return nil, nil, err
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.Match == nil {
		room.Logger.Warn("match not found")
		return nil, nil, fmt.Errorf("%w: roomID=%s", errMatchNotFound, roomID)
	}

	teamID := room.Match.WorkingState.ActiveTeam
	if err := room.validatePlayerToken(teamID, token); err != nil {
		return nil, nil, err
	}
	if err := room.rejectDuringCPUTurn(); err != nil {
		return nil, nil, err
	}

	planGameEvents, resolveTurnGameEvents := room.Match.ResolveTurn()
	room.LastActivity = time.Now()

	return planGameEvents, resolveTurnGameEvents, nil
}

func (s *ServerStateManager) surrender(roomID string, teamID int, token string) ([]engine.GameEvent, error) {
	if teamID != 1 && teamID != 2 {
		return nil, fmt.Errorf("%w: team must be 1 or 2", errInvalidConfig)
	}

	room, err := s.loadRoom(roomID)
	if err != nil {
		return nil, err
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.Match == nil {
		room.Logger.Warn("match not found")
		return nil, fmt.Errorf("%w: roomID=%s", errMatchNotFound, roomID)
	}

	if err := room.validatePlayerToken(teamID, token); err != nil {
		return nil, err
	}

	gameEvents := room.Match.Surrender(teamID)
	room.LastActivity = time.Now()

	return gameEvents, nil
}

func (s *ServerStateManager) getMatchConfig(roomID string) (*engine.GameCfg, error) {
	room, err := s.loadRoom(roomID)
	if err != nil {
		return nil, err
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	if room.Match == nil {
		room.Logger.Warn("match not found")
		return nil, fmt.Errorf("%w: roomID=%s", errMatchNotFound, roomID)
	}

	gameCfg := room.Match.GameCfg
	return &gameCfg, nil
}

func (s *ServerStateManager) getAllowedTiles(roomID string, unitID engine.UnitID, turnCmdType engine.TurnCmdType) ([]engine.Coordinate, error) {
	room, err := s.loadRoom(roomID)
	if err != nil {
		return nil, err
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	if room.Match == nil {
		room.Logger.Warn("match not found")
		return nil, fmt.Errorf("%w: roomID=%s", errMatchNotFound, roomID)
	}

	allowedTiles, err := room.Match.WorkingState.FindAllowedTilesForCommand(unitID, turnCmdType)

	if err != nil {
		return nil, err
	}

	return slices.Collect(maps.Keys(allowedTiles)), nil
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
			s.cleanupInactiveRooms()
		}
	}
}

const roomInactivityTimeout = 60 * time.Minute

func (s *ServerStateManager) cleanupInactiveRooms() {
	now := time.Now()
	s.Rooms.Range(func(key, value any) bool {
		room, ok := value.(*matchRoom)
		if !ok {
			s.Rooms.Delete(key)
			s.Logger.Error("removed room of unexpected type", "roomID", key, "type", fmt.Sprintf("%T", value))
			return true
		}
		room.mu.Lock()
		inactive := now.Sub(room.LastActivity) > roomInactivityTimeout
		room.mu.Unlock()

		if inactive {
			s.Rooms.Delete(key)
			s.Logger.Info("removed room", "roomID", key, "inactive", inactive)
		}
		return true
	})
}
