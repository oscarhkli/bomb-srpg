package server

import (
	"bomb-srpg/engine"
	"encoding/json/v2"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

// Handler wraps HTTP handlers with a logger.
type Handler struct {
	Manager *ServerStateManager
	Logger  *slog.Logger
}

// HandlerOption configures a Handler.
type HandlerOption func(*Handler)

// WithHandlerLogger sets the logger for the Handler.
func WithHandlerLogger(logger *slog.Logger) HandlerOption {
	return func(h *Handler) {
		h.Logger = logger
	}
}

// NewHandler creates a new Handler with the given ServerStateManager.
func NewHandler(m *ServerStateManager, opts ...HandlerOption) *Handler {
	h := &Handler{
		Manager: m,
		Logger:  slog.Default(),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func (h *Handler) requireToken(w http.ResponseWriter, r *http.Request) (string, bool) {
	token, err := h.extractBearerToken(r)
	if err != nil {
		code, msg := mapError(err)
		http.Error(w, msg, code)
		return "", false
	}
	return token, true
}

func (h *Handler) handleError(w http.ResponseWriter, err error, logMsg string, args ...any) {
	code, msg := mapError(err)
	h.Logger.Warn(logMsg, append(args, "error", err)...)
	http.Error(w, msg, code)
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, res any, logMsg string, args ...any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.MarshalWrite(w, res); err != nil {
		h.Logger.Error(logMsg, append(args, "error", err)...)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *Handler) extractBearerToken(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", errInvalidToken
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == auth {
		return "", errInvalidToken
	}
	return token, nil
}

type healthCheckResponse struct {
	Status string `json:"status"`
}

func (h *Handler) handleHealthCheck(w http.ResponseWriter, _ *http.Request) {
	res := healthCheckResponse{Status: "ok"}
	h.writeJSON(w, http.StatusOK, res, "encode health check failed")
}

type catalogResponse struct {
	Archetypes   []engine.Archetype   `json:"archetypes"`
	StagePresets []engine.StagePreset `json:"stagePresets"`
}

func (h *Handler) handleGetCatalog(w http.ResponseWriter, _ *http.Request) {
	archetypes := engine.GetAllArchetypes()
	stagePresets := engine.GetAllStagePresets()

	res := catalogResponse{Archetypes: archetypes, StagePresets: stagePresets}
	h.writeJSON(w, http.StatusOK, res, "encode catalog failed")
}

type createMatchRoomResponse struct {
	ID string `json:"id"`
}

func (h *Handler) handleCreateMatchRoom(w http.ResponseWriter, _ *http.Request) {
	id, err := h.Manager.createMatchRoom()
	if err != nil {
		h.Logger.Error("create match room failed", "error", err)
		http.Error(w, "Failed to create new MatchRoom", http.StatusInternalServerError)
		return
	}

	location := fmt.Sprintf("/api/match-rooms/%s", id)
	w.Header().Set("Location", location)

	res := createMatchRoomResponse{ID: id}
	h.writeJSON(w, http.StatusCreated, res, "encode match room failed", "roomID", id)

	h.Logger.Info("match room created", "roomID", id)
}

type createMatchRequest struct {
	GameCfg engine.GameCfg `json:"gameCfg"`
}

type createMatchResponse struct {
	Success      bool      `json:"success"`
	PlayerTokens [2]string `json:"playerTokens"`
}

func (h *Handler) handleCreateMatch(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	var req createMatchRequest
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		h.Logger.Warn("invalid config format", "error", err)
		http.Error(w, "Invalid configuration format", http.StatusBadRequest)
		return
	}

	tokens, err := h.Manager.createMatch(roomID, req.GameCfg)

	if !h.writeMatchCreated(tokens, err, roomID, w) {
		return
	}

	h.Logger.Info("match created", "roomID", roomID)
}

func (h *Handler) writeMatchCreated(tokens [2]string, err error, roomID string, w http.ResponseWriter) bool {
	if err != nil {
		h.handleError(w, err, "create match failed", "roomID", roomID)
		return false
	}

	res := createMatchResponse{Success: true, PlayerTokens: tokens}
	h.writeJSON(w, http.StatusCreated, res, "encode match failed", "roomID", roomID)
	return true
}

func (h *Handler) handleRematch(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	token, ok := h.requireToken(w, r)
	if !ok {
		return
	}

	tokens, err := h.Manager.rematch(roomID, token)

	if !h.writeMatchCreated(tokens, err, roomID, w) {
		return
	}

	h.Logger.Info("rematch created", "roomID", roomID)
}

func (h *Handler) handleDeleteMatch(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	token, ok := h.requireToken(w, r)
	if !ok {
		return
	}

	if err := h.Manager.deleteMatch(roomID, token); err != nil {
		h.handleError(w, err, "delete match failed", "roomID", roomID)
		return
	}

	w.WriteHeader(http.StatusNoContent)

	h.Logger.Info("match deleted", "roomID", roomID)
}

func (h *Handler) handleGetMatchState(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	gs, err := h.Manager.getMatchState(roomID)
	if err != nil {
		h.handleError(w, err, "get match state failed", "roomID", roomID)
		return
	}

	h.writeJSON(w, http.StatusOK, gs, "encode gameState failed")
}

func (h *Handler) handleSubmitTurnCommand(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	var req engine.TurnCommand
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		h.Logger.Warn("invalid turnCommand format", "error", err)
		http.Error(w, "Invalid turnCommand format", http.StatusBadRequest)
		return
	}

	token, ok := h.requireToken(w, r)
	if !ok {
		return
	}

	gameEvents, err := h.Manager.submitTurnCommand(roomID, req, token)
	if err != nil {
		h.handleError(w, err, "submit turn command failed", "roomID", roomID)
		return
	}

	h.writeJSON(w, http.StatusOK, gameEvents, "encode gameEvents failed")
}

// cpuStatusResponse wraps CPUTurnPhase and the CPU turn's gameEvents, split by phase so the
// client can animate the CPU's actions and their resolution apart.
type cpuStatusResponse struct {
	TurnPhase             engine.CPUTurnPhase `json:"turnPhase"`
	PlanGameEvents        []engine.GameEvent  `json:"planGameEvents"`
	ResolveTurnGameEvents []engine.GameEvent  `json:"resolveTurnGameEvents"`
}

// handleConsumeCPUStatus returns the CPU turn phase and events, and resets a Ready phase to Idle.
// It launches the CPU turn first if one is due.
func (h *Handler) handleConsumeCPUStatus(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	token, ok := h.requireToken(w, r)
	if !ok {
		return
	}

	turnPhase, planGameEvents, resolveTurnGameEvents, err := h.Manager.consumeCPUStatus(roomID, token)
	if err != nil {
		h.handleError(w, err, "consume cpu status failed", "roomID", roomID)
		return
	}
	h.Logger.Debug("consume cpu status", "roomID", roomID, "turnPhase", turnPhase, "planGameEvents", planGameEvents, "resolveTurnGameEvents", resolveTurnGameEvents)

	res := cpuStatusResponse{TurnPhase: turnPhase, PlanGameEvents: planGameEvents, ResolveTurnGameEvents: resolveTurnGameEvents}
	h.writeJSON(w, http.StatusOK, res, "encode cpu status response failed")
}

func (h *Handler) handleResetTurn(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	token, ok := h.requireToken(w, r)
	if !ok {
		return
	}

	if err := h.Manager.resetTurn(roomID, token); err != nil {
		h.handleError(w, err, "reset turn failed", "roomID", roomID)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// resolveTurnResponse splits the turn's events by phase so the client can animate the resolution apart.
type resolveTurnResponse struct {
	PlanGameEvents        []engine.GameEvent `json:"planGameEvents"`
	ResolveTurnGameEvents []engine.GameEvent `json:"resolveTurnGameEvents"`
}

func (h *Handler) handleResolveTurn(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	token, ok := h.requireToken(w, r)
	if !ok {
		return
	}

	planGameEvents, resolveTurnGameEvents, err := h.Manager.resolveTurn(roomID, token)
	if err != nil {
		h.handleError(w, err, "resolve turn failed", "roomID", roomID)
		return
	}
	h.Logger.Debug("resolve turn", "roomID", roomID, "planGameEvents", planGameEvents, "resolveTurnGameEvents", resolveTurnGameEvents)

	res := resolveTurnResponse{PlanGameEvents: planGameEvents, ResolveTurnGameEvents: resolveTurnGameEvents}
	h.writeJSON(w, http.StatusOK, res, "encode resolve turn response failed")
}

type surrenderRequest struct {
	TeamID int `json:"teamId"`
}

func (h *Handler) handleSurrender(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	var req surrenderRequest
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		h.Logger.Warn("invalid surrender request format", "error", err)
		http.Error(w, "Invalid surrenderRequest format", http.StatusBadRequest)
		return
	}

	token, ok := h.requireToken(w, r)
	if !ok {
		return
	}

	gameEvents, err := h.Manager.surrender(roomID, req.TeamID, token)
	if err != nil {
		h.handleError(w, err, "surrender failed", "roomID", roomID)
		return
	}

	h.writeJSON(w, http.StatusOK, gameEvents, "encode gameEvents failed")
}

func (h *Handler) handleGetMatchConfig(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")
	gameCfg, err := h.Manager.getMatchConfig(roomID)
	if err != nil {
		h.handleError(w, err, "get match config failed", "roomID", roomID)
		return
	}

	h.writeJSON(w, http.StatusOK, gameCfg, "encode gameConfig failed")
}

func (h *Handler) handleGetAllowedTiles(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")
	unitIDStr := r.URL.Query().Get("unitId")
	turnCmdType := r.URL.Query().Get("turnCmdType")

	if unitIDStr == "" || turnCmdType == "" {
		http.Error(w, "missing required query parameters: unitId and turnCmdType are required", http.StatusBadRequest)
		return
	}

	unitID, err := strconv.ParseUint(unitIDStr, 10, 8)
	if err != nil {
		http.Error(w, "Invalid unitId parameter", http.StatusBadRequest)
		return
	}

	allowed, err := h.Manager.getAllowedTiles(roomID, engine.UnitID(unitID), engine.TurnCmdType(turnCmdType))
	if err != nil {
		h.handleError(w, err, "get allowed tiles failed", "roomID", roomID)
		return
	}

	h.writeJSON(w, http.StatusOK, allowed, "encode allowedTiles failed")
}
