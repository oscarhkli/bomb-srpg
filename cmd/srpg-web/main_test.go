package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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

			if got := runHealthCheck(context.Background(), url); got != tt.want {
				t.Errorf("runHealthCheck() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRunHealthCheck_DeadlineExceeded(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer srv.Close()
	defer close(release)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if got := runHealthCheck(ctx, srv.URL); got != 1 {
		t.Errorf("runHealthCheck() = %d, want 1", got)
	}
}

func TestListenAddr(t *testing.T) {
	tests := []struct {
		name    string
		port    string
		want    string
		wantErr bool
	}{
		{name: "unset defaults to 8080", port: "", want: ":8080"},
		{name: "valid port", port: "54321", want: ":54321"},
		{name: "lowest port", port: "1", want: ":1"},
		{name: "highest port", port: "65535", want: ":65535"},
		{name: "zero", port: "0", wantErr: true},
		{name: "above range", port: "65536", wantErr: true},
		{name: "negative", port: "-1", wantErr: true},
		{name: "service name", port: "http", wantErr: true},
		{name: "host and port", port: "127.0.0.1:80", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := listenAddr(func(string) string { return tt.port })
			if (err != nil) != tt.wantErr {
				t.Fatalf("listenAddr(%q) error = %v, wantErr %v", tt.port, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("listenAddr(%q) = %q, want %q", tt.port, got, tt.want)
			}
		})
	}
}
