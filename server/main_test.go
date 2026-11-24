package server

import (
	"log/slog"
	"testing"
)

func TestMain(m *testing.M) {
	InstallJSONCodec()
	slog.SetDefault(slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{
		Level: slog.LevelError + 1,
	})))
	m.Run()
}
