package server

import (
	"bomb-srpg/engine"
	"bomb-srpg/server/api"
	"bytes"
	"encoding/json/v2"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPRouting(t *testing.T) {
	mux := http.NewServeMux()
	serverState := NewServerStateManager()
	handler := NewHandler(serverState)

	RegisterRoutes(mux, handler)

	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantAllow  string
	}{
		{
			name:       "GET /api/health",
			method:     "GET",
			path:       "/api/health",
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST /api/health (405)",
			method:     "POST",
			path:       "/api/health",
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  "GET, HEAD",
		},
		{
			name:       "GET /api/catalog",
			method:     "GET",
			path:       "/api/catalog",
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST /api/match-rooms",
			method:     "POST",
			path:       "/api/match-rooms",
			wantStatus: http.StatusCreated,
		},
		{
			name:       "POST /api/match-rooms/{roomID}/match",
			method:     "POST",
			path:       "/api/match-rooms/DUMMY/match",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "GET /api/match-rooms/{roomID}/match (405)",
			method:     "GET",
			path:       "/api/match-rooms/DUMMY/match",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "PUT /api/match-rooms/{roomID}/match (405)",
			method:     "PUT",
			path:       "/api/match-rooms/DUMMY/match",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "DELETE /api/match-rooms/{roomID}/match (404 - no room)",
			method:     "DELETE",
			path:       "/api/match-rooms/DUMMY/match",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "POST /api/match-rooms/{roomID}/rematch (404 - no room)",
			method:     "POST",
			path:       "/api/match-rooms/DUMMY/rematch",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "GET /api/match-rooms/{roomID}/rematch (405)",
			method:     "GET",
			path:       "/api/match-rooms/DUMMY/rematch",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "PUT /api/match-rooms/{roomID}/rematch (405)",
			method:     "PUT",
			path:       "/api/match-rooms/DUMMY/rematch",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "DELETE /api/match-rooms/{roomID}/rematch (405)",
			method:     "DELETE",
			path:       "/api/match-rooms/DUMMY/rematch",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "POST /api/catalog (405)",
			method:     "POST",
			path:       "/api/catalog",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "GET /api/match-rooms (405)",
			method:     "GET",
			path:       "/api/match-rooms",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "PUT /api/catalog (405)",
			method:     "PUT",
			path:       "/api/catalog",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "DELETE /api/match-rooms (405)",
			method:     "DELETE",
			path:       "/api/match-rooms",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "Unknown route",
			method:     "GET",
			path:       "/unknown",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "GET /api/match-rooms/{roomID}/match/state (404 - no room)",
			method:     "GET",
			path:       "/api/match-rooms/DUMMY/match/state",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "POST /api/match-rooms/{roomID}/match/state (405)",
			method:     "POST",
			path:       "/api/match-rooms/DUMMY/match/state",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "PUT /api/match-rooms/{roomID}/match/state (405)",
			method:     "PUT",
			path:       "/api/match-rooms/DUMMY/match/state",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "DELETE /api/match-rooms/{roomID}/match/state (405)",
			method:     "DELETE",
			path:       "/api/match-rooms/DUMMY/match/state",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "POST /api/match-rooms/{roomID}/match/turn-commands (404 - no room)",
			method:     "POST",
			path:       "/api/match-rooms/DUMMY/match/turn-commands",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "GET /api/match-rooms/{roomID}/match/turn-commands (405)",
			method:     "GET",
			path:       "/api/match-rooms/DUMMY/match/turn-commands",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "POST /api/match-rooms/{roomID}/match/reset (404 - no room)",
			method:     "POST",
			path:       "/api/match-rooms/DUMMY/match/reset",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "GET /api/match-rooms/{roomID}/match/reset (405)",
			method:     "GET",
			path:       "/api/match-rooms/DUMMY/match/reset",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "PUT /api/match-rooms/{roomID}/match/reset (405)",
			method:     "PUT",
			path:       "/api/match-rooms/DUMMY/match/reset",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "DELETE /api/match-rooms/{roomID}/match/reset (405)",
			method:     "DELETE",
			path:       "/api/match-rooms/DUMMY/match/reset",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "POST /api/match-rooms/{roomID}/match/resolve (404 - no room)",
			method:     "POST",
			path:       "/api/match-rooms/DUMMY/match/resolve",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "GET /api/match-rooms/{roomID}/match/resolve (405)",
			method:     "GET",
			path:       "/api/match-rooms/DUMMY/match/resolve",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "PUT /api/match-rooms/{roomID}/match/resolve (405)",
			method:     "PUT",
			path:       "/api/match-rooms/DUMMY/match/resolve",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "DELETE /api/match-rooms/{roomID}/match/resolve (405)",
			method:     "DELETE",
			path:       "/api/match-rooms/DUMMY/match/resolve",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "POST /api/match-rooms/{roomID}/match/cpu-status/consume (404 - no room)",
			method:     "POST",
			path:       "/api/match-rooms/DUMMY/match/cpu-status/consume",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "GET /api/match-rooms/{roomID}/match/cpu-status/consume (405)",
			method:     "GET",
			path:       "/api/match-rooms/DUMMY/match/cpu-status/consume",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "PUT /api/match-rooms/{roomID}/match/cpu-status/consume (405)",
			method:     "PUT",
			path:       "/api/match-rooms/DUMMY/match/cpu-status/consume",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "DELETE /api/match-rooms/{roomID}/match/cpu-status/consume (405)",
			method:     "DELETE",
			path:       "/api/match-rooms/DUMMY/match/cpu-status/consume",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "POST /api/match-rooms/{roomID}/match/surrender (404 - no room)",
			method:     "POST",
			path:       "/api/match-rooms/DUMMY/match/surrender",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "GET /api/match-rooms/{roomID}/match/surrender (405)",
			method:     "GET",
			path:       "/api/match-rooms/DUMMY/match/surrender",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "PUT /api/match-rooms/{roomID}/match/surrender (405)",
			method:     "PUT",
			path:       "/api/match-rooms/DUMMY/match/surrender",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "DELETE /api/match-rooms/{roomID}/match/surrender (405)",
			method:     "DELETE",
			path:       "/api/match-rooms/DUMMY/match/surrender",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "GET /api/match-rooms/{roomID}/match/config (404 - no room)",
			method:     "GET",
			path:       "/api/match-rooms/DUMMY/match/config",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "POST /api/match-rooms/{roomID}/match/config (405)",
			method:     "POST",
			path:       "/api/match-rooms/DUMMY/match/config",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "PUT /api/match-rooms/{roomID}/match/config (405)",
			method:     "PUT",
			path:       "/api/match-rooms/DUMMY/match/config",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "DELETE /api/match-rooms/{roomID}/match/config (405)",
			method:     "DELETE",
			path:       "/api/match-rooms/DUMMY/match/config",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "GET /api/match-rooms/{roomID}/match/allowed-tiles (404 - no room)",
			method:     "GET",
			path:       "/api/match-rooms/DUMMY/match/allowed-tiles?unitId=16&turnCmdType=placeBomb",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "POST /api/match-rooms/{roomID}/match/allowed-tiles (405)",
			method:     "POST",
			path:       "/api/match-rooms/DUMMY/match/allowed-tiles",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "PUT /api/match-rooms/{roomID}/match/allowed-tiles (405)",
			method:     "PUT",
			path:       "/api/match-rooms/DUMMY/match/allowed-tiles",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "DELETE /api/match-rooms/{roomID}/match/allowed-tiles (405)",
			method:     "DELETE",
			path:       "/api/match-rooms/DUMMY/match/allowed-tiles",
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	gameCfgBody, _ := json.Marshal(api.CreateMatchRequest{GameCfg: api.GameCfg{
		StagePreset: "Plain",
		P1Slots:     []api.TeamSlot{{Archetype: "King", Role: api.King}},
		P2Slots:     []api.TeamSlot{{Archetype: "King", Role: api.King}},
		MaxTurns:    10,
	}})

	turnCmdBody, _ := json.Marshal(wireMoveCmd(engine.NewUnitID(1, 0), 4, 7))

	surrenderReqBody, _ := json.Marshal(api.SurrenderRequest{TeamID: 1})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			switch {
			case tt.name == "POST /api/match-rooms/{roomID}/match":
				body = bytes.NewReader(gameCfgBody)
			case strings.HasPrefix(tt.name, "POST /api/match-rooms/{roomID}/match/turn-commands"):
				body = bytes.NewBuffer(turnCmdBody)
			case strings.HasPrefix(tt.name, "POST /api/match-rooms/{roomID}/match/surrender"):
				body = bytes.NewBuffer(surrenderReqBody)
			}

			req, err := http.NewRequest(tt.method, server.URL+tt.path, body)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			if body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			if strings.Contains(tt.name, "turn-commands") || strings.Contains(tt.name, "reset") || strings.Contains(tt.name, "resolve") ||
				strings.Contains(tt.name, "cpu-status") ||
				strings.Contains(tt.name, "surrender") || strings.Contains(tt.name, "rematch") ||
				(tt.method == http.MethodDelete && strings.HasSuffix(tt.path, "/match")) {
				req.Header.Set("Authorization", "Bearer dummy-token")
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("Status code: got %d want %d", resp.StatusCode, tt.wantStatus)
			}

			respBody, _ := io.ReadAll(resp.Body)
			if tt.wantAllow == "" {
				return
			}
			if got := resp.Header.Get("Allow"); got != tt.wantAllow {
				t.Errorf("Allow = %q, want %q", got, tt.wantAllow)
			}
			if got, want := string(respBody), "Method Not Allowed\n"; got != want {
				t.Errorf("body = %q, want %q", got, want)
			}
		})
	}
}

func TestSecurityHeadersOnEveryResponse(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, NewHandler(NewServerStateManager()))
	logger := slog.New(slog.DiscardHandler)
	handler := SecurityHeaders(RecoverPanic(logger, mux))

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{"success", "/api/health", http.StatusOK},
		{"error", "/api/match-rooms/NONEXISTENT/match/state", http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
			if got, want := rr.Header().Get("X-Content-Type-Options"), "nosniff"; got != want {
				t.Errorf("X-Content-Type-Options = %q, want %q", got, want)
			}
		})
	}
}

func TestStartTurnRouteRemoved(t *testing.T) {
	roomID, tokens, s := createTestRoom(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, NewHandler(s))

	req := httptest.NewRequest(http.MethodPost, "/api/match-rooms/"+roomID+"/match/start-turn", nil)
	req.Header.Set("Authorization", "Bearer "+tokens[0])
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusNotFound; got != want {
		t.Errorf("POST start-turn status = %d, want %d", got, want)
	}
}
