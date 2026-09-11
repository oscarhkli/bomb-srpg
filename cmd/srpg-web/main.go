package main

import (
	"bomb-srpg/server"
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func logLevelFromEnv(env string) slog.Level {
	switch strings.ToLower(env) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevelFromEnv(os.Getenv("LOG_LEVEL"))}))

	// Context cancelled on SIGINT (Ctrl+C) or SIGTERM
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	serverState := server.NewServerStateManager(server.WithLogger(logger))
	handler := server.NewHandler(serverState, server.WithHandlerLogger(logger))

	// Start background cleanup
	serverState.StartCleanupLoop(ctx, server.CleanupInterval)

	r := http.NewServeMux()
	fs := http.FileServer(http.Dir("./web/dist"))
	r.Handle("GET /", fs)

	server.RegisterRoutes(r, handler)

	s := &http.Server{
		Addr:         ":8080",
		Handler:      server.SecurityHeaders(server.RecoverPanic(logger, r)),
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
