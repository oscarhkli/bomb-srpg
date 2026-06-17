package server

import (
	"bomb-srpg/engine"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

		var response []ArchetypeResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		expectedCount := len(engine.GetAllArchetypes())
		if len(response) != expectedCount {
			t.Errorf("Handler returned unexpected number of archetypes: got %d want %d", len(response), expectedCount)
		}
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/archetypes", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		brokenWriter := &BrokenResponseWriter{}

		http.HandlerFunc(s.HandleGetAllArchetypes).ServeHTTP(brokenWriter, req)

		if brokenWriter.Code != http.StatusOK {
			t.Errorf("Expected initial header setup to attempt status 200, got %d", brokenWriter.Code)
		}
	})

	t.Run("Test Contract", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/archetypes", nil)
		rr := httptest.NewRecorder()

		http.HandlerFunc(s.HandleGetAllArchetypes).ServeHTTP(rr, req)

		assertArchetypeContract(t, rr.Body.Bytes())
	})
}

func assertArchetypeContract(t *testing.T, body []byte) {
	var raw []map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	if len(raw) == 0 {
		t.Skip("No archetypes found to validate")
	}
	expectedFields := []string{"name", "speed", "bombMaxRange", "skills"}
	targetObj := raw[0]

	if len(targetObj) != len(expectedFields) {
		t.Errorf("Total number of fields exceeded, want %d, got %d", len(expectedFields), len(targetObj))
	}

	for _, field := range expectedFields {
		if _, exists := targetObj[field]; !exists {
			t.Errorf("Phaser Contract Broken: JavaScript code expects key '%s', but it was missing in the HTTP response payload.", field)
		}
	}
}

func TestHandleCreateServerRoom(t *testing.T) {
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
		req, err := http.NewRequest("POST", "/api/match-rooms", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		brokenWriter := &BrokenResponseWriter{}

		http.HandlerFunc(s.HandleCreateMatchRoom).ServeHTTP(brokenWriter, req)

		if brokenWriter.Code != http.StatusCreated {
			t.Errorf("Expected initial header setup to attempt status 201, got %d", brokenWriter.Code)
		}
	})

	t.Run("Test Contract", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/match-rooms", nil)
		rr := httptest.NewRecorder()

		http.HandlerFunc(s.HandleCreateMatchRoom).ServeHTTP(rr, req)

		assertMatchRoomContract(t, rr.Body.Bytes())
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

func assertMatchRoomContract(t *testing.T, body []byte) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	expectedFields := []string{"id"}
	targetObj := raw

	if len(targetObj) != len(expectedFields) {
		t.Errorf("Total number of fields exceeded, want %d, got %d", len(expectedFields), len(targetObj))
	}

	for _, field := range expectedFields {
		if _, exists := targetObj[field]; !exists {
			t.Errorf("Phaser Contract Broken: JavaScript code expects key '%s', but it was missing in the HTTP response payload.", field)
		}
	}
}

func TestHandleCreateNewMatch(t *testing.T) {
	setupMux := func(s *ServerStateManager) *http.ServeMux {
		mux := http.NewServeMux()
		mux.HandleFunc("POST /api/match-rooms/{roomID}/match", s.HandleCreateMatch)
		return mux
	}

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
		jsonBody, _ := json.Marshal(gameCfg)
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		setupMux(s).ServeHTTP(rr, req)

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
		setupMux(s).ServeHTTP(rr, req)

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
		jsonBody, _ := json.Marshal(gameCfg)
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		setupMux(s).ServeHTTP(rr, req)

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
		setupMux(s).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "invalid config") {
			t.Errorf("Expected error message 'invalid config', got: %s", rr.Body.String())
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
		setupMux(s).ServeHTTP(rr, req)

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
		jsonBody, _ := json.Marshal(gameCfg)
		req, err := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match", strings.NewReader(string(jsonBody)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		brokenWriter := &BrokenResponseWriter{}
		setupMux(s).ServeHTTP(brokenWriter, req)

		// Expect 201 because the match creation succeeded before encoding failed
		if brokenWriter.Code != http.StatusCreated {
			t.Errorf("Expected initial header setup to attempt status 201, got %d", brokenWriter.Code)
		}
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
		jsonBody, _ := json.Marshal(gameCfg)
		req, _ := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match", strings.NewReader(string(jsonBody)))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		setupMux(s).ServeHTTP(rr, req)

		assertCreateMatchContract(t, rr.Body.Bytes())
	})
}

func assertCreateMatchContract(t *testing.T, body []byte) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	expectedFields := []string{"success"}
	targetObj := raw

	if len(targetObj) != len(expectedFields) {
		t.Errorf("Total number of fields exceeded, want %d, got %d", len(expectedFields), len(targetObj))
	}

	for _, field := range expectedFields {
		if _, exists := targetObj[field]; !exists {
			t.Errorf("Phaser Contract Broken: JavaScript code expects key '%s', but it was missing in the HTTP response payload.", field)
		}
	}
}
