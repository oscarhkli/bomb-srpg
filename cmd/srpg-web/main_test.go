package main

import (
	"log/slog"
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
