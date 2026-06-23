package main

import (
	"bomb-srpg/server"
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Context cancelled on SIGINT (Ctrl+C) or SIGTERM
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	r := http.NewServeMux()
	serverState := server.NewServerStateManager()

	// Start background cleanup
	serverState.StartCleanupLoop(ctx, server.CleanupInterval)

	fs := http.FileServer(http.Dir("./web/public"))
	r.Handle("GET /", fs)

	// all other HTTP endpoints
	r.HandleFunc("GET /api/archetypes", serverState.HandleGetAllArchetypes)
	r.HandleFunc("POST /api/match-rooms", serverState.HandleCreateMatchRoom)
	r.HandleFunc("POST /api/match-rooms/{roomID}/match", serverState.HandleCreateMatch)
	r.HandleFunc("GET /api/match-rooms/{roomID}/match/state", serverState.HandleGetMatchState)
	r.HandleFunc("POST /api/match-rooms/{roomID}/match/turn-commands", serverState.HandleSubmitTurnCommand)
	r.HandleFunc("POST /api/match-rooms/{roomID}/match/start-turn", serverState.HandleStartTurn)
	r.HandleFunc("POST /api/match-rooms/{roomID}/match/reset", serverState.HandleResetTurn)
	r.HandleFunc("POST /api/match-rooms/{roomID}/match/resolve", serverState.HandleResolveTurn)
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

	// Run server in goroutine
	go func() {
		slog.Info("Bomb Tactics Server running on http://localhost:8080")
		slog.Info("Open http://localhost:8080 in your browser to view the Title Screen!")
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", "error", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	slog.Info("Shutdown signal received")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}
	slog.Info("Server stopped gracefully")
}
