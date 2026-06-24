package server

import "net/http"

// RegisterRoutes registers all HTTP routes for the Bomb Tactics REST API on the given ServeMux.
func RegisterRoutes(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("GET /api/archetypes", h.HandleGetAllArchetypes)
	mux.HandleFunc("POST /api/match-rooms", h.HandleCreateMatchRoom)
	mux.HandleFunc("POST /api/match-rooms/{roomID}/match", h.HandleCreateMatch)
	mux.HandleFunc("GET /api/match-rooms/{roomID}/match/state", h.HandleGetMatchState)
	mux.HandleFunc("POST /api/match-rooms/{roomID}/match/turn-commands", h.HandleSubmitTurnCommand)
	mux.HandleFunc("POST /api/match-rooms/{roomID}/match/start-turn", h.HandleStartTurn)
	mux.HandleFunc("POST /api/match-rooms/{roomID}/match/reset", h.HandleResetTurn)
	mux.HandleFunc("POST /api/match-rooms/{roomID}/match/resolve", h.HandleResolveTurn)
	mux.HandleFunc("POST /api/match-rooms/{roomID}/match/surrender", h.HandleSurrender)
	mux.HandleFunc("GET /api/match-rooms/{roomID}/match/config", h.HandleGetMatchConfig)
	mux.HandleFunc("GET /api/match-rooms/{roomID}/match/victory", h.HandleGetMatchVictoryResult)
	mux.HandleFunc("GET /api/match-rooms/{roomID}/match/allowed-tiles", h.HandleGetAllowedTiles)
}
