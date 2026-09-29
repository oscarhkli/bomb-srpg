package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogLevelFromEnv(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want slog.Level
	}{
		{name: "empty defaults to info", env: "", want: slog.LevelInfo},
		{name: "debug", env: "debug", want: slog.LevelDebug},
		{name: "case-insensitive", env: "DEBUG", want: slog.LevelDebug},
		{name: "warn", env: "warn", want: slog.LevelWarn},
		{name: "error", env: "error", want: slog.LevelError},
		{name: "unrecognized defaults to info", env: "bogus", want: slog.LevelInfo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := logLevelFromEnv(tt.env); got != tt.want {
				t.Errorf("logLevelFromEnv(%q) = %v, want %v", tt.env, got, tt.want)
			}
		})
	}
}

func TestCheckHealth(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "no args", args: nil, want: false},
		{name: "healthcheck arg", args: []string{"healthcheck"}, want: true},
		{name: "other arg", args: []string{"serve"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkHealth(tt.args); got != tt.want {
				t.Errorf("checkHealth(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestRunHealthCheck(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		unreachable bool
		want        int
	}{
		{name: "200 response", statusCode: http.StatusOK, want: 0},
		{name: "non-200 response", statusCode: http.StatusInternalServerError, want: 1},
		{name: "unreachable", unreachable: true, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			url := srv.URL
			if tt.unreachable {
				srv.Close()
			} else {
				defer srv.Close()
			}

			if got := runHealthCheck(url); got != tt.want {
				t.Errorf("runHealthCheck() = %d, want %d", got, tt.want)
			}
		})
	}
}
