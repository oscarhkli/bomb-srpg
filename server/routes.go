package server

import "net/http"

// RegisterRoutes registers all HTTP routes for the Bomb Tactics REST API on the given ServeMux.
func RegisterRoutes(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("GET /api/health", h.handleHealthCheck)
	mux.HandleFunc("GET /api/catalog", h.handleGetCatalog)
	mux.HandleFunc("POST /api/match-rooms", h.handleCreateMatchRoom)
	mux.HandleFunc("POST /api/match-rooms/{roomID}/match", h.handleCreateMatch)
	mux.HandleFunc("POST /api/match-rooms/{roomID}/rematch", h.handleRematch)
	mux.HandleFunc("DELETE /api/match-rooms/{roomID}/match", h.handleDeleteMatch)
	mux.HandleFunc("GET /api/match-rooms/{roomID}/match/state", h.handleGetMatchState)
	mux.HandleFunc("POST /api/match-rooms/{roomID}/match/turn-commands", h.handleSubmitTurnCommand)
	mux.HandleFunc("POST /api/match-rooms/{roomID}/match/cpu-status/consume", h.handleConsumeCPUStatus)
	mux.HandleFunc("POST /api/match-rooms/{roomID}/match/reset", h.handleResetTurn)
	mux.HandleFunc("POST /api/match-rooms/{roomID}/match/resolve", h.handleResolveTurn)
	mux.HandleFunc("POST /api/match-rooms/{roomID}/match/surrender", h.handleSurrender)
	mux.HandleFunc("GET /api/match-rooms/{roomID}/match/config", h.handleGetMatchConfig)
	mux.HandleFunc("GET /api/match-rooms/{roomID}/match/allowed-tiles", h.handleGetAllowedTiles)
}
