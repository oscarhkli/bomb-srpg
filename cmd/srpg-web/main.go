package main

import (
	"bomb-srpg/server"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	r := http.NewServeMux()
	serverState := server.NewServerStateManager()

	fs := http.FileServer(http.Dir("./web/public"))
	r.Handle("GET /", fs)

	// all other HTTP endpoints
	r.HandleFunc("GET /api/archetypes", serverState.HandleGetAllArchetypes)
	r.HandleFunc("POST /api/match-rooms", serverState.HandleCreateMatchRoom)
	r.HandleFunc("POST /api/match-rooms/{roomID}/match", serverState.HandleCreateMatch)
	r.HandleFunc("GET /api/match-rooms/{roomID}/match/state", serverState.HandleGetMatchState)
	r.HandleFunc("POST /api/match-rooms/{roomID}/match/turn-commands", serverState.HandleSubmitTurnCommand)
	r.HandleFunc("POST /api/match-rooms/{roomID}/match/start-turn", serverState.HandleStartTurn)
	r.HandleFunc("POST /api/match-rooms/{roomID}/match/commit", serverState.HandleCommitTurn)
	r.HandleFunc("POST /api/match-rooms/{roomID}/match/reset", serverState.HandleResetTurn)
	r.HandleFunc("POST /api/match-rooms/{roomID}/match/surrender", serverState.HandleSurrender)
	r.HandleFunc("GET /api/match-rooms/{roomID}/match/config", serverState.HandleGetMatchConfig)
	r.HandleFunc("GET /api/match-rooms/{roomID}/match/victory", serverState.HandleGetMatchVictoryResult)
	r.HandleFunc("GET /api/match-rooms/{roomID}/match/allowed-tiles", serverState.HandleGetAllowedTiles)

	s := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	slog.Info("Bomb Tactics Server running on http://localhost:8080")
	slog.Info("Open http://localhost:8080 in your browser to view the Title Screen!")

	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server crashed", "error", err)
	}
}
