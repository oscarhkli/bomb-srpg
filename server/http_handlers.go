package server

import (
	"bomb-srpg/engine"
	"encoding/json"
	"net/http"
)

func (s *ServerStateManager) HandleGetAllArchetypes(w http.ResponseWriter, r *http.Request) {
	archetypes := engine.GetAllArchetypes()
	responsePayload := make([]ArchetypeResponse, len(archetypes))
	for i, arch := range archetypes {
		responsePayload[i] = MapToArchetypeResponse(arch)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(responsePayload); err != nil {
		http.Error(w, "Failed to encode archetype definitions", http.StatusInternalServerError)
		return
	}
}

func (s *ServerStateManager) HandleCreateNewMatchRoom(w http.ResponseWriter, r *http.Request) {
	id, err := s.CreateMatchRoom()

	if err != nil {
		http.Error(w, "Failed to create new MatchRoom", http.StatusInternalServerError)
	}

	responsePayload := CreateMatchRoomResponse{
		ID: id,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(responsePayload); err != nil {
		http.Error(w, "Failed to encode MatchRoom ID", http.StatusInternalServerError)
		return
	}
}
