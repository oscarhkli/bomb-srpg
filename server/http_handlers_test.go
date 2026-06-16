package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bomb-srpg/engine"
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

		var rawPayload []map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &rawPayload); err != nil {
			t.Fatalf("Failed to parse raw JSON body: %v", err)
		}

		if len(rawPayload) == 0 {
			t.Skip("No archetypes found to validate")
		}

		targetObj := rawPayload[0]
		expectedFields := []string{
			"name",
			"speed",
			"bombMaxRange",
			"skills",
		}

		if len(targetObj) != len(expectedFields) {
			t.Errorf("Total number of fields exceeded, want %d, got %d", len(expectedFields), len(targetObj))
		}

		for _, field := range expectedFields {
			if _, exists := targetObj[field]; !exists {
				t.Errorf("Phaser Contract Broken: JavaScript code expects key '%s', but it was missing in the HTTP response payload.", field)
			}
		}
	})
}

func TestHandleCreateServerRoom(t *testing.T) {
	s := NewServerStateManager()

	t.Run("Success: called server.CreateMatchRoom", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/api/match-rooms", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()

		http.HandlerFunc(s.HandleCreateNewMatchRoom).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
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

		http.HandlerFunc(s.HandleGetAllArchetypes).ServeHTTP(brokenWriter, req)

		if brokenWriter.Code != http.StatusOK {
			t.Errorf("Expected initial header setup to attempt status 200, got %d", brokenWriter.Code)
		}
	})

	t.Run("Test Contract", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/match-rooms", nil)
		rr := httptest.NewRecorder()

		http.HandlerFunc(s.HandleCreateNewMatchRoom).ServeHTTP(rr, req)

		var rawPayload map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &rawPayload); err != nil {
			t.Fatalf("Failed to parse raw JSON body: %v", err)
		}

		if len(rawPayload) == 0 {
			t.Skip("No archetypes found to validate")
		}

		targetObj := rawPayload
		expectedFields := []string{
			"id",
		}

		if len(targetObj) != len(expectedFields) {
			t.Errorf("Total number of fields exceeded, want %d, got %d", len(expectedFields), len(targetObj))
		}

		for _, field := range expectedFields {
			if _, exists := targetObj[field]; !exists {
				t.Errorf("Phaser Contract Broken: JavaScript code expects key '%s', but it was missing in the HTTP response payload.", field)
			}
		}
	})
}
