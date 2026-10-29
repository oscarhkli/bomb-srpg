package server

import (
	"bomb-srpg/engine"
	"encoding/json/v2"
	"net/http"
	"slices"
	"strings"
	"testing"
)

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
		startTurnForTest(t, s, roomID, tokens[0])
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

	queryFailureCfg := engine.GameCfg{
		StagePreset: "Divided",
		P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
		P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
		MaxTurns:    10,
	}
	queryFailureTests := []struct {
		name    string
		query   string
		wantMsg string
	}{
		{"Failure: missing query string", "", "missing required query parameters"},
		{"Failure: invalid query string", "?unitId=abc&turnCmdType=move", "Invalid unitId parameter"},
	}
	for _, tt := range queryFailureTests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, s, h := newTestRoom(t)
			mustCreateMatch(t, s, roomID, queryFailureCfg)

			rr := doRequest(t, h.HandleGetAllowedTiles, "GET /api/match-rooms/{roomID}/match/allowed-tiles", "GET", "/api/match-rooms/"+roomID+"/match/allowed-tiles"+tt.query, "", "")

			if got := rr.Code; got != http.StatusBadRequest {
				t.Errorf("Handler returned wrong status code: got %v want %v", got, http.StatusBadRequest)
			}
			if got := rr.Body.String(); !strings.Contains(got, tt.wantMsg) {
				t.Errorf("Expected error message %q, got: %s", tt.wantMsg, got)
			}
		})
	}

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
