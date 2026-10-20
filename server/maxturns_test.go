package server

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleCreateMatch_MaxTurnsBound(t *testing.T) {
	tests := []struct {
		name     string
		maxTurns int
		want     int
	}{
		{name: "negative", maxTurns: -1, want: http.StatusBadRequest},
		{name: "zero", maxTurns: 0, want: http.StatusCreated},
		{name: "upper bound", maxTurns: 100, want: http.StatusCreated},
		{name: "above upper bound", maxTurns: 101, want: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServerStateManager()
			mux := http.NewServeMux()
			RegisterRoutes(mux, NewHandler(s))
			roomID, err := s.CreateMatchRoom()
			if err != nil {
				t.Fatalf("Failed to create room: %v", err)
			}

			cfg := validGameCfg()
			cfg.MaxTurns = tt.maxTurns
			body, err := json.Marshal(CreateMatchRequest{GameCfg: cfg})
			if err != nil {
				t.Fatalf("Failed to encode request: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/match-rooms/%s/match", roomID), bytes.NewReader(body))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Errorf("got status %d, want %d (body %q)", rec.Code, tt.want, rec.Body.String())
			}
		})
	}
}
