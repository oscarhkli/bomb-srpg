package main

import (
	"bomb-srpg/server"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
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

const defaultPort = "8080"

func listenAddr(getenv func(string) string) (string, error) {
	port := getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("invalid PORT %q: want a number from 1 to 65535", port)
	}
	return ":" + strconv.Itoa(n), nil
}

const (
	readTimeout     = 5 * time.Second
	writeTimeout    = 5 * time.Second
	shutdownTimeout = 5 * time.Second
	healthTimeout   = 2 * time.Second
)

func main() {
	addr, err := listenAddr(os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if checkHealth(os.Args[1:]) {
		ctx, cancel := context.WithTimeout(context.Background(), healthTimeout)
		code := runHealthCheck(ctx, "http://localhost"+addr+"/api/health")
		cancel()
		os.Exit(code)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevelFromEnv(os.Getenv("LOG_LEVEL"))}))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	serverState := server.NewServerStateManager(server.WithLogger(logger))
	handler := server.NewHandler(serverState, server.WithHandlerLogger(logger))

	go serverState.RunCleanupLoop(ctx, server.CleanupInterval)

	r := http.NewServeMux()
	fs := http.FileServer(http.Dir("./web/dist"))
	r.Handle("GET /", fs)

	server.RegisterRoutes(r, handler)

	s := &http.Server{
		Addr:         addr,
		Handler:      server.SecurityHeaders(server.RecoverPanic(logger, r)),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	go func() {
		logger.Info("Bomb Tactics Server running on http://localhost" + addr)
		logger.Info("Open http://localhost" + addr + " in your browser to view the Title Screen!")
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server error", "error", err)
		}
	}()

	<-ctx.Done()
	logger.Info("Shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := s.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}
	logger.Info("Server stopped gracefully")
}
