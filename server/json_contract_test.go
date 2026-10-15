package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bomb-srpg/engine"
)

func TestSurrenderRequestDecoding(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"exact keys", `{"teamId":1}`, http.StatusOK},
		{"keys match case-sensitively", `{"TeamID":1}`, http.StatusBadRequest},
		{"unknown member ignored", `{"teamId":1,"extra":[true]}`, http.StatusOK},
		{"duplicate key", `{"teamId":3,"teamId":1}`, http.StatusBadRequest},
		{"invalid UTF-8 in unknown member", "{\"teamId\":1,\"x\":\"\xff\"}", http.StatusBadRequest},
		{"trailing data after value", `{"teamId":1} garbage`, http.StatusBadRequest},
		{"second value after value", `{"teamId":1}{"teamId":3}`, http.StatusBadRequest},
		{"truncated", `{"teamId":`, http.StatusBadRequest},
		{"wrong member type", `{"teamId":"1"}`, http.StatusBadRequest},
		{"fractional number", `{"teamId":1.5}`, http.StatusBadRequest},
		{"array body", `[1]`, http.StatusBadRequest},
		{"empty body", ``, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, playerTokens, _, h := createTestRoomWithMatch(t)
			req := httptest.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/surrender", strings.NewReader(tt.body))
			req.Header.Set("Authorization", "Bearer "+playerTokens[0])
			rr := httptest.NewRecorder()
			testMux("POST /api/match-rooms/{roomID}/match/surrender", h.HandleSurrender).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
			if tt.wantStatus != http.StatusOK {
				return
			}
			var gameEvents []engine.GameEvent
			if err := json.NewDecoder(rr.Body).Decode(&gameEvents); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if len(gameEvents) != 1 || gameEvents[0].WinnerTeamID != 2 {
				t.Errorf("gameEvents = %+v, want one event with WinnerTeamID 2", gameEvents)
			}
		})
	}
}
