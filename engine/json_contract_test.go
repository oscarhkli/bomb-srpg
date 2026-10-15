package engine

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"
)

type encodingCase struct {
	name string
	v    any
	want string
}

func checkEncoding(t *testing.T, tests []encodingCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := jsontext.Value(tt.want)
			if err := want.Compact(); err != nil {
				t.Fatalf("invalid want: %v", err)
			}
			got, err := json.Marshal(tt.v)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(got) != string(want) {
				t.Errorf("Marshal() =\n%s\nwant\n%s", got, want)
			}
		})
	}
}

func TestEventAndCommandEncoding(t *testing.T) {
	pos := Coordinate{X: 1, Y: 2}

	checkEncoding(t, []encodingCase{
		{"unit moved", NewUnitMovedEvent(17, pos, Coordinate{X: 3, Y: 4}), `{
			"type": "unitMoved", "unitId": 17,
			"from": {"x": 1, "y": 2}, "to": {"x": 3, "y": 4},
			"newHp": 0, "countdown": 0}`},
		{"unit damaged", NewUnitDamagedEvent(17, pos, 0), `{
			"type": "unitDamaged", "unitId": 17,
			"position": {"x": 1, "y": 2}, "newHp": 0, "countdown": 0}`},
		{"bomb placed", NewBombPlacedEvent(17, 70000, pos, 2, 3), `{
			"type": "bombPlaced", "unitId": 17, "bombId": 70000,
			"position": {"x": 1, "y": 2}, "newHp": 0, "range": 2, "countdown": 3}`},
		{"bomb exploded", NewBombExplodedEvent(70000, pos, []Coordinate{{X: 1, Y: 2}, {X: 1, Y: 3}}), `{
			"type": "bombExploded", "unitId": 0, "bombId": 70000,
			"position": {"x": 1, "y": 2}, "newHp": 0, "countdown": 0,
			"affectedPositions": [{"x": 1, "y": 2}, {"x": 1, "y": 3}]}`},
		{"soft block destroyed", NewSoftBlockDestroyedEvent(5, pos), `{
			"type": "softBlockDestroyed", "unitId": 0, "softBlockId": 5,
			"position": {"x": 1, "y": 2}, "newHp": 0, "countdown": 0}`},
		{"match ended", NewMatchEndedEvent(2), `{
			"type": "matchEnded", "unitId": 0, "newHp": 0, "countdown": 0,
			"winnerTeamId": 2}`},
		{"move command", NewMoveCommand(17, Coordinate{X: 4, Y: 7}), `{
			"type": "move", "unitId": 17, "target": {"x": 4, "y": 7}}`},
	})
}

func TestEntityEncoding(t *testing.T) {
	pos := Coordinate{X: 1, Y: 2}

	checkEncoding(t, []encodingCase{
		{"archetype with skills", Archetype{Name: "Witch", BaseSpeed: 3, BombMaxRange: 4, PresetSkills: SkillCanJump | SkillCanFly}, `{
			"name": "Witch", "speed": 3, "bombMaxRange": 4, "skills": ["Jump", "Fly"]}`},
		{"archetype without skills", Archetype{Name: "King", BaseSpeed: 3, BombMaxRange: 4}, `{
			"name": "King", "speed": 3, "bombMaxRange": 4, "skills": []}`},
		{"archetype name keeps HTML characters", Archetype{Name: "<a&b>"}, `{
			"name": "<a&b>", "speed": 0, "bombMaxRange": 0, "skills": []}`},
		{"unit", Unit{ID: 17, Type: Archetype{Name: "King"}, Position: pos, Speed: 3, BombMaxRange: 4, BombPower: 1, MaxBombCount: 2, Team: 1, HP: 1, Skills: SkillCanFly, HasMoved: true}, `{
			"id": 17, "type": "King", "position": {"x": 1, "y": 2},
			"speed": 3, "bombMaxRange": 4, "bombPower": 1, "maxBombCount": 2, "bombUsed": 0,
			"team": 1, "hp": 1, "skills": ["Fly"], "hasMoved": true, "hasUsedSkill": false}`},
		{"bomb", Bomb{ID: 70000, OwnerID: 17, Position: pos, Range: 2, Countdown: 3}, `{
			"id": 70000, "ownerId": 17, "position": {"x": 1, "y": 2}, "range": 2, "countdown": 3}`},
	})
}

func TestGameStateEncoding(t *testing.T) {
	pos := Coordinate{X: 1, Y: 2}
	grid := [][]Tile{{{Type: TerrainWater, OccupantType: OccupantBomb, OccupantID: 9}}}
	const gridJSON = `[[{"type": "TerrainWater", "occupantType": "OccupantBomb", "occupantId": 9}]]`

	checkEncoding(t, []encodingCase{
		{"without collections", GameState{Turn: 1, ActiveTeam: 1, Grid: grid}, `{
			"turn": 1, "inSuddenDeath": false, "activeTeam": 1, "grid": ` + gridJSON + `,
			"units": [], "bombs": [], "softBlocks": [], "turnCommands": []}`},
		{"with collections", GameState{
			Turn: 2, InSuddenDeath: true, ActiveTeam: 2, Grid: grid,
			Units:        map[UnitID]*Unit{17: {ID: 17, Type: Archetype{Name: "King"}, Position: pos, Team: 1, HP: 1}},
			Bombs:        map[BombID]*Bomb{70000: {ID: 70000, OwnerID: 17, Position: pos, Range: 2, Countdown: 3}},
			SoftBlocks:   map[int]*SoftBlock{4: {ID: 4, Position: pos}},
			TurnCommands: []TurnCommand{NewMoveCommand(17, pos)},
		}, `{
			"turn": 2, "inSuddenDeath": true, "activeTeam": 2, "grid": ` + gridJSON + `,
			"units": [{
				"id": 17, "type": "King", "position": {"x": 1, "y": 2},
				"speed": 0, "bombMaxRange": 0, "bombPower": 0, "maxBombCount": 0, "bombUsed": 0,
				"team": 1, "hp": 1, "skills": [], "hasMoved": false, "hasUsedSkill": false}],
			"bombs": [{"id": 70000, "ownerId": 17, "position": {"x": 1, "y": 2}, "range": 2, "countdown": 3}],
			"softBlocks": [{"id": 4, "position": {"x": 1, "y": 2}}],
			"turnCommands": [{"type": "move", "unitId": 17, "target": {"x": 1, "y": 2}}]}`},
	})
}

func TestConfigEncoding(t *testing.T) {
	checkEncoding(t, []encodingCase{
		{"game cfg omits test overrides", GameCfg{VSCpu: true, StagePreset: "Plain", P1Slots: []TeamSlot{{Archetype: "King", Role: RoleKing}}, P2Slots: []TeamSlot{}, GlobalSpeedOverride: 9}, `{
			"vsCpu": true, "stagePreset": "Plain",
			"p1Slots": [{"archetype": "King", "role": "King"}], "p2Slots": [],
			"maxTurns": 0, "allowResetTurn": false}`},
		{"game cfg nil slots", GameCfg{StagePreset: "Plain"}, `{
			"vsCpu": false, "stagePreset": "Plain", "p1Slots": [], "p2Slots": [],
			"maxTurns": 0, "allowResetTurn": false}`},
		{"stage preset omits layout", StagePreset{Name: "Plain", Description: "d", Width: 2, Height: 1, MaxTurns: 5, LayoutGrid: []string{"ab"}}, `{
			"name": "Plain", "description": "d", "width": 2, "height": 1, "maxTurns": 5}`},
	})
}

func TestEnumEncoding(t *testing.T) {
	checkEncoding(t, []encodingCase{
		{"cpu phase", TurnPhaseReady, `"TurnPhaseReady"`},
		{"unknown values", []any{TerrainType(99), OccupantType(99), UnitRole(99), CPUTurnPhase(99)},
			`["TerrainUnknown", "OccupantUnknown", "Unknown", "TurnPhaseUnknown"]`},
	})
}

func TestGameCfgDecodingIgnoresTestOverrides(t *testing.T) {
	var got GameCfg
	if err := json.Unmarshal([]byte(`{"GlobalSpeedOverride":9,"globalSpeedOverride":9}`), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.GlobalSpeedOverride != 0 {
		t.Errorf("GlobalSpeedOverride = %d, want 0", got.GlobalSpeedOverride)
	}
}
