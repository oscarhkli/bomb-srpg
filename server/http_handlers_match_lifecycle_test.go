package server

import (
	"bomb-srpg/engine"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleCreateMatchRoom(t *testing.T) {
	s := NewServerStateManager()
	h := NewHandler(s)

	t.Run("Success: called server.CreateMatchRoom", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/api/match-rooms", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()

		http.HandlerFunc(h.HandleCreateMatchRoom).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusCreated)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response CreateMatchRoomResponse
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		if got, want := len(response.ID), 10; got != want {
			t.Errorf("Handler returned unexpected Match Room ID: got %v want length of %v", response.ID, want)
		}
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		testEncodeFailure(t, http.HandlerFunc(h.HandleCreateMatchRoom),
			func() *http.Request {
				req, _ := http.NewRequest("POST", "/api/match-rooms", nil)
				return req
			}, http.StatusCreated)
	})

	t.Run("Test Contract", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/match-rooms", nil)
		rr := httptest.NewRecorder()

		http.HandlerFunc(h.HandleCreateMatchRoom).ServeHTTP(rr, req)

		assertObjectContract(t, rr.Body.Bytes(), []string{"id"}, nil)
	})

	t.Run("Failure: CreateMatchRoom exhausted retries", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)

		roomIDs := []string{"ID001", "ID002", "ID003", "ID004", "ID005"}
		for _, id := range roomIDs {
			s.Rooms.Store(id, &MatchRoom{ID: id})
		}
		callCount := 0
		s.generateRoomID = func(int) (string, error) {
			if callCount < len(roomIDs) {
				id := roomIDs[callCount]
				callCount++
				return id, nil
			}
			return "SHOULD_NOT_REACH", nil
		}

		req, _ := http.NewRequest("POST", "/api/match-rooms", nil)
		rr := httptest.NewRecorder()

		http.HandlerFunc(h.HandleCreateMatchRoom).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", status)
		}

		body := rr.Body.String()
		if !strings.Contains(body, "Failed to create new MatchRoom") {
			t.Errorf("Expected error message in body, got: %s", body)
		}
	})
}

func TestHandleCreateNewMatch(t *testing.T) {
	t.Run("Success: creates a new match in an existing room", func(t *testing.T) {
		roomID, s, h := newTestRoom(t)

		gameCfg := validGameCfg()
		jsonBody, _ := json.Marshal(CreateMatchRequest{GameCfg: gameCfg})
		rr := doRequest(t, h.HandleCreateMatch, "POST /api/match-rooms/{roomID}/match", "POST", "/api/match-rooms/"+roomID+"/match", string(jsonBody), "")

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusCreated)
		}
		if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, "application/json")
		}

		var response CreateMatchResponse
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}
		if !response.Success {
			t.Error("Expected success=true in response, got false")
		}

		room := mustRoom(t, s, roomID)
		if room.Match == nil {
			t.Error("Match was not created in the server state manager")
		}

		if len(response.PlayerTokens) != 2 || response.PlayerTokens[0] == "" || response.PlayerTokens[1] == "" || response.PlayerTokens[0] == response.PlayerTokens[1] {
			t.Errorf("Expected 2 unique non-empty PlayerToken, got %v", response.PlayerTokens)
		}
		if response.PlayerTokens != room.PlayerTokens {
			t.Errorf("Expected response and MatchRoom shareTokens, response %v vs MatchRoom %v", response.PlayerTokens, room.PlayerTokens)
		}
	})

	testRouteFailures(t, failureRoute{
		handler: (*Handler).HandleCreateMatch,
		pattern: "POST /api/match-rooms/{roomID}/match",
		body:    CreateMatchRequest{GameCfg: validGameCfg()},
		cases:   failRoom,
	})

	t.Run("Failure: match already exists", func(t *testing.T) {
		roomID, s, h := newTestRoom(t)
		room := mustRoom(t, s, roomID)
		room.Match = &engine.Match{}

		gameCfg := validGameCfg()
		jsonBody, _ := json.Marshal(CreateMatchRequest{GameCfg: gameCfg})
		rr := doRequest(t, h.HandleCreateMatch, "POST /api/match-rooms/{roomID}/match", "POST", "/api/match-rooms/"+roomID+"/match", string(jsonBody), "")

		if status := rr.Code; status != http.StatusConflict {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusConflict)
		}
		if !strings.Contains(rr.Body.String(), "match already exists") {
			t.Errorf("Expected error message 'match already exists', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid game config", func(t *testing.T) {
		roomID, _, h := newTestRoom(t)

		gameCfg := engine.GameCfg{
			StagePreset: "INVALID_STAGE",
			MaxTurns:    10,
		}
		jsonBody, _ := json.Marshal(CreateMatchRequest{GameCfg: gameCfg})
		rr := doRequest(t, h.HandleCreateMatch, "POST /api/match-rooms/{roomID}/match", "POST", "/api/match-rooms/"+roomID+"/match", string(jsonBody), "")

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "invalid game config") {
			t.Errorf("Expected error message 'invalid game config', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid JSON format", func(t *testing.T) {
		roomID, _, h := newTestRoom(t)

		rr := doRequest(t, h.HandleCreateMatch, "POST /api/match-rooms/{roomID}/match", "POST", "/api/match-rooms/"+roomID+"/match", "{invalid json", "")

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "Invalid configuration format") {
			t.Errorf("Expected error message 'Invalid configuration format', got: %s", rr.Body.String())
		}
	})

	roleTests := []struct {
		name       string
		role       string
		wantStatus int
	}{
		{"valid role", `"Normal"`, http.StatusCreated},
		{"wrong case", `"normal"`, http.StatusBadRequest},
		{"unknown name", `"Wizard"`, http.StatusBadRequest},
		{"number", `1`, http.StatusBadRequest},
	}
	for _, tt := range roleTests {
		t.Run("role: "+tt.name, func(t *testing.T) {
			roomID, _, h := newTestRoom(t)

			body := `{"gameCfg":{"stagePreset":"Plain","maxTurns":10,` +
				`"p1Slots":[{"archetype":"King","role":"King"},{"archetype":"Fighter","role":` + tt.role + `}],` +
				`"p2Slots":[{"archetype":"King","role":"King"},{"archetype":"Witch","role":"Normal"}]}}`
			req := httptest.NewRequest("POST", "/api/match-rooms/"+roomID+"/match", strings.NewReader(body))

			rr := httptest.NewRecorder()
			testMux("POST /api/match-rooms/{roomID}/match", h.HandleCreateMatch).ServeHTTP(rr, req)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.wantStatus)
			}
		})
	}

	t.Run("Failure: failed to Encode response", func(t *testing.T) {
		roomID, _, h := newTestRoom(t)

		gameCfg := validGameCfg()
		jsonBody, _ := json.Marshal(CreateMatchRequest{GameCfg: gameCfg})
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		testEncodeFailure(t, testMux("POST /api/match-rooms/{roomID}/match", h.HandleCreateMatch),
			func() *http.Request {
				req, _ := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match", strings.NewReader(string(jsonBody)))
				req.Header.Set("Content-Type", "application/json")
				return req
			}, http.StatusCreated)
	})

	t.Run("Test Contract", func(t *testing.T) {
		roomID, _, h := newTestRoom(t)
		gameCfg := validGameCfg()
		jsonBody, _ := json.Marshal(CreateMatchRequest{GameCfg: gameCfg})
		rr := doRequest(t, h.HandleCreateMatch, "POST /api/match-rooms/{roomID}/match", "POST", "/api/match-rooms/"+roomID+"/match", string(jsonBody), "")

		assertObjectContract(t, rr.Body.Bytes(), []string{"success", "playerTokens"}, nil)
	})
}

func TestHandleRematch(t *testing.T) {
	t.Run("Success: wipes existing Match and creates a new one", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)

		rr := doRequest(t, h.HandleRematch, "POST /api/match-rooms/{roomID}/rematch", "POST", "/api/match-rooms/"+roomID+"/rematch", "", playerTokens[0])

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusCreated)
		}
		if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, "application/json")
		}

		var response CreateMatchResponse
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}
		if !response.Success {
			t.Error("Expected success=true in response, got false")
		}

		room := mustRoom(t, s, roomID)
		if room.Match == nil {
			t.Error("Match was not created in the server state manager")
		}

		if len(response.PlayerTokens) != 2 || response.PlayerTokens[0] == "" || response.PlayerTokens[1] == "" || response.PlayerTokens[0] == response.PlayerTokens[1] {
			t.Errorf("Expected 2 unique non-empty PlayerToken, got %v", response.PlayerTokens)
		}
		if response.PlayerTokens != room.PlayerTokens {
			t.Errorf("Expected response and MatchRoom shareTokens, response %v vs MatchRoom %v", response.PlayerTokens, room.PlayerTokens)
		}
		if response.PlayerTokens != playerTokens {
			t.Errorf("Expected Rematch to reuse existing PlayerTokens, got %v want %v", response.PlayerTokens, playerTokens)
		}
	})

	testRouteFailures(t, failureRoute{
		handler:   (*Handler).HandleRematch,
		pattern:   "POST /api/match-rooms/{roomID}/rematch",
		cases:     failRoom | failMatch | failMissingAuth | failInvalidToken,
		matchGone: func(room *MatchRoom) { room.GameCfg = nil },
	})

	t.Run("Failure: failed to Encode response", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		testEncodeFailure(t, testMux("POST /api/match-rooms/{roomID}/rematch", h.HandleRematch),
			func() *http.Request {
				req, _ := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/rematch", nil)
				req.Header.Set("Authorization", "Bearer "+playerTokens[0])
				return req
			}, http.StatusCreated)
	})

	t.Run("Test Contract", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		rr := doRequest(t, h.HandleRematch, "POST /api/match-rooms/{roomID}/rematch", "POST", "/api/match-rooms/"+roomID+"/rematch", "", playerTokens[0])

		assertObjectContract(t, rr.Body.Bytes(), []string{"success", "playerTokens"}, nil)
	})
}

func TestHandleDeleteMatch(t *testing.T) {
	t.Run("Success: deletes an existing concluded Match", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		room := mustRoom(t, s, roomID)
		room.Match.WinnerTeamID = 1

		rr := doRequest(t, h.HandleDeleteMatch, "DELETE /api/match-rooms/{roomID}/match", "DELETE", "/api/match-rooms/"+roomID+"/match", "", playerTokens[0])

		if status := rr.Code; status != http.StatusNoContent {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
		}
		if room.Match != nil {
			t.Errorf("Expected Match to be deleted, got %p", room.Match)
		}
	})

	t.Run("Success: no-op when Match already cleared", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		room := mustRoom(t, s, roomID)
		room.Match = nil

		rr := doRequest(t, h.HandleDeleteMatch, "DELETE /api/match-rooms/{roomID}/match", "DELETE", "/api/match-rooms/"+roomID+"/match", "", playerTokens[0])

		if status := rr.Code; status != http.StatusNoContent {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
		}
	})

	t.Run("Failure: match still in progress", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		rr := doRequest(t, h.HandleDeleteMatch, "DELETE /api/match-rooms/{roomID}/match", "DELETE", "/api/match-rooms/"+roomID+"/match", "", playerTokens[0])

		if status := rr.Code; status != http.StatusConflict {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusConflict)
		}
		if !strings.Contains(rr.Body.String(), "match still in progress") {
			t.Errorf("Expected error message 'match still in progress', got: %s", rr.Body.String())
		}
	})

	testRouteFailures(t, failureRoute{
		handler: (*Handler).HandleDeleteMatch,
		pattern: "DELETE /api/match-rooms/{roomID}/match",
		cases:   failRoom | failMissingAuth | failInvalidToken,
	})
}
