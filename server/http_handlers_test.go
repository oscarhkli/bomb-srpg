package server

import (
	"bomb-srpg/engine"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
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
		assertKeys(t, rawMap, expectedFields, "Contract Broken: client code expects key '%s', but it was missing")
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
		assertKeys(t, targetObj, expectedFields, "Contract Broken: client code expects key '%s', but it was missing")
		if itemChecks != nil {
			itemChecks(t, targetObj)
		}
	}
}

func firstElement(v any) (any, bool) {
	arr, ok := v.([]any)
	if !ok || len(arr) == 0 {
		return nil, false
	}
	return arr[0], true
}

func firstObject(v any) (map[string]any, bool) {
	first, ok := firstElement(v)
	if !ok {
		return nil, false
	}
	obj, ok := first.(map[string]any)
	return obj, ok
}

func doRequest(t *testing.T, handler http.HandlerFunc, pattern, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequest(method, path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	testMux(pattern, handler).ServeHTTP(rr, req)
	return rr
}

type failureSet uint

const (
	failRoom failureSet = 1 << iota
	failMatch
	failMissingAuth
	failInvalidToken
	failWrongTeam
)

type failureRoute struct {
	handler   func(*Handler, http.ResponseWriter, *http.Request)
	pattern   string
	query     string
	body      any
	cases     failureSet
	matchGone func(room *MatchRoom)
	bareRoom  bool
}

func testRouteFailures(t *testing.T, route failureRoute) {
	t.Helper()
	method, routePath, _ := strings.Cut(route.pattern, " ")
	body := ""
	if route.body != nil {
		jsonBody, err := json.Marshal(route.body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		body = string(jsonBody)
	}
	authed := route.cases&(failMissingAuth|failInvalidToken|failWrongTeam) != 0

	call := func(t *testing.T, h *Handler, roomID, token string) *httptest.ResponseRecorder {
		t.Helper()
		handler := func(w http.ResponseWriter, r *http.Request) { route.handler(h, w, r) }
		path := strings.Replace(routePath, "{roomID}", roomID, 1) + route.query
		return doRequest(t, handler, route.pattern, method, path, body, token)
	}
	assertFailure := func(t *testing.T, rr *httptest.ResponseRecorder, wantStatus int, wantMsg string) {
		t.Helper()
		if status := rr.Code; status != wantStatus {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, wantStatus)
		}
		if !strings.Contains(rr.Body.String(), wantMsg) {
			t.Errorf("Expected error message '%s', got: %s", wantMsg, rr.Body.String())
		}
	}
	assertInvalidToken := func(t *testing.T, token func(tokens [2]string) string) {
		t.Helper()
		roomID, tokens, _, h := createTestRoomWithMatch(t)
		assertFailure(t, call(t, h, roomID, token(tokens)), http.StatusUnauthorized, "invalid player token")
	}

	if route.cases&failRoom != 0 {
		t.Run("Failure: room not found", func(t *testing.T) {
			token := ""
			if authed {
				token = "dummy-token"
			}
			h := NewHandler(NewServerStateManager())
			assertFailure(t, call(t, h, "NONEXISTENT", token), http.StatusNotFound, "room not found")
		})
	}
	if route.cases&failMatch != 0 {
		t.Run("Failure: match not found", func(t *testing.T) {
			if route.bareRoom {
				roomID, _, h := newTestRoom(t)
				assertFailure(t, call(t, h, roomID, ""), http.StatusNotFound, "match not found")
				return
			}
			roomID, tokens, s, h := createTestRoomWithMatch(t)
			room := mustRoom(t, s, roomID)
			if route.matchGone != nil {
				route.matchGone(room)
			} else {
				room.Match = nil
			}
			assertFailure(t, call(t, h, roomID, tokens[0]), http.StatusNotFound, "match not found")
		})
	}
	if route.cases&failMissingAuth != 0 {
		t.Run("Failure: missing Authorization header", func(t *testing.T) {
			assertInvalidToken(t, func([2]string) string { return "" })
		})
	}
	if route.cases&failInvalidToken != 0 {
		t.Run("Failure: invalid token", func(t *testing.T) {
			assertInvalidToken(t, func([2]string) string { return "invalid-token" })
		})
	}
	if route.cases&failWrongTeam != 0 {
		t.Run("Failure: wrong team token", func(t *testing.T) {
			assertInvalidToken(t, func(tokens [2]string) string { return tokens[1] })
		})
	}
}

func assertKeys(t *testing.T, obj map[string]any, fields []string, format string) {
	t.Helper()
	for _, field := range fields {
		if _, exists := obj[field]; !exists {
			t.Errorf(format, field)
		}
	}
}

func assertMatchStateNested(t *testing.T, raw map[string]any) {
	t.Helper()
	row, ok := firstElement(raw["grid"])
	if !ok {
		t.Errorf("grid is empty or not an array: %#v", raw["grid"])
	} else if tile, ok := firstObject(row); !ok {
		t.Errorf("grid[0] is empty or not an array of objects: %#v", row)
	} else {
		assertKeys(t, tile, []string{"type", "occupantType", "occupantId"}, "grid[0][0] missing key '%s'")
	}

	if sb, ok := firstObject(raw["softBlocks"]); !ok {
		t.Errorf("softBlocks is empty or not an array of objects: %#v", raw["softBlocks"])
	} else {
		assertKeys(t, sb, []string{"id", "position"}, "softBlock missing key '%s'")
		if coord, ok := sb["position"].(map[string]any); !ok {
			t.Errorf("softBlock.position is not an object: %#v", sb["position"])
		} else {
			assertKeys(t, coord, []string{"x", "y"}, "softBlock.position missing key '%s'")
		}
	}

	if unit, ok := firstObject(raw["units"]); !ok {
		t.Errorf("units is empty or not an array of objects: %#v", raw["units"])
	} else {
		unitFields := []string{"id", "type", "position", "speed", "bombMaxRange", "bombPower", "maxBombCount", "bombUsed", "team", "hp", "skills", "hasMoved", "hasUsedSkill"}
		assertKeys(t, unit, unitFields, "Contract Broken: unit missing key '%s'")
		if pos, ok := unit["position"].(map[string]any); !ok {
			t.Errorf("unit.position is not an object: %#v", unit["position"])
		} else {
			assertKeys(t, pos, []string{"x", "y"}, "unit.position missing key '%s'")
		}
	}

	if bomb, ok := firstObject(raw["bombs"]); !ok {
		t.Errorf("bombs is empty or not an array of objects: %#v", raw["bombs"])
	} else {
		assertKeys(t, bomb, []string{"id", "ownerId", "position", "range", "countdown"}, "Contract Broken: bomb missing key '%s'")
	}

	if tc, ok := firstObject(raw["turnCommands"]); !ok {
		t.Errorf("turnCommands is empty or not an array of objects: %#v", raw["turnCommands"])
	} else {
		assertKeys(t, tc, []string{"type", "unitId", "target"}, "Contract Broken: turnCommand missing key '%s'")
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

func TestHandler_extractBearerToken(t *testing.T) {
	tests := []struct {
		name      string
		authValue string
		omitAuth  bool
		wantToken string
		wantErr   error
	}{
		{"Missing header", "", true, "", ErrInvalidToken},
		{"Missing Bearer prefix", "abc123", false, "", ErrInvalidToken},
		{"Valid Bearer token", "Bearer abc123", false, "abc123", nil},
	}

	h := NewHandler(NewServerStateManager())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if !tt.omitAuth {
				r.Header.Set("Authorization", tt.authValue)
			}

			token, err := h.extractBearerToken(r)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("extractBearerToken() error = %v, want %v", err, tt.wantErr)
			}
			if token != tt.wantToken {
				t.Errorf("extractBearerToken() token = %q, want %q", token, tt.wantToken)
			}
		})
	}
}

func TestHandleHealthCheck(t *testing.T) {
	h := NewHandler(NewServerStateManager())

	t.Run("Success: returns ok status", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/health", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()

		http.HandlerFunc(h.HandleHealthCheck).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response HealthCheckResponse
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		if got, want := response.Status, "ok"; got != want {
			t.Errorf("Handler returned unexpected status: got %v want %v", got, want)
		}
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		testEncodeFailure(t, http.HandlerFunc(h.HandleHealthCheck),
			func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/health", nil)
				return req
			}, http.StatusOK)
	})

	t.Run("Test Contract", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/health", nil)
		rr := httptest.NewRecorder()

		http.HandlerFunc(h.HandleHealthCheck).ServeHTTP(rr, req)

		assertObjectContract(t, rr.Body.Bytes(), []string{"status"}, nil)
	})
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

		var response struct {
			Archetypes   []jsontext.Value `json:"archetypes"`
			StagePresets []jsontext.Value `json:"stagePresets"`
		}
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
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

// ClientUnit mimics the client's view of Unit in GetMatchState response.
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

// ClientTile mimics the client's view of Tile in GetMatchState response.
type ClientTile struct {
	Type         string `json:"type"`
	OccupantType string `json:"occupantType"`
	OccupantID   int64  `json:"occupantId"`
}

// ClientMatchStateResponse mimics the client's view of GetMatchState response.
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
		roomID, s, h := newTestRoom(t)

		gameCfg := engine.GameCfg{
			StagePreset: "Divided",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		mustCreateMatch(t, s, roomID, gameCfg)

		rr := doRequest(t, h.HandleGetMatchState, "GET /api/match-rooms/{roomID}/match/state", "GET", "/api/match-rooms/"+roomID+"/match/state", "", "")

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response ClientMatchStateResponse
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
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

	testRouteFailures(t, failureRoute{
		handler:  (*Handler).HandleGetMatchState,
		pattern:  "GET /api/match-rooms/{roomID}/match/state",
		cases:    failRoom | failMatch,
		bareRoom: true,
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		roomID, s, h := newTestRoom(t)

		gameCfg := validGameCfg()
		mustCreateMatch(t, s, roomID, gameCfg)

		testEncodeFailure(t, testMux("GET /api/match-rooms/{roomID}/match/state", h.HandleGetMatchState),
			func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/state", nil)
				return req
			}, http.StatusOK)
	})

	t.Run("Test Contract", func(t *testing.T) {
		roomID, s, h := newTestRoom(t)

		gameCfg := engine.GameCfg{
			StagePreset: "Divided",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		tokens := mustCreateMatch(t, s, roomID, gameCfg)
		unitID := engine.NewUnitID(1, 0)
		pos := engine.Coordinate{X: 4, Y: 7}
		if _, _, err := s.StartTurn(roomID, tokens[0]); err != nil {
			t.Fatalf("StartTurn() error = %v", err)
		}
		if _, err := s.SubmitTurnCommand(roomID, engine.NewPlaceBombCommand(unitID, pos), tokens[0]); err != nil {
			t.Fatalf("SubmitTurnCommand(place bomb) error = %v", err)
		}
		if _, _, err := s.ResolveTurn(roomID, tokens[0]); err != nil {
			t.Fatalf("ResolveTurn() error = %v", err)
		}
		room := mustRoom(t, s, roomID)
		room.mu.Lock()
		room.Match.WorkingState.TurnCommands = []engine.TurnCommand{engine.NewMoveCommand(unitID, pos)}
		room.mu.Unlock()

		rr := doRequest(t, h.HandleGetMatchState, "GET /api/match-rooms/{roomID}/match/state", "GET", "/api/match-rooms/"+roomID+"/match/state", "", "")

		assertObjectContract(t, rr.Body.Bytes(),
			[]string{"turn", "inSuddenDeath", "activeTeam", "grid", "units", "bombs", "softBlocks", "turnCommands"},
			assertMatchStateNested)
	})
}

func newTestRoom(t *testing.T) (string, *ServerStateManager, *Handler) {
	t.Helper()
	s := NewServerStateManager()
	h := NewHandler(s)
	roomID, err := s.CreateMatchRoom()
	if err != nil {
		t.Fatalf("Failed to create room: %v", err)
	}
	return roomID, s, h
}

func createTestRoomWithMatch(t *testing.T) (string, [2]string, *ServerStateManager, *Handler) {
	t.Helper()

	roomID, s, h := newTestRoom(t)

	playerTokens := mustCreateMatch(t, s, roomID, validGameCfg())

	return roomID, playerTokens, s, h
}

func TestHandleSubmitTurnCommand(t *testing.T) {
	t.Run("Success: submit a valid TurnCommand in an existing room", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)

		unitID := engine.NewUnitID(1, 0)
		newPos := engine.Coordinate{X: 4, Y: 7}
		jsonBody, _ := json.Marshal(engine.NewMoveCommand(unitID, newPos))

		rr := doRequest(t, h.HandleSubmitTurnCommand, "POST /api/match-rooms/{roomID}/match/turn-commands", "POST", "/api/match-rooms/"+roomID+"/match/turn-commands", string(jsonBody), playerTokens[0])

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response []engine.GameEvent
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
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

		rr := doRequest(t, h.HandleSubmitTurnCommand, "POST /api/match-rooms/{roomID}/match/turn-commands", "POST", "/api/match-rooms/"+roomID+"/match/turn-commands", string(jsonBody), playerTokens[0])

		if status := rr.Code; status != http.StatusConflict {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusConflict)
		}
		if !strings.Contains(rr.Body.String(), "invalid turn command") {
			t.Errorf("Expected error message 'invalid turn command', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid JSON format", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		rr := doRequest(t, h.HandleSubmitTurnCommand, "POST /api/match-rooms/{roomID}/match/turn-commands", "POST", "/api/match-rooms/"+roomID+"/match/turn-commands", "{invalid json", playerTokens[0])

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "Invalid turnCommand format") {
			t.Errorf("Expected error message 'Invalid turnCommand format', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: unitId overflows UnitID", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		body := `{"type":"move","unitId":300,"target":{"x":4,"y":7}}`
		req := httptest.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/turn-commands", strings.NewReader(body))
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

	testRouteFailures(t, failureRoute{
		handler: (*Handler).HandleSubmitTurnCommand,
		pattern: "POST /api/match-rooms/{roomID}/match/turn-commands",
		body:    engine.NewMoveCommand(engine.NewUnitID(1, 0), engine.Coordinate{X: 4, Y: 7}),
		cases:   failRoom | failMatch | failMissingAuth | failInvalidToken | failWrongTeam,
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
		rr := doRequest(t, h.HandleSubmitTurnCommand, "POST /api/match-rooms/{roomID}/match/turn-commands", "POST", "/api/match-rooms/"+roomID+"/match/turn-commands", string(jsonBody), playerTokens[0])

		assertArrayContract(t, rr.Body.Bytes(),
			[]string{"type", "unitId", "from", "to",
				"countdown", "newHp"},
			func(t *testing.T, item map[string]any) {
				t.Helper()
				fromField, ok := item["from"].(map[string]any)
				if !ok {
					t.Fatalf("from is not an object: %#v", item["from"])
				}
				toField, ok := item["to"].(map[string]any)
				if !ok {
					t.Fatalf("to is not an object: %#v", item["to"])
				}
				assertKeys(t, fromField, []string{"x", "y"}, "Contract Broken: from missing key '%s'")
				assertKeys(t, toField, []string{"x", "y"}, "Contract Broken: to missing key '%s'")
			})
	})
}

func TestHandleStartTurn(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		room := mustRoom(t, s, roomID)
		room.Match.TrueState.Turn = 1000
		room.Match.WorkingState.Turn = 1000

		rr := doRequest(t, h.HandleStartTurn, "POST /api/match-rooms/{roomID}/match/start-turn", "POST", "/api/match-rooms/"+roomID+"/match/start-turn", "", playerTokens[0])

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response StartTurnResponse
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
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

	testRouteFailures(t, failureRoute{
		handler: (*Handler).HandleStartTurn,
		pattern: "POST /api/match-rooms/{roomID}/match/start-turn",
		cases:   failRoom | failMatch | failMissingAuth | failInvalidToken | failWrongTeam,
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

		rr := doRequest(t, h.HandleStartTurn, "POST /api/match-rooms/{roomID}/match/start-turn", "POST", "/api/match-rooms/"+roomID+"/match/start-turn", "", playerTokens[0])

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
						"newHp"},
					func(t *testing.T, item map[string]any) {
						t.Helper()
						positionField, ok := item["position"].(map[string]any)
						if !ok {
							t.Fatalf("position is not an object: %#v", item["position"])
						}
						assertKeys(t, positionField, []string{"x", "y"}, "Contract Broken: position missing key '%s'")
					})
			})
	})
}

func TestHandleConsumeCPUStatus(t *testing.T) {
	unitID := engine.NewUnitID(1, 0)

	t.Run("Success: TurnPhaseReady returns and clears pending events", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		room := mustRoom(t, s, roomID)
		room.Match.CPU.Phase = engine.TurnPhaseReady
		room.Match.CPU.PlanGameEvents = []engine.GameEvent{
			engine.NewUnitMovedEvent(unitID, engine.Coordinate{X: 1, Y: 2}, engine.Coordinate{X: 2, Y: 2}),
		}
		room.Match.CPU.ResolveTurnGameEvents = []engine.GameEvent{
			engine.NewUnitDiedEvent(unitID, engine.Coordinate{X: 2, Y: 2}),
		}

		rr := doRequest(t, h.HandleConsumeCPUStatus, "POST /api/match-rooms/{roomID}/match/cpu-status/consume", "POST", "/api/match-rooms/"+roomID+"/match/cpu-status/consume", "", playerTokens[0])

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response struct {
			TurnPhase             string             `json:"turnPhase"`
			PlanGameEvents        []engine.GameEvent `json:"planGameEvents"`
			ResolveTurnGameEvents []engine.GameEvent `json:"resolveTurnGameEvents"`
		}
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		if got, want := response.TurnPhase, engine.TurnPhaseReady.String(); got != want {
			t.Errorf("Expected turnPhase %v, got %v", want, got)
		}
		if got, want := len(response.PlanGameEvents), 1; got != want {
			t.Errorf("Expected %d planGameEvents returned, got %#v", want, response.PlanGameEvents)
		}
		if got, want := len(response.ResolveTurnGameEvents), 1; got != want {
			t.Errorf("Expected %d resolveTurnGameEvents returned, got %#v", want, response.ResolveTurnGameEvents)
		}

		if got, want := room.Match.CPU.Phase, engine.TurnPhaseIdle; got != want {
			t.Errorf("Expected CPU.Phase reset to %v, got %v", want, got)
		}
		if got, want := len(room.Match.CPU.PlanGameEvents), 0; got != want {
			t.Errorf("Expected CPU.PlanGameEvents cleared, got %#v", room.Match.CPU.PlanGameEvents)
		}
		if got, want := len(room.Match.CPU.ResolveTurnGameEvents), 0; got != want {
			t.Errorf("Expected CPU.ResolveTurnGameEvents cleared, got %#v", room.Match.CPU.ResolveTurnGameEvents)
		}
	})

	t.Run("Success: TurnPhasePlanning leaves state untouched", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		room := mustRoom(t, s, roomID)
		room.Match.CPU.Phase = engine.TurnPhasePlanning

		rr := doRequest(t, h.HandleConsumeCPUStatus, "POST /api/match-rooms/{roomID}/match/cpu-status/consume", "POST", "/api/match-rooms/"+roomID+"/match/cpu-status/consume", "", playerTokens[0])

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		body := rr.Body.Bytes()

		var response struct {
			TurnPhase             string             `json:"turnPhase"`
			PlanGameEvents        []engine.GameEvent `json:"planGameEvents"`
			ResolveTurnGameEvents []engine.GameEvent `json:"resolveTurnGameEvents"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		if got, want := response.TurnPhase, engine.TurnPhasePlanning.String(); got != want {
			t.Errorf("Expected turnPhase %v, got %v", want, got)
		}
		if got, want := len(response.PlanGameEvents), 0; got != want {
			t.Errorf("Expected no planGameEvents, got %#v", response.PlanGameEvents)
		}
		if got, want := len(response.ResolveTurnGameEvents), 0; got != want {
			t.Errorf("Expected no resolveTurnGameEvents, got %#v", response.ResolveTurnGameEvents)
		}
		if got, want := room.Match.CPU.Phase, engine.TurnPhasePlanning; got != want {
			t.Errorf("Expected CPU.Phase to stay at %v, got %v", want, got)
		}

		var raw map[string]any
		if err := json.Unmarshal(body, &raw); err != nil {
			t.Fatalf("Failed to parse raw JSON: %v", err)
		}
		for _, field := range []string{"planGameEvents", "resolveTurnGameEvents"} {
			if _, ok := raw[field].([]any); !ok {
				t.Errorf("Expected %s to serialize as [], got %#v (client types this field as non-nullable)", field, raw[field])
			}
		}
	})

	t.Run("Success: either player's token is accepted", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		room := mustRoom(t, s, roomID)
		room.Match.CPU.Phase = engine.TurnPhaseReady

		rr := doRequest(t, h.HandleConsumeCPUStatus, "POST /api/match-rooms/{roomID}/match/cpu-status/consume", "POST", "/api/match-rooms/"+roomID+"/match/cpu-status/consume", "", playerTokens[1])

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})

	testRouteFailures(t, failureRoute{
		handler: (*Handler).HandleConsumeCPUStatus,
		pattern: "POST /api/match-rooms/{roomID}/match/cpu-status/consume",
		cases:   failRoom | failMatch | failMissingAuth | failInvalidToken,
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
		room.Match.CPU.PlanGameEvents = []engine.GameEvent{
			engine.NewUnitMovedEvent(unitID, engine.Coordinate{X: 1, Y: 2}, engine.Coordinate{X: 2, Y: 2}),
		}
		room.Match.CPU.ResolveTurnGameEvents = []engine.GameEvent{
			engine.NewUnitDiedEvent(unitID, engine.Coordinate{X: 2, Y: 2}),
		}

		rr := doRequest(t, h.HandleConsumeCPUStatus, "POST /api/match-rooms/{roomID}/match/cpu-status/consume", "POST", "/api/match-rooms/"+roomID+"/match/cpu-status/consume", "", playerTokens[0])

		assertObjectContract(t, rr.Body.Bytes(), []string{"turnPhase", "planGameEvents", "resolveTurnGameEvents"},
			func(t *testing.T, raw map[string]any) {
				t.Helper()
				events, ok := raw["planGameEvents"].([]any)
				if !ok || len(events) == 0 {
					t.Fatalf("Expected non-empty planGameEvents, got %#v", raw["planGameEvents"])
				}
				evt, ok := events[0].(map[string]any)
				if !ok {
					t.Fatalf("planGameEvents[0] is not an object: %#v", events[0])
				}
				assertKeys(t, evt, []string{"type", "unitId", "from", "to"}, "Contract Broken: planGameEvents[0] missing key '%s'")
				if _, ok := raw["resolveTurnGameEvents"].([]any); !ok {
					t.Errorf("Expected resolveTurnGameEvents array, got %#v", raw["resolveTurnGameEvents"])
				}
			})
	})
}

func TestHandleResetTurn(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		unitID := engine.NewUnitID(1, 0)
		room := mustRoom(t, s, roomID)
		room.Match.WorkingState.Units[unitID].HasMoved = true

		rr := doRequest(t, h.HandleResetTurn, "POST /api/match-rooms/{roomID}/match/reset", "POST", "/api/match-rooms/"+roomID+"/match/reset", "", playerTokens[0])

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

	testRouteFailures(t, failureRoute{
		handler: (*Handler).HandleResetTurn,
		pattern: "POST /api/match-rooms/{roomID}/match/reset",
		cases:   failRoom | failMatch | failMissingAuth | failInvalidToken | failWrongTeam,
	})
}

func TestHandleResolveTurn(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		unitID := engine.NewUnitID(1, 0)
		if _, err := s.SubmitTurnCommand(roomID, engine.NewPlaceBombCommand(unitID, engine.Coordinate{X: 4, Y: 7}), playerTokens[0]); err != nil {
			t.Fatalf("SubmitTurnCommand() error = %v", err)
		}
		room := mustRoom(t, s, roomID)
		room.Match.WorkingState.Bombs[engine.NewBombID(1, 1, unitID)].Countdown = 1

		rr := doRequest(t, h.HandleResolveTurn, "POST /api/match-rooms/{roomID}/match/resolve", "POST", "/api/match-rooms/"+roomID+"/match/resolve", "", playerTokens[0])

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response ResolveTurnResponse
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		if got, want := response.PlanGameEvents, 1; len(got) != want {
			t.Errorf("Expected %d PlanGameEvents returned, got %#v", want, got)
		}
		if got, want := response.ResolveTurnGameEvents, 5; len(got) != want {
			t.Errorf("Expected %d ResolveTurnGameEvents returned, got %#v", want, got)
		}
		if got, want := room.Match.WorkingState.Units[unitID].HP, 0; got != want {
			t.Errorf("Expected Unit %#X HP %v, got %v", 16, want, got)
		}
		if got, want := room.Match.WinnerTeamID, 2; got != want {
			t.Errorf("Expected match winner = %v, got %v", want, got)
		}
	})

	testRouteFailures(t, failureRoute{
		handler: (*Handler).HandleResolveTurn,
		pattern: "POST /api/match-rooms/{roomID}/match/resolve",
		cases:   failRoom | failMatch | failMissingAuth | failInvalidToken | failWrongTeam,
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
		if _, err := s.SubmitTurnCommand(roomID, engine.NewPlaceBombCommand(unitID, engine.Coordinate{X: 4, Y: 7}), playerTokens[0]); err != nil {
			t.Fatalf("SubmitTurnCommand() error = %v", err)
		}

		rr := doRequest(t, h.HandleResolveTurn, "POST /api/match-rooms/{roomID}/match/resolve", "POST", "/api/match-rooms/"+roomID+"/match/resolve", "", playerTokens[0])

		assertObjectContract(t, rr.Body.Bytes(), []string{"planGameEvents", "resolveTurnGameEvents"},
			func(t *testing.T, raw map[string]any) {
				t.Helper()
				events, ok := raw["planGameEvents"].([]any)
				if !ok || len(events) == 0 {
					t.Fatalf("Expected non-empty planGameEvents, got %#v", raw["planGameEvents"])
				}
				evt, ok := events[0].(map[string]any)
				if !ok {
					t.Fatalf("planGameEvents[0] is not an object: %#v", events[0])
				}
				assertKeys(t, evt, []string{"type", "unitId", "bombId", "position", "range", "countdown"}, "Contract Broken: planGameEvents[0] missing key '%s'")
				if _, ok := raw["resolveTurnGameEvents"].([]any); !ok {
					t.Errorf("Expected resolveTurnGameEvents array, got %#v", raw["resolveTurnGameEvents"])
				}
			})

	})
}

func TestHandleSurrender(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)

		jsonBody, _ := json.Marshal(SurrenderRequest{TeamID: 1})
		rr := doRequest(t, h.HandleSurrender, "POST /api/match-rooms/{roomID}/match/surrender", "POST", "/api/match-rooms/"+roomID+"/match/surrender", string(jsonBody), playerTokens[0])

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response []engine.GameEvent
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		if got, want := response, 1; len(got) != want {
			t.Errorf("Expected %d gameEvents returned, got %#v", want, got)
		}
		if got, want := response[0].WinnerTeamID, 2; got != want {
			t.Errorf("Expected gameEvent WinnerTeamID = %v, got %v", want, got)
		}
		if _, ok := s.Rooms.Load(roomID); !ok {
			t.Error("Expected room not to be deleted after surrender")
		}
	})

	t.Run("Failure: invalid SurrenderRequest", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		jsonBody, _ := json.Marshal(SurrenderRequest{TeamID: 3})
		rr := doRequest(t, h.HandleSurrender, "POST /api/match-rooms/{roomID}/match/surrender", "POST", "/api/match-rooms/"+roomID+"/match/surrender", string(jsonBody), playerTokens[0])

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "invalid game config") {
			t.Errorf("Expected error message 'invalid game config', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid JSON format", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		rr := doRequest(t, h.HandleSurrender, "POST /api/match-rooms/{roomID}/match/surrender", "POST", "/api/match-rooms/"+roomID+"/match/surrender", "{invalid json}", playerTokens[0])

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "Invalid surrenderRequest format") {
			t.Errorf("Expected error message 'Invalid surrenderRequest format', got: %s", rr.Body.String())
		}
	})

	testRouteFailures(t, failureRoute{
		handler: (*Handler).HandleSurrender,
		pattern: "POST /api/match-rooms/{roomID}/match/surrender",
		body:    SurrenderRequest{TeamID: 1},
		cases:   failRoom | failMatch | failMissingAuth | failInvalidToken | failWrongTeam,
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
		if _, err := s.SubmitTurnCommand(roomID, engine.NewPlaceBombCommand(unitID, engine.Coordinate{X: 4, Y: 7}), playerTokens[0]); err != nil {
			t.Fatalf("SubmitTurnCommand() error = %v", err)
		}

		jsonBody, _ := json.Marshal(SurrenderRequest{TeamID: 1})
		rr := doRequest(t, h.HandleSurrender, "POST /api/match-rooms/{roomID}/match/surrender", "POST", "/api/match-rooms/"+roomID+"/match/surrender", string(jsonBody), playerTokens[0])

		assertArrayContract(t, rr.Body.Bytes(), []string{"type", "winnerTeamId",
			"unitId", "countdown", "newHp"},
			nil)
	})
}

func TestHandleGetMatchConfig(t *testing.T) {
	t.Run("Success: get the gameConfig in an existing room", func(t *testing.T) {
		roomID, s, h := newTestRoom(t)

		gameCfg := engine.GameCfg{
			StagePreset: "Divided",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		mustCreateMatch(t, s, roomID, gameCfg)

		rr := doRequest(t, h.HandleGetMatchConfig, "GET /api/match-rooms/{roomID}/match/config", "GET", "/api/match-rooms/"+roomID+"/match/config", "", "")

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response engine.GameCfg
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		if response.StagePreset != gameCfg.StagePreset {
			t.Errorf("Expected stagePreset %v, got %v", validGameCfg().StagePreset, gameCfg.StagePreset)
		}
	})

	testRouteFailures(t, failureRoute{
		handler:  (*Handler).HandleGetMatchConfig,
		pattern:  "GET /api/match-rooms/{roomID}/match/config",
		cases:    failRoom | failMatch,
		bareRoom: true,
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		roomID, s, h := newTestRoom(t)

		gameCfg := validGameCfg()
		mustCreateMatch(t, s, roomID, gameCfg)

		testEncodeFailure(t, testMux("GET /api/match-rooms/{roomID}/match/config", h.HandleGetMatchConfig),
			func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/config", nil)
				return req
			}, http.StatusOK)
	})

	t.Run("Test Contract", func(t *testing.T) {
		roomID, s, h := newTestRoom(t)

		gameCfg := engine.GameCfg{
			StagePreset: "Divided",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		mustCreateMatch(t, s, roomID, gameCfg)

		rr := doRequest(t, h.HandleGetMatchConfig, "GET /api/match-rooms/{roomID}/match/config", "GET", "/api/match-rooms/"+roomID+"/match/config", "", "")

		assertObjectContract(t, rr.Body.Bytes(),
			[]string{"vsCpu", "stagePreset", "p1Slots", "p2Slots", "maxTurns", "allowResetTurn"}, nil)
	})
}

func TestHandleGetAllowedTiles(t *testing.T) {
	t.Run("Success: get the allowedTiles in an existing room", func(t *testing.T) {
		roomID, s, h := newTestRoom(t)

		gameCfg := engine.GameCfg{
			StagePreset: "Divided",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		mustCreateMatch(t, s, roomID, gameCfg)

		rr := doRequest(t, h.HandleGetAllowedTiles, "GET /api/match-rooms/{roomID}/match/allowed-tiles", "GET", "/api/match-rooms/"+roomID+"/match/allowed-tiles?unitId=16&turnCmdType=placeBomb", "", "")

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response []engine.Coordinate
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
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
		roomID, s, h := newTestRoom(t)
		mustCreateMatch(t, s, roomID, validGameCfg())

		// Unit 0x10 spawns at {4,8} with an ally on {3,8}; soft blocks on {5,8} and {4,7}
		// plus the stage edge below leave it no legal landing tile.
		gs := mustRoom(t, s, roomID).Match.WorkingState
		gs.UpdateStageOccupant(engine.Coordinate{X: 5, Y: 8}, engine.OccupantSoftBlock, 1)
		gs.UpdateStageOccupant(engine.Coordinate{X: 4, Y: 7}, engine.OccupantSoftBlock, 2)

		rr := doRequest(t, h.HandleGetAllowedTiles, "GET /api/match-rooms/{roomID}/match/allowed-tiles", "GET", "/api/match-rooms/"+roomID+"/match/allowed-tiles?unitId=16&turnCmdType=move", "", "")

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		if got := strings.TrimSpace(rr.Body.String()); got != "[]" {
			t.Errorf("Expected an empty JSON array, got %q", got)
		}
	})

	t.Run("Failure: missing query string", func(t *testing.T) {
		roomID, s, h := newTestRoom(t)

		gameCfg := engine.GameCfg{
			StagePreset: "Divided",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		mustCreateMatch(t, s, roomID, gameCfg)

		rr := doRequest(t, h.HandleGetAllowedTiles, "GET /api/match-rooms/{roomID}/match/allowed-tiles", "GET", "/api/match-rooms/"+roomID+"/match/allowed-tiles", "", "")

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "missing required query parameters") {
			t.Errorf("Expected error message 'missing required query parameters', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid query string", func(t *testing.T) {
		roomID, s, h := newTestRoom(t)

		gameCfg := engine.GameCfg{
			StagePreset: "Divided",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		mustCreateMatch(t, s, roomID, gameCfg)

		rr := doRequest(t, h.HandleGetAllowedTiles, "GET /api/match-rooms/{roomID}/match/allowed-tiles", "GET", "/api/match-rooms/"+roomID+"/match/allowed-tiles?unitId=abc&turnCmdType=move", "", "")

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "Invalid unitId parameter") {
			t.Errorf("Expected error message 'Invalid unitId parameter', got: %s", rr.Body.String())
		}
	})

	testRouteFailures(t, failureRoute{
		handler:  (*Handler).HandleGetAllowedTiles,
		pattern:  "GET /api/match-rooms/{roomID}/match/allowed-tiles",
		query:    "?unitId=16&turnCmdType=placeBomb",
		cases:    failRoom | failMatch,
		bareRoom: true,
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		roomID, s, h := newTestRoom(t)

		gameCfg := validGameCfg()
		mustCreateMatch(t, s, roomID, gameCfg)

		testEncodeFailure(t, testMux("GET /api/match-rooms/{roomID}/match/allowed-tiles", h.HandleGetAllowedTiles),
			func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/allowed-tiles?unitId=16&turnCmdType="+string(engine.TurnCmdPlaceBomb), nil)
				return req
			}, http.StatusOK)
	})

	t.Run("Test Contract", func(t *testing.T) {
		roomID, s, h := newTestRoom(t)

		gameCfg := engine.GameCfg{
			StagePreset: "Divided",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		mustCreateMatch(t, s, roomID, gameCfg)

		rr := doRequest(t, h.HandleGetAllowedTiles, "GET /api/match-rooms/{roomID}/match/allowed-tiles", "GET", "/api/match-rooms/"+roomID+"/match/allowed-tiles?unitId=16&turnCmdType=placeBomb", "", "")

		assertArrayContract(t, rr.Body.Bytes(),
			[]string{"x", "y"}, nil)
	})
}
