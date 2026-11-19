package room

import (
	"bomb-srpg/engine"
	cryptorand "crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"sync"
	"time"
)

var (
	ErrMatchInProgress   = errors.New("match still in progress")
	ErrMatchExists       = errors.New("match already exists")
	ErrMatchNotFound     = errors.New("match not found")
	ErrInvalidConfig     = errors.New("invalid game config")
	ErrInvalidTurnCmd    = errors.New("invalid turn command")
	ErrInvalidToken      = errors.New("invalid player token")
	ErrCPUTurnInProgress = errors.New("cpu turn in progress")
)

type Deps struct {
	Logger   *slog.Logger
	Decide   func(*engine.GameState) ([]engine.TurnCommand, error)
	Spawn    func(func())
	NewMatch func(engine.GameCfg) (*engine.Match, error)
	Now      func() time.Time
}

type Room struct {
	Mu           sync.RWMutex
	ID           string
	Match        *engine.Match
	GameCfg      *engine.GameCfg
	LastActivity time.Time
	PlayerTokens [2]string
	Logger       *slog.Logger
	deps         Deps
}

func New(id string, deps Deps) *Room {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	return &Room{
		ID:           id,
		LastActivity: deps.Now(),
		Logger:       deps.Logger.With("roomID", id),
		deps:         deps,
	}
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

func matchToken(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func (r *Room) validatePlayerToken(teamID int, token string) error {
	idx := teamID - 1
	if idx < 0 || idx > 1 {
		return ErrInvalidConfig
	}

	if !matchToken(r.PlayerTokens[idx], token) {
		return ErrInvalidToken
	}
	return nil
}

func (r *Room) rejectDuringCPUTurn() error {
	if r.Match.CPU.Phase == engine.TurnPhasePlanning {
		return ErrCPUTurnInProgress
	}
	return nil
}

func (r *Room) validateAnyMatchPlayerToken(token string) error {
	if !matchToken(r.PlayerTokens[0], token) && !matchToken(r.PlayerTokens[1], token) {
		return ErrInvalidToken
	}
	return nil
}

func (r *Room) buildMatchLocked(cfg engine.GameCfg) (*engine.Match, error) {
	match, err := r.deps.NewMatch(cfg)
	if err != nil {
		r.Logger.Error("invalid game config", "gameCfg", cfg, "error", err)
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}
	return match, nil
}

func (r *Room) Start(cfg engine.GameCfg) ([2]string, error) {
	cfg.P1Slots = slices.Clone(cfg.P1Slots)
	cfg.P2Slots = slices.Clone(cfg.P2Slots)

	r.Mu.Lock()
	defer r.Mu.Unlock()

	if r.Match != nil {
		r.Logger.Warn("match already exists")
		return [2]string{}, fmt.Errorf("%w: roomID=%s", ErrMatchExists, r.ID)
	}

	match, err := r.buildMatchLocked(cfg)
	if err != nil {
		return [2]string{}, err
	}

	tokens, err := generatePlayerTokens()
	if err != nil {
		r.Logger.Warn("failed to generate player tokens", "roomID", r.ID, "error", err)
		return [2]string{}, err
	}

	r.Match = match
	r.GameCfg = &cfg
	r.PlayerTokens = tokens
	r.LastActivity = r.deps.Now()

	return tokens, nil
}

func (r *Room) Rematch(token string) ([2]string, error) {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	if err := r.validateAnyMatchPlayerToken(token); err != nil {
		return [2]string{}, err
	}

	if r.GameCfg == nil {
		r.Logger.Warn("previous match not found")
		return [2]string{}, fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, r.ID)
	}

	match, err := r.buildMatchLocked(*r.GameCfg)
	if err != nil {
		return [2]string{}, err
	}

	r.Match = match
	r.LastActivity = r.deps.Now()

	return r.PlayerTokens, nil
}

func (r *Room) Delete(token string) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	if err := r.validateAnyMatchPlayerToken(token); err != nil {
		return err
	}

	if r.Match == nil {
		r.Logger.Info("match not found, no-op")
		return nil
	}

	if r.Match.WinnerTeamID == 0 {
		r.Logger.Warn("match still in progress")
		return fmt.Errorf("%w: roomID=%s", ErrMatchInProgress, r.ID)
	}

	r.Match = nil

	return nil
}

// State returns a copy of the WorkingState, safe to marshal after the lock is released.
func (r *Room) State() (*engine.GameState, error) {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	if r.Match == nil {
		r.Logger.Warn("match not found")
		return nil, fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, r.ID)
	}

	return r.Match.WorkingState.DeepCopy(), nil
}

func (r *Room) Config() (engine.GameCfg, error) {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	if r.Match == nil {
		r.Logger.Warn("match not found")
		return engine.GameCfg{}, fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, r.ID)
	}

	cfg := r.Match.GameCfg
	cfg.P1Slots = slices.Clone(cfg.P1Slots)
	cfg.P2Slots = slices.Clone(cfg.P2Slots)
	return cfg, nil
}

func (r *Room) AllowedTiles(unitID engine.UnitID, turnCmdType engine.TurnCmdType) ([]engine.Coordinate, error) {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	if r.Match == nil {
		r.Logger.Warn("match not found")
		return nil, fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, r.ID)
	}

	allowedTiles, err := r.Match.WorkingState.FindAllowedTilesForCommand(unitID, turnCmdType)
	if err != nil {
		return nil, err
	}

	return slices.Collect(maps.Keys(allowedTiles)), nil
}

func (r *Room) Submit(cmd engine.TurnCommand, token string) ([]engine.GameEvent, error) {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	if r.Match == nil {
		r.Logger.Warn("match not found")
		return nil, fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, r.ID)
	}

	teamID, _ := cmd.UnitID.Decode()
	if err := r.validatePlayerToken(teamID, token); err != nil {
		return nil, err
	}
	if err := r.rejectDuringCPUTurn(); err != nil {
		return nil, err
	}

	gameEvents, err := r.Match.ApplyTurnCommand(cmd)
	if err != nil {
		r.Logger.Error("invalid turn command", "turnCommand", cmd, "error", err)
		return nil, fmt.Errorf("%w: %w", ErrInvalidTurnCmd, err)
	}

	r.LastActivity = r.deps.Now()
	return gameEvents, nil
}

func (r *Room) Reset(token string) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	if r.Match == nil {
		r.Logger.Warn("match not found")
		return fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, r.ID)
	}

	teamID := r.Match.WorkingState.ActiveTeam
	if err := r.validatePlayerToken(teamID, token); err != nil {
		return err
	}
	if err := r.rejectDuringCPUTurn(); err != nil {
		return err
	}

	r.Match.ResetTurn()

	r.LastActivity = r.deps.Now()
	return nil
}

func (r *Room) Resolve(token string) (plan, resolved []engine.GameEvent, err error) {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	if r.Match == nil {
		r.Logger.Warn("match not found")
		return nil, nil, fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, r.ID)
	}

	teamID := r.Match.WorkingState.ActiveTeam
	if err := r.validatePlayerToken(teamID, token); err != nil {
		return nil, nil, err
	}
	if err := r.rejectDuringCPUTurn(); err != nil {
		return nil, nil, err
	}

	plan, resolved = r.Match.ResolveTurn()
	r.LastActivity = r.deps.Now()

	return plan, resolved, nil
}

func (r *Room) Surrender(teamID int, token string) ([]engine.GameEvent, error) {
	if teamID != 1 && teamID != 2 {
		return nil, fmt.Errorf("%w: team must be 1 or 2", ErrInvalidConfig)
	}

	r.Mu.Lock()
	defer r.Mu.Unlock()

	if r.Match == nil {
		r.Logger.Warn("match not found")
		return nil, fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, r.ID)
	}

	if err := r.validatePlayerToken(teamID, token); err != nil {
		return nil, err
	}

	gameEvents := r.Match.Surrender(teamID)
	r.LastActivity = r.deps.Now()

	return gameEvents, nil
}

func (r *Room) Expired(now time.Time, idle time.Duration) bool {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	return now.Sub(r.LastActivity) > idle
}
