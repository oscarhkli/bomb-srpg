package server

import (
	"bomb-srpg/engine"
	"cmp"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestHandleSubmitTurnCommand(t *testing.T) {
	t.Run("Success: submit a valid TurnCommand in an existing room", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)

		unitID := engine.NewUnitID(1, 0)
		newPos := engine.Coordinate{X: 4, Y: 7}
		jsonBody, _ := json.Marshal(engine.NewMoveCommand(unitID, newPos))

		rr := doRequest(t, h.handleSubmitTurnCommand, "POST /api/match-rooms/{roomID}/match/turn-commands", "POST", "/api/match-rooms/"+roomID+"/match/turn-commands", string(jsonBody), playerTokens[0])

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

		rr := doRequest(t, h.handleSubmitTurnCommand, "POST /api/match-rooms/{roomID}/match/turn-commands", "POST", "/api/match-rooms/"+roomID+"/match/turn-commands", string(jsonBody), playerTokens[0])

		if status := rr.Code; status != http.StatusConflict {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusConflict)
		}
		if !strings.Contains(rr.Body.String(), "invalid turn command") {
			t.Errorf("Expected error message 'invalid turn command', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid JSON format", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		rr := doRequest(t, h.handleSubmitTurnCommand, "POST /api/match-rooms/{roomID}/match/turn-commands", "POST", "/api/match-rooms/"+roomID+"/match/turn-commands", "{invalid json", playerTokens[0])

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
		testMux("POST /api/match-rooms/{roomID}/match/turn-commands", h.handleSubmitTurnCommand).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "Invalid turnCommand format") {
			t.Errorf("Expected error message 'Invalid turnCommand format', got: %s", rr.Body.String())
		}
	})

	testRouteFailures(t, failureRoute{
		handler: (*Handler).handleSubmitTurnCommand,
		pattern: "POST /api/match-rooms/{roomID}/match/turn-commands",
		body:    engine.NewMoveCommand(engine.NewUnitID(1, 0), engine.Coordinate{X: 4, Y: 7}),
		cases:   failRoom | failMatch | failMissingAuth | failInvalidToken | failWrongTeam,
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)
		unitID := engine.NewUnitID(1, 0)
		newPos := engine.Coordinate{X: 4, Y: 7}
		jsonBody, _ := json.Marshal(engine.NewMoveCommand(unitID, newPos))

		testEncodeFailure(t, testMux("POST /api/match-rooms/{roomID}/match/turn-commands", h.handleSubmitTurnCommand),
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
		rr := doRequest(t, h.handleSubmitTurnCommand, "POST /api/match-rooms/{roomID}/match/turn-commands", "POST", "/api/match-rooms/"+roomID+"/match/turn-commands", string(jsonBody), playerTokens[0])

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

func createSuddenDeathRoom(t *testing.T) (string, [2]string, *ServerStateManager, *Handler) {
	t.Helper()
	roomID, s, h := newTestRoom(t)
	gameCfg := validGameCfg()
	gameCfg.MaxTurns = 0
	gameCfg.AllowResetTurn = true
	tokens := mustCreateMatch(t, s, roomID, gameCfg)

	return roomID, tokens, s, h
}

func TestHandleConsumeCPUStatus(t *testing.T) {
	const consumePattern = "POST /api/match-rooms/{roomID}/match/cpu-status/consume"
	cpuUnitID := engine.NewUnitID(2, 0)

	consume := func(t *testing.T, h *Handler, roomID, token string) *httptest.ResponseRecorder {
		t.Helper()
		return doRequest(t, h.handleConsumeCPUStatus, consumePattern, "POST", "/api/match-rooms/"+roomID+"/match/cpu-status/consume", "", token)
	}

	readyCPUTurn := func(t *testing.T) (string, [2]string, *Handler) {
		t.Helper()
		roomID, tokens, s, h := newCPUTurnHandlerRoom(t)
		s.decide = (&recordingDecider{plan: func(*engine.GameState, int) []engine.TurnCommand {
			return []engine.TurnCommand{engine.NewMoveCommand(cpuUnitID, engine.Coordinate{X: 4, Y: 1})}
		}}).decide
		held := holdCPUTurn(t, s)
		consume(t, h, roomID, tokens[0])
		held.runOutsideRoomLock()
		return roomID, tokens, h
	}

	type consumeResponse struct {
		TurnPhase             string             `json:"turnPhase"`
		PlanGameEvents        []engine.GameEvent `json:"planGameEvents"`
		ResolveTurnGameEvents []engine.GameEvent `json:"resolveTurnGameEvents"`
	}

	t.Run("Success: TurnPhaseReady returns and clears pending events", func(t *testing.T) {
		roomID, playerTokens, h := readyCPUTurn(t)

		rr := consume(t, h, roomID, playerTokens[0])

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response consumeResponse
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		if got, want := response.TurnPhase, engine.TurnPhaseReady.String(); got != want {
			t.Errorf("Expected turnPhase %v, got %v", want, got)
		}
		if !hasGameEvent(response.PlanGameEvents, engine.GameEvtUnitMoved, cpuUnitID) {
			t.Errorf("Expected the CPU unit's move in planGameEvents, got %#v", response.PlanGameEvents)
		}

		var again consumeResponse
		if err := json.UnmarshalRead(consume(t, h, roomID, playerTokens[0]).Body, &again); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}
		if got, want := again.TurnPhase, engine.TurnPhaseIdle.String(); got != want {
			t.Errorf("Expected turnPhase reset to %v, got %v", want, got)
		}
		if len(again.PlanGameEvents) != 0 || len(again.ResolveTurnGameEvents) != 0 {
			t.Errorf("Expected events cleared, got %#v and %#v", again.PlanGameEvents, again.ResolveTurnGameEvents)
		}
	})

	t.Run("Success: TurnPhasePlanning leaves state untouched", func(t *testing.T) {
		roomID, playerTokens, s, h := newCPUTurnHandlerRoom(t)
		holdCPUTurn(t, s)
		consume(t, h, roomID, playerTokens[0])

		rr := consume(t, h, roomID, playerTokens[0])

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		body := rr.Body.Bytes()

		var response consumeResponse
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
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		rr := consume(t, h, roomID, playerTokens[1])

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		var response consumeResponse
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}
		if got, want := response.TurnPhase, engine.TurnPhaseIdle.String(); got != want {
			t.Errorf("Expected turnPhase %v, got %v", want, got)
		}
	})

	testRouteFailures(t, failureRoute{
		handler: (*Handler).handleConsumeCPUStatus,
		pattern: consumePattern,
		cases:   failRoom | failMatch | failMissingAuth | failInvalidToken,
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		testEncodeFailure(t, testMux(consumePattern, h.handleConsumeCPUStatus),
			func() *http.Request {
				req, _ := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/cpu-status/consume", nil)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+playerTokens[0])
				return req
			}, http.StatusOK)
	})

	t.Run("Test Contract", func(t *testing.T) {
		roomID, playerTokens, h := readyCPUTurn(t)

		rr := consume(t, h, roomID, playerTokens[0])

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

func TestHandleResolveTurn_DuringCPUTurn(t *testing.T) {
	roomID, playerTokens, s, h := newCPUTurnHandlerRoom(t)
	holdCPUTurn(t, s)
	if _, _, _, err := s.consumeCPUStatus(roomID, playerTokens[0]); err != nil {
		t.Fatalf("ConsumeCPUStatus() error = %v", err)
	}

	rr := doRequest(t, h.handleResolveTurn, "POST /api/match-rooms/{roomID}/match/resolve", "POST", "/api/match-rooms/"+roomID+"/match/resolve", "", playerTokens[1])

	if got, want := rr.Code, http.StatusConflict; got != want {
		t.Errorf("Handler returned wrong status code: got %v want %v", got, want)
	}
}

func TestHandleResetTurn(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		unitID := engine.NewUnitID(1, 0)
		room := mustRoom(t, s, roomID)
		room.Match.WorkingState.Units[unitID].HasMoved = true

		rr := doRequest(t, h.handleResetTurn, "POST /api/match-rooms/{roomID}/match/reset", "POST", "/api/match-rooms/"+roomID+"/match/reset", "", playerTokens[0])

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

	t.Run("GET state keeps the turn start events after a reset", func(t *testing.T) {
		roomID, playerTokens, s, h := createSuddenDeathRoom(t)
		room := mustRoom(t, s, roomID)

		getState := func() clientMatchStateResponse {
			t.Helper()
			rr := doRequest(t, h.handleGetMatchState, "GET /api/match-rooms/{roomID}/match/state", "GET", "/api/match-rooms/"+roomID+"/match/state", "", "")
			var state clientMatchStateResponse
			if err := json.UnmarshalRead(rr.Body, &state); err != nil {
				t.Fatalf("Failed to decode state: %v", err)
			}

			return state
		}

		wantEvents := getState().TurnStartEvents
		if len(wantEvents) == 0 {
			t.Fatal("state at creation has no hazard events")
		}
		byID := func(a, b *engine.Bomb) int { return cmp.Compare(a.ID, b.ID) }
		bombsBefore := getState().Bombs
		slices.SortFunc(bombsBefore, byID)

		unitID := engine.NewUnitID(1, 0)
		room.Match.WorkingState.Units[unitID].HasMoved = true
		doRequest(t, h.handleResetTurn, "POST /api/match-rooms/{roomID}/match/reset", "POST", "/api/match-rooms/"+roomID+"/match/reset", "", playerTokens[0])

		if room.Match.WorkingState.Units[unitID].HasMoved {
			t.Fatal("reset did not discard the planned action")
		}
		state := getState()
		if !reflect.DeepEqual(state.TurnStartEvents, wantEvents) {
			t.Errorf("turnStartEvents after reset = %+v, want %+v", state.TurnStartEvents, wantEvents)
		}
		slices.SortFunc(state.Bombs, byID)
		if !reflect.DeepEqual(state.Bombs, bombsBefore) {
			t.Errorf("bombs after reset = %+v, want %+v", state.Bombs, bombsBefore)
		}
	})

	testRouteFailures(t, failureRoute{
		handler: (*Handler).handleResetTurn,
		pattern: "POST /api/match-rooms/{roomID}/match/reset",
		cases:   failRoom | failMatch | failMissingAuth | failInvalidToken | failWrongTeam,
	})
}

func TestHandleResolveTurn(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		unitID := engine.NewUnitID(1, 0)
		if _, err := s.submitTurnCommand(roomID, engine.NewPlaceBombCommand(unitID, engine.Coordinate{X: 4, Y: 7}), playerTokens[0]); err != nil {
			t.Fatalf("SubmitTurnCommand() error = %v", err)
		}
		room := mustRoom(t, s, roomID)
		room.Match.WorkingState.Bombs[engine.NewBombID(1, 1, unitID)].Countdown = 1

		rr := doRequest(t, h.handleResolveTurn, "POST /api/match-rooms/{roomID}/match/resolve", "POST", "/api/match-rooms/"+roomID+"/match/resolve", "", playerTokens[0])

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response resolveTurnResponse
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
		handler: (*Handler).handleResolveTurn,
		pattern: "POST /api/match-rooms/{roomID}/match/resolve",
		cases:   failRoom | failMatch | failMissingAuth | failInvalidToken | failWrongTeam,
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		testEncodeFailure(t, testMux("POST /api/match-rooms/{roomID}/match/resolve", h.handleResolveTurn),
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
		if _, err := s.submitTurnCommand(roomID, engine.NewPlaceBombCommand(unitID, engine.Coordinate{X: 4, Y: 7}), playerTokens[0]); err != nil {
			t.Fatalf("SubmitTurnCommand() error = %v", err)
		}

		rr := doRequest(t, h.handleResolveTurn, "POST /api/match-rooms/{roomID}/match/resolve", "POST", "/api/match-rooms/"+roomID+"/match/resolve", "", playerTokens[0])

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

		jsonBody, _ := json.Marshal(surrenderRequest{TeamID: 1})
		rr := doRequest(t, h.handleSurrender, "POST /api/match-rooms/{roomID}/match/surrender", "POST", "/api/match-rooms/"+roomID+"/match/surrender", string(jsonBody), playerTokens[0])

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

		jsonBody, _ := json.Marshal(surrenderRequest{TeamID: 3})
		rr := doRequest(t, h.handleSurrender, "POST /api/match-rooms/{roomID}/match/surrender", "POST", "/api/match-rooms/"+roomID+"/match/surrender", string(jsonBody), playerTokens[0])

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "invalid game config") {
			t.Errorf("Expected error message 'invalid game config', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid JSON format", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		rr := doRequest(t, h.handleSurrender, "POST /api/match-rooms/{roomID}/match/surrender", "POST", "/api/match-rooms/"+roomID+"/match/surrender", "{invalid json}", playerTokens[0])

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "Invalid surrenderRequest format") {
			t.Errorf("Expected error message 'Invalid surrenderRequest format', got: %s", rr.Body.String())
		}
	})

	testRouteFailures(t, failureRoute{
		handler: (*Handler).handleSurrender,
		pattern: "POST /api/match-rooms/{roomID}/match/surrender",
		body:    surrenderRequest{TeamID: 1},
		cases:   failRoom | failMatch | failMissingAuth | failInvalidToken | failWrongTeam,
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		testEncodeFailure(t, testMux("POST /api/match-rooms/{roomID}/match/surrender", h.handleSurrender),
			func() *http.Request {
				jsonBody, _ := json.Marshal(surrenderRequest{TeamID: 1})
				req, _ := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/surrender", strings.NewReader(string(jsonBody)))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+playerTokens[0])
				return req
			}, http.StatusOK)
	})

	t.Run("Test Contract", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		unitID := engine.NewUnitID(1, 0)
		if _, err := s.submitTurnCommand(roomID, engine.NewPlaceBombCommand(unitID, engine.Coordinate{X: 4, Y: 7}), playerTokens[0]); err != nil {
			t.Fatalf("SubmitTurnCommand() error = %v", err)
		}

		jsonBody, _ := json.Marshal(surrenderRequest{TeamID: 1})
		rr := doRequest(t, h.handleSurrender, "POST /api/match-rooms/{roomID}/match/surrender", "POST", "/api/match-rooms/"+roomID+"/match/surrender", string(jsonBody), playerTokens[0])

		assertArrayContract(t, rr.Body.Bytes(), []string{"type", "winnerTeamId",
			"unitId", "countdown", "newHp"},
			nil)
	})
}

func TestHandleResolveTurn_SuddenDeathMatchEnd(t *testing.T) {
	roomID, s, h := newTestRoom(t)
	gameCfg := validGameCfg()
	gameCfg.MaxTurns = 0
	tokens := mustCreateMatch(t, s, roomID, gameCfg)

	room := mustRoom(t, s, roomID)
	hazardCountdowns := make(map[engine.BombID]int)
	for id, bomb := range room.Match.WorkingState.Bombs {
		if bomb.OwnerID == engine.SystemUnitID {
			hazardCountdowns[id] = bomb.Countdown
		}
	}
	if got, want := len(hazardCountdowns), 2; got != want {
		t.Fatalf("Expected exactly %d hazards after starting the turn, found %d", want, got)
	}

	kingID := engine.NewUnitID(1, 0)
	kingPos := room.Match.WorkingState.Units[kingID].Position
	target := freeNeighbour(t, room.Match.WorkingState, kingPos)
	if _, err := s.submitTurnCommand(roomID, engine.NewPlaceBombCommand(kingID, target), tokens[0]); err != nil {
		t.Fatalf("SubmitTurnCommand() error = %v", err)
	}
	for _, bomb := range room.Match.WorkingState.Bombs {
		if bomb.OwnerID == kingID {
			bomb.Countdown = 1
		}
	}

	rr := doRequest(t, h.handleResolveTurn, "POST /api/match-rooms/{roomID}/match/resolve", "POST", "/api/match-rooms/"+roomID+"/match/resolve", "", tokens[0])
	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("Expected status %v, got %v", want, got)
	}
	var response resolveTurnResponse
	if err := json.UnmarshalRead(rr.Body, &response); err != nil {
		t.Fatalf("Failed to decode response JSON payload: %v", err)
	}

	if !slices.ContainsFunc(response.ResolveTurnGameEvents, func(evt engine.GameEvent) bool {
		return evt.Type == engine.GameEvtMatchEnded && evt.WinnerTeamID == 2
	}) {
		t.Errorf("Expected a matchEnded event with winner 2, got %+v", response.ResolveTurnGameEvents)
	}
	if got, want := room.Match.WinnerTeamID, 2; got != want {
		t.Errorf("Expected winner %v, got %v", want, got)
	}
	for name, gameEvents := range map[string][]engine.GameEvent{
		"plan":    response.PlanGameEvents,
		"resolve": response.ResolveTurnGameEvents,
	} {
		if hasGameEvent(gameEvents, engine.GameEvtBombPlaced, engine.SystemUnitID) {
			t.Errorf("Expected no hazard bombPlaced event in the %s events, got %+v", name, gameEvents)
		}
	}

	stateRR := doRequest(t, h.handleGetMatchState, "GET /api/match-rooms/{roomID}/match/state", "GET", "/api/match-rooms/"+roomID+"/match/state", "", "")
	var state struct {
		Turn          int           `json:"turn"`
		InSuddenDeath bool          `json:"inSuddenDeath"`
		Bombs         []engine.Bomb `json:"bombs"`
	}
	if err := json.UnmarshalRead(stateRR.Body, &state); err != nil {
		t.Fatalf("Failed to decode state JSON payload: %v", err)
	}
	if !state.InSuddenDeath {
		t.Error("Expected the state to stay in sudden death")
	}
	if got, want := state.Turn, 1; got != want {
		t.Errorf("Expected turn %d to stay, got %d", want, got)
	}
	remaining := 0
	for _, bomb := range state.Bombs {
		if bomb.OwnerID != engine.SystemUnitID {
			continue
		}
		remaining++
		before, ok := hazardCountdowns[bomb.ID]
		if !ok {
			t.Errorf("Expected hazard %#x to be one of the recorded ones", bomb.ID)
			continue
		}
		if got, want := bomb.Countdown, before-1; got != want {
			t.Errorf("Expected hazard %#x countdown %d, got %d", bomb.ID, want, got)
		}
	}
	if remaining > len(hazardCountdowns) {
		t.Errorf("Expected at most %d hazards in the state, got %d", len(hazardCountdowns), remaining)
	}
}
