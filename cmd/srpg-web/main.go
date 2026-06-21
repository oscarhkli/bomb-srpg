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
