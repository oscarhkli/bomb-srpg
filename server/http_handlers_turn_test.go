package server

import (
	"bomb-srpg/engine"
	"bomb-srpg/room"
	"bomb-srpg/server/api"
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
		jsonBody, _ := json.Marshal(wireMoveCmd(unitID, newPos.X, newPos.Y))

		rr := doRequest(t, h, "POST", "/api/match-rooms/"+roomID+"/match/turn-commands", string(jsonBody), playerTokens[0])

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response []api.GameEvent
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}
		if len(response) != 1 {
			t.Fatalf("expected 1 GameEvent returned, got %d", len(response))
		}
		resEvt := response[0]
		validFrom := api.Coordinate{X: 4, Y: 8}
		if resEvt.Type != api.UnitMoved || resEvt.UnitID != int(unitID) || resEvt.From == nil || *resEvt.From != validFrom ||
			resEvt.To == nil || *resEvt.To != (api.Coordinate{X: newPos.X, Y: newPos.Y}) {
			t.Errorf("malformed UnitMoveEvent returned: %+v", resEvt)
		}

		u := mustMatchState(t, s, roomID).Units[unitID]

		if u.Position != newPos {
			t.Errorf("Expected Unit %#X new position %#v, got %#v", unitID, newPos, u.Position)
		}
	})

	t.Run("Failure: invalid TurnCommand", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		unitID := engine.NewUnitID(1, 0)
		newPos := engine.Coordinate{X: 4, Y: 777}
		jsonBody, _ := json.Marshal(wireMoveCmd(unitID, newPos.X, newPos.Y))

		rr := doRequest(t, h, "POST", "/api/match-rooms/"+roomID+"/match/turn-commands", string(jsonBody), playerTokens[0])

		if status := rr.Code; status != http.StatusConflict {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusConflict)
		}
		if !strings.Contains(rr.Body.String(), "invalid turn command") {
			t.Errorf("Expected error message 'invalid turn command', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid JSON format", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		rr := doRequest(t, h, "POST", "/api/match-rooms/"+roomID+"/match/turn-commands", "{invalid json", playerTokens[0])

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
		rr := doRequest(t, h, "POST", "/api/match-rooms/"+roomID+"/match/turn-commands", body, playerTokens[0])

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "Invalid turnCommand format") {
			t.Errorf("Expected error message 'Invalid turnCommand format', got: %s", rr.Body.String())
		}
	})

	testRouteFailures(t, failureRoute{
		pattern: "POST /api/match-rooms/{roomID}/match/turn-commands",
		body:    wireMoveCmd(engine.NewUnitID(1, 0), 4, 7),
		cases:   failRoom | failMatch | failMissingAuth | failInvalidToken | failWrongTeam | failRoomNoToken | failRoomBadBody,
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)
		unitID := engine.NewUnitID(1, 0)
		newPos := engine.Coordinate{X: 4, Y: 7}
		jsonBody, _ := json.Marshal(wireMoveCmd(unitID, newPos.X, newPos.Y))

		testEncodeFailure(t, routedMux(h),
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
		jsonBody, _ := json.Marshal(wireMoveCmd(unitID, newPos.X, newPos.Y))
		rr := doRequest(t, h, "POST", "/api/match-rooms/"+roomID+"/match/turn-commands", string(jsonBody), playerTokens[0])

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

func TestSubmitTurnCommandDecoding(t *testing.T) {
	const (
		formatMsg = "Invalid turnCommand format"
		target    = `"target":{"x":4,"y":7}`
	)
	tests := []struct {
		name       string
		body       string
		noToken    bool
		wantStatus int
		wantMsg    string
	}{
		{"exact keys", `{"type":"move","unitId":16,` + target + `}`, false, http.StatusOK, ""},
		{"keys match case-sensitively", `{"Type":"move","unitId":16,` + target + `}`, false, http.StatusConflict, ""},
		{"duplicate key", `{"type":"move","type":"placeBomb","unitId":16,` + target + `}`, false, http.StatusBadRequest, formatMsg},
		{"trailing data after value", `{"type":"move","unitId":16,` + target + `} garbage`, false, http.StatusBadRequest, formatMsg},
		{"second value after value", `{"type":"move","unitId":16,` + target + `}{"type":"move","unitId":16,` + target + `}`, false, http.StatusBadRequest, formatMsg},
		{"invalid UTF-8 in unknown member", "{\"type\":\"move\",\"unitId\":16,\"x\":\"\xff\"," + target + `}`, false, http.StatusBadRequest, formatMsg},
		{"invalid UTF-8 body", "\xff", false, http.StatusBadRequest, formatMsg},
		{"byte order mark before value", "\xef\xbb\xbf{\"type\":\"move\",\"unitId\":16," + target + `}`, false, http.StatusBadRequest, formatMsg},
		{"null body", `null`, false, http.StatusBadRequest, "invalid game config"},
		{"empty body", ``, false, http.StatusBadRequest, formatMsg},
		{"whitespace-only body", " \n\t", false, http.StatusBadRequest, formatMsg},
		{"unitId written with a fraction", `{"type":"move","unitId":1.0,` + target + `}`, false, http.StatusBadRequest, formatMsg},
		{"unitId written with an exponent", `{"type":"move","unitId":2e0,` + target + `}`, false, http.StatusBadRequest, formatMsg},
		{"unitId of one million", `{"type":"move","unitId":1000000,` + target + `}`, false, http.StatusBadRequest, formatMsg},
		{"unitId of negative zero", `{"type":"move","unitId":-0,` + target + `}`, false, http.StatusBadRequest, formatMsg},
		{"target x written with a fraction", `{"type":"move","unitId":16,"target":{"x":4.0,"y":7}}`, false, http.StatusBadRequest, formatMsg},
		{"target x of one million", `{"type":"move","unitId":16,"target":{"x":1000000,"y":7}}`, false, http.StatusConflict, "target out of move range"},
		{"target x beyond float64", `{"type":"move","unitId":16,"target":{"x":1e400,"y":7}}`, false, http.StatusBadRequest, formatMsg},
		{"unitId above 255", `{"type":"move","unitId":256,` + target + `}`, false, http.StatusBadRequest, formatMsg},
		{"negative unitId", `{"type":"move","unitId":-1,` + target + `}`, false, http.StatusBadRequest, formatMsg},
		{"unknown type", `{"type":"fly","unitId":16,` + target + `}`, false, http.StatusConflict, ""},
		{"malformed body without a token", `{`, true, http.StatusBadRequest, formatMsg},
		{"valid body without a token", `{"type":"move","unitId":16,` + target + `}`, true, http.StatusUnauthorized, "invalid player token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, playerTokens, _, h := createTestRoomWithMatch(t)
			header := bearerHeader(playerTokens[0])
			if tt.noToken {
				header = nil
			}

			rr := serveRequest(t, h, "POST", "/api/match-rooms/"+roomID+"/match/turn-commands", tt.body, header)

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body %q", rr.Code, tt.wantStatus, rr.Body.String())
			}
			if !strings.Contains(rr.Body.String(), tt.wantMsg) {
				t.Errorf("body = %q, want it to contain %q", rr.Body.String(), tt.wantMsg)
			}
		})
	}
}

func createSuddenDeathRoom(t *testing.T, opts ...Option) (string, [2]string, *ServerStateManager, *Handler) {
	t.Helper()
	roomID, s, h := newTestRoom(t, opts...)
	gameCfg := validGameCfg()
	gameCfg.MaxTurns = 0
	gameCfg.AllowResetTurn = true
	tokens := mustCreateMatch(t, s, roomID, gameCfg)

	return roomID, tokens, s, h
}

func TestHandleConsumeCPUStatus(t *testing.T) {
	cpuUnitID := engine.NewUnitID(2, 0)

	consume := func(t *testing.T, h *Handler, roomID, token string) *httptest.ResponseRecorder {
		t.Helper()
		return doRequest(t, h, "POST", "/api/match-rooms/"+roomID+"/match/cpu-status/consume", "", token)
	}

	readyCPUTurn := func(t *testing.T) (string, [2]string, *Handler) {
		t.Helper()
		decider := &recordingDecider{plan: moveThenBombPlan(t, cpuUnitID, engine.Coordinate{X: 4, Y: 1})}
		held := holdCPUTurn(t)
		roomID, tokens, _, h := newCPUTurnHandlerRoom(t, WithDecider(decider.decide), WithSpawner(held.spawn))
		consume(t, h, roomID, tokens[0])
		held.runOutsideRoomLock()
		return roomID, tokens, h
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

		var response api.CPUStatusResponse
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		if got, want := string(response.TurnPhase), room.TurnPhaseReady.String(); got != want {
			t.Errorf("Expected turnPhase %v, got %v", want, got)
		}
		if !hasWireEvent(response.PlanGameEvents, api.UnitMoved, cpuUnitID) {
			t.Errorf("Expected the CPU unit's move in planGameEvents, got %#v", response.PlanGameEvents)
		}

		var again api.CPUStatusResponse
		if err := json.UnmarshalRead(consume(t, h, roomID, playerTokens[0]).Body, &again); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}
		if got, want := string(again.TurnPhase), room.TurnPhaseIdle.String(); got != want {
			t.Errorf("Expected turnPhase reset to %v, got %v", want, got)
		}
		if len(again.PlanGameEvents) != 0 || len(again.ResolveTurnGameEvents) != 0 {
			t.Errorf("Expected events cleared, got %#v and %#v", again.PlanGameEvents, again.ResolveTurnGameEvents)
		}
	})

	t.Run("Success: TurnPhasePlanning leaves state untouched", func(t *testing.T) {
		roomID, playerTokens, _, h := newCPUTurnHandlerRoom(t, WithSpawner(holdCPUTurn(t).spawn))
		consume(t, h, roomID, playerTokens[0])

		rr := consume(t, h, roomID, playerTokens[0])

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		body := rr.Body.Bytes()

		var response api.CPUStatusResponse
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		if got, want := string(response.TurnPhase), room.TurnPhasePlanning.String(); got != want {
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
		var response api.CPUStatusResponse
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}
		if got, want := string(response.TurnPhase), room.TurnPhaseIdle.String(); got != want {
			t.Errorf("Expected turnPhase %v, got %v", want, got)
		}
	})

	testRouteFailures(t, failureRoute{
		pattern: "POST /api/match-rooms/{roomID}/match/cpu-status/consume",
		cases:   failRoom | failMatch | failMissingAuth | failInvalidToken | failRoomNoToken,
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		testEncodeFailure(t, routedMux(h),
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
				resolved, ok := raw["resolveTurnGameEvents"].([]any)
				if !ok || len(resolved) == 0 {
					t.Fatalf("Expected non-empty resolveTurnGameEvents, got %#v", raw["resolveTurnGameEvents"])
				}
				resolvedEvt, ok := resolved[0].(map[string]any)
				if !ok {
					t.Fatalf("resolveTurnGameEvents[0] is not an object: %#v", resolved[0])
				}
				assertKeys(t, resolvedEvt, []string{"type", "bombId", "countdown", "position"}, "Contract Broken: resolveTurnGameEvents[0] missing key '%s'")
			})
	})
}

func TestHandleResolveTurn_DuringCPUTurn(t *testing.T) {
	roomID, playerTokens, s, h := newCPUTurnHandlerRoom(t, WithSpawner(holdCPUTurn(t).spawn))
	if _, err := mustRoom(t, s, roomID).ConsumeCPU(playerTokens[0]); err != nil {
		t.Fatalf("ConsumeCPUStatus() error = %v", err)
	}

	rr := doRequest(t, h, "POST", "/api/match-rooms/"+roomID+"/match/resolve", "", playerTokens[1])

	if got, want := rr.Code, http.StatusConflict; got != want {
		t.Errorf("Handler returned wrong status code: got %v want %v", got, want)
	}
}

func TestHandleResetTurn(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		unitID := engine.NewUnitID(1, 0)
		roomID, playerTokens, s, h := createTestRoomWithMatch(t, arrangedMatch(unitAlreadyMoved(unitID)))
		if !mustMatchState(t, s, roomID).Units[unitID].HasMoved {
			t.Fatal("arranged unit has not moved")
		}

		rr := doRequest(t, h, "POST", "/api/match-rooms/"+roomID+"/match/reset", "", playerTokens[0])

		if got, want := rr.Code, http.StatusNoContent; got != want {
			t.Errorf("Handler returned wrong status code: got %v want %v", got, want)
		}

		if contentType := rr.Header().Get("Content-Type"); contentType != "" {
			t.Errorf("Expected no Content-Type on 204, got %v", contentType)
		}

		if rr.Body.Len() > 0 {
			t.Errorf("Expect empty body, got %q", rr.Body.String())
		}

		if got, want := mustMatchState(t, s, roomID).Units[unitID].HasMoved, false; got != want {
			t.Errorf("Expected Unit %#X HasMoved reset to %v, got %v", unitID, want, got)
		}
	})

	t.Run("GET state keeps the turn start events after a reset", func(t *testing.T) {
		unitID := engine.NewUnitID(1, 0)
		roomID, playerTokens, s, h := createSuddenDeathRoom(t, arrangedMatch(unitAlreadyMoved(unitID)))

		getState := func() api.GameState {
			t.Helper()
			rr := doRequest(t, h, "GET", "/api/match-rooms/"+roomID+"/match/state", "", "")
			var state api.GameState
			if err := json.UnmarshalRead(rr.Body, &state); err != nil {
				t.Fatalf("Failed to decode state: %v", err)
			}

			return state
		}

		wantEvents := getState().TurnStartEvents
		if len(wantEvents) == 0 {
			t.Fatal("state at creation has no hazard events")
		}
		byID := func(a, b api.Bomb) int { return cmp.Compare(a.ID, b.ID) }
		bombsBefore := getState().Bombs
		slices.SortFunc(bombsBefore, byID)

		if !mustMatchState(t, s, roomID).Units[unitID].HasMoved {
			t.Fatal("arranged unit has not moved")
		}
		doRequest(t, h, "POST", "/api/match-rooms/"+roomID+"/match/reset", "", playerTokens[0])

		if mustMatchState(t, s, roomID).Units[unitID].HasMoved {
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
		pattern: "POST /api/match-rooms/{roomID}/match/reset",
		cases:   failRoom | failMatch | failMissingAuth | failInvalidToken | failWrongTeam | failRoomNoToken,
	})
}

func TestHandleResolveTurn(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		unitID := engine.NewUnitID(1, 0)
		roomID, playerTokens, s, h := createTestRoomWithMatch(t, arrangedMatch(bombAboutToExplode(unitID)))

		rr := doRequest(t, h, "POST", "/api/match-rooms/"+roomID+"/match/resolve", "", playerTokens[0])

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response api.ResolveTurnResponse
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		if got, want := response.PlanGameEvents, 1; len(got) != want {
			t.Errorf("Expected %d PlanGameEvents returned, got %#v", want, got)
		}
		if got, want := response.ResolveTurnGameEvents, 5; len(got) != want {
			t.Errorf("Expected %d ResolveTurnGameEvents returned, got %#v", want, got)
		}
		if got, want := mustMatchState(t, s, roomID).Units[unitID].HP, 0; got != want {
			t.Errorf("Expected Unit %#X HP %v, got %v", 16, want, got)
		}
		if !slices.ContainsFunc(response.ResolveTurnGameEvents, func(evt api.GameEvent) bool {
			return evt.Type == api.MatchEnded && wireWinner(evt) == 2
		}) {
			t.Errorf("Expected a matchEnded event with winner 2, got %+v", response.ResolveTurnGameEvents)
		}
	})

	testRouteFailures(t, failureRoute{
		pattern: "POST /api/match-rooms/{roomID}/match/resolve",
		cases:   failRoom | failMatch | failMissingAuth | failInvalidToken | failWrongTeam | failRoomNoToken,
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		testEncodeFailure(t, routedMux(h),
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
		if _, err := mustRoom(t, s, roomID).Submit(engine.NewPlaceBombCommand(unitID, engine.Coordinate{X: 4, Y: 7}), playerTokens[0]); err != nil {
			t.Fatalf("SubmitTurnCommand() error = %v", err)
		}

		rr := doRequest(t, h, "POST", "/api/match-rooms/"+roomID+"/match/resolve", "", playerTokens[0])

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
				resolved, ok := raw["resolveTurnGameEvents"].([]any)
				if !ok || len(resolved) == 0 {
					t.Fatalf("Expected non-empty resolveTurnGameEvents, got %#v", raw["resolveTurnGameEvents"])
				}
				resolvedEvt, ok := resolved[0].(map[string]any)
				if !ok {
					t.Fatalf("resolveTurnGameEvents[0] is not an object: %#v", resolved[0])
				}
				assertKeys(t, resolvedEvt, []string{"type", "bombId", "countdown", "position"}, "Contract Broken: resolveTurnGameEvents[0] missing key '%s'")
			})

	})
}

func TestHandleSurrender(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)

		jsonBody, _ := json.Marshal(api.SurrenderRequest{TeamID: 1})
		rr := doRequest(t, h, "POST", "/api/match-rooms/"+roomID+"/match/surrender", string(jsonBody), playerTokens[0])

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response []api.GameEvent
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		if got, want := response, 1; len(got) != want {
			t.Fatalf("Expected %d gameEvents returned, got %#v", want, got)
		}
		if got, want := wireWinner(response[0]), 2; got != want {
			t.Errorf("Expected gameEvent WinnerTeamID = %v, got %v", want, got)
		}
		if _, ok := s.rooms.Load(roomID); !ok {
			t.Error("Expected room not to be deleted after surrender")
		}
	})

	t.Run("Failure: invalid SurrenderRequest", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		jsonBody, _ := json.Marshal(api.SurrenderRequest{TeamID: 3})
		rr := doRequest(t, h, "POST", "/api/match-rooms/"+roomID+"/match/surrender", string(jsonBody), playerTokens[0])

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "invalid game config") {
			t.Errorf("Expected error message 'invalid game config', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid team on an unknown room", func(t *testing.T) {
		h := NewHandler(NewServerStateManager())

		jsonBody, _ := json.Marshal(api.SurrenderRequest{TeamID: 3})
		rr := doRequest(t, h, "POST", "/api/match-rooms/NONEXISTENT/match/surrender", string(jsonBody), "dummy-token")

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "invalid game config") {
			t.Errorf("Expected error message 'invalid game config', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid team, unknown room, no Authorization header", func(t *testing.T) {
		h := NewHandler(NewServerStateManager())

		jsonBody, _ := json.Marshal(api.SurrenderRequest{TeamID: 3})
		rr := doRequest(t, h, "POST", "/api/match-rooms/NONEXISTENT/match/surrender", string(jsonBody), "")

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "invalid player token") {
			t.Errorf("Expected error message 'invalid player token', got: %s", rr.Body.String())
		}
	})

	t.Run("Failure: invalid JSON format", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		rr := doRequest(t, h, "POST", "/api/match-rooms/"+roomID+"/match/surrender", "{invalid json}", playerTokens[0])

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "Invalid surrenderRequest format") {
			t.Errorf("Expected error message 'Invalid surrenderRequest format', got: %s", rr.Body.String())
		}
	})

	testRouteFailures(t, failureRoute{
		pattern: "POST /api/match-rooms/{roomID}/match/surrender",
		body:    api.SurrenderRequest{TeamID: 1},
		cases:   failRoom | failMatch | failMissingAuth | failInvalidToken | failWrongTeam | failRoomNoToken | failRoomBadBody,
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		roomID, playerTokens, _, h := createTestRoomWithMatch(t)

		testEncodeFailure(t, routedMux(h),
			func() *http.Request {
				jsonBody, _ := json.Marshal(api.SurrenderRequest{TeamID: 1})
				req, _ := http.NewRequest("POST", "/api/match-rooms/"+roomID+"/match/surrender", strings.NewReader(string(jsonBody)))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+playerTokens[0])
				return req
			}, http.StatusOK)
	})

	t.Run("Test Contract", func(t *testing.T) {
		roomID, playerTokens, s, h := createTestRoomWithMatch(t)
		unitID := engine.NewUnitID(1, 0)
		if _, err := mustRoom(t, s, roomID).Submit(engine.NewPlaceBombCommand(unitID, engine.Coordinate{X: 4, Y: 7}), playerTokens[0]); err != nil {
			t.Fatalf("SubmitTurnCommand() error = %v", err)
		}

		jsonBody, _ := json.Marshal(api.SurrenderRequest{TeamID: 1})
		rr := doRequest(t, h, "POST", "/api/match-rooms/"+roomID+"/match/surrender", string(jsonBody), playerTokens[0])

		assertArrayContract(t, rr.Body.Bytes(), []string{"type", "winnerTeamId",
			"unitId", "countdown", "newHp"},
			nil)
	})
}

func TestSurrenderCheckOrder(t *testing.T) {
	const surrenderPath = "/match/surrender"
	tests := []struct {
		name       string
		room       string
		body       string
		header     func(tokens [2]string) http.Header
		wantStatus int
		wantMsg    string
	}{
		{"malformed body wins over the token and the room", "", `{`, func(tokens [2]string) http.Header { return bearerHeader(tokens[1]) }, http.StatusBadRequest, "Invalid surrenderRequest format"},
		{"other team's token", "", `{"teamId":1}`, func(tokens [2]string) http.Header { return bearerHeader(tokens[1]) }, http.StatusUnauthorized, "invalid player token"},
		{"no token", "", `{"teamId":1}`, func([2]string) http.Header { return nil }, http.StatusUnauthorized, "invalid player token"},
		{"bad team without Authorization", "", `{"teamId":3}`, func([2]string) http.Header { return nil }, http.StatusUnauthorized, "invalid player token"},
		{"bad team with a non-Bearer header", "", `{"teamId":3}`, func([2]string) http.Header { return http.Header{"Authorization": {"Token abc"}} }, http.StatusUnauthorized, "invalid player token"},
		{"bad team with a valid token", "", `{"teamId":3}`, func(tokens [2]string) http.Header { return bearerHeader(tokens[0]) }, http.StatusBadRequest, "team must be 1 or 2"},
		{"bad team in a nonexistent room", "NONEXISTENT", `{"teamId":3}`, func(tokens [2]string) http.Header { return bearerHeader(tokens[0]) }, http.StatusBadRequest, "team must be 1 or 2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, playerTokens, _, h := createTestRoomWithMatch(t)
			if tt.room != "" {
				roomID = tt.room
			}

			rr := serveRequest(t, h, "POST", "/api/match-rooms/"+roomID+surrenderPath, tt.body, tt.header(playerTokens))

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body %q", rr.Code, tt.wantStatus, rr.Body.String())
			}
			if !strings.Contains(rr.Body.String(), tt.wantMsg) {
				t.Errorf("body = %q, want it to contain %q", rr.Body.String(), tt.wantMsg)
			}
		})
	}
}

func TestHandleResolveTurn_SuddenDeathMatchEnd(t *testing.T) {
	kingID := engine.NewUnitID(1, 0)
	roomID, s, h := newTestRoom(t, arrangedMatch(bombAboutToExplode(kingID)))
	gameCfg := validGameCfg()
	gameCfg.MaxTurns = 0
	tokens := mustCreateMatch(t, s, roomID, gameCfg)

	hazardCountdowns := make(map[int64]int)
	for id, bomb := range mustMatchState(t, s, roomID).Bombs {
		if bomb.OwnerID == engine.SystemUnitID {
			hazardCountdowns[int64(id)] = bomb.Countdown
		}
	}
	if got, want := len(hazardCountdowns), 2; got != want {
		t.Fatalf("Expected exactly %d hazards after starting the turn, found %d", want, got)
	}

	rr := doRequest(t, h, "POST", "/api/match-rooms/"+roomID+"/match/resolve", "", tokens[0])
	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("Expected status %v, got %v", want, got)
	}
	var response api.ResolveTurnResponse
	if err := json.UnmarshalRead(rr.Body, &response); err != nil {
		t.Fatalf("Failed to decode response JSON payload: %v", err)
	}

	if !slices.ContainsFunc(response.ResolveTurnGameEvents, func(evt api.GameEvent) bool {
		return evt.Type == api.MatchEnded && wireWinner(evt) == 2
	}) {
		t.Errorf("Expected a matchEnded event with winner 2, got %+v", response.ResolveTurnGameEvents)
	}
	for name, gameEvents := range map[string][]api.GameEvent{
		"plan":    response.PlanGameEvents,
		"resolve": response.ResolveTurnGameEvents,
	} {
		if hasWireEvent(gameEvents, api.BombPlaced, engine.SystemUnitID) {
			t.Errorf("Expected no hazard bombPlaced event in the %s events, got %+v", name, gameEvents)
		}
	}

	stateRR := doRequest(t, h, "GET", "/api/match-rooms/"+roomID+"/match/state", "", "")
	var state struct {
		Turn          int        `json:"turn"`
		InSuddenDeath bool       `json:"inSuddenDeath"`
		Bombs         []api.Bomb `json:"bombs"`
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
		if bomb.OwnerID != int(engine.SystemUnitID) {
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
