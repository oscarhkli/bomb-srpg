package server

import (
	"bomb-srpg/engine"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
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

// BrokenResponseWriter simulates a severed connection or dropped TCP socket mid-transit.
type BrokenResponseWriter struct {
	header http.Header
	Code   int
}

func (b *BrokenResponseWriter) Header() http.Header {
	if b.header == nil {
		b.header = make(http.Header)
	}
	return b.header
}

func (b *BrokenResponseWriter) Write(bytes []byte) (int, error) {
	return 0, fmt.Errorf("network connection systematically aborted by peer")
}

func (b *BrokenResponseWriter) WriteHeader(statusCode int) {
	if b.Code != 0 {
		return
	}
	b.Code = statusCode
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

func createTestRoomWithMatch(t *testing.T) (string, [2]string, *ServerStateManager, *Handler) {
	t.Helper()

	roomID, s, h := newTestRoom(t)

	playerTokens := mustCreateMatch(t, s, roomID, validGameCfg())

	return roomID, playerTokens, s, h
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
