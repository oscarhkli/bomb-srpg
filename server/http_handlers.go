package server

import (
	"bomb-srpg/engine"
	"encoding/json"
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
	if err := json.NewEncoder(w).Encode(res); err != nil {
		h.Logger.Error(logMsg, append(args, "error", err)...)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *Handler) extractBearerToken(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", ErrInvalidToken
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == auth {
		return "", ErrInvalidToken
	}
	return token, nil
}

// HealthCheckResponse wraps the status (typically OK) when health check is called.
type HealthCheckResponse struct {
	Status string `json:"status"`
}

// HandleHealthCheck returns OK as a server health check.
func (h *Handler) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	res := HealthCheckResponse{Status: "ok"}
	h.writeJSON(w, http.StatusOK, res, "encode health check failed")
}

// CatalogResponse is returned when the Match Catalog is requested.
type CatalogResponse struct {
	Archetypes   []engine.Archetype   `json:"archetypes"`
	StagePresets []engine.StagePreset `json:"stagePresets"`
}

// HandleGetCatalog returns all available unit archetypes and stages for the client to display in the lobby.
// It encodes the archetype and stagePreset definitions as JSON and writes them to the response.
func (h *Handler) HandleGetCatalog(w http.ResponseWriter, r *http.Request) {
	archetypes := engine.GetAllArchetypes()
	stagePresets := engine.GetAllStagePresets()

	res := CatalogResponse{Archetypes: archetypes, StagePresets: stagePresets}
	h.writeJSON(w, http.StatusOK, res, "encode catalog failed")
}

// CreateMatchRoomResponse is returned when a new match room is created.
type CreateMatchRoomResponse struct {
	ID string `json:"id"`
}

// HandleCreateMatchRoom creates a new match room and returns its unique ID.
// The room is initialized without a match instance; the match is created when players join.
func (h *Handler) HandleCreateMatchRoom(w http.ResponseWriter, r *http.Request) {
	id, err := h.Manager.CreateMatchRoom()
	if err != nil {
		h.Logger.Error("create match room failed", "error", err)
		http.Error(w, "Failed to create new MatchRoom", http.StatusInternalServerError)
		return
	}

	location := fmt.Sprintf("/api/match-rooms/%s", id)
	w.Header().Set("Location", location)

	res := CreateMatchRoomResponse{ID: id}
	h.writeJSON(w, http.StatusCreated, res, "encode match room failed", "roomID", id)

	h.Logger.Info("match room created", "roomID", id)
}

// CreateMatchRequest wraps GameCfg for backward compatibility with existing clients.
type CreateMatchRequest struct {
	GameCfg engine.GameCfg `json:"gameCfg"`
}

// CreateMatchResponse is returned when a new match is created.
type CreateMatchResponse struct {
	Success      bool      `json:"success"`
	PlayerTokens [2]string `json:"playerTokens"`
}

// HandleCreateMatch creates a new match with given RoomID and GameCfg
func (h *Handler) HandleCreateMatch(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	var req CreateMatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Warn("invalid config format", "error", err)
		http.Error(w, "Invalid configuration format", http.StatusBadRequest)
		return
	}

	tokens, err := h.Manager.CreateMatch(roomID, req.GameCfg)

	if !h.handleCreateMatch(tokens, err, roomID, w) {
		return
	}

	h.Logger.Info("match created", "roomID", roomID)
}

func (h *Handler) handleCreateMatch(tokens [2]string, err error, roomID string, w http.ResponseWriter) bool {
	if err != nil {
		h.handleError(w, err, "create match failed", "roomID", roomID)
		return false
	}

	res := CreateMatchResponse{Success: true, PlayerTokens: tokens}
	h.writeJSON(w, http.StatusCreated, res, "encode match failed", "roomID", roomID)
	return true
}

// HandleRematch wipes the existing Match in a given MatchRoom and recreate one using GameCfg
func (h *Handler) HandleRematch(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	token, ok := h.requireToken(w, r)
	if !ok {
		return
	}

	tokens, err := h.Manager.Rematch(roomID, token)

	if !h.handleCreateMatch(tokens, err, roomID, w) {
		return
	}

	h.Logger.Info("rematch created", "roomID", roomID)
}

// HandleDeleteMatch removes the existing concluded Match in a given MatchRoom.
func (h *Handler) HandleDeleteMatch(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	token, ok := h.requireToken(w, r)
	if !ok {
		return
	}

	if err := h.Manager.DeleteMatch(roomID, token); err != nil {
		h.handleError(w, err, "delete match failed", "roomID", roomID)
		return
	}

	w.WriteHeader(http.StatusNoContent)

	h.Logger.Info("match deleted", "roomID", roomID)
}

// GetMatchState gets the WorkingState of the Match in a given MatchRoom.
// It encodes the gameState as JSON and writes them to the response.
func (h *Handler) HandleGetMatchState(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	gs, err := h.Manager.GetMatchState(roomID)
	if err != nil {
		h.handleError(w, err, "get match state failed", "roomID", roomID)
		return
	}

	h.writeJSON(w, http.StatusOK, gs, "encode gameState failed")
}

// HandleSubmitTurnCommand delivers TurnCommand to engine to move a Unit or place a bomb in a given MatchRoom.
// It encodes the gameEvents as JSON and writes them to the response.
func (h *Handler) HandleSubmitTurnCommand(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	var req engine.TurnCommand
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Warn("invalid turnCommand format", "error", err)
		http.Error(w, "Invalid turnCommand format", http.StatusBadRequest)
		return
	}

	token, ok := h.requireToken(w, r)
	if !ok {
		return
	}

	gameEvents, err := h.Manager.SubmitTurnCommand(roomID, req, token)
	if err != nil {
		h.handleError(w, err, "submit turn command failed", "roomID", roomID)
		return
	}

	h.writeJSON(w, http.StatusOK, gameEvents, "encode gameEvents failed")
}

// StartTurnResponse is returned to provide the result of Sudden Death
type StartTurnResponse struct {
	InSuddenDeath bool               `json:"inSuddenDeath"`
	GameEvents    []engine.GameEvent `json:"gameEvents"`
}

// HandleStartTurn sends StartTurn signal engine to start a new turn in a given MatchRoom.
// It encodes the gameEvents as JSON and writes them to the response.
func (h *Handler) HandleStartTurn(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	token, ok := h.requireToken(w, r)
	if !ok {
		return
	}

	inSuddenDeath, gameEvents, err := h.Manager.StartTurn(roomID, token)
	if err != nil {
		h.handleError(w, err, "start turn failed", "roomID", roomID)
		return
	}
	h.Logger.Debug("start turn", "roomID", roomID, "gameEvents", gameEvents)

	res := StartTurnResponse{InSuddenDeath: inSuddenDeath, GameEvents: gameEvents}
	h.writeJSON(w, http.StatusOK, res, "encode start turn response failed")
}

// CPUStatusResponse wraps CPUTurnPhase and the CPU turn's gameEvents, split by phase so the
// client can animate the CPU's actions and their resolution apart.
type CPUStatusResponse struct {
	TurnPhase             engine.CPUTurnPhase `json:"turnPhase"`
	PlanGameEvents        []engine.GameEvent  `json:"planGameEvents"`
	ResolveTurnGameEvents []engine.GameEvent  `json:"resolveTurnGameEvents"`
}

// HandleConsumeCPUStatus consumes current CPU Turn States in given MatchRoom and reset the state if CPU TurnPhase is in Ready - planning is done.
// It encodes the cpuTurnPhase, gameEvents as JSON and writes them to the response.
func (h *Handler) HandleConsumeCPUStatus(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	token, ok := h.requireToken(w, r)
	if !ok {
		return
	}

	turnPhase, planGameEvents, resolveTurnGameEvents, err := h.Manager.ConsumeCPUStatus(roomID, token)
	if err != nil {
		h.handleError(w, err, "consume cpu status failed", "roomID", roomID)
		return
	}
	h.Logger.Debug("consume cpu status", "roomID", roomID, "turnPhase", turnPhase, "planGameEvents", planGameEvents, "resolveTurnGameEvents", resolveTurnGameEvents)

	res := CPUStatusResponse{TurnPhase: turnPhase, PlanGameEvents: planGameEvents, ResolveTurnGameEvents: resolveTurnGameEvents}
	h.writeJSON(w, http.StatusOK, res, "encode cpu status response failed")
}

// HandleResetTurn sends ResetTurn signal to engine to drop the current WorkingState and reset to TrueState in a given MatchRoom.
// It writes HTTP 204 with no content to the response.
func (h *Handler) HandleResetTurn(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	token, ok := h.requireToken(w, r)
	if !ok {
		return
	}

	if err := h.Manager.ResetTurn(roomID, token); err != nil {
		h.handleError(w, err, "reset turn failed", "roomID", roomID)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ResolveTurnResponse wraps HumanTurnPhase and the Human Player turn's gameEvents, split by phase so the
// client can animate only the resolveTurnGameEvents.
type ResolveTurnResponse struct {
	PlanGameEvents        []engine.GameEvent `json:"planGameEvents"`
	ResolveTurnGameEvents []engine.GameEvent `json:"resolveTurnGameEvents"`
}

// HandleResolveTurn sends ResolveTurn signal to engine to calculate the impacts of the Player's action in a given MatchRoom.
// It encodes the ResolveTurnStatusResponse as JSON and writes them to the response.
func (h *Handler) HandleResolveTurn(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	token, ok := h.requireToken(w, r)
	if !ok {
		return
	}

	planGameEvents, resolveTurnGameEvents, err := h.Manager.ResolveTurn(roomID, token)
	if err != nil {
		h.handleError(w, err, "resolve turn failed", "roomID", roomID)
		return
	}
	h.Logger.Debug("resolve turn", "roomID", roomID, "planGameEvents", planGameEvents, "resolveTurnGameEvents", resolveTurnGameEvents)

	res := ResolveTurnResponse{PlanGameEvents: planGameEvents, ResolveTurnGameEvents: resolveTurnGameEvents}
	h.writeJSON(w, http.StatusOK, res, "encode resolve turn response failed")
}

// SurrenderRequest wraps TeamID for backward compatibility with existing clients.
type SurrenderRequest struct {
	TeamID int `json:"teamId"`
}

// HandleSurrender sends Surrender signal to engine to egnd the current Match in a given MatchRoom.
// It encodes the gameEvents as JSON and writes them to the response.
func (h *Handler) HandleSurrender(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	var req SurrenderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Warn("invalid surrender request format", "error", err)
		http.Error(w, "Invalid surrenderRequest format", http.StatusBadRequest)
		return
	}

	token, ok := h.requireToken(w, r)
	if !ok {
		return
	}

	gameEvents, err := h.Manager.Surrender(roomID, req.TeamID, token)
	if err != nil {
		h.handleError(w, err, "surrender failed", "roomID", roomID)
		return
	}

	h.writeJSON(w, http.StatusOK, gameEvents, "encode gameEvents failed")
}

// HandleGetMatchConfig gets the GameCfg of the current Match in a given MatchRoom
func (h *Handler) HandleGetMatchConfig(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")
	gameCfg, err := h.Manager.GetMatchConfig(roomID)
	if err != nil {
		h.handleError(w, err, "get match config failed", "roomID", roomID)
		return
	}

	h.writeJSON(w, http.StatusOK, gameCfg, "encode gameConfig failed")
}

// HandlesGetAllowedTiles gets the hints for Player to identify which tiles are available according to the TurnCmdAction
func (h *Handler) HandleGetAllowedTiles(w http.ResponseWriter, r *http.Request) {
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

	allowed, err := h.Manager.GetAllowedTiles(roomID, engine.UnitID(unitID), engine.TurnCmdType(turnCmdType))
	if err != nil {
		h.handleError(w, err, "get allowed tiles failed", "roomID", roomID)
		return
	}

	h.writeJSON(w, http.StatusOK, allowed, "encode allowedTiles failed")
}
