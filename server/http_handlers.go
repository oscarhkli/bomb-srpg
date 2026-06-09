package server

import (
	"bomb-srpg/engine"
	"encoding/json"
	"net/http"
)

func (s *ServerStateManager) HandleGetAllArchetypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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
