package server

import (
	"bomb-srpg/engine"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestHandleGetAllArchetypes(t *testing.T) {
	s := NewServerStateManager()

	t.Run("Success: called engine.GetAllArchetypes", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/archetypes", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()

		http.HandlerFunc(s.HandleGetAllArchetypes).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response []engine.Archetype
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		expectedCount := len(engine.GetAllArchetypes())
		if len(response) != expectedCount {
			t.Errorf("Handler returned unexpected number of archetypes: got %d want %d", len(response), expectedCount)
		}
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		testEncodeFailure(t, http.HandlerFunc(s.HandleGetAllArchetypes),
			func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/archetypes", nil)
				return req
			}, http.StatusOK)
	})

	t.Run("Test Contract", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/archetypes", nil)
		rr := httptest.NewRecorder()

		http.HandlerFunc(s.HandleGetAllArchetypes).ServeHTTP(rr, req)

		assertArrayContract(t, rr.Body.Bytes(), []string{"name", "speed", "bombMaxRange", "skills"},
			func(t *testing.T, item map[string]any) {
				t.Helper()
				for _, field := range []string{"name", "speed", "bombMaxRange", "skills"} {
					if _, exists := item[field]; !exists {
						t.Errorf("Contract Broken: archetype missing key '%s'", field)
					}
				}
			})
	})
}

func TestHandleCreateMatchRoom(t *testing.T) {
	s := NewServerStateManager()

	t.Run("Success: called server.CreateMatchRoom", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/api/match-rooms", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()

		http.HandlerFunc(s.HandleCreateMatchRoom).ServeHTTP(rr, req)

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

		if len(response.ID) != 5 {
			t.Errorf("Handler returned unexpected Match Room ID: got %v want length of 5", response.ID)
		}
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		testEncodeFailure(t, http.HandlerFunc(s.HandleCreateMatchRoom),
			func() *http.Request {
				req, _ := http.NewRequest("POST", "/api/match-rooms", nil)
				return req
			}, http.StatusCreated)
	})

	t.Run("Test Contract", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/match-rooms", nil)
		rr := httptest.NewRecorder()

		http.HandlerFunc(s.HandleCreateMatchRoom).ServeHTTP(rr, req)

		assertObjectContract(t, rr.Body.Bytes(), []string{"id"}, nil)
	})

	t.Run("Failure: CreateMatchRoom exhausted retries", func(t *testing.T) {
		s := NewServerStateManager()

		roomIDs := []string{"ID001", "ID002", "ID003", "ID004", "ID005"}
		for _, id := range roomIDs {
			s.Rooms[id] = &MatchRoom{ID: id}
		}
		callCount := 0
		s.generateRoomID = func(int) string {
			if callCount < len(roomIDs) {
				id := roomIDs[callCount]
				callCount++
				return id
			}
			return "SHOULD_NOT_REACH"
		}

		req, _ := http.NewRequest("POST", "/api/match-rooms", nil)
		rr := httptest.NewRecorder()

		http.HandlerFunc(s.HandleCreateMatchRoom).ServeHTTP(rr, req)

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
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		gameCfg := engine.GameCfg{
			StagePreset: "MAP01",
			P1Teams:     []string{"King", "Fighter"},
			P2Teams:     []string{"King", "Witch"},
			MaxTurns:    10,
		}
		jsonBody, _ := json.Marshal(CreateMatchRequest{GameCfg: gameCfg})
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match", s.HandleCreateMatch).ServeHTTP(rr, req)

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
		room, ok := s.Rooms[roomID]
		if !ok || room.Match == nil {
			t.Error("Match was not created in the server state manager")
		}
	})

	t.Run("Failure: room not found", func(t *testing.T) {
		s := NewServerStateManager()
		gameCfg := engine.GameCfg{
			StagePreset: "MAP01",
			P1Teams:     []string{"King", "Fighter"},
			P2Teams:     []string{"King", "Witch"},
			MaxTurns:    10,
		}
		jsonBody, _ := json.Marshal(gameCfg)
		req, err := http.NewRequest("POST", "/api/match-rooms/NONEXISTENT/match", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match", s.HandleCreateMatch).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "room not found") {
			t.Errorf("Expected error message 'room not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: match already exists", func(t *testing.T) {
		s := NewServerStateManager()
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}
		s.Rooms[roomID].Match = &engine.Match{}

		gameCfg := engine.GameCfg{
			StagePreset: "MAP01",
			P1Teams:     []string{"King", "Fighter"},
			P2Teams:     []string{"King", "Witch"},
			MaxTurns:    10,
		}
		jsonBody, _ := json.Marshal(CreateMatchRequest{GameCfg: gameCfg})
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match", s.HandleCreateMatch).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusConflict {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusConflict)
		}
		if !strings.Contains(rr.Body.String(), "match already exists") {
			t.Errorf("Expected error message 'match already exists', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid game config", func(t *testing.T) {
		s := NewServerStateManager()
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
		testMux("POST /api/match-rooms/{roomID}/match", s.HandleCreateMatch).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "invalid game config") {
			t.Errorf("Expected error message 'invalid game config', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid JSON format", func(t *testing.T) {
		s := NewServerStateManager()
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
		testMux("POST /api/match-rooms/{roomID}/match", s.HandleCreateMatch).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "Invalid configuration format") {
			t.Errorf("Expected error message 'Invalid configuration format', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: failed to Encode response", func(t *testing.T) {
		s := NewServerStateManager()
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		gameCfg := engine.GameCfg{
			StagePreset: "MAP01",
			P1Teams:     []string{"King", "Fighter"},
			P2Teams:     []string{"King", "Witch"},
			MaxTurns:    10,
		}
		jsonBody, _ := json.Marshal(CreateMatchRequest{GameCfg: gameCfg})
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		testEncodeFailure(t, testMux("POST /api/match-rooms/{roomID}/match", s.HandleCreateMatch),
			func() *http.Request {
				req, _ := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match", strings.NewReader(string(jsonBody)))
				req.Header.Set("Content-Type", "application/json")
				return req
			}, http.StatusCreated)
	})

	t.Run("Test Contract", func(t *testing.T) {
		s := NewServerStateManager()
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}
		gameCfg := engine.GameCfg{
			StagePreset: "MAP01",
			P1Teams:     []string{"King", "Fighter"},
			P2Teams:     []string{"King", "Witch"},
			MaxTurns:    10,
		}
		jsonBody, _ := json.Marshal(CreateMatchRequest{GameCfg: gameCfg})
		req, _ := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match", strings.NewReader(string(jsonBody)))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match", s.HandleCreateMatch).ServeHTTP(rr, req)

		assertObjectContract(t, rr.Body.Bytes(), []string{"success"}, nil)
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
	t.Run("Success: creates a new match in an existing room", func(t *testing.T) {
		s := NewServerStateManager()
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		gameCfg := engine.GameCfg{
			StagePreset: "MAP03",
			P1Teams:     []string{"King", "Fighter"},
			P2Teams:     []string{"King", "Witch"},
			MaxTurns:    10,
		}
		err = s.CreateMatch(roomID, gameCfg)
		if err != nil {
			t.Fatalf("Failed to create match: %v", err)
		}

		req, err := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/state", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("GET /api/match-rooms/{roomID}/match/state", s.HandleGetMatchState).ServeHTTP(rr, req)

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
		req, err := http.NewRequest("GET", "/api/match-rooms/NONEXISTENT/match/state", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("GET /api/match-rooms/{roomID}/match/state", s.HandleGetMatchState).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "room not found") {
			t.Errorf("Expected error message 'room not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: match not found", func(t *testing.T) {
		s := NewServerStateManager()
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		req, err := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/state", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("GET /api/match-rooms/{roomID}/match/state", s.HandleGetMatchState).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "match not found") {
			t.Errorf("Expected error message 'match not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		s := NewServerStateManager()
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		gameCfg := engine.GameCfg{
			StagePreset: "MAP01",
			P1Teams:     []string{"King", "Fighter"},
			P2Teams:     []string{"King", "Witch"},
			MaxTurns:    10,
		}
		err = s.CreateMatch(roomID, gameCfg)
		if err != nil {
			t.Fatalf("Failed to create match: %v", err)
		}

		testEncodeFailure(t, testMux("GET /api/match-rooms/{roomID}/match/state", s.HandleGetMatchState),
			func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/state", nil)
				return req
			}, http.StatusOK)
	})

	t.Run("Test Contract", func(t *testing.T) {
		s := NewServerStateManager()
		roomID, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("Failed to create room: %v", err)
		}

		gameCfg := engine.GameCfg{
			StagePreset: "MAP03",
			P1Teams:     []string{"King", "Fighter"},
			P2Teams:     []string{"King", "Witch"},
			MaxTurns:    10,
		}
		err = s.CreateMatch(roomID, gameCfg)
		if err != nil {
			t.Fatalf("Failed to create match: %v", err)
		}

		req, _ := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/state", nil)
		rr := httptest.NewRecorder()
		testMux("GET /api/match-rooms/{roomID}/match/state", s.HandleGetMatchState).ServeHTTP(rr, req)

		assertObjectContract(t, rr.Body.Bytes(),
			[]string{"turn", "activeTeam", "grid", "units", "bombs", "softBlocks", "turnCommands"},
			assertMatchStateNested)
	})
}

func createTestRoomWithMatch(t *testing.T) (string, *ServerStateManager) {
	t.Helper()

	s := NewServerStateManager()
	roomID, err := s.CreateMatchRoom()
	if err != nil {
		t.Fatalf("Failed to create room: %v", err)
	}

	gameCfg := engine.GameCfg{
		StagePreset: "MAP01",
		P1Teams:     []string{"King", "Fighter"},
		P2Teams:     []string{"King", "Witch"},
		MaxTurns:    10,
		SuddenDeath: true,
	}
	err = s.CreateMatch(roomID, gameCfg)
	if err != nil {
		t.Fatalf("Failed to create match: %v", err)
	}

	return roomID, s
}

func TestHandleSubmitTurnCommand(t *testing.T) {
	t.Run("Success: submit a valid TurnCommand in an existing room", func(t *testing.T) {
		roomID, s := createTestRoomWithMatch(t)

		uID := engine.NewUnitID(1, 0)
		newPos := engine.Coordinate{X: 4, Y: 7}
		jsonBody, _ := json.Marshal(engine.NewMoveCommand(uID, newPos))

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/turn-commands", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/turn-commands", s.HandleSubmitTurnCommand).ServeHTTP(rr, req)

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

		for _, u := range response.Units {
			if u.ID == uID {
				if u.Position != newPos {
					t.Errorf("Expected Unit %#X new position %#v, got %#v", uID, newPos, u.Position)
				}
				return
			}
		}
		t.Errorf("Expected Unit %#X is missing from the result", uID)
	})

	t.Run("Failure: invalid TurnCommand", func(t *testing.T) {
		roomID, s := createTestRoomWithMatch(t)

		uID := engine.NewUnitID(1, 0)
		newPos := engine.Coordinate{X: 4, Y: 777}
		jsonBody, _ := json.Marshal(engine.NewMoveCommand(uID, newPos))

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/turn-commands", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/turn-commands", s.HandleSubmitTurnCommand).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusConflict {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusConflict)
		}
		if !strings.Contains(rr.Body.String(), "invalid turn command") {
			t.Errorf("Expected error message 'invalid turn command', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid JSON format", func(t *testing.T) {
		roomID, s := createTestRoomWithMatch(t)

		// Malformed JSON body
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/turn-commands", strings.NewReader("{invalid json"))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/turn-commands", s.HandleSubmitTurnCommand).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "Invalid turnCommand format") {
			t.Errorf("Expected error message 'Invalid turnCommand format', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: room not found", func(t *testing.T) {
		s := NewServerStateManager()

		uID := engine.NewUnitID(1, 0)
		newPos := engine.Coordinate{X: 4, Y: 7}
		jsonBody, _ := json.Marshal(engine.NewMoveCommand(uID, newPos))

		req, err := http.NewRequest("POST", "/api/match-rooms/NONEXISTENT/match/turn-commands", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/turn-commands", s.HandleSubmitTurnCommand).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "room not found") {
			t.Errorf("Expected error message 'room not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: match not found", func(t *testing.T) {
		roomID, s := createTestRoomWithMatch(t)
		s.Rooms[roomID].Match = nil

		uID := engine.NewUnitID(1, 0)
		newPos := engine.Coordinate{X: 4, Y: 7}
		jsonBody, _ := json.Marshal(engine.NewMoveCommand(uID, newPos))

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/turn-commands", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/turn-commands", s.HandleSubmitTurnCommand).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "match not found") {
			t.Errorf("Expected error message 'match not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		roomID, s := createTestRoomWithMatch(t)
		uID := engine.NewUnitID(1, 0)
		newPos := engine.Coordinate{X: 4, Y: 7}
		jsonBody, _ := json.Marshal(engine.NewMoveCommand(uID, newPos))

		testEncodeFailure(t, testMux("POST /api/match-rooms/{roomID}/match/turn-commands", s.HandleSubmitTurnCommand),
			func() *http.Request {
				req, _ := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/turn-commands", strings.NewReader(string(jsonBody)))
				req.Header.Set("Content-Type", "application/json")
				return req
			}, http.StatusOK)
	})

	t.Run("Test Contract", func(t *testing.T) {
		roomID, s := createTestRoomWithMatch(t)
		uID := engine.NewUnitID(1, 0)
		newPos := engine.Coordinate{X: 4, Y: 7}
		jsonBody, _ := json.Marshal(engine.NewMoveCommand(uID, newPos))
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/turn-commands", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/turn-commands", s.HandleSubmitTurnCommand).ServeHTTP(rr, req)

		assertObjectContract(t, rr.Body.Bytes(),
			[]string{"turn", "activeTeam", "grid", "units", "bombs", "softBlocks", "turnCommands"},
			assertMatchStateNested)
	})
}

func TestHandleStartTurn(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		roomID, s := createTestRoomWithMatch(t)
		s.Rooms[roomID].Match.TrueState.Turn = 1000
		s.Rooms[roomID].Match.WorkingState.Turn = 1000

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/start-turn", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/start-turn", s.HandleStartTurn).ServeHTTP(rr, req)

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

		if got, want := len(s.Rooms[roomID].Match.WorkingState.Bombs), 2; got != want {
			t.Errorf("Expected SuddenDeath triggered and drop %d bombs, got %d", want, got)
		}
	})

	t.Run("Failure: room not found", func(t *testing.T) {
		s := NewServerStateManager()

		req, err := http.NewRequest("POST", "/api/match-rooms/NONEXISTENT/match/start-turn", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/start-turn", s.HandleStartTurn).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "room not found") {
			t.Errorf("Expected error message 'room not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: match not found", func(t *testing.T) {
		roomID, s := createTestRoomWithMatch(t)
		s.Rooms[roomID].Match = nil

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/start-turn", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/start-turn", s.HandleStartTurn).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "match not found") {
			t.Errorf("Expected error message 'match not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		roomID, s := createTestRoomWithMatch(t)

		testEncodeFailure(t, testMux("POST /api/match-rooms/{roomID}/match/start-turn", s.HandleStartTurn),
			func() *http.Request {
				req, _ := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/start-turn", nil)
				req.Header.Set("Content-Type", "application/json")
				return req
			}, http.StatusOK)
	})

	t.Run("Test Contract", func(t *testing.T) {
		roomID, s := createTestRoomWithMatch(t)
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/start-turn", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/start-turn", s.HandleStartTurn).ServeHTTP(rr, req)

		assertObjectContract(t, rr.Body.Bytes(),
			[]string{"turn", "activeTeam", "grid", "units", "bombs", "softBlocks", "turnCommands"},
			assertMatchStateNested)
	})
}

func TestHandleResetTurn(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		roomID, s := createTestRoomWithMatch(t)
		uID := engine.NewUnitID(1, 0)
		s.Rooms[roomID].Match.WorkingState.Units[uID].HasMoved = true

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/reset-turn", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/reset-turn", s.HandleResetTurn).ServeHTTP(rr, req)

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

		if got, want := s.Rooms[roomID].Match.WorkingState.Units[uID].HasMoved, false; got != want {
			t.Errorf("Expected Unit %#X HasMoved reset to %v, got %v", uID, want, got)
		}
	})

	t.Run("Failure: room not found", func(t *testing.T) {
		s := NewServerStateManager()

		req, err := http.NewRequest("POST", "/api/match-rooms/NONEXISTENT/match/reset-turn", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/reset-turn", s.HandleResetTurn).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "room not found") {
			t.Errorf("Expected error message 'room not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: match not found", func(t *testing.T) {
		roomID, s := createTestRoomWithMatch(t)
		s.Rooms[roomID].Match = nil

		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/reset-turn", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/reset-turn", s.HandleResetTurn).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
		if !strings.Contains(rr.Body.String(), "match not found") {
			t.Errorf("Expected error message 'match not found', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		roomID, s := createTestRoomWithMatch(t)

		testEncodeFailure(t, testMux("POST /api/match-rooms/{roomID}/match/reset-turn", s.HandleResetTurn),
			func() *http.Request {
				req, _ := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/reset-turn", nil)
				req.Header.Set("Content-Type", "application/json")
				return req
			}, http.StatusOK)
	})

	t.Run("Test Contract", func(t *testing.T) {
		roomID, s := createTestRoomWithMatch(t)
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/reset-turn", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		testMux("POST /api/match-rooms/{roomID}/match/reset-turn", s.HandleResetTurn).ServeHTTP(rr, req)

		assertObjectContract(t, rr.Body.Bytes(),
			[]string{"turn", "activeTeam", "grid", "units", "bombs", "softBlocks", "turnCommands"},
			assertMatchStateNested)
	})
}
