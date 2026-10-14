package engine

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestJSONEncoding(t *testing.T) {
	pos := Coordinate{X: 1, Y: 2}
	grid := [][]Tile{{{Type: TerrainWater, OccupantType: OccupantBomb, OccupantID: 9}}}
	const gridJSON = `[[{"type":"TerrainWater","occupantType":"OccupantBomb","occupantId":9}]]`

	tests := []struct {
		name string
		v    any
		want string
	}{
		{"unit moved", NewUnitMovedEvent(17, pos, Coordinate{X: 3, Y: 4}),
			`{"type":"unitMoved","unitId":17,"from":{"x":1,"y":2},"to":{"x":3,"y":4},"newHp":0,"countdown":0}`},
		{"unit damaged", NewUnitDamagedEvent(17, pos, 0),
			`{"type":"unitDamaged","unitId":17,"position":{"x":1,"y":2},"newHp":0,"countdown":0}`},
		{"bomb placed", NewBombPlacedEvent(17, 70000, pos, 2, 3),
			`{"type":"bombPlaced","unitId":17,"bombId":70000,"position":{"x":1,"y":2},"newHp":0,"range":2,"countdown":3}`},
		{"bomb exploded", NewBombExplodedEvent(70000, pos, []Coordinate{{X: 1, Y: 2}, {X: 1, Y: 3}}),
			`{"type":"bombExploded","unitId":0,"bombId":70000,"position":{"x":1,"y":2},"newHp":0,"countdown":0,"affectedPositions":[{"x":1,"y":2},{"x":1,"y":3}]}`},
		{"soft block destroyed", NewSoftBlockDestroyedEvent(5, pos),
			`{"type":"softBlockDestroyed","unitId":0,"softBlockId":5,"position":{"x":1,"y":2},"newHp":0,"countdown":0}`},
		{"match ended", NewMatchEndedEvent(2),
			`{"type":"matchEnded","unitId":0,"newHp":0,"countdown":0,"winnerTeamId":2}`},
		{"move command", NewMoveCommand(17, Coordinate{X: 4, Y: 7}),
			`{"type":"move","unitId":17,"target":{"x":4,"y":7}}`},
		{"archetype with skills", Archetype{Name: "Witch", BaseSpeed: 3, BombMaxRange: 4, PresetSkills: SkillCanJump | SkillCanFly},
			`{"name":"Witch","speed":3,"bombMaxRange":4,"skills":["Jump","Fly"]}`},
		{"archetype without skills", Archetype{Name: "King", BaseSpeed: 3, BombMaxRange: 4},
			`{"name":"King","speed":3,"bombMaxRange":4,"skills":[]}`},
		{"unit", Unit{ID: 17, Type: Archetype{Name: "King"}, Position: pos, Speed: 3, BombMaxRange: 4, BombPower: 1, MaxBombCount: 2, Team: 1, HP: 1, Skills: SkillCanFly, HasMoved: true},
			`{"id":17,"type":"King","position":{"x":1,"y":2},"speed":3,"bombMaxRange":4,"bombPower":1,"maxBombCount":2,"bombUsed":0,"team":1,"hp":1,"skills":["Fly"],"hasMoved":true,"hasUsedSkill":false}`},
		{"bomb", Bomb{ID: 70000, OwnerID: 17, Position: pos, Range: 2, Countdown: 3},
			`{"id":70000,"ownerId":17,"position":{"x":1,"y":2},"range":2,"countdown":3}`},
		{"game state without collections", GameState{Turn: 1, ActiveTeam: 1, Grid: grid},
			`{"turn":1,"inSuddenDeath":false,"activeTeam":1,"grid":` + gridJSON + `,"units":[],"bombs":[],"softBlocks":[],"turnCommands":[]}`},
		{"game state with collections", GameState{
			Turn: 2, InSuddenDeath: true, ActiveTeam: 2, Grid: grid,
			Units:        map[UnitID]*Unit{17: {ID: 17, Type: Archetype{Name: "King"}, Position: pos, Team: 1, HP: 1}},
			Bombs:        map[BombID]*Bomb{70000: {ID: 70000, OwnerID: 17, Position: pos, Range: 2, Countdown: 3}},
			SoftBlocks:   map[int]*SoftBlock{4: {ID: 4, Position: pos}},
			TurnCommands: []TurnCommand{NewMoveCommand(17, pos)},
		},
			`{"turn":2,"inSuddenDeath":true,"activeTeam":2,"grid":` + gridJSON +
				`,"units":[{"id":17,"type":"King","position":{"x":1,"y":2},"speed":0,"bombMaxRange":0,"bombPower":0,"maxBombCount":0,"bombUsed":0,"team":1,"hp":1,"skills":[],"hasMoved":false,"hasUsedSkill":false}]` +
				`,"bombs":[{"id":70000,"ownerId":17,"position":{"x":1,"y":2},"range":2,"countdown":3}]` +
				`,"softBlocks":[{"id":4,"position":{"x":1,"y":2}}]` +
				`,"turnCommands":[{"type":"move","unitId":17,"target":{"x":1,"y":2}}]}`},
		{"game cfg", GameCfg{VSCpu: true, StagePreset: "Plain", P1Slots: []TeamSlot{{Archetype: "King", Role: RoleKing}}, P2Slots: []TeamSlot{}, MaxTurns: 0, GlobalSpeedOverride: 9},
			`{"vsCpu":true,"stagePreset":"Plain","p1Slots":[{"archetype":"King","role":"King"}],"p2Slots":[],"maxTurns":0,"allowResetTurn":false}`},
		{"game cfg nil slots", GameCfg{StagePreset: "Plain"},
			`{"vsCpu":false,"stagePreset":"Plain","p1Slots":null,"p2Slots":null,"maxTurns":0,"allowResetTurn":false}`},
		{"stage preset", StagePreset{Name: "Plain", Description: "d", Width: 2, Height: 1, MaxTurns: 5, LayoutGrid: []string{"ab"}},
			`{"name":"Plain","description":"d","width":2,"height":1,"maxTurns":5}`},
		{"HTML characters are escaped", Archetype{Name: "<a&b>"},
			`{"name":"\u003ca\u0026b\u003e","speed":0,"bombMaxRange":0,"skills":[]}`},
		{"cpu phase", TurnPhaseReady, `"TurnPhaseReady"`},
		{"unknown enums", []any{TerrainType(99), OccupantType(99), UnitRole(99), CPUTurnPhase(99)},
			`["TerrainUnknown","OccupantUnknown","Unknown","TurnPhaseUnknown"]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.v)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("Marshal() =\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}

func TestJSONDecoding(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    TurnCommand
		wantErr bool
	}{
		{"exact keys", `{"type":"move","unitId":17,"target":{"x":4,"y":7}}`,
			TurnCommand{Type: TurnCmdMove, UnitID: 17, Target: Coordinate{X: 4, Y: 7}}, false},
		{"keys match case-insensitively", `{"TYPE":"move","UnitID":17,"TARGET":{"X":4,"Y":7}}`,
			TurnCommand{Type: TurnCmdMove, UnitID: 17, Target: Coordinate{X: 4, Y: 7}}, false},
		{"unknown member ignored", `{"type":"move","unitId":17,"target":{"x":4,"y":7},"extra":[1]}`,
			TurnCommand{Type: TurnCmdMove, UnitID: 17, Target: Coordinate{X: 4, Y: 7}}, false},
		{"duplicate key last wins", `{"type":"move","type":"placeBomb","unitId":17,"target":{"x":4,"y":7}}`,
			TurnCommand{Type: TurnCmdPlaceBomb, UnitID: 17, Target: Coordinate{X: 4, Y: 7}}, false},
		{"invalid UTF-8 is replaced", "{\"type\":\"mo\xffve\",\"unitId\":17,\"target\":{\"x\":4,\"y\":7}}",
			TurnCommand{Type: "mo\ufffdve", UnitID: 17, Target: Coordinate{X: 4, Y: 7}}, false},
		{"missing members stay zero", `{}`, TurnCommand{}, false},
		{"null leaves zero", `{"type":null,"unitId":null}`, TurnCommand{}, false},
		{"unit id overflows uint8", `{"type":"move","unitId":300,"target":{"x":4,"y":7}}`, TurnCommand{}, true},
		{"wrong member type", `{"type":"move","unitId":"17"}`, TurnCommand{}, true},
		{"trailing data", `{"type":"move"} x`, TurnCommand{}, true},
		{"truncated", `{"type":"move"`, TurnCommand{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got TurnCommand
			err := json.Unmarshal([]byte(tt.input), &got)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Unmarshal() error = %v, want error %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Unmarshal() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestGameCfgDecoding(t *testing.T) {
	t.Run("slots and roles", func(t *testing.T) {
		var got GameCfg
		in := `{"vsCpu":true,"stagePreset":"Plain","p1Slots":[{"archetype":"King","role":"King"}],"p2Slots":null,"maxTurns":3,"allowResetTurn":true}`
		if err := json.Unmarshal([]byte(in), &got); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		want := GameCfg{VSCpu: true, StagePreset: "Plain", P1Slots: []TeamSlot{{Archetype: "King", Role: RoleKing}}, MaxTurns: 3, AllowResetTurn: true}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Unmarshal() = %+v, want %+v", got, want)
		}
	})

	t.Run("hidden fields are not settable", func(t *testing.T) {
		var got GameCfg
		if err := json.Unmarshal([]byte(`{"GlobalSpeedOverride":9}`), &got); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		if got.GlobalSpeedOverride != 0 {
			t.Errorf("GlobalSpeedOverride = %d, want 0", got.GlobalSpeedOverride)
		}
	})

	roleTests := []struct {
		name    string
		role    string
		wantErr error
	}{
		{"wrong case", `"king"`, ErrInvalidUnitRole},
		{"empty", `""`, ErrInvalidUnitRole},
		{"null", `null`, ErrInvalidUnitRole},
		{"number", `1`, nil},
	}
	for _, tt := range roleTests {
		t.Run("role "+tt.name, func(t *testing.T) {
			var got TeamSlot
			err := json.Unmarshal([]byte(`{"archetype":"King","role":`+tt.role+`}`), &got)
			if err == nil {
				t.Fatalf("Unmarshal() error = nil, want error")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("Unmarshal() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
