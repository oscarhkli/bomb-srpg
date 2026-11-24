package server

import (
	"bomb-srpg/engine"
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

		rr := doRequest(t, h, "GET", "/api/match-rooms/"+roomID+"/match/state", "", "")

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response api.GameState
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
			if u.Hp != 1 {
				t.Errorf("Expected unit HP 1, got %d", u.Hp)
			}
		}
	})

	t.Run("carries the turn start events until the turn is resolved", func(t *testing.T) {
		roomID, tokens, s, h := createSuddenDeathRoom(t)

		getState := func() api.GameState {
			t.Helper()
			rr := doRequest(t, h, "GET", "/api/match-rooms/"+roomID+"/match/state", "", "")
			var state api.GameState
			if err := json.UnmarshalRead(rr.Body, &state); err != nil {
				t.Fatalf("Failed to decode state: %v", err)
			}

			return state
		}

		state := getState()
		firstEvents := state.TurnStartEvents
		if got, want := len(firstEvents), 2; got != want {
			t.Fatalf("turnStartEvents at creation = %+v, want %d", firstEvents, want)
		}
		hazards := make(map[int64]api.Coordinate)
		for _, bomb := range state.Bombs {
			if bomb.OwnerID == int(engine.SystemUnitID) {
				hazards[bomb.ID] = bomb.Position
			}
		}
		if len(hazards) != len(firstEvents) {
			t.Fatalf("hazard bombs = %d, want %d", len(hazards), len(firstEvents))
		}
		for _, evt := range firstEvents {
			if pos, ok := hazards[wireBombID(evt)]; !ok || evt.Position == nil || pos != *evt.Position {
				t.Errorf("event %+v has no hazard bomb at its position; hazards = %+v", evt, hazards)
			}
		}
		if got := getState().TurnStartEvents; !reflect.DeepEqual(got, firstEvents) {
			t.Errorf("second GET turnStartEvents = %+v, want %+v", got, firstEvents)
		}

		if _, _, err := mustRoom(t, s, roomID).Resolve(tokens[0]); err != nil {
			t.Fatalf("ResolveTurn() error = %v", err)
		}
		nextEvents := getState().TurnStartEvents
		if got, want := len(nextEvents), 2; got != want {
			t.Fatalf("turnStartEvents after resolve = %+v, want %d", nextEvents, want)
		}
		for _, evt := range nextEvents {
			if slices.ContainsFunc(firstEvents, func(old api.GameEvent) bool { return wireBombID(old) == wireBombID(evt) }) {
				t.Errorf("turnStartEvents after resolve reuse bomb ID %v", wireBombID(evt))
			}
		}
	})

	testRouteFailures(t, failureRoute{
		pattern:  "GET /api/match-rooms/{roomID}/match/state",
		cases:    failRoom | failMatch,
		bareRoom: true,
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		roomID, s, h := newTestRoom(t)

		gameCfg := validGameCfg()
		mustCreateMatch(t, s, roomID, gameCfg)

		testEncodeFailure(t, routedMux(h),
			func() *http.Request {
				req, _ := http.NewRequest("GET", "/api/match-rooms/"+roomID+"/match/state", nil)
				return req
			}, http.StatusOK)
	})

	t.Run("Test Contract", func(t *testing.T) {
		unitID := engine.NewUnitID(1, 0)
		pos := engine.Coordinate{X: 4, Y: 7}
		roomID, s, h := newTestRoom(t, arrangedMatch(turnCommandsQueued(engine.NewMoveCommand(unitID, pos))))

		gameCfg := engine.GameCfg{
			StagePreset: "Divided",
			P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
			P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
			MaxTurns:    10,
		}
		tokens := mustCreateMatch(t, s, roomID, gameCfg)
		if _, err := mustRoom(t, s, roomID).Submit(engine.NewPlaceBombCommand(unitID, pos), tokens[0]); err != nil {
			t.Fatalf("SubmitTurnCommand(place bomb) error = %v", err)
		}
		if _, _, err := mustRoom(t, s, roomID).Resolve(tokens[0]); err != nil {
			t.Fatalf("ResolveTurn() error = %v", err)
		}

		rr := doRequest(t, h, "GET", "/api/match-rooms/"+roomID+"/match/state", "", "")

		assertObjectContract(t, rr.Body.Bytes(),
			[]string{"turn", "inSuddenDeath", "activeTeam", "grid", "units", "bombs", "softBlocks", "turnCommands", "turnStartEvents"},
			assertMatchStateNested)
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

		rr := doRequest(t, h, "GET", "/api/match-rooms/"+roomID+"/match/config", "", "")

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response api.GameCfg
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		if response.StagePreset != gameCfg.StagePreset {
			t.Errorf("Expected stagePreset %v, got %v", validGameCfg().StagePreset, gameCfg.StagePreset)
		}
	})

	testRouteFailures(t, failureRoute{
		pattern:  "GET /api/match-rooms/{roomID}/match/config",
		cases:    failRoom | failMatch,
		bareRoom: true,
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		roomID, s, h := newTestRoom(t)

		gameCfg := validGameCfg()
		mustCreateMatch(t, s, roomID, gameCfg)

		testEncodeFailure(t, routedMux(h),
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

		rr := doRequest(t, h, "GET", "/api/match-rooms/"+roomID+"/match/config", "", "")

		assertObjectContract(t, rr.Body.Bytes(),
			[]string{"vsCpu", "stagePreset", "p1Slots", "p2Slots", "maxTurns", "allowResetTurn"}, nil)
	})
}

func dividedGameCfg() engine.GameCfg {
	return engine.GameCfg{
		StagePreset: "Divided",
		P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
		P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
		MaxTurns:    10,
	}
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

		rr := doRequest(t, h, "GET", "/api/match-rooms/"+roomID+"/match/allowed-tiles?unitId=16&turnCmdType=placeBomb", "", "")

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedHeader := "application/json"
		if contentType := rr.Header().Get("Content-Type"); contentType != expectedHeader {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedHeader)
		}

		var response []api.Coordinate
		if err := json.UnmarshalRead(rr.Body, &response); err != nil {
			t.Fatalf("Failed to decode response JSON payload: %v", err)
		}

		cmpFunc := func(a, b api.Coordinate) int {
			if a.X != b.X {
				return a.X - b.X
			}
			return a.Y - b.Y
		}
		want := []api.Coordinate{
			{X: 2, Y: 8}, {X: 5, Y: 8}, {X: 6, Y: 8}, {X: 4, Y: 7}, {X: 4, Y: 6},
		}
		slices.SortFunc(want, cmpFunc)
		slices.SortFunc(response, cmpFunc)
		if !slices.Equal(want, response) {
			t.Errorf("Expected coordinates %#v, got %#v", want, response)
		}
	})

	t.Run("Success: a boxed-in unit encodes an empty array, not null", func(t *testing.T) {
		// Unit 0x10 spawns at {4,8} with an ally on {3,8}; soft blocks on {5,8} and {4,7}
		// plus the stage edge below leave it no legal landing tile.
		roomID, s, h := newTestRoom(t, arrangedMatch(softBlocksAt(engine.Coordinate{X: 5, Y: 8}, engine.Coordinate{X: 4, Y: 7})))
		mustCreateMatch(t, s, roomID, validGameCfg())

		rr := doRequest(t, h, "GET", "/api/match-rooms/"+roomID+"/match/allowed-tiles?unitId=16&turnCmdType=move", "", "")

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		if got := strings.TrimSpace(rr.Body.String()); got != "[]" {
			t.Errorf("Expected an empty JSON array, got %q", got)
		}
	})

	const (
		missingMsg   = "missing required query parameters: unitId and turnCmdType are required"
		invalidIDMsg = "Invalid unitId parameter"
	)
	queryTests := []struct {
		name       string
		query      string
		wantStatus int
		wantMsg    string
	}{
		{"no query", "", http.StatusBadRequest, missingMsg},
		{"unitId only", "?unitId=16", http.StatusBadRequest, missingMsg},
		{"turnCmdType only", "?turnCmdType=move", http.StatusBadRequest, missingMsg},
		{"empty unitId", "?unitId=&turnCmdType=move", http.StatusBadRequest, missingMsg},
		{"empty turnCmdType", "?unitId=16&turnCmdType=", http.StatusBadRequest, missingMsg},
		{"non-numeric unitId without turnCmdType", "?unitId=abc", http.StatusBadRequest, missingMsg},
		{"non-numeric unitId", "?unitId=abc&turnCmdType=move", http.StatusBadRequest, invalidIDMsg},
		{"negative unitId", "?unitId=-1&turnCmdType=move", http.StatusBadRequest, invalidIDMsg},
		{"unitId above 255", "?unitId=256&turnCmdType=move", http.StatusBadRequest, invalidIDMsg},
		{"unitId 255 names no unit", "?unitId=255&turnCmdType=move", http.StatusConflict, "unit not found"},
		{"leading zero reads as 1, which names no unit", "?unitId=01&turnCmdType=move", http.StatusConflict, "unit not found"},
	}
	for _, tt := range queryTests {
		t.Run("Failure: "+tt.name, func(t *testing.T) {
			roomID, s, h := newTestRoom(t)
			mustCreateMatch(t, s, roomID, dividedGameCfg())

			rr := serveRequest(t, h, "GET", "/api/match-rooms/"+roomID+"/match/allowed-tiles"+tt.query, "", nil)

			if got := rr.Code; got != tt.wantStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", got, tt.wantStatus)
			}
			if got := rr.Body.String(); !strings.Contains(got, tt.wantMsg) {
				t.Errorf("Expected error message %q, got: %s", tt.wantMsg, got)
			}
		})
	}

	t.Run("Success: a zero-padded unitId reads as the same unit", func(t *testing.T) {
		roomID, s, h := newTestRoom(t)
		mustCreateMatch(t, s, roomID, dividedGameCfg())
		base := "/api/match-rooms/" + roomID + "/match/allowed-tiles?turnCmdType=move&unitId="

		plain := serveRequest(t, h, "GET", base+"16", "", nil)
		padded := serveRequest(t, h, "GET", base+"016", "", nil)

		if plain.Code != http.StatusOK || padded.Code != http.StatusOK {
			t.Fatalf("status = %d and %d, want both %d", plain.Code, padded.Code, http.StatusOK)
		}
		sorted := func(rr *httptest.ResponseRecorder) []api.Coordinate {
			var tiles []api.Coordinate
			if err := json.UnmarshalRead(rr.Body, &tiles); err != nil {
				t.Fatalf("Failed to decode response JSON payload: %v", err)
			}
			slices.SortFunc(tiles, func(a, b api.Coordinate) int { return cmp.Or(cmp.Compare(a.X, b.X), cmp.Compare(a.Y, b.Y)) })
			return tiles
		}
		if got, want := sorted(padded), sorted(plain); len(want) == 0 || !slices.Equal(got, want) {
			t.Errorf("unitId=016 tiles = %v, want the tiles of unitId=16, %v", got, want)
		}
	})

	testRouteFailures(t, failureRoute{
		pattern:  "GET /api/match-rooms/{roomID}/match/allowed-tiles",
		query:    "?unitId=16&turnCmdType=placeBomb",
		cases:    failRoom | failMatch,
		bareRoom: true,
	})

	t.Run("Failure: failed to Encode", func(t *testing.T) {
		roomID, s, h := newTestRoom(t)

		gameCfg := validGameCfg()
		mustCreateMatch(t, s, roomID, gameCfg)

		testEncodeFailure(t, routedMux(h),
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

		rr := doRequest(t, h, "GET", "/api/match-rooms/"+roomID+"/match/allowed-tiles?unitId=16&turnCmdType=placeBomb", "", "")

		assertArrayContract(t, rr.Body.Bytes(),
			[]string{"x", "y"}, nil)
	})
}
