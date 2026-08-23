package server

import (
	"bomb-srpg/engine"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func testMux(pattern string, h http.HandlerFunc) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc(pattern, h)
	return mux
}

func assertObjectContract(t *testing.T, body []byte, expectedFields []string, nestedChecks func(t *testing.T, raw map[string]any)) {
	t.Helper()
	var rawMap map[string]any
	if err := json.Unmarshal(body, &rawMap); err != nil {
		t.Fatalf("Failed to parse JSON object: %v", err)
	}
	if len(expectedFields) > 0 {
		if len(rawMap) != len(expectedFields) {
			t.Errorf("Field count mismatch: want %d, got %d", len(expectedFields), len(rawMap))
		}
		for _, field := range expectedFields {
			if _, exists := rawMap[field]; !exists {
				t.Errorf("Contract Broken: client code expects key '%s', but it was missing", field)
			}
		}
	}
	if nestedChecks != nil {
		nestedChecks(t, rawMap)
	}
}

func assertArrayContract(t *testing.T, body []byte, expectedFields []string, itemChecks func(t *testing.T, item map[string]any)) {
	t.Helper()
	var rawArr []map[string]any
	if err := json.Unmarshal(body, &rawArr); err != nil {
		t.Fatalf("Failed to parse JSON array: %v", err)
	}
	if len(rawArr) == 0 {
		t.Fatalf("Expected non-empty array response")
	}
	if len(expectedFields) > 0 {
		targetObj := rawArr[0]
		if len(targetObj) != len(expectedFields) {
			t.Errorf("Total number of fields exceeded, want %d, got %d", len(expectedFields), len(targetObj))
		}
		for _, field := range expectedFields {
			if _, exists := targetObj[field]; !exists {
				t.Errorf("Contract Broken: client code expects key '%s', but it was missing", field)
			}
		}
		if itemChecks != nil {
			itemChecks(t, targetObj)
		}
	}
}

func assertMatchStateNested(t *testing.T, raw map[string]any) {
	t.Helper()
	if gridRaw, ok := raw["grid"].([]any); ok && len(gridRaw) > 0 {
		if row, ok := gridRaw[0].([]any); ok && len(row) > 0 {
			if tile, ok := row[0].(map[string]any); ok {
				for _, field := range []string{"type", "occupantType", "occupantId"} {
					if _, exists := tile[field]; !exists {
						t.Errorf("grid[0][0] missing key '%s'", field)
					}
				}
			}
		}
	}

	if sbRaw, ok := raw["softBlocks"].([]any); ok && len(sbRaw) > 0 {
		if sb, ok := sbRaw[0].(map[string]any); ok {
			for _, field := range []string{"id", "position"} {
				if _, exists := sb[field]; !exists {
					t.Errorf("softBlock missing key '%s'", field)
				}
			}
			if coord, ok := sb["position"].(map[string]any); ok {
				for _, field := range []string{"x", "y"} {
					if _, exists := coord[field]; !exists {
						t.Errorf("softBlock.position missing key '%s'", field)
					}
				}
			}
		}
	}

	if unitsRaw, ok := raw["units"].([]any); ok && len(unitsRaw) > 0 {
		if unit, ok := unitsRaw[0].(map[string]any); ok {
			unitFields := []string{"id", "type", "position", "speed", "bombMaxRange", "bombPower", "maxBombCount", "bombUsed", "team", "hp", "skills", "hasMoved", "hasUsedSkill"}
			for _, field := range unitFields {
				if _, exists := unit[field]; !exists {
					t.Errorf("Contract Broken: unit missing key '%s'", field)
				}
			}
			if pos, ok := unit["position"].(map[string]any); ok {
				for _, field := range []string{"x", "y"} {
					if _, exists := pos[field]; !exists {
						t.Errorf("unit.position missing key '%s'", field)
					}
				}
			}
		}
	}

	if bombsRaw, ok := raw["bombs"].([]any); ok && len(bombsRaw) > 0 {
		if bomb, ok := bombsRaw[0].(map[string]any); ok {
			bombFields := []string{"id", "ownerUnitID", "position", "range", "placedTurn", "countdown"}
			for _, field := range bombFields {
				if _, exists := bomb[field]; !exists {
					t.Errorf("Contract Broken: bomb missing key '%s'", field)
				}
			}
		}
	}

	if tcRaw, ok := raw["turnCommands"].([]any); ok && len(tcRaw) > 0 {
		if tc, ok := tcRaw[0].(map[string]any); ok {
			tcFields := []string{"type", "unitID"}
			for _, field := range tcFields {
				if _, exists := tc[field]; !exists {
					t.Errorf("Contract Broken: turnCommand missing key '%s'", field)
				}
			}
		}
	}
}

func testEncodeFailure(t *testing.T, handler http.Handler, setup func() *http.Request, expectedStatus int) {
	t.Helper()
	req := setup()
	brokenWriter := &BrokenResponseWriter{}
	handler.ServeHTTP(brokenWriter, req)
	if brokenWriter.Code != expectedStatus {
		t.Errorf("Expected initial header setup to attempt status %d, got %d", expectedStatus, brokenWriter.Code)
	}
}

func TestHandleGetCatalog(t *testing.T) {
	h := NewHandler(NewServerStateManager())

	t.Run("Success: called engine.GetAllArchetypes and engine.GetAllStagePresets", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/catalog", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()

		http.HandlerFunc(h.HandleGetCatalog).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response CatalogResopnse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		if got, want := len(response.Archetypes), len(engine.GetAllArchetypes()); got != want {
			t.Errorf("Handler returned unexpected number of archetypes: got %d want %d", got, want)
		}
		if got, want := len(response.StagePresets), len(engine.GetAllStagePresets()); got != want {
			t.Errorf("Handler returned unexpected number of stagePresets: got %d want %d", got, want)
		}
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		testEncodeFailure(t, http.HandlerFunc(h.HandleGetCatalog),
			func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/catalog", nil)
				return req
			}, http.StatusOK)
	})

	t.Run("Test Contract", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/catalog", nil)
		rr := httptest.NewRecorder()

		http.HandlerFunc(h.HandleGetCatalog).ServeHTTP(rr, req)

		assertObjectContract(t, rr.Body.Bytes(), []string{"archetypes", "stagePresets"}, func(t *testing.T, raw map[string]any) {
			t.Helper()
			archetypesBytes, err := json.Marshal(raw["archetypes"])
			if err != nil {
				t.Fatalf("Failed to re-marshal archetypes: %v", err)
			}
			assertArrayContract(t, archetypesBytes, []string{"name", "speed", "bombMaxRange", "skills"}, nil)

			stagePresetsBytes, err := json.Marshal(raw["stagePresets"])
			if err != nil {
				t.Fatalf("Failed to re-marshal stagePresets: %v", err)
			}
			assertArrayContract(t, stagePresetsBytes, []string{"name", "description", "width", "height", "maxTurns"}, nil)
		})
	})
}

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
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
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
		s := NewServerStateManager()
		h := NewHandler(s)
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		gameCfg := engine.GameCfg{
			StagePreset: "Plain",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		jsonBody, _ := json.Marshal(CreateMatchRequest{GameCfg: gameCfg})
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match", h.HandleCreateMatch).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusCreated)
		}
		if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, "application/json")
		}

		var response CreateMatchResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}
		if !response.Success {
			t.Error("Expected success=true in response, got false")
		}

		// Verify match was actually created in the server state manager
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

	t.Run("Failure: room not found", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		gameCfg := engine.GameCfg{
			StagePreset: "Plain",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		jsonBody, _ := json.Marshal(gameCfg)
		req, err := http.NewRequest("POST", "/api/match-rooms/NONEXISTENT/match", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match", h.HandleCreateMatch).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "room not found") {
			t.Errorf("Expected error message 'room not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: match already exists", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}
		room := mustRoom(t, s, roomID)
		room.Match = &engine.Match{}

		gameCfg := engine.GameCfg{
			StagePreset: "Plain",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		jsonBody, _ := json.Marshal(CreateMatchRequest{GameCfg: gameCfg})
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match", h.HandleCreateMatch).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusConflict {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusConflict)
		}
		if !strings.Contains(rr.Body.String(), "match already exists") {
			t.Errorf("Expected error message 'match already exists', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid game config", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		gameCfg := engine.GameCfg{
			StagePreset: "INVALID_STAGE", // Invalid stage ID
			MaxTurns:    10,
		}
		jsonBody, _ := json.Marshal(CreateMatchRequest{GameCfg: gameCfg})
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match", h.HandleCreateMatch).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "invalid game config") {
			t.Errorf("Expected error message 'invalid game config', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid JSON format", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		// Malformed JSON body
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match", strings.NewReader("{invalid json"))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match", h.HandleCreateMatch).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "Invalid configuration format") {
			t.Errorf("Expected error message 'Invalid configuration format', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: failed to Encode response", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		gameCfg := engine.GameCfg{
			StagePreset: "Plain",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
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
		s := NewServerStateManager()
		h := NewHandler(s)
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}
		gameCfg := engine.GameCfg{
			StagePreset: "Plain",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		jsonBody, _ := json.Marshal(CreateMatchRequest{GameCfg: gameCfg})
		req, _ := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match", strings.NewReader(string(jsonBody)))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match", h.HandleCreateMatch).ServeHTTP(rr, req)

		assertObjectContract(t, rr.Body.Bytes(), []string{"success", "playerTokens"}, nil)
	})
}

func TestHandleRematch(t *testing.T) {
	t.Run("Success: wipes existing Match and creates a new one", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/rematch", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/rematch", h.HandleRematch).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusCreated)
		}
		if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, "application/json")
		}

		var response CreateMatchResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}
		if !response.Success {
			t.Error("Expected success=true in response, got false")
		}

		// Verify match was actually recreated in the server state manager
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

	t.Run("Failure: room not found", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)

		req, err := http.NewRequest("POST", "/api/match-rooms/NONEXISTENT/rematch", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer dummy-token")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/rematch", h.HandleRematch).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "room not found") {
			t.Errorf("Expected error message 'room not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: match not found", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		room := mustRoom(t, s, roomID)
		room.GameCfg = nil

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/rematch", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/rematch", h.HandleRematch).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "match not found") {
			t.Errorf("Expected error message 'match not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: missing Authorization header", func(t *testing.T) {
		roomID, _, _, h := createTestRoomWithMatch(t)

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/rematch", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/rematch", h.HandleRematch).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "invalid player token") {
			t.Errorf("Expected error message 'invalid player token', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid token", func(t *testing.T) {
		roomID, _, _, h := createTestRoomWithMatch(t)

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/rematch", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer invalid-token")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/rematch", h.HandleRematch).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "invalid player token") {
			t.Errorf("Expected error message 'invalid player token', got: %s", rr.Body.String())
		}
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

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/rematch", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/rematch", h.HandleRematch).ServeHTTP(rr, req)

		assertObjectContract(t, rr.Body.Bytes(), []string{"success", "playerTokens"}, nil)
	})
}

func TestHandleDeleteMatch(t *testing.T) {
	t.Run("Success: deletes an existing concluded Match", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		room := mustRoom(t, s, roomID)
		room.Match.WinnerTeamID = 1 // conclude the match

		req, err := http.NewRequest("DELETE", "/api/match-rooms/"+roomID+"/match", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("DELETE /api/match-rooms/{roomID}/match", h.HandleDeleteMatch).ServeHTTP(rr, req)

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

		req, err := http.NewRequest("DELETE", "/api/match-rooms/"+roomID+"/match", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("DELETE /api/match-rooms/{roomID}/match", h.HandleDeleteMatch).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNoContent {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
		}
	})

	t.Run("Failure: match still in progress", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		req, err := http.NewRequest("DELETE", "/api/match-rooms/"+roomID+"/match", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("DELETE /api/match-rooms/{roomID}/match", h.HandleDeleteMatch).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusConflict {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusConflict)
		}
		if !strings.Contains(rr.Body.String(), "match still in progress") {
			t.Errorf("Expected error message 'match still in progress', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: room not found", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)

		req, err := http.NewRequest("DELETE", "/api/match-rooms/NONEXISTENT/match", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer dummy-token")

		rr := httptest.NewRecorder()
		testMux("DELETE /api/match-rooms/{roomID}/match", h.HandleDeleteMatch).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "room not found") {
			t.Errorf("Expected error message 'room not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid token", func(t *testing.T) {
		roomID, _, _, h := createTestRoomWithMatch(t)

		req, err := http.NewRequest("DELETE", "/api/match-rooms/"+roomID+"/match", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer invalid-token")

		rr := httptest.NewRecorder()
		testMux("DELETE /api/match-rooms/{roomID}/match", h.HandleDeleteMatch).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "invalid player token") {
			t.Errorf("Expected error message 'invalid player token', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: missing Authorization header", func(t *testing.T) {
		roomID, _, _, h := createTestRoomWithMatch(t)

		req, err := http.NewRequest("DELETE", "/api/match-rooms/"+roomID+"/match", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("DELETE /api/match-rooms/{roomID}/match", h.HandleDeleteMatch).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "invalid player token") {
			t.Errorf("Expected error message 'invalid player token', got: %s", rr.Body.String())
		}
	})
}

// ClientUnit mimics the client's view of Unit in GetMatchState response
type ClientUnit struct {
	ID           engine.UnitID     `json:"id"`
	Type         string            `json:"type"`
	Position     engine.Coordinate `json:"position"`
	Speed        int               `json:"speed"`
	BombMaxRange int               `json:"bombMaxRange"`
	BombPower    int               `json:"bombPower"`
	MaxBombCount int               `json:"maxBombCount"`
	BombUsed     int               `json:"bombUsed"`
	Team         int               `json:"team"`
	HP           int               `json:"hp"`
	Skills       []string          `json:"skills"`
	HasMoved     bool              `json:"hasMoved"`
	HasUsedSkill bool              `json:"hasUsedSkill"`
}

// ClientTile mimics the client's view of Tile in GetMatchState response
type ClientTile struct {
	Type         string `json:"type"`
	OccupantType string `json:"occupantType"`
	OccupantID   int64  `json:"occupantId"`
}

// ClientMatchStateResponse mimics the client's view of GetMatchState response
type ClientMatchStateResponse struct {
	Turn         int                  `json:"turn"`
	ActiveTeam   int                  `json:"activeTeam"`
	Grid         [][]ClientTile       `json:"grid"`
	Units        []ClientUnit         `json:"units"`
	Bombs        []*engine.Bomb       `json:"bombs"`
	SoftBlocks   []*engine.SoftBlock  `json:"softBlocks"`
	TurnCommands []engine.TurnCommand `json:"turnCommands"`
}

func TestHandleGetMatchState(t *testing.T) {
	t.Run("Success: get match state in an existing room", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		gameCfg := engine.GameCfg{
			StagePreset: "Divided",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		_, err = s.CreateMatch(roomID, gameCfg)
		if err != nil {
			t.Fatalf("Failed to create match: %v", err)
		}

		req, err := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/state", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("GET /api/match-rooms/{roomID}/match/state", h.HandleGetMatchState).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response ClientMatchStateResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		if response.Turn != 1 {
			t.Errorf("Expected turn 1, got %d", response.Turn)
		}
		if response.ActiveTeam != 1 {
			t.Errorf("Expected activeTeam 1, got %d", response.ActiveTeam)
		}
		if len(response.Units) == 0 {
			t.Error("Expected units to be populated")
		}
		for _, u := range response.Units {
			if u.ID == 0 {
				t.Error("Unit missing ID")
			}
			if u.Type == "" {
				t.Error("Unit missing type")
			}
			if u.HP != 1 {
				t.Errorf("Expected unit HP 1, got %d", u.HP)
			}
		}
	})

	t.Run("Failure: room not found", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		req, err := http.NewRequest("GET", "/api/match-rooms/NONEXISTENT/match/state", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("GET /api/match-rooms/{roomID}/match/state", h.HandleGetMatchState).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "room not found") {
			t.Errorf("Expected error message 'room not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: match not found", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		req, err := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/state", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("GET /api/match-rooms/{roomID}/match/state", h.HandleGetMatchState).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "match not found") {
			t.Errorf("Expected error message 'match not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		gameCfg := engine.GameCfg{
			StagePreset: "Plain",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		_, err = s.CreateMatch(roomID, gameCfg)
		if err != nil {
			t.Fatalf("Failed to create match: %v", err)
		}

		testEncodeFailure(t, testMux("GET /api/match-rooms/{roomID}/match/state", h.HandleGetMatchState),
			func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/state", nil)
				return req
			}, http.StatusOK)
	})

	t.Run("Test Contract", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		gameCfg := engine.GameCfg{
			StagePreset: "Divided",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		_, err = s.CreateMatch(roomID, gameCfg)
		if err != nil {
			t.Fatalf("Failed to create match: %v", err)
		}

		req, _ := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/state", nil)
		rr := httptest.NewRecorder()
		testMux("GET /api/match-rooms/{roomID}/match/state", h.HandleGetMatchState).ServeHTTP(rr, req)

		assertObjectContract(t, rr.Body.Bytes(),
			[]string{"turn", "inSuddenDeath", "activeTeam", "grid", "units", "bombs", "softBlocks", "turnCommands"},
			assertMatchStateNested)
	})
}

func createTestRoomWithMatch(t *testing.T) (string, [2]string, *ServerStateManager, *Handler) {
	t.Helper()

	s := NewServerStateManager()
	h := NewHandler(s)
	roomID, err := s.CreateMatchRoom()
	if err != nil {
		t.Fatalf("Failed to create room: %v", err)
	}

	gameCfg := engine.GameCfg{
		StagePreset: "Plain",
		P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
		P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
		MaxTurns:    10,
	}
	playerTokens, err := s.CreateMatch(roomID, gameCfg)
	if err != nil {
		t.Fatalf("Failed to create match: %v", err)
	}

	return roomID, playerTokens, s, h
}

func TestHandleSubmitTurnCommand(t *testing.T) {
	t.Run("Success: submit a valid TurnCommand in an existing room", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)

		unitID := engine.NewUnitID(1, 0)
		newPos := engine.Coordinate{X: 4, Y: 7}
		jsonBody, _ := json.Marshal(engine.NewMoveCommand(unitID, newPos))

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/turn-commands", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/turn-commands", h.HandleSubmitTurnCommand).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response []engine.GameEvent
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}
		if len(response) != 1 {
			t.Errorf("expected 1 GameEvent returned, got %d", len(response))
		}
		resEvt := response[0]
		validFrom := engine.Coordinate{X: 4, Y: 8}
		if resEvt.Type != engine.GameEvtUnitMoved || resEvt.UnitID != unitID || *resEvt.From != validFrom || *resEvt.To != newPos {
			t.Errorf("malformed UnitMoveEvent returned: %+v", resEvt)
		}

		room := mustRoom(t, s, roomID)
		u := room.Match.WorkingState.Units[unitID]

		if u.Position != newPos {
			t.Errorf("Expected Unit %#X new position %#v, got %#v", unitID, newPos, u.Position)
		}
	})

	t.Run("Failure: invalid TurnCommand", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		unitID := engine.NewUnitID(1, 0)
		newPos := engine.Coordinate{X: 4, Y: 777}
		jsonBody, _ := json.Marshal(engine.NewMoveCommand(unitID, newPos))

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/turn-commands", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/turn-commands", h.HandleSubmitTurnCommand).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusConflict {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusConflict)
		}
		if !strings.Contains(rr.Body.String(), "invalid turn command") {
			t.Errorf("Expected error message 'invalid turn command', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid JSON format", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		// Malformed JSON body
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/turn-commands", strings.NewReader("{invalid json"))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/turn-commands", h.HandleSubmitTurnCommand).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "Invalid turnCommand format") {
			t.Errorf("Expected error message 'Invalid turnCommand format', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: room not found", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)

		unitID := engine.NewUnitID(1, 0)
		newPos := engine.Coordinate{X: 4, Y: 7}
		jsonBody, _ := json.Marshal(engine.NewMoveCommand(unitID, newPos))

		req, err := http.NewRequest("POST", "/api/match-rooms/NONEXISTENT/match/turn-commands", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer dummy-token")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/turn-commands", h.HandleSubmitTurnCommand).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "room not found") {
			t.Errorf("Expected error message 'room not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: match not found", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		room := mustRoom(t, s, roomID)
		room.Match = nil

		unitID := engine.NewUnitID(1, 0)
		newPos := engine.Coordinate{X: 4, Y: 7}
		jsonBody, _ := json.Marshal(engine.NewMoveCommand(unitID, newPos))

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/turn-commands", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/turn-commands", h.HandleSubmitTurnCommand).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "match not found") {
			t.Errorf("Expected error message 'match not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)
		unitID := engine.NewUnitID(1, 0)
		newPos := engine.Coordinate{X: 4, Y: 7}
		jsonBody, _ := json.Marshal(engine.NewMoveCommand(unitID, newPos))

		testEncodeFailure(t, testMux("POST /api/match-rooms/{roomID}/match/turn-commands", h.HandleSubmitTurnCommand),
			func() *http.Request {
				req, _ := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/turn-commands", strings.NewReader(string(jsonBody)))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+playerTokens[0])
				return req
			}, http.StatusOK)
	})

	t.Run("Test Contract", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)
		unitID := engine.NewUnitID(1, 0)
		newPos := engine.Coordinate{X: 4, Y: 7}
		jsonBody, _ := json.Marshal(engine.NewMoveCommand(unitID, newPos))
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/turn-commands", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/turn-commands", h.HandleSubmitTurnCommand).ServeHTTP(rr, req)

		assertArrayContract(t, rr.Body.Bytes(),
			[]string{"type", "unitId", "from", "to",
				"countdown", "newHp"}, // unrelated to this GameEvent
			func(t *testing.T, item map[string]any) {
				t.Helper()
				fromField := item["from"].(map[string]any)
				toField := item["from"].(map[string]any)
				for _, field := range []string{"x", "y"} {
					if _, exists := fromField[field]; !exists {
						t.Errorf("Contract Broken: from missing key '%s'", field)
					}
					if _, exists := toField[field]; !exists {
						t.Errorf("Contract Broken: to missing key '%s'", field)
					}
				}
			})
	})

	t.Run("Failure: missing Authorization header", func(t *testing.T) {
		roomID, _, _, h := createTestRoomWithMatch(t)

		unitID := engine.NewUnitID(1, 0)
		newPos := engine.Coordinate{X: 4, Y: 7}
		jsonBody, _ := json.Marshal(engine.NewMoveCommand(unitID, newPos))

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/turn-commands", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/turn-commands", h.HandleSubmitTurnCommand).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "invalid player token") {
			t.Errorf("Expected error message 'invalid player token', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid token", func(t *testing.T) {
		roomID, _, _, h := createTestRoomWithMatch(t)

		unitID := engine.NewUnitID(1, 0)
		newPos := engine.Coordinate{X: 4, Y: 7}
		jsonBody, _ := json.Marshal(engine.NewMoveCommand(unitID, newPos))

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/turn-commands", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer invalid-token")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/turn-commands", h.HandleSubmitTurnCommand).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "invalid player token") {
			t.Errorf("Expected error message 'invalid player token', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: wrong team token", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		unitID := engine.NewUnitID(1, 0)
		newPos := engine.Coordinate{X: 4, Y: 7}
		jsonBody, _ := json.Marshal(engine.NewMoveCommand(unitID, newPos))

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/turn-commands", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+playerTokens[1])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/turn-commands", h.HandleSubmitTurnCommand).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "invalid player token") {
			t.Errorf("Expected error message 'invalid player token', got: %s", rr.Body.String())
		}
	})
}

func TestHandleStartTurn(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		room := mustRoom(t, s, roomID)
		room.Match.TrueState.Turn = 1000
		room.Match.WorkingState.Turn = 1000

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/start-turn", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+playerTokens[0]) // Team 1's turn

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/start-turn", h.HandleStartTurn).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response StartTurnResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		if got, want := len(room.Match.WorkingState.Bombs), 2; got != want {
			t.Errorf("Expected SuddenDeath triggered and drop %d bombs, got %d", want, got)
		}
		if len(response.GameEvents) != 2 {
			t.Errorf("expected 2 GameEvent returned, got %d", len(response.GameEvents))
		}
		for _, evt := range response.GameEvents {
			if evt.Type != engine.GameEvtBombPlaced {
				t.Errorf("malformed EvtBombPlaced returned: %+v", evt)
			}
		}
		if want, got := response.InSuddenDeath, true; want != got {
			t.Errorf("Expected suddenDeath to be true, got %v", got)
		}
	})

	t.Run("Failure: room not found", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)

		req, err := http.NewRequest("POST", "/api/match-rooms/NONEXISTENT/match/start-turn", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer dummy-token")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/start-turn", h.HandleStartTurn).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "room not found") {
			t.Errorf("Expected error message 'room not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: match not found", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		room := mustRoom(t, s, roomID)
		room.Match = nil

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/start-turn", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/start-turn", h.HandleStartTurn).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "match not found") {
			t.Errorf("Expected error message 'match not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		testEncodeFailure(t, testMux("POST /api/match-rooms/{roomID}/match/start-turn", h.HandleStartTurn),
			func() *http.Request {
				req, _ := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/start-turn", nil)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+playerTokens[0])
				return req
			}, http.StatusOK)
	})

	t.Run("Test Contract", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		room := mustRoom(t, s, roomID)
		room.Match.TrueState.Turn = 99999
		room.Match.WorkingState.Turn = 99999

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/start-turn", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/start-turn", h.HandleStartTurn).ServeHTTP(rr, req)

		assertObjectContract(t, rr.Body.Bytes(),
			[]string{"inSuddenDeath", "gameEvents"},
			func(t *testing.T, raw map[string]any) {
				t.Helper()
				gameEventsBytes, err := json.Marshal(raw["gameEvents"])
				if err != nil {
					t.Fatalf("Failed to re-marshal gameEvents: %v", err)
				}
				assertArrayContract(t, gameEventsBytes,
					[]string{"type", "unitId", "bombId", "position", "range", "countdown",
						"newHp"}, // unrelated to this GameEvent
					func(t *testing.T, item map[string]any) {
						t.Helper()
						positionField := item["position"].(map[string]any)
						for _, field := range []string{"x", "y"} {
							if _, exists := positionField[field]; !exists {
								t.Errorf("Contract Broken: from missing key '%s'", field)
							}
						}
					})
			})
	})

	t.Run("Failure: missing Authorization header", func(t *testing.T) {
		roomID, _, _, h := createTestRoomWithMatch(t)

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/start-turn", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/start-turn", h.HandleStartTurn).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "invalid player token") {
			t.Errorf("Expected error message 'invalid player token', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid token", func(t *testing.T) {
		roomID, _, _, h := createTestRoomWithMatch(t)

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/start-turn", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer invalid-token")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/start-turn", h.HandleStartTurn).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "invalid player token") {
			t.Errorf("Expected error message 'invalid player token', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: wrong team token", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/start-turn", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+playerTokens[1]) // Team 2 token for Team 1's turn

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/start-turn", h.HandleStartTurn).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "invalid player token") {
			t.Errorf("Expected error message 'invalid player token', got: %s", rr.Body.String())
		}
	})
}

func TestHandleResetTurn(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		unitID := engine.NewUnitID(1, 0)
		room := mustRoom(t, s, roomID)
		room.Match.WorkingState.Units[unitID].HasMoved = true

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/reset", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/reset", h.HandleResetTurn).ServeHTTP(rr, req)

		if got, want := rr.Code, http.StatusNoContent; got != want {
			t.Errorf("Handler returned wrong status code: got %v want %v", got, want)
		}

		if contentType := rr.Header().Get("Content-Type"); contentType != "" {
			t.Errorf("Expected no Content-Type on 204, got %v", contentType)
		}

		if rr.Body.Len() > 0 {
			t.Errorf("Expect empty body, got %q", rr.Body.String())
		}

		if got, want := room.Match.WorkingState.Units[unitID].HasMoved, false; got != want {
			t.Errorf("Expected Unit %#X HasMoved reset to %v, got %v", unitID, want, got)
		}
	})

	t.Run("Failure: room not found", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)

		req, err := http.NewRequest("POST", "/api/match-rooms/NONEXISTENT/match/reset", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer dummy-token")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/reset", h.HandleResetTurn).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "room not found") {
			t.Errorf("Expected error message 'room not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: match not found", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		room := mustRoom(t, s, roomID)
		room.Match = nil

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/reset", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/reset", h.HandleResetTurn).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "match not found") {
			t.Errorf("Expected error message 'match not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: missing Authorization header", func(t *testing.T) {
		roomID, _, _, h := createTestRoomWithMatch(t)

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/reset", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/reset", h.HandleResetTurn).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "invalid player token") {
			t.Errorf("Expected error message 'invalid player token', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid token", func(t *testing.T) {
		roomID, _, _, h := createTestRoomWithMatch(t)

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/reset", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer invalid-token")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/reset", h.HandleResetTurn).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "invalid player token") {
			t.Errorf("Expected error message 'invalid player token', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: wrong team token", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/reset", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+playerTokens[1])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/reset", h.HandleResetTurn).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "invalid player token") {
			t.Errorf("Expected error message 'invalid player token', got: %s", rr.Body.String())
		}
	})
}

func TestHandleResolveTurn(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		unitID := engine.NewUnitID(1, 0)
		s.SubmitTurnCommand(roomID, engine.NewPlaceBombCommand(unitID, engine.Coordinate{X: 4, Y: 7}), playerTokens[0])
		room := mustRoom(t, s, roomID)
		room.Match.WorkingState.Bombs[engine.NewBombID(1, 1, unitID)].Countdown = 1

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/resolve", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/resolve", h.HandleResolveTurn).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response []engine.GameEvent
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		if got, want := response, 6; len(got) != want {
			t.Errorf("Expected %d gameEvents returned, got %#v", want, got)
		}
		if got, want := room.Match.WorkingState.Units[unitID].HP, 0; got != want {
			t.Errorf("Expected Unit %#X HP %v, got %v", 16, want, got)
		}
		if got, want := room.Match.WinnerTeamID, 2; got != want {
			t.Errorf("Expected match winner = %v, got %v", want, got)
		}
	})

	t.Run("Failure: room not found", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)

		req, err := http.NewRequest("POST", "/api/match-rooms/NONEXISTENT/match/resolve", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer dummy-token")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/resolve", h.HandleResolveTurn).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "room not found") {
			t.Errorf("Expected error message 'room not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: match not found", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		room := mustRoom(t, s, roomID)
		room.Match = nil

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/resolve", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/resolve", h.HandleResolveTurn).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "match not found") {
			t.Errorf("Expected error message 'match not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		testEncodeFailure(t, testMux("POST /api/match-rooms/{roomID}/match/resolve", h.HandleResolveTurn),
			func() *http.Request {
				req, _ := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/resolve", nil)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+playerTokens[0])
				return req
			}, http.StatusOK)
	})

	t.Run("Test Contract", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		unitID := engine.NewUnitID(1, 0)
		s.SubmitTurnCommand(roomID, engine.NewPlaceBombCommand(unitID, engine.Coordinate{X: 4, Y: 7}), playerTokens[0])

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/resolve", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/resolve", h.HandleResolveTurn).ServeHTTP(rr, req)

		assertArrayContract(t, rr.Body.Bytes(),
			[]string{"type", "unitId", "bombId", "position", "range", "countdown",
				"newHp"}, // unrelated to this GameEvent
			func(t *testing.T, item map[string]any) {
				t.Helper()
				positionField := item["position"].(map[string]any)
				for _, field := range []string{"x", "y"} {
					if _, exists := positionField[field]; !exists {
						t.Errorf("Contract Broken: position missing key '%s'", field)
					}
				}
			})
	})

	t.Run("Failure: missing Authorization header", func(t *testing.T) {
		roomID, _, _, h := createTestRoomWithMatch(t)

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/resolve", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/resolve", h.HandleResolveTurn).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "invalid player token") {
			t.Errorf("Expected error message 'invalid player token', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid token", func(t *testing.T) {
		roomID, _, _, h := createTestRoomWithMatch(t)

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/resolve", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer invalid-token")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/resolve", h.HandleResolveTurn).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "invalid player token") {
			t.Errorf("Expected error message 'invalid player token', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: wrong team token", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/resolve", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+playerTokens[1])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/resolve", h.HandleResolveTurn).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "invalid player token") {
			t.Errorf("Expected error message 'invalid player token', got: %s", rr.Body.String())
		}
	})
}

func TestHandleConsumeCPUStatus(t *testing.T) {
	unitID := engine.NewUnitID(1, 0)

	t.Run("Success: TurnPhaseReady returns and clears pending events", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		room := mustRoom(t, s, roomID)
		room.Match.CPU.Phase = engine.TurnPhaseReady
		room.Match.CPU.PendingEvents = []engine.GameEvent{
			engine.NewUnitMovedEvent(unitID, engine.Coordinate{X: 1, Y: 2}, engine.Coordinate{X: 2, Y: 2}),
		}

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/cpu-status/consume", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/cpu-status/consume", h.HandleConsumeCPUStatus).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response struct {
			TurnPhase     string             `json:"turnPhase"`
			PendingEvents []engine.GameEvent `json:"pendingGameEvents"`
		}
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		if got, want := response.TurnPhase, engine.TurnPhaseReady.String(); got != want {
			t.Errorf("Expected turnPhase %v, got %v", want, got)
		}
		if got, want := len(response.PendingEvents), 1; got != want {
			t.Errorf("Expected %d pendingGameEvents returned, got %#v", want, response.PendingEvents)
		}

		if got, want := room.Match.CPU.Phase, engine.TurnPhaseIdle; got != want {
			t.Errorf("Expected CPU.Phase reset to %v, got %v", want, got)
		}
		if got, want := len(room.Match.CPU.PendingEvents), 0; got != want {
			t.Errorf("Expected CPU.PendingEvents cleared, got %#v", room.Match.CPU.PendingEvents)
		}
	})

	t.Run("Success: TurnPhasePlanning leaves state untouched", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		room := mustRoom(t, s, roomID)
		room.Match.CPU.Phase = engine.TurnPhasePlanning

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/cpu-status/consume", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/cpu-status/consume", h.HandleConsumeCPUStatus).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		body := rr.Body.Bytes()

		var response struct {
			TurnPhase     string             `json:"turnPhase"`
			PendingEvents []engine.GameEvent `json:"pendingGameEvents"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		if got, want := response.TurnPhase, engine.TurnPhasePlanning.String(); got != want {
			t.Errorf("Expected turnPhase %v, got %v", want, got)
		}
		if got, want := len(response.PendingEvents), 0; got != want {
			t.Errorf("Expected no pendingGameEvents, got %#v", response.PendingEvents)
		}
		if got, want := room.Match.CPU.Phase, engine.TurnPhasePlanning; got != want {
			t.Errorf("Expected CPU.Phase to stay at %v, got %v", want, got)
		}

		var raw map[string]any
		if err := json.Unmarshal(body, &raw); err != nil {
			t.Fatalf("Failed to parse raw JSON: %v", err)
		}
		if _, ok := raw["pendingGameEvents"].([]any); !ok {
			t.Errorf("Expected pendingGameEvents to serialize as [], got %#v (client types this field as non-nullable)", raw["pendingGameEvents"])
		}
	})

	t.Run("Success: either player's token is accepted", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		room := mustRoom(t, s, roomID)
		room.Match.CPU.Phase = engine.TurnPhaseReady

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/cpu-status/consume", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+playerTokens[1])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/cpu-status/consume", h.HandleConsumeCPUStatus).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})

	t.Run("Failure: room not found", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)

		req, err := http.NewRequest("POST", "/api/match-rooms/NONEXISTENT/match/cpu-status/consume", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer dummy-token")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/cpu-status/consume", h.HandleConsumeCPUStatus).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "room not found") {
			t.Errorf("Expected error message 'room not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: match not found", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		room := mustRoom(t, s, roomID)
		room.Match = nil

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/cpu-status/consume", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/cpu-status/consume", h.HandleConsumeCPUStatus).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "match not found") {
			t.Errorf("Expected error message 'match not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		testEncodeFailure(t, testMux("POST /api/match-rooms/{roomID}/match/cpu-status/consume", h.HandleConsumeCPUStatus),
			func() *http.Request {
				req, _ := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/cpu-status/consume", nil)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+playerTokens[0])
				return req
			}, http.StatusOK)
	})

	t.Run("Test Contract", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		room := mustRoom(t, s, roomID)
		room.Match.CPU.Phase = engine.TurnPhaseReady
		room.Match.CPU.PendingEvents = []engine.GameEvent{
			engine.NewUnitMovedEvent(unitID, engine.Coordinate{X: 1, Y: 2}, engine.Coordinate{X: 2, Y: 2}),
		}

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/cpu-status/consume", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/cpu-status/consume", h.HandleConsumeCPUStatus).ServeHTTP(rr, req)

		assertObjectContract(t, rr.Body.Bytes(), []string{"turnPhase", "pendingGameEvents"},
			func(t *testing.T, raw map[string]any) {
				t.Helper()
				events, ok := raw["pendingGameEvents"].([]any)
				if !ok || len(events) == 0 {
					t.Fatalf("Expected non-empty pendingGameEvents, got %#v", raw["pendingGameEvents"])
				}
				evt := events[0].(map[string]any)
				for _, field := range []string{"type", "unitId", "from", "to"} {
					if _, exists := evt[field]; !exists {
						t.Errorf("Contract Broken: pendingGameEvents[0] missing key '%s'", field)
					}
				}
			})
	})

	t.Run("Failure: missing Authorization header", func(t *testing.T) {
		roomID, _, _, h := createTestRoomWithMatch(t)

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/cpu-status/consume", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/cpu-status/consume", h.HandleConsumeCPUStatus).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "invalid player token") {
			t.Errorf("Expected error message 'invalid player token', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid token", func(t *testing.T) {
		roomID, _, _, h := createTestRoomWithMatch(t)

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/cpu-status/consume", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer invalid-token")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/cpu-status/consume", h.HandleConsumeCPUStatus).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "invalid player token") {
			t.Errorf("Expected error message 'invalid player token', got: %s", rr.Body.String())
		}
	})
}

func TestHandleSurrender(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)

		jsonBody, _ := json.Marshal(SurrenderRequest{TeamID: 1})
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/surrender", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/surrender", h.HandleSurrender).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response []engine.GameEvent
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		if got, want := response, 1; len(got) != want {
			t.Errorf("Expected %d gameEvents returned, got %#v", want, got)
		}
		if got, want := response[0].WinnerTeamID, 2; got != want {
			t.Errorf("Expected gameEvent WinnerTeamID = %v, got %v", want, got)
		}
		// Room should not be deleted after surrender, verify it's still here
		if _, ok := s.Rooms.Load(roomID); !ok {
			t.Error("Expected room not to be deleted after surrender")
		}
	})

	t.Run("Failure: invalid SurrenderRequest", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		jsonBody, _ := json.Marshal(SurrenderRequest{TeamID: 3})
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/surrender", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/surrender", h.HandleSurrender).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "invalid game config") {
			t.Errorf("Expected error message 'invalid game config', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid JSON format", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/surrender", strings.NewReader("{invalid json}"))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/surrender", h.HandleSurrender).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "Invalid surrenderRequest format") {
			t.Errorf("Expected error message 'Invalid surrenderRequest format', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: room not found", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)

		jsonBody, _ := json.Marshal(SurrenderRequest{TeamID: 1})
		req, err := http.NewRequest("POST", "/api/match-rooms/NONEXISTENT/match/surrender", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer dummy-token")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/surrender", h.HandleSurrender).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "room not found") {
			t.Errorf("Expected error message 'room not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: match not found", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		room := mustRoom(t, s, roomID)
		room.Match = nil

		jsonBody, _ := json.Marshal(SurrenderRequest{TeamID: 1})
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/surrender", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/surrender", h.HandleSurrender).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "match not found") {
			t.Errorf("Expected error message 'match not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		testEncodeFailure(t, testMux("POST /api/match-rooms/{roomID}/match/surrender", h.HandleSurrender),
			func() *http.Request {
				jsonBody, _ := json.Marshal(SurrenderRequest{TeamID: 1})
				req, _ := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/surrender", strings.NewReader(string(jsonBody)))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+playerTokens[0])
				return req
			}, http.StatusOK)
	})

	t.Run("Test Contract", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		unitID := engine.NewUnitID(1, 0)
		s.SubmitTurnCommand(roomID, engine.NewPlaceBombCommand(unitID, engine.Coordinate{X: 4, Y: 7}), playerTokens[0])

		jsonBody, _ := json.Marshal(SurrenderRequest{TeamID: 1})
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/surrender", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+playerTokens[0])

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/surrender", h.HandleSurrender).ServeHTTP(rr, req)

		assertArrayContract(t, rr.Body.Bytes(), []string{"type", "winnerTeamId",
			"unitId", "countdown", "newHp"}, // unrelated to this GameEvent
			nil)
	})

	t.Run("Failure: missing Authorization header", func(t *testing.T) {
		roomID, _, _, h := createTestRoomWithMatch(t)

		jsonBody, _ := json.Marshal(SurrenderRequest{TeamID: 1})
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/surrender", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/surrender", h.HandleSurrender).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "invalid player token") {
			t.Errorf("Expected error message 'invalid player token', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid token", func(t *testing.T) {
		roomID, _, _, h := createTestRoomWithMatch(t)

		jsonBody, _ := json.Marshal(SurrenderRequest{TeamID: 1})
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/surrender", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer invalid-token")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/surrender", h.HandleSurrender).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "invalid player token") {
			t.Errorf("Expected error message 'invalid player token', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: wrong team token", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		jsonBody, _ := json.Marshal(SurrenderRequest{TeamID: 1})
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/surrender", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+playerTokens[1]) // Team 2 token for Team 1 surrender

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/surrender", h.HandleSurrender).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "invalid player token") {
			t.Errorf("Expected error message 'invalid player token', got: %s", rr.Body.String())
		}
	})
}

func TestHandleGetMatchConfig(t *testing.T) {
	t.Run("Success: get the gameConfig in an existing room", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		gameCfg := engine.GameCfg{
			StagePreset: "Divided",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		_, err = s.CreateMatch(roomID, gameCfg)
		if err != nil {
			t.Fatalf("Failed to create match: %v", err)
		}

		req, err := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/config", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("GET /api/match-rooms/{roomID}/match/config", h.HandleGetMatchConfig).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response engine.GameCfg
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		if response.StagePreset != gameCfg.StagePreset {
			t.Errorf("Expected stagePreset %v, got %v", validGameCfg().StagePreset, gameCfg.StagePreset)
		}
	})

	t.Run("Failure: room not found", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		req, err := http.NewRequest("GET", "/api/match-rooms/NONEXISTENT/match/config", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("GET /api/match-rooms/{roomID}/match/config", h.HandleGetMatchConfig).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "room not found") {
			t.Errorf("Expected error message 'room not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: match not found", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		req, err := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/config", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("GET /api/match-rooms/{roomID}/match/config", h.HandleGetMatchConfig).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "match not found") {
			t.Errorf("Expected error message 'match not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		gameCfg := engine.GameCfg{
			StagePreset: "Plain",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		_, err = s.CreateMatch(roomID, gameCfg)
		if err != nil {
			t.Fatalf("Failed to create match: %v", err)
		}

		testEncodeFailure(t, testMux("GET /api/match-rooms/{roomID}/match/config", h.HandleGetMatchConfig),
			func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/config", nil)
				return req
			}, http.StatusOK)
	})

	t.Run("Test Contract", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		gameCfg := engine.GameCfg{
			StagePreset: "Divided",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		_, err = s.CreateMatch(roomID, gameCfg)
		if err != nil {
			t.Fatalf("Failed to create match: %v", err)
		}

		req, _ := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/config", nil)
		rr := httptest.NewRecorder()
		testMux("GET /api/match-rooms/{roomID}/match/config", h.HandleGetMatchConfig).ServeHTTP(rr, req)

		assertObjectContract(t, rr.Body.Bytes(),
			[]string{"vsCpu", "stagePreset", "p1Slots", "p2Slots", "maxTurns", "allowResetTurn"}, nil)
	})
}

func TestHandleGetAllowedTiles(t *testing.T) {
	t.Run("Success: get the allowedTiles in an existing room", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		gameCfg := engine.GameCfg{
			StagePreset: "Divided",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		_, err = s.CreateMatch(roomID, gameCfg)
		if err != nil {
			t.Fatalf("Failed to create match: %v", err)
		}

		req, err := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/allowed-tiles?unitId=16&turnCmdType=placeBomb", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("GET /api/match-rooms/{roomID}/match/allowed-tiles", h.HandleGetAllowedTiles).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response []engine.Coordinate
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		cmpFunc := func(a, b engine.Coordinate) int {
			if a.X != b.X {
				return a.X - b.X
			}
			return a.Y - b.Y
		}
		want := []engine.Coordinate{
			{X: 2, Y: 8}, {X: 5, Y: 8}, {X: 6, Y: 8}, {X: 4, Y: 7}, {X: 4, Y: 6},
		}
		slices.SortFunc(want, cmpFunc)
		slices.SortFunc(response, cmpFunc)
		if !slices.Equal(want, response) {
			t.Errorf("Expected coordinates %#v, got %#v", want, response)
		}
	})

	t.Run("Success: a boxed-in unit encodes an empty array, not null", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}
		if _, err := s.CreateMatch(roomID, validGameCfg()); err != nil {
			t.Fatalf("Failed to create match: %v", err)
		}

		// Unit 0x10 spawns at {4,8} with an ally on {3,8}; soft blocks on {5,8} and {4,7}
		// plus the stage edge below leave it no legal landing tile.
		gs := mustRoom(t, s, roomID).Match.WorkingState
		gs.UpdateStageOccupant(engine.Coordinate{X: 5, Y: 8}, engine.OccupantSoftBlock, 1)
		gs.UpdateStageOccupant(engine.Coordinate{X: 4, Y: 7}, engine.OccupantSoftBlock, 2)

		req, err := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/allowed-tiles?unitId=16&turnCmdType=move", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("GET /api/match-rooms/{roomID}/match/allowed-tiles", h.HandleGetAllowedTiles).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		if got := strings.TrimSpace(rr.Body.String()); got != "[]" {
			t.Errorf("Expected an empty JSON array, got %q", got)
		}
	})

	t.Run("Failure: missing query string", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		gameCfg := engine.GameCfg{
			StagePreset: "Divided",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		_, err = s.CreateMatch(roomID, gameCfg)
		if err != nil {
			t.Fatalf("Failed to create match: %v", err)
		}

		req, err := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/allowed-tiles", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("GET /api/match-rooms/{roomID}/match/allowed-tiles", h.HandleGetAllowedTiles).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "missing required query parameters") {
			t.Errorf("Expected error message 'missing required query parameters', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid query string", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		gameCfg := engine.GameCfg{
			StagePreset: "Divided",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		_, err = s.CreateMatch(roomID, gameCfg)
		if err != nil {
			t.Fatalf("Failed to create match: %v", err)
		}

		req, err := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/allowed-tiles?unitId=abc&turnCmdType=move", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("GET /api/match-rooms/{roomID}/match/allowed-tiles", h.HandleGetAllowedTiles).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "Invalid unitId parameter") {
			t.Errorf("Expected error message 'Invalid unitId parameter', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: room not found", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		req, err := http.NewRequest("GET", "/api/match-rooms/NONEXISTENT/match/allowed-tiles?unitId=16&turnCmdType=placeBomb", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("GET /api/match-rooms/{roomID}/match/allowed-tiles", h.HandleGetAllowedTiles).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "room not found") {
			t.Errorf("Expected error message 'room not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: match not found", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		req, err := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/allowed-tiles?unitId=16&turnCmdType=placeBomb", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("GET /api/match-rooms/{roomID}/match/allowed-tiles", h.HandleGetAllowedTiles).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "match not found") {
			t.Errorf("Expected error message 'match not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		gameCfg := engine.GameCfg{
			StagePreset: "Plain",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		_, err = s.CreateMatch(roomID, gameCfg)
		if err != nil {
			t.Fatalf("Failed to create match: %v", err)
		}

		testEncodeFailure(t, testMux("GET /api/match-rooms/{roomID}/match/allowed-tiles", h.HandleGetAllowedTiles),
			func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/allowed-tiles?unitId=16&turnCmdType="+string(engine.TurnCmdPlaceBomb), nil)
				return req
			}, http.StatusOK)
	})

	t.Run("Test Contract", func(t *testing.T) {
		s := NewServerStateManager()
		h := NewHandler(s)
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		gameCfg := engine.GameCfg{
			StagePreset: "Divided",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		_, err = s.CreateMatch(roomID, gameCfg)
		if err != nil {
			t.Fatalf("Failed to create match: %v", err)
		}

		req, _ := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/allowed-tiles?unitId=16&turnCmdType=placeBomb", nil)
		rr := httptest.NewRecorder()
		testMux("GET /api/match-rooms/{roomID}/match/allowed-tiles", h.HandleGetAllowedTiles).ServeHTTP(rr, req)

		assertArrayContract(t, rr.Body.Bytes(),
			[]string{"x", "y"}, nil)
	})
}
