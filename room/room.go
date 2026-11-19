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
	mu           sync.RWMutex
	id           string
	match        *engine.Match
	cfg          *engine.GameCfg
	lastActivity time.Time
	tokens       [2]string
	log          *slog.Logger
	deps         Deps
}

func New(id string, deps Deps) *Room {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	return &Room{
		id:           id,
		lastActivity: deps.Now(),
		log:          deps.Logger.With("roomID", id),
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

	if !matchToken(r.tokens[idx], token) {
		return ErrInvalidToken
	}
	return nil
}

func (r *Room) rejectDuringCPUTurn() error {
	if r.match.CPU.Phase == engine.TurnPhasePlanning {
		return ErrCPUTurnInProgress
	}
	return nil
}

func (r *Room) validateAnyMatchPlayerToken(token string) error {
	if !matchToken(r.tokens[0], token) && !matchToken(r.tokens[1], token) {
		return ErrInvalidToken
	}
	return nil
}

func (r *Room) buildMatchLocked(cfg engine.GameCfg) (*engine.Match, error) {
	match, err := r.deps.NewMatch(cfg)
	if err != nil {
		r.log.Error("invalid game config", "gameCfg", cfg, "error", err)
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}
	return match, nil
}

func (r *Room) Start(cfg engine.GameCfg) ([2]string, error) {
	cfg.P1Slots = slices.Clone(cfg.P1Slots)
	cfg.P2Slots = slices.Clone(cfg.P2Slots)

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.match != nil {
		r.log.Warn("match already exists")
		return [2]string{}, fmt.Errorf("%w: roomID=%s", ErrMatchExists, r.id)
	}

	match, err := r.buildMatchLocked(cfg)
	if err != nil {
		return [2]string{}, err
	}

	tokens, err := generatePlayerTokens()
	if err != nil {
		r.log.Warn("failed to generate player tokens", "roomID", r.id, "error", err)
		return [2]string{}, err
	}

	r.match = match
	r.cfg = &cfg
	r.tokens = tokens
	r.lastActivity = r.deps.Now()

	return tokens, nil
}

func (r *Room) Rematch(token string) ([2]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.validateAnyMatchPlayerToken(token); err != nil {
		return [2]string{}, err
	}

	if r.cfg == nil {
		r.log.Warn("previous match not found")
		return [2]string{}, fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, r.id)
	}

	match, err := r.buildMatchLocked(*r.cfg)
	if err != nil {
		return [2]string{}, err
	}

	r.match = match
	r.lastActivity = r.deps.Now()

	return r.tokens, nil
}

func (r *Room) Delete(token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.validateAnyMatchPlayerToken(token); err != nil {
		return err
	}

	if r.match == nil {
		r.log.Info("match not found, no-op")
		return nil
	}

	if r.match.WinnerTeamID == 0 {
		r.log.Warn("match still in progress")
		return fmt.Errorf("%w: roomID=%s", ErrMatchInProgress, r.id)
	}

	r.match = nil

	return nil
}

func (r *Room) matchNotFound() error {
	r.log.Warn("match not found")
	return fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, r.id)
}

func (r *Room) read(f func(*engine.Match) error) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.match == nil {
		return r.matchNotFound()
	}
	return f(r.match)
}

type cpuPolicy int

const (
	rejectInCPUTurn cpuPolicy = iota
	allowInCPUTurn
)

func activeTeam(match *engine.Match) int { return match.WorkingState.ActiveTeam }

func (r *Room) command(token string, teamOf func(*engine.Match) int, policy cpuPolicy, f func(*engine.Match) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.match == nil {
		return r.matchNotFound()
	}
	if err := r.validatePlayerToken(teamOf(r.match), token); err != nil {
		return err
	}
	if policy == rejectInCPUTurn {
		if err := r.rejectDuringCPUTurn(); err != nil {
			return err
		}
	}
	if err := f(r.match); err != nil {
		return err
	}
	r.lastActivity = r.deps.Now()
	return nil
}

// State returns a copy of the WorkingState, safe to marshal after the lock is released.
func (r *Room) State() (*engine.GameState, error) {
	var gs *engine.GameState
	err := r.read(func(match *engine.Match) error {
		gs = match.WorkingState.DeepCopy()
		return nil
	})
	return gs, err
}

func (r *Room) Config() (engine.GameCfg, error) {
	var cfg engine.GameCfg
	err := r.read(func(match *engine.Match) error {
		cfg = match.GameCfg
		cfg.P1Slots = slices.Clone(cfg.P1Slots)
		cfg.P2Slots = slices.Clone(cfg.P2Slots)
		return nil
	})
	return cfg, err
}

func (r *Room) AllowedTiles(unitID engine.UnitID, turnCmdType engine.TurnCmdType) ([]engine.Coordinate, error) {
	var tiles []engine.Coordinate
	err := r.read(func(match *engine.Match) error {
		allowedTiles, err := match.WorkingState.FindAllowedTilesForCommand(unitID, turnCmdType)
		if err != nil {
			return err
		}
		tiles = slices.Collect(maps.Keys(allowedTiles))
		return nil
	})
	return tiles, err
}

func (r *Room) Submit(cmd engine.TurnCommand, token string) ([]engine.GameEvent, error) {
	teamID, _ := cmd.UnitID.Decode()
	var gameEvents []engine.GameEvent
	err := r.command(token, func(*engine.Match) int { return teamID }, rejectInCPUTurn, func(match *engine.Match) error {
		events, err := match.ApplyTurnCommand(cmd)
		if err != nil {
			r.log.Error("invalid turn command", "turnCommand", cmd, "error", err)
			return fmt.Errorf("%w: %w", ErrInvalidTurnCmd, err)
		}
		gameEvents = events
		return nil
	})
	return gameEvents, err
}

func (r *Room) Reset(token string) error {
	return r.command(token, activeTeam, rejectInCPUTurn, func(match *engine.Match) error {
		match.ResetTurn()
		return nil
	})
}

func (r *Room) Resolve(token string) (plan, resolved []engine.GameEvent, err error) {
	err = r.command(token, activeTeam, rejectInCPUTurn, func(match *engine.Match) error {
		plan, resolved = match.ResolveTurn()
		return nil
	})
	return plan, resolved, err
}

func (r *Room) Surrender(teamID int, token string) ([]engine.GameEvent, error) {
	if teamID != 1 && teamID != 2 {
		return nil, fmt.Errorf("%w: team must be 1 or 2", ErrInvalidConfig)
	}

	var gameEvents []engine.GameEvent
	err := r.command(token, func(*engine.Match) int { return teamID }, allowInCPUTurn, func(match *engine.Match) error {
		gameEvents = match.Surrender(teamID)
		return nil
	})
	return gameEvents, err
}

func (r *Room) Expired(now time.Time, idle time.Duration) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return now.Sub(r.lastActivity) > idle
}
