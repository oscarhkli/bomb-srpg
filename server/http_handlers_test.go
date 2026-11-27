package server

import (
	"bomb-srpg/engine"
	"bomb-srpg/room"
	"bomb-srpg/server/api"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

type brokenResponseWriter struct {
	header http.Header
	Code   int
}

func (b *brokenResponseWriter) Header() http.Header {
	if b.header == nil {
		b.header = make(http.Header)
	}
	return b.header
}

func (b *brokenResponseWriter) Write(bytes []byte) (int, error) {
	return 0, fmt.Errorf("network connection systematically aborted by peer")
}

func (b *brokenResponseWriter) WriteHeader(statusCode int) {
	if b.Code != 0 {
		return
	}
	b.Code = statusCode
}

func testEncodeFailure(t *testing.T, handler http.Handler, setup func() *http.Request, expectedStatus int) {
	t.Helper()
	req := setup()
	brokenWriter := &brokenResponseWriter{}
	handler.ServeHTTP(brokenWriter, req)
	if brokenWriter.Code != expectedStatus {
		t.Errorf("Expected initial header setup to attempt status %d, got %d", expectedStatus, brokenWriter.Code)
	}
}

func createTestRoomWithMatch(t *testing.T, opts ...Option) (string, [2]string, *ServerStateManager, *Handler) {
	t.Helper()

	roomID, s, h := newTestRoom(t, opts...)

	playerTokens := mustCreateMatch(t, s, roomID, validGameCfg())

	return roomID, playerTokens, s, h
}

func routedMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	RegisterRoutes(mux, h)
	return mux
}

func serveRequest(t *testing.T, h *Handler, method, path, body string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	maps.Copy(req.Header, header)
	rr := httptest.NewRecorder()
	routedMux(h).ServeHTTP(rr, req)
	return rr
}

func bearerHeader(token string) http.Header {
	return http.Header{"Authorization": {"Bearer " + token}}
}

func doRequest(t *testing.T, h *Handler, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	var header http.Header
	if token != "" {
		header = bearerHeader(token)
	}
	return serveRequest(t, h, method, path, body, header)
}

func wireMoveCmd(unitID engine.UnitID, x, y int) api.TurnCommand {
	var cmd api.TurnCommand
	if err := cmd.FromMoveCommand(api.MoveCommand{Type: api.MoveCommandTypeMove, UnitID: int(unitID), Target: api.Coordinate{X: x, Y: y}}); err != nil {
		panic(err)
	}
	return cmd
}

func wirePlaceBombCmd(unitID engine.UnitID, x, y int) api.TurnCommand {
	var cmd api.TurnCommand
	if err := cmd.FromPlaceBombCommand(api.PlaceBombCommand{Type: api.PlaceBombCommandTypePlaceBomb, UnitID: int(unitID), Target: api.Coordinate{X: x, Y: y}}); err != nil {
		panic(err)
	}
	return cmd
}

func wireUnitMoved(evt api.GameEvent) (api.UnitMovedEvent, bool) {
	if got, err := evt.Discriminator(); err != nil || got != "unitMoved" {
		return api.UnitMovedEvent{}, false
	}
	v, err := evt.AsUnitMovedEvent()
	return v, err == nil
}

func wireBombPlaced(evt api.GameEvent) (api.BombPlacedEvent, bool) {
	if got, err := evt.Discriminator(); err != nil || got != "bombPlaced" {
		return api.BombPlacedEvent{}, false
	}
	v, err := evt.AsBombPlacedEvent()
	return v, err == nil
}

func wireMatchEnded(evt api.GameEvent) (api.MatchEndedEvent, bool) {
	if got, err := evt.Discriminator(); err != nil || got != "matchEnded" {
		return api.MatchEndedEvent{}, false
	}
	v, err := evt.AsMatchEndedEvent()
	return v, err == nil
}

func hasWireUnitMoved(events []api.GameEvent, unitID engine.UnitID) bool {
	return slices.ContainsFunc(events, func(evt api.GameEvent) bool {
		moved, ok := wireUnitMoved(evt)
		return ok && moved.UnitID == int(unitID)
	})
}

func hasWireBombPlacedBy(events []api.GameEvent, unitID engine.UnitID) bool {
	return slices.ContainsFunc(events, func(evt api.GameEvent) bool {
		placed, ok := wireBombPlaced(evt)
		return ok && placed.UnitID == int(unitID)
	})
}

func hasWireWinner(events []api.GameEvent, teamID int) bool {
	return slices.ContainsFunc(events, func(evt api.GameEvent) bool {
		ended, ok := wireMatchEnded(evt)
		return ok && ended.WinnerTeamID == teamID
	})
}

func TestWireEventHelpersCheckTheKind(t *testing.T) {
	var matchEnded api.GameEvent
	if err := matchEnded.FromMatchEndedEvent(api.MatchEndedEvent{WinnerTeamID: 2}); err != nil {
		t.Fatalf("building a matchEnded event: %v", err)
	}

	if _, ok := wireBombPlaced(matchEnded); ok {
		t.Error("wireBombPlaced accepted a matchEnded event")
	}
	if _, ok := wireUnitMoved(matchEnded); ok {
		t.Error("wireUnitMoved accepted a matchEnded event")
	}
	if ended, ok := wireMatchEnded(matchEnded); !ok || ended.WinnerTeamID != 2 {
		t.Errorf("wireMatchEnded = %+v, %v, want winner 2", ended, ok)
	}
}

func newTestRoom(t *testing.T, opts ...Option) (string, *ServerStateManager, *Handler) {
	t.Helper()
	s := NewServerStateManager(opts...)
	h := NewHandler(s)
	roomID, err := s.createMatchRoom()
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
	failRoomNoToken
	failRoomBadBody
)

type failureRoute struct {
	pattern  string
	query    string
	body     any
	cases    failureSet
	bareRoom bool
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

	callWithBody := func(t *testing.T, h *Handler, roomID, token, reqBody string) *httptest.ResponseRecorder {
		t.Helper()
		path := strings.Replace(routePath, "{roomID}", roomID, 1) + route.query
		return doRequest(t, h, method, path, reqBody, token)
	}
	call := func(t *testing.T, h *Handler, roomID, token string) *httptest.ResponseRecorder {
		t.Helper()
		return callWithBody(t, h, roomID, token, body)
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
	if route.cases&failRoomNoToken != 0 {
		t.Run("Failure: room not found, missing Authorization header", func(t *testing.T) {
			h := NewHandler(NewServerStateManager())
			assertFailure(t, call(t, h, "NONEXISTENT", ""), http.StatusUnauthorized, "invalid player token")
		})
	}
	if route.cases&failRoomBadBody != 0 {
		t.Run("Failure: room not found, invalid body", func(t *testing.T) {
			token := ""
			if authed {
				token = "dummy-token"
			}
			h := NewHandler(NewServerStateManager())
			assertFailure(t, callWithBody(t, h, "NONEXISTENT", token, "{"), http.StatusBadRequest, "Invalid")
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
			mustEndAndDeleteMatch(t, s, roomID, tokens[0])
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

func TestErrorResponseShape(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       func(roomID string) string
		body       string
		withToken  bool
		wantStatus int
	}{
		{"400", "POST", func(r string) string { return "/api/match-rooms/" + r + "/match" }, "{", false, http.StatusBadRequest},
		{"401", "POST", func(r string) string { return "/api/match-rooms/" + r + "/match/reset" }, "", false, http.StatusUnauthorized},
		{"404", "GET", func(string) string { return "/api/match-rooms/NONEXISTENT/match/state" }, "", false, http.StatusNotFound},
		{"409", "DELETE", func(r string) string { return "/api/match-rooms/" + r + "/match" }, "", true, http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, playerTokens, _, h := createTestRoomWithMatch(t)
			var header http.Header
			if tt.withToken {
				header = bearerHeader(playerTokens[0])
			}

			rr := serveRequest(t, h, tt.method, tt.path(roomID), tt.body, header)

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body %q", rr.Code, tt.wantStatus, rr.Body.String())
			}
			if got, want := rr.Header().Get("Content-Type"), "text/plain; charset=utf-8"; got != want {
				t.Errorf("Content-Type = %q, want %q", got, want)
			}
			if got, want := rr.Header().Get("X-Content-Type-Options"), "nosniff"; got != want {
				t.Errorf("X-Content-Type-Options = %q, want %q", got, want)
			}
			if body := rr.Body.String(); !strings.HasSuffix(body, "\n") {
				t.Errorf("body = %q, want it to end with a newline", body)
			}
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
		{"Missing header", "", true, "", room.ErrInvalidToken},
		{"Missing Bearer prefix", "abc123", false, "", room.ErrInvalidToken},
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
		rr := doRequest(t, h, "GET", "/api/health", "", "")

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response api.Health
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		if got, want := response.Status, "ok"; got != want {
			t.Errorf("Handler returned unexpected status: got %v want %v", got, want)
		}
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		testEncodeFailure(t, routedMux(h),
			func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/health", nil)
				return req
			}, http.StatusOK)
	})
}

func TestHandleGetCatalog(t *testing.T) {
	h := NewHandler(NewServerStateManager())

	t.Run("Success: called engine.GetAllArchetypes and engine.GetAllStagePresets", func(t *testing.T) {
		rr := doRequest(t, h, "GET", "/api/catalog", "", "")

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
		testEncodeFailure(t, routedMux(h),
			func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/catalog", nil)
				return req
			}, http.StatusOK)
	})
}
