package server

import (
	"bomb-srpg/engine"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

// CreateMatchRoomResponse is returned when a new match room is created.
type CreateMatchRoomResponse struct {
	ID string `json:"id"`
}

// CreateMatchResponse is returned when a new match is created.
type CreateMatchResponse struct {
	Success bool `json:"success"`
}

// CreateMatchRequest wraps GameCfg for backward compatibility with existing clients.
type CreateMatchRequest struct {
	GameCfg engine.GameCfg `json:"gameCfg"`
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

	err := s.CreateMatch(roomID, req.GameCfg)

	if err != nil {
		switch {
		case errors.Is(err, ErrRoomNotFound):
			slog.Warn("create match room not found", "roomID", roomID)
			http.Error(w, "room not found", http.StatusNotFound)
		case errors.Is(err, ErrMatchExists):
			slog.Warn("create match exists", "roomID", roomID)
			http.Error(w, "match already exists", http.StatusConflict)
		case errors.Is(err, ErrInvalidConfig):
			slog.Warn("create match invalid config", "roomID", roomID, "error", err)
			http.Error(w, "invalid config", http.StatusBadRequest)
		default:
			slog.Error("create match internal error", "roomID", roomID, "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	res := CreateMatchResponse{Success: true}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		slog.Error("encode match response failed", "roomID", roomID, "error", err)
		http.Error(w, "Failed to encode success indicator", http.StatusInternalServerError)
		return
	}

	slog.Info("match created", "roomID", roomID)
}

// GetMatchState get the WorkingState of the Match in a given MatchRoom.
// It encodes the gameState definitions as JSON and writes them to the response.
func (s *ServerStateManager) HandleGetMatchState(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	gameState, err := s.GetMatchState(roomID)

	if err != nil {
		switch {
		case errors.Is(err, ErrRoomNotFound):
			slog.Warn("get match state match room not found", "roomID", roomID)
			http.Error(w, "room not found", http.StatusNotFound)
		case errors.Is(err, ErrMatchNotFound):
			slog.Warn("get match state match not found", "roomID", roomID)
			http.Error(w, "match not found", http.StatusNotFound)
		default:
			slog.Error("get match state internal error", "roomID", roomID, "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(gameState); err != nil {
		slog.Error("encode gameState failed", "error", err)
		http.Error(w, "Failed to encode gameState definitions", http.StatusInternalServerError)
		return
	}
}
