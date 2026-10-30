package server

import (
	"bomb-srpg/engine"
	"fmt"
	"reflect"
	"testing"
)

func validGameCfg() engine.GameCfg {
	return engine.GameCfg{
		StagePreset: "Plain",
		P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
		P2Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Witch", Role: engine.RoleNormal}},
		MaxTurns:    10,
	}
}

func assertGameCfgSynced(t *testing.T, room *MatchRoom) {
	t.Helper()
	if room.GameCfg == nil {
		t.Fatal("Expected room.GameCfg to be set")
	}
	if !reflect.DeepEqual(*room.GameCfg, room.Match.GameCfg) {
		t.Errorf("Expected MatchRoom and Match to have equal GameCfg values, MatchRoom %+v vs Match %+v", *room.GameCfg, room.Match.GameCfg)
	}
}

func createTestRoomWithCfg(t *testing.T, gameCfg engine.GameCfg) (string, [2]string, *ServerStateManager) {
	t.Helper()
	s := NewServerStateManager()
	roomID, _ := s.CreateMatchRoom()
	tokens, _ := s.CreateMatch(roomID, gameCfg)
	return roomID, tokens, s
}

func createTestRoom(t *testing.T) (string, [2]string, *ServerStateManager) {
	t.Helper()
	return createTestRoomWithCfg(t, validGameCfg())
}

func mustRoom(t *testing.T, s *ServerStateManager, roomID string) *MatchRoom {
	t.Helper()
	roomVal, ok := s.Rooms.Load(roomID)
	if !ok {
		t.Fatalf("Room %s not found", roomID)
	}
	room, ok := roomVal.(*MatchRoom)
	if !ok {
		t.Fatalf("Room %s has type %T, want *MatchRoom", roomID, roomVal)
	}
	return room
}

func mustCreateMatch(t *testing.T, s *ServerStateManager, roomID string, gameCfg engine.GameCfg) [2]string {
	t.Helper()
	tokens, err := s.CreateMatch(roomID, gameCfg)
	if err != nil {
		t.Fatalf("CreateMatch() error = %v", err)
	}
	return tokens
}

func startTurnForTest(t *testing.T, s *ServerStateManager, roomID, token string) {
	t.Helper()
	if _, _, err := s.StartTurn(roomID, token); err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}
}

func freeNeighbour(t *testing.T, gs *engine.GameState, pos engine.Coordinate) engine.Coordinate {
	t.Helper()
	for _, step := range []engine.Coordinate{{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0}} {
		target := engine.Coordinate{X: pos.X + step.X, Y: pos.Y + step.Y}
		if gs.IsLandingLegal(target, engine.OccupantBomb) == nil {
			return target
		}
	}
	t.Fatalf("no free neighbour next to %v", pos)
	return engine.Coordinate{}
}

func TestMapError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
		wantMsg  string
	}{
		{"room not found", ErrRoomNotFound, 404, "room not found"},
		{"match not found", ErrMatchNotFound, 404, "match not found"},
		{"match exists", ErrMatchExists, 409, "match already exists"},
		{"cpu turn in progress", ErrCPUTurnInProgress, 409, "cpu turn in progress"},
		{"invalid config", ErrInvalidConfig, 400, "invalid game config"},
		{"invalid player token", ErrInvalidToken, 401, "invalid player token"},

		{"invalid stage preset", engine.ErrInvalidStagePreset, 400, "invalid stage preset"},
		{"invalid team size", engine.ErrInvalidTeamSize, 400, "invalid team size"},
		{"missing king", engine.ErrMissingKing, 400, "missing king"},
		{"invalid stage layout", engine.ErrInvalidStageLayout, 400, "invalid stage layout"},
		{"invalid terrain", engine.ErrInvalidTerrain, 400, "invalid terrain"},
		{"unknown archetype", engine.ErrUnknownArchetype, 400, "unknown archetype"},

		{"unit not found", engine.ErrUnitNotFound, 409, "unit not found"},
		{"unit dead", engine.ErrUnitDead, 409, "unit is dead"},
		{"not active team", engine.ErrNotActiveTeam, 409, "not active team"},
		{"already moved", engine.ErrAlreadyMoved, 409, "unit already moved this turn"},
		{"already used skill", engine.ErrAlreadyUsedSkill, 409, "unit already used skill this turn"},
		{"out of move range", engine.ErrOutOfMoveRange, 409, "target out of move range"},
		{"out of bomb range", engine.ErrOutOfBombRange, 409, "target out of bomb range"},
		{"cell occupied", engine.ErrCellOccupied, 409, "cell occupied"},
		{"out of bombs", engine.ErrOutOfBombs, 409, "out of bombs"},
		{"unsupported command", engine.ErrUnsupportedCommand, 409, "unsupported command type"},
		{"invalid landing", engine.ErrInvalidLanding, 409, "invalid landing position"},

		{"desynced", fmt.Errorf("%w: unit %#x desynced at %v", engine.ErrDesynced, 1, engine.Coordinate{}), 409, "desynced: unit 0x1 desynced at {0 0}"},
		{"out of bounds", fmt.Errorf("%w: unit %#x out of bounds", engine.ErrOutOfBounds, 1), 409, "out of bounds: unit 0x1 out of bounds"},

		{"unknown", fmt.Errorf("something else"), 500, "internal error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, msg := mapError(tt.err)
			if code != tt.wantCode || msg != tt.wantMsg {
				t.Errorf("got (%d, %q) want (%d, %q)", code, msg, tt.wantCode, tt.wantMsg)
			}
		})
	}
}
