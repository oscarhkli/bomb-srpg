package server

import (
	"bomb-srpg/engine"
	"bomb-srpg/room"
	"bomb-srpg/server/api"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func withSkills(t *testing.T, field any, bits uint64) {
	t.Helper()
	target := reflect.ValueOf(field).Elem()
	if !target.CanUint() {
		t.Fatalf("%T is not an unsigned integer set", field)
	}
	target.SetUint(bits)
}

func writtenBytes(t *testing.T, visit func(w http.ResponseWriter) error) string {
	t.Helper()
	rr := httptest.NewRecorder()
	if err := visit(rr); err != nil {
		t.Fatalf("writing the response: %v", err)
	}
	return rr.Body.String()
}

func TestWireEventBytes(t *testing.T) {
	pos := engine.Coordinate{X: 1, Y: 2}
	tests := []struct {
		name string
		evt  engine.GameEvent
		want string
	}{
		{
			"unit moved",
			engine.NewUnitMovedEvent(17, pos, engine.Coordinate{X: 3, Y: 4}),
			`{"countdown":0,"from":{"x":1,"y":2},"newHp":0,"to":{"x":3,"y":4},"type":"unitMoved","unitId":17}`,
		},
		{
			"unit damaged",
			engine.GameEvent{Type: engine.GameEvtUnitDamaged, UnitID: 17, Position: &pos, NewHP: 4},
			`{"countdown":0,"newHp":4,"position":{"x":1,"y":2},"type":"unitDamaged","unitId":17}`,
		},
		{
			"unit died",
			engine.NewUnitDiedEvent(17, pos),
			`{"countdown":0,"newHp":0,"position":{"x":1,"y":2},"type":"unitDied","unitId":17}`,
		},
		{
			"bomb placed",
			engine.GameEvent{Type: engine.GameEvtBombPlaced, UnitID: 17, BombID: 70000, Position: &pos, Range: 2, Countdown: 3},
			`{"bombId":70000,"countdown":3,"newHp":0,"position":{"x":1,"y":2},"range":2,"type":"bombPlaced","unitId":17}`,
		},
		{
			"bomb countdown updated",
			engine.GameEvent{Type: engine.GameEvtBombCountdownUpdated, BombID: 70000, Position: &pos, Countdown: 2},
			`{"bombId":70000,"countdown":2,"newHp":0,"position":{"x":1,"y":2},"type":"bombCountdownUpdated","unitId":0}`,
		},
		{
			"bomb exploded",
			engine.GameEvent{Type: engine.GameEvtBombExploded, BombID: 70000, Position: &pos, AffectedPositions: []engine.Coordinate{{X: 1, Y: 2}, {X: 1, Y: 3}}},
			`{"affectedPositions":[{"x":1,"y":2},{"x":1,"y":3}],"bombId":70000,"countdown":0,"newHp":0,"position":{"x":1,"y":2},"type":"bombExploded","unitId":0}`,
		},
		{
			"soft block destroyed",
			engine.GameEvent{Type: engine.GameEvtSoftBlockDestroyed, SoftBlockID: 5, Position: &pos},
			`{"countdown":0,"newHp":0,"position":{"x":1,"y":2},"softBlockId":5,"type":"softBlockDestroyed","unitId":0}`,
		},
		{
			"item id is sent when set",
			engine.GameEvent{Type: engine.GameEvtUnitMoved, ItemID: 3},
			`{"countdown":0,"itemId":3,"newHp":0,"type":"unitMoved","unitId":0}`,
		},
		{
			"match ended",
			engine.NewMatchEndedEvent(2),
			`{"countdown":0,"newHp":0,"type":"matchEnded","unitId":0,"winnerTeamId":2}`,
		},
		{
			"match drawn",
			engine.NewMatchEndedEvent(-1),
			`{"countdown":0,"newHp":0,"type":"matchEnded","unitId":0,"winnerTeamId":-1}`,
		},
		{
			"position at the origin is still sent",
			engine.GameEvent{Type: engine.GameEvtUnitDamaged, UnitID: 1, Position: &engine.Coordinate{}},
			`{"countdown":0,"newHp":0,"position":{"x":0,"y":0},"type":"unitDamaged","unitId":1}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := writtenBytes(t, api.SubmitTurnCommand200JSONResponse(mapSlice([]engine.GameEvent{tt.evt}, wireGameEvent)).VisitSubmitTurnCommandResponse)

			if want := "[" + tt.want + "]\n"; got != want {
				t.Errorf("bytes =\n%s\nwant\n%s", got, want)
			}
		})
	}

	t.Run("no events is an empty array", func(t *testing.T) {
		got := writtenBytes(t, api.SubmitTurnCommand200JSONResponse(mapSlice([]engine.GameEvent(nil), wireGameEvent)).VisitSubmitTurnCommandResponse)

		if want := "[]\n"; got != want {
			t.Errorf("bytes = %q, want %q", got, want)
		}
	})
}

func TestWireStateBytes(t *testing.T) {
	pos := engine.Coordinate{X: 1, Y: 2}
	grid := [][]engine.Tile{{{Type: engine.TerrainWater, OccupantType: engine.OccupantBomb, OccupantID: 9}}}
	const gridJSON = `[[{"occupantId":9,"occupantType":"OccupantBomb","type":"TerrainWater"}]]`

	stats := func(u engine.Unit) engine.Unit {
		u.Speed, u.BombMaxRange, u.BombPower, u.MaxBombCount, u.BombUsed, u.HP = 3, 4, 5, 6, 2, 7
		return u
	}
	skilled := stats(engine.Unit{ID: 18, Type: engine.Archetype{Name: "Witch"}, Position: pos, Team: 2, HasUsedSkill: true})
	withSkills(t, &skilled.Skills, 3)

	full := &engine.GameState{
		Turn: 2, InSuddenDeath: true, ActiveTeam: 2, Grid: grid,
		Units: map[engine.UnitID]*engine.Unit{
			20: new(stats(engine.Unit{ID: 20, Type: engine.Archetype{Name: "Fighter"}, Position: pos, Team: 2})),
			17: new(stats(engine.Unit{ID: 17, Type: engine.Archetype{Name: "King"}, Position: pos, Team: 1, HasMoved: true})),
			19: new(stats(engine.Unit{ID: 19, Type: engine.Archetype{Name: "Bandit"}, Position: pos, Team: 2})),
			18: &skilled,
		},
		Bombs: map[engine.BombID]*engine.Bomb{
			70001: {ID: 70001, OwnerID: 17, Position: pos, Range: 2, Countdown: 3},
			70000: {ID: 70000, OwnerID: 17, Position: pos, Range: 2, Countdown: 3},
		},
		SoftBlocks: map[int]*engine.SoftBlock{
			5: {ID: 5, Position: pos},
			4: {ID: 4, Position: pos},
		},
		TurnCommands: []engine.TurnCommand{
			engine.NewMoveCommand(17, pos),
			engine.NewPlaceBombCommand(17, engine.Coordinate{X: 1, Y: 3}),
		},
		TurnStartEvents: []engine.GameEvent{{Type: engine.GameEvtBombPlaced, UnitID: 17, BombID: 70000, Position: &pos, Range: 2, Countdown: 3}},
	}
	const unitHead = `{"bombMaxRange":4,"bombPower":5,"bombUsed":2,"hasMoved":`
	const unitMid = `,"hp":7,"id":`
	const unitTail = `,"maxBombCount":6,"position":{"x":1,"y":2},"skills":`
	wantFull := `{"activeTeam":2,` +
		`"bombs":[` +
		`{"countdown":3,"id":70000,"ownerId":17,"position":{"x":1,"y":2},"range":2},` +
		`{"countdown":3,"id":70001,"ownerId":17,"position":{"x":1,"y":2},"range":2}],` +
		`"grid":` + gridJSON + `,"inSuddenDeath":true,` +
		`"softBlocks":[{"id":4,"position":{"x":1,"y":2}},{"id":5,"position":{"x":1,"y":2}}],` +
		`"turn":2,` +
		`"turnCommands":[` +
		`{"target":{"x":1,"y":2},"type":"move","unitId":17},` +
		`{"target":{"x":1,"y":3},"type":"placeBomb","unitId":17}],` +
		`"turnStartEvents":[{"bombId":70000,"countdown":3,"newHp":0,"position":{"x":1,"y":2},"range":2,"type":"bombPlaced","unitId":17}],` +
		`"units":[` +
		unitHead + `true,"hasUsedSkill":false` + unitMid + `17` + unitTail + `[],"speed":3,"team":1,"type":"King"},` +
		unitHead + `false,"hasUsedSkill":true` + unitMid + `18` + unitTail + `["Jump","Fly"],"speed":3,"team":2,"type":"Witch"},` +
		unitHead + `false,"hasUsedSkill":false` + unitMid + `19` + unitTail + `[],"speed":3,"team":2,"type":"Bandit"},` +
		unitHead + `false,"hasUsedSkill":false` + unitMid + `20` + unitTail + `[],"speed":3,"team":2,"type":"Fighter"}]}`

	tests := []struct {
		name  string
		state *engine.GameState
		want  string
	}{
		{
			"without collections",
			&engine.GameState{Turn: 1, ActiveTeam: 1, Grid: grid},
			`{"activeTeam":1,"bombs":[],"grid":` + gridJSON + `,"inSuddenDeath":false,"softBlocks":[],"turn":1,"turnCommands":[],"turnStartEvents":[],"units":[]}`,
		},
		{
			"a move and a placeBomb in turnCommands",
			&engine.GameState{Turn: 1, ActiveTeam: 1, Grid: grid, TurnCommands: []engine.TurnCommand{
				engine.NewMoveCommand(17, pos),
				engine.NewPlaceBombCommand(17, engine.Coordinate{X: 1, Y: 3}),
			}},
			`{"activeTeam":1,"bombs":[],"grid":` + gridJSON + `,"inSuddenDeath":false,"softBlocks":[],"turn":1,` +
				`"turnCommands":[{"target":{"x":1,"y":2},"type":"move","unitId":17},{"target":{"x":1,"y":3},"type":"placeBomb","unitId":17}],` +
				`"turnStartEvents":[],"units":[]}`,
		},
		{"units, bombs and soft blocks in ID order", full, wantFull},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := writtenBytes(t, api.GetMatchState200JSONResponse(wireGameState(tt.state)).VisitGetMatchStateResponse)

			if want := tt.want + "\n"; got != want {
				t.Errorf("bytes =\n%s\nwant\n%s", got, want)
			}
		})
	}
}

func TestWireConfigBytes(t *testing.T) {
	tests := []struct {
		name string
		cfg  engine.GameCfg
		want string
	}{
		{
			"slots and no test overrides",
			engine.GameCfg{
				VSCpu: true, StagePreset: "Plain", MaxTurns: 7, AllowResetTurn: true,
				P1Slots:             []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
				P2Slots:             []engine.TeamSlot{{Archetype: "Boss", Role: engine.RoleBoss}},
				GlobalSpeedOverride: 9,
			},
			`{"allowResetTurn":true,"maxTurns":7,` +
				`"p1Slots":[{"archetype":"King","role":"King"},{"archetype":"Fighter","role":"Normal"}],` +
				`"p2Slots":[{"archetype":"Boss","role":"Boss"}],"stagePreset":"Plain","vsCpu":true}`,
		},
		{
			"nil slots are empty arrays",
			engine.GameCfg{StagePreset: "Plain"},
			`{"allowResetTurn":false,"maxTurns":0,"p1Slots":[],"p2Slots":[],"stagePreset":"Plain","vsCpu":false}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := writtenBytes(t, api.GetMatchConfig200JSONResponse(wireGameCfg(tt.cfg)).VisitGetMatchConfigResponse)

			if want := tt.want + "\n"; got != want {
				t.Errorf("bytes =\n%s\nwant\n%s", got, want)
			}
		})
	}
}

func TestWireCatalogBytes(t *testing.T) {
	witch := engine.Archetype{Name: "Witch", BaseSpeed: 3, BombMaxRange: 4}
	withSkills(t, &witch.PresetSkills, 1)
	tests := []struct {
		name       string
		archetypes []engine.Archetype
		want       string
	}{
		{
			"an archetype with skills",
			[]engine.Archetype{witch},
			`{"archetypes":[{"bombMaxRange":4,"name":"Witch","skills":["Jump"],"speed":3}],` +
				`"stagePresets":[{"description":"d","height":1,"maxTurns":5,"name":"Plain","width":2}]}`,
		},
		{
			"an archetype without skills",
			[]engine.Archetype{{Name: "King", BaseSpeed: 3, BombMaxRange: 4}},
			`{"archetypes":[{"bombMaxRange":4,"name":"King","skills":[],"speed":3}],` +
				`"stagePresets":[{"description":"d","height":1,"maxTurns":5,"name":"Plain","width":2}]}`,
		},
		{
			"HTML characters in a name are escaped",
			[]engine.Archetype{{Name: "A<B>&C"}},
			`{"archetypes":[{"bombMaxRange":0,"name":"A\u003cB\u003e\u0026C","skills":[],"speed":0}],` +
				`"stagePresets":[{"description":"d","height":1,"maxTurns":5,"name":"Plain","width":2}]}`,
		},
	}
	presets := []engine.StagePreset{{Name: "Plain", Description: "d", Width: 2, Height: 1, MaxTurns: 5, LayoutGrid: []string{"ab"}}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := writtenBytes(t, api.GetCatalog200JSONResponse(wireCatalog(tt.archetypes, presets)).VisitGetCatalogResponse)

			if want := tt.want + "\n"; got != want {
				t.Errorf("bytes =\n%s\nwant\n%s", got, want)
			}
		})
	}
}

func TestWireCoordinatesBytes(t *testing.T) {
	tests := []struct {
		name string
		in   []engine.Coordinate
		want string
	}{
		{"none", nil, "[]\n"},
		{"x and y keep their places", []engine.Coordinate{{X: 4, Y: 7}}, `[{"x":4,"y":7}]` + "\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := writtenBytes(t, api.GetAllowedTiles200JSONResponse(wireCoordinates(tt.in)).VisitGetAllowedTilesResponse)

			if got != tt.want {
				t.Errorf("bytes = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResponseErrorHandler(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{"an unknown error is a 500 without detail", errors.New("disk on fire"), http.StatusInternalServerError, "internal error\n"},
		{"a known error keeps its mapped status and text", room.ErrMatchNotFound, http.StatusNotFound, room.ErrMatchNotFound.Error() + "\n"},
		{"a fixed-text error keeps its status and text", badRequest("Invalid configuration format"), http.StatusBadRequest, "Invalid configuration format\n"},
	}

	h := NewHandler(NewServerStateManager(), WithHandlerLogger(discardLogger()))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			h.responseError(rr, httptest.NewRequest(http.MethodGet, "/", nil), tt.err)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
			if got := rr.Body.String(); got != tt.wantBody {
				t.Errorf("body = %q, want %q", got, tt.wantBody)
			}
			if got := rr.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
				t.Errorf("Content-Type = %q, want text/plain", got)
			}
		})
	}
}
