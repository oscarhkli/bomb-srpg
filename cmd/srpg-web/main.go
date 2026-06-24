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

	// Context cancelled on SIGINT (Ctrl+C) or SIGTERM
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	serverState := server.NewServerStateManager(server.WithLogger(logger))
	handler := server.NewHandler(serverState, server.WithHandlerLogger(logger))

	// Start background cleanup
	serverState.StartCleanupLoop(ctx, server.CleanupInterval)

	r := http.NewServeMux()
	fs := http.FileServer(http.Dir("./web/public"))
	r.Handle("GET /", fs)

	// all other HTTP endpoints
	r.HandleFunc("GET /api/archetypes", handler.HandleGetAllArchetypes)
	r.HandleFunc("POST /api/match-rooms", handler.HandleCreateMatchRoom)
	r.HandleFunc("POST /api/match-rooms/{roomID}/match", handler.HandleCreateMatch)
	r.HandleFunc("GET /api/match-rooms/{roomID}/match/state", handler.HandleGetMatchState)
	r.HandleFunc("POST /api/match-rooms/{roomID}/match/turn-commands", handler.HandleSubmitTurnCommand)
	r.HandleFunc("POST /api/match-rooms/{roomID}/match/start-turn", handler.HandleStartTurn)
	r.HandleFunc("POST /api/match-rooms/{roomID}/match/reset", handler.HandleResetTurn)
	r.HandleFunc("POST /api/match-rooms/{roomID}/match/resolve", handler.HandleResolveTurn)
	r.HandleFunc("POST /api/match-rooms/{roomID}/match/surrender", handler.HandleSurrender)
	r.HandleFunc("GET /api/match-rooms/{roomID}/match/config", handler.HandleGetMatchConfig)
	r.HandleFunc("GET /api/match-rooms/{roomID}/match/victory", handler.HandleGetMatchVictoryResult)
	r.HandleFunc("GET /api/match-rooms/{roomID}/match/allowed-tiles", handler.HandleGetAllowedTiles)

	s := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	// Run server in goroutine
	go func() {
		logger.Info("Bomb Tactics Server running on http://localhost:8080")
		logger.Info("Open http://localhost:8080 in your browser to view the Title Screen!")
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error", "error", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info("Shutdown signal received")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}
	logger.Info("Server stopped gracefully")
}
