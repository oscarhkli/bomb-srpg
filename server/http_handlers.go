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

// CreateMatchRoomResponse is returned when a new match room is created.
type CreateMatchRoomResponse struct {
	ID string `json:"id"`
}

// CreateMatchResponse is returned when a new match is created.
type CreateMatchResponse struct {
	Success      bool      `json:"success"`
	PlayerTokens [2]string `json:"playerTokens"`
}

// CreateMatchRequest wraps GameCfg for backward compatibility with existing clients.
type CreateMatchRequest struct {
	GameCfg engine.GameCfg `json:"gameCfg"`
}

// SurrenderRequest wraps TeamID for backward compatibility with existing clients.
type SurrenderRequest struct {
	TeamID int `json:"teamId"`
}

// HandleGetAllArchetypes returns all available unit archetypes for the client to display in the lobby.
// It encodes the archetype definitions as JSON and writes them to the response.
func (s *ServerStateManager) HandleGetAllArchetypes(w http.ResponseWriter, r *http.Request) {
	archetypes := engine.GetAllArchetypes()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(archetypes); err != nil {
		slog.Error("encode archetypes failed", "error", err)
		http.Error(w, "Failed to encode archetype definitions", http.StatusInternalServerError)
		return
	}
}

// HandleCreateMatchRoom creates a new match room and returns its unique ID.
// The room is initialized without a match instance; the match is created when players join.
func (s *ServerStateManager) HandleCreateMatchRoom(w http.ResponseWriter, r *http.Request) {
	id, err := s.CreateMatchRoom()

	if err != nil {
		slog.Error("create match room failed", "error", err)
		http.Error(w, "Failed to create new MatchRoom", http.StatusInternalServerError)
		return
	}

	location := fmt.Sprintf("/api/match-rooms/%s", id)
	w.Header().Set("Location", location)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	res := CreateMatchRoomResponse{ID: id}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		slog.Error("encode match room response failed", "roomID", id, "error", err)
		http.Error(w, "Failed to encode MatchRoom ID", http.StatusInternalServerError)
		return
	}

	slog.Info("match room created", "roomID", id)
}

// HandleCreateMatch creates a new match with given RoomID and GameCfg
func (s *ServerStateManager) HandleCreateMatch(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	var req CreateMatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("invalid config format", "error", err)
		http.Error(w, "Invalid configuration format", http.StatusBadRequest)
		return
	}

	tokens, err := s.CreateMatch(roomID, req.GameCfg)

	if err != nil {
		code, msg := mapError(err)
		slog.Warn("create match failed", "roomID", roomID, "error", err)
		http.Error(w, msg, code)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	res := CreateMatchResponse{Success: true, PlayerTokens: tokens}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		slog.Error("encode match response failed", "roomID", roomID, "error", err)
		http.Error(w, "Failed to encode success indicator", http.StatusInternalServerError)
		return
	}

	slog.Info("match created", "roomID", roomID)
}

// GetMatchState gets the WorkingState of the Match in a given MatchRoom.
// It encodes the gameState as JSON and writes them to the response.
func (s *ServerStateManager) HandleGetMatchState(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	gs, err := s.GetMatchState(roomID)

	if err != nil {
		code, msg := mapError(err)
		slog.Warn("get match state failed", "roomID", roomID, "error", err)
		http.Error(w, msg, code)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(gs); err != nil {
		slog.Error("encode gameState failed", "error", err)
		http.Error(w, "Failed to encode gameState", http.StatusInternalServerError)
		return
	}
}

// HandleSubmitTurnCommand delivers TurnCommand to engine to move a Unit or place a bomb in a given MatchRoom.
// It encodes the gameState as JSON and writes them to the response.
func (s *ServerStateManager) HandleSubmitTurnCommand(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	var req engine.TurnCommand
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("invalid turnCommand format", "error", err)
		http.Error(w, "Invalid turnCommand format", http.StatusBadRequest)
		return
	}

	token, err := s.extractBearerToken(r)
	if err != nil {
		code, msg := mapError(err)
		http.Error(w, msg, code)
		return
	}

	gs, err := s.SubmitTurnCommand(roomID, req, token)
	if err != nil {
		code, msg := mapError(err)
		slog.Warn("submit turn command failed", "roomID", roomID, "error", err)
		http.Error(w, msg, code)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(gs); err != nil {
		slog.Error("encode gameState failed", "error", err)
		http.Error(w, "Failed to encode gameState", http.StatusInternalServerError)
		return
	}
}

func (s *ServerStateManager) extractBearerToken(r *http.Request) (string, error) {
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

// HandleStartTurn sends StartTurn signal engine to start a new turn in a given MatchRoom.
// It encodes the gameState as JSON and writes them to the response.
func (s *ServerStateManager) HandleStartTurn(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	token, err := s.extractBearerToken(r)
	if err != nil {
		code, msg := mapError(err)
		http.Error(w, msg, code)
		return
	}

	gs, err := s.StartTurn(roomID, token)
	if err != nil {
		code, msg := mapError(err)
		slog.Warn("start turn failed", "roomID", roomID, "error", err)
		http.Error(w, msg, code)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(gs); err != nil {
		slog.Error("encode gameState failed", "error", err)
		http.Error(w, "Failed to encode gameState", http.StatusInternalServerError)
		return
	}
}

// HandleResetTurn sends ResetTurn signal to engine to drop the current WorkingState and reset to TrueState in a given MatchRoom.
// It encodes the gameState as JSON and writes them to the response.
func (s *ServerStateManager) HandleResetTurn(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	token, err := s.extractBearerToken(r)
	if err != nil {
		code, msg := mapError(err)
		http.Error(w, msg, code)
		return
	}

	gs, err := s.ResetTurn(roomID, token)
	if err != nil {
		code, msg := mapError(err)
		slog.Warn("reset turn failed", "roomID", roomID, "error", err)
		http.Error(w, msg, code)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(gs); err != nil {
		slog.Error("encode gameState failed", "error", err)
		http.Error(w, "Failed to encode gameState", http.StatusInternalServerError)
		return
	}
}

// HandleResolveTurn sends ResolveTurn signal to engine to calculate the impacts of the Player's action in a given MatchRoom.
// It encodes the gameEvents as JSON and writes them to the response.
func (s *ServerStateManager) HandleResolveTurn(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	token, err := s.extractBearerToken(r)
	if err != nil {
		code, msg := mapError(err)
		http.Error(w, msg, code)
		return
	}

	gameEvents, err := s.ResolveTurn(roomID, token)
	if err != nil {
		code, msg := mapError(err)
		slog.Warn("res turn failed", "roomID", roomID, "error", err)
		http.Error(w, msg, code)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(gameEvents); err != nil {
		slog.Error("encode gameState failed", "error", err)
		http.Error(w, "Failed to encode gameEvents", http.StatusInternalServerError)
		return
	}
}

// HandleSurrender sends Surrender signal to engine to egnd the current Match in a given MatchRoom.
// It encodes the gameEvents as JSON and writes them to the response.
func (s *ServerStateManager) HandleSurrender(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	var req SurrenderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("invalid surrender request format", "error", err)
		http.Error(w, "Invalid surrenderRequest format", http.StatusBadRequest)
		return
	}

	token, err := s.extractBearerToken(r)
	if err != nil {
		code, msg := mapError(err)
		http.Error(w, msg, code)
		return
	}

	gameEvents, err := s.Surrender(roomID, req.TeamID, token)
	if err != nil {
		code, msg := mapError(err)
		slog.Warn("res turn failed", "roomID", roomID, "error", err)
		http.Error(w, msg, code)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(gameEvents); err != nil {
		slog.Error("encode gameState failed", "error", err)
		http.Error(w, "Failed to encode gameEvents", http.StatusInternalServerError)
		return
	}
}

// HandleGetMatchConfig gets the GameCfg of the current Match in a given MatchRoom
func (s *ServerStateManager) HandleGetMatchConfig(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")
	gameCfg, err := s.GetMatchConfig(roomID)
	if err != nil {
		code, msg := mapError(err)
		slog.Warn("res turn failed", "roomID", roomID, "error", err)
		http.Error(w, msg, code)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(gameCfg); err != nil {
		slog.Error("encode gameConfig failed", "error", err)
		http.Error(w, "Failed to encode gameConfig", http.StatusInternalServerError)
		return
	}
}

// HandleGetMatchVictoryResult gets the VictoryResult of the current Match in a given MatchRoom
// TODO: Phase 3 frontend will determine what to provide
func (s *ServerStateManager) HandleGetMatchVictoryResult(w http.ResponseWriter, r *http.Request) {
	//roomID := r.PathValue("roomID")
	http.Error(w, "not yet implemented", http.StatusNotImplemented)
}

// HandlesGetAllowedTiles gets the hints for Player to identify which tiles are available according to the TurnCmdAction
func (s *ServerStateManager) HandleGetAllowedTiles(w http.ResponseWriter, r *http.Request) {
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

	allowed, err := s.GetAllowedTiles(roomID, engine.UnitID(unitID), engine.TurnCmdType(turnCmdType))
	if err != nil {
		code, msg := mapError(err)
		slog.Warn("res turn failed", "roomID", roomID, "error", err)
		http.Error(w, msg, code)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(allowed); err != nil {
		slog.Error("encode gameConfig failed", "error", err)
		http.Error(w, "Failed to encode gameConfig", http.StatusInternalServerError)
		return
	}
}
