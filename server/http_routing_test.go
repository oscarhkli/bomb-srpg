package server

import (
	"bomb-srpg/engine"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPRouting(t *testing.T) {
	mux := http.NewServeMux()
	serverState := NewServerStateManager()

	mux.HandleFunc("GET /api/archetypes", serverState.HandleGetAllArchetypes)
	mux.HandleFunc("POST /api/match-rooms", serverState.HandleCreateMatchRoom)
	mux.HandleFunc("POST /api/match-rooms/{roomID}/match", serverState.HandleCreateMatch)
	mux.HandleFunc("GET /api/match-rooms/{roomID}/match/state", serverState.HandleGetMatchState)
	mux.HandleFunc("POST /api/match-rooms/{roomID}/match/turn-commands", serverState.HandleSubmitTurnCommand)
	mux.HandleFunc("POST /api/match-rooms/{roomID}/match/start-turn", serverState.HandleStartTurn)
	mux.HandleFunc("POST /api/match-rooms/{roomID}/match/reset-turn", serverState.HandleResetTurn)
	mux.HandleFunc("POST /api/match-rooms/{roomID}/match/resolve-turn", serverState.HandleResolveTurn)
	mux.HandleFunc("POST /api/match-rooms/{roomID}/match/surrender", serverState.HandleSurrender)
	mux.HandleFunc("GET /api/match-rooms/{roomID}/match/config", serverState.HandleGetMatchConfig)
	mux.HandleFunc("GET /api/match-rooms/{roomID}/match/allowed-tiles", serverState.HandleGetAllowedTiles)

	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{
			name:       "GET /api/archetypes",
			method:     "GET",
			path:       "/api/archetypes",
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
			path:       "/api/match-rooms/DUMMY/match", // Use a dummy roomID for routing test
			wantStatus: http.StatusNotFound,            // Room doesn't exist yet
		},
		{
			name:       "GET /api/match-rooms/{roomID}/match (405)",
			method:     "GET",
			path:       "/api/match-rooms/DUMMY/match",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "POST /api/archetypes (405)",
			method:     "POST",
			path:       "/api/archetypes",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "GET /api/match-rooms (405)",
			method:     "GET",
			path:       "/api/match-rooms",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "PUT /api/archetypes (405)",
			method:     "PUT",
			path:       "/api/archetypes",
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
			name:       "POST /api/match-rooms/{roomID}/match/start-turn (404 - no room)",
			method:     "POST",
			path:       "/api/match-rooms/DUMMY/match/start-turn",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "GET /api/match-rooms/{roomID}/match/start-turn (405)",
			method:     "GET",
			path:       "/api/match-rooms/DUMMY/match/start-turn",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "PUT /api/match-rooms/{roomID}/match/start-turn (405)",
			method:     "PUT",
			path:       "/api/match-rooms/DUMMY/match/start-turn",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "DELETE /api/match-rooms/{roomID}/match/start-turn (405)",
			method:     "DELETE",
			path:       "/api/match-rooms/DUMMY/match/start-turn",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "POST /api/match-rooms/{roomID}/match/reset-turn (404 - no room)",
			method:     "POST",
			path:       "/api/match-rooms/DUMMY/match/reset-turn",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "GET /api/match-rooms/{roomID}/match/reset-turn (405)",
			method:     "GET",
			path:       "/api/match-rooms/DUMMY/match/reset-turn",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "PUT /api/match-rooms/{roomID}/match/reset-turn (405)",
			method:     "PUT",
			path:       "/api/match-rooms/DUMMY/match/reset-turn",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "DELETE /api/match-rooms/{roomID}/match/reset-turn (405)",
			method:     "DELETE",
			path:       "/api/match-rooms/DUMMY/match/reset-turn",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "POST /api/match-rooms/{roomID}/match/resolve-turn (404 - no room)",
			method:     "POST",
			path:       "/api/match-rooms/DUMMY/match/resolve-turn",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "GET /api/match-rooms/{roomID}/match/resolve-turn (405)",
			method:     "GET",
			path:       "/api/match-rooms/DUMMY/match/resolve-turn",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "PUT /api/match-rooms/{roomID}/match/resolve-turn (405)",
			method:     "PUT",
			path:       "/api/match-rooms/DUMMY/match/resolve-turn",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "DELETE /api/match-rooms/{roomID}/match/resolve-turn (405)",
			method:     "DELETE",
			path:       "/api/match-rooms/DUMMY/match/resolve-turn",
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

	gameCfgBody, _ := json.Marshal(engine.GameCfg{
		StagePreset: "MAP01",
		P1Teams:     []string{"King"},
		P2Teams:     []string{"King"},
		MaxTurns:    10,
	})

	turnCmdBody, _ := json.Marshal(engine.NewMoveCommand(engine.NewUnitID(1, 0), engine.Coordinate{X: 4, Y: 7}))

	surrenderReqBody, _ := json.Marshal(SurrenderRequest{TeamID: 1})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			if tt.name == "POST /api/match-rooms/{roomID}/match" {
				body = bytes.NewReader(gameCfgBody)
			} else if strings.HasPrefix(tt.name, "POST /api/match-rooms/{roomID}/match/turn-commands") {
				body = bytes.NewBuffer(turnCmdBody)
			} else if strings.HasPrefix(tt.name, "POST /api/match-rooms/{roomID}/match/surrender") {
				body = bytes.NewBuffer(surrenderReqBody)
			}

			req, err := http.NewRequest(tt.method, server.URL+tt.path, body)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			if body != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("Status code: got %d want %d", resp.StatusCode, tt.wantStatus)
			}

			// Consume body to allow connection reuse
			_, _ = io.Copy(io.Discard, resp.Body)
		})
	}
}
