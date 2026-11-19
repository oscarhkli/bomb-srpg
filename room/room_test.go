package room

import (
	"bomb-srpg/engine"
	"bytes"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"
)

func TestRoom_Rematch_KeepsMatchWhenBuildFails(t *testing.T) {
	buildErr := errors.New("build failed")
	builds := 0
	r := New("room", Deps{
		Logger: slog.New(slog.DiscardHandler),
		NewMatch: func(cfg engine.GameCfg) (*engine.Match, error) {
			builds++
			if builds > 1 {
				return nil, buildErr
			}
			return engine.InitGame(cfg)
		},
	})
	tokens, err := r.Start(validGameCfg())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	want, err := r.State()
	if err != nil {
		t.Fatalf("State() error = %v", err)
	}

	_, err = r.Rematch(tokens[0])
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("Rematch() error = %v, want %v", err, ErrInvalidConfig)
	}

	got, err := r.State()
	if err != nil {
		t.Fatalf("State() after failed Rematch error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("State() after failed Rematch = %+v, want the first match's %+v", got, want)
	}
}

func TestRoom_Rematch_ReplacesMatchAndKeepsTokens(t *testing.T) {
	r := New("room", Deps{Logger: slog.New(slog.DiscardHandler), NewMatch: engine.InitGame})
	tokens, err := r.Start(validGameCfg())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	first := r.match

	got, err := r.Rematch(tokens[0])
	if err != nil {
		t.Fatalf("Rematch() error = %v", err)
	}

	if got != tokens {
		t.Errorf("Rematch() tokens = %v, want %v", got, tokens)
	}
	if r.match == nil || r.match == first {
		t.Errorf("Match after Rematch = %p, want a new match distinct from %p", r.match, first)
	}
	if _, err := r.State(); err != nil {
		t.Errorf("State() after Rematch error = %v", err)
	}
}

func TestRoom_Rematch_WithoutConfig(t *testing.T) {
	r := New("room", Deps{Logger: slog.New(slog.DiscardHandler), NewMatch: engine.InitGame})
	tokens, err := r.Start(validGameCfg())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	r.mu.Lock()
	r.cfg = nil
	r.mu.Unlock()

	_, err = r.Rematch(tokens[0])
	if !errors.Is(err, ErrMatchNotFound) {
		t.Errorf("Rematch() error = %v, want %v", err, ErrMatchNotFound)
	}
}

func TestRoom_LogsCarryRoomID(t *testing.T) {
	var out bytes.Buffer
	r := New("room-42", Deps{Logger: slog.New(slog.NewTextHandler(&out, nil)), NewMatch: engine.InitGame})

	if _, err := r.State(); !errors.Is(err, ErrMatchNotFound) {
		t.Fatalf("State() error = %v, want %v", err, ErrMatchNotFound)
	}

	if got := out.String(); !strings.Contains(got, "roomID=room-42") {
		t.Errorf("log output = %q, want it to carry roomID=room-42", got)
	}
}
