package server

import (
	"encoding/json/v2"
	"net/http"
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
			roomID, _, h := newTestRoom(t)

			cfg := validGameCfg()
			cfg.MaxTurns = tt.maxTurns
			body, err := json.Marshal(CreateMatchRequest{GameCfg: cfg})
			if err != nil {
				t.Fatalf("Failed to encode request: %v", err)
			}

			rr := doRequest(t, h.HandleCreateMatch, "POST /api/match-rooms/{roomID}/match", http.MethodPost, "/api/match-rooms/"+roomID+"/match", string(body), "")

			if rr.Code != tt.want {
				t.Errorf("got status %d, want %d (body %q)", rr.Code, tt.want, rr.Body.String())
			}
		})
	}
}
