package server

import (
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"
)

const turnCommandsPath = "/match/turn-commands"

func TestTurnCommandWireRows(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		noToken    bool
		wantStatus int
		want       string
	}{
		{"bomb", "{\"type\":\"placeBomb\",\"unitId\":16,\"target\":{\"x\":4,\"y\":6}}", false, 200, "[{\"bombId\":268500993,\"countdown\":5,\"position\":{\"x\":4,\"y\":6},\"range\":2,\"type\":\"bombPlaced\",\"unitId\":16}]\n"},
		{"TYPE value cap", "{\"type\":\"Move\",\"unitId\":16,\"target\":{\"x\":4,\"y\":7}}", false, 409, "invalid turn command: unsupported command type: Move\n"},
		{"UnitId cap", "{\"type\":\"move\",\"UnitId\":16,\"target\":{\"x\":4,\"y\":7}}", false, 400, "invalid game config\n"},
		{"Target cap", "{\"type\":\"move\",\"unitId\":16,\"Target\":{\"x\":4,\"y\":7}}", false, 409, "invalid turn command: target out of move range\n"},
		{"dup unitId", "{\"type\":\"move\",\"unitId\":16,\"unitId\":16,\"target\":{\"x\":4,\"y\":7}}", false, 400, "Invalid turnCommand format\n"},
		{"dup target key", "{\"type\":\"move\",\"unitId\":16,\"target\":{\"x\":4,\"x\":4,\"y\":7}}", false, 400, "Invalid turnCommand format\n"},
		{"array", "[]", false, 400, "Invalid turnCommand format\n"},
		{"string", "\"move\"", false, 400, "Invalid turnCommand format\n"},
		{"number", "1", false, 400, "Invalid turnCommand format\n"},
		{"empty obj", "{}", false, 409, "invalid turn command: unsupported command type: \n"},
		{"unitId string", "{\"type\":\"move\",\"unitId\":\"16\",\"target\":{\"x\":4,\"y\":7}}", false, 400, "Invalid turnCommand format\n"},
		{"unitId null", "{\"type\":\"move\",\"unitId\":null,\"target\":{\"x\":4,\"y\":7}}", false, 400, "invalid game config\n"},
		{"unitId missing", "{\"type\":\"move\",\"target\":{\"x\":4,\"y\":7}}", false, 400, "invalid game config\n"},
		{"target missing", "{\"type\":\"move\",\"unitId\":16}", false, 409, "invalid turn command: target out of move range\n"},
		{"target null", "{\"type\":\"move\",\"unitId\":16,\"target\":null}", false, 409, "invalid turn command: target out of move range\n"},
		{"target string", "{\"type\":\"move\",\"unitId\":16,\"target\":\"x\"}", false, 400, "Invalid turnCommand format\n"},
		{"target x string", "{\"type\":\"move\",\"unitId\":16,\"target\":{\"x\":\"4\",\"y\":7}}", false, 400, "Invalid turnCommand format\n"},
		{"target y missing", "{\"type\":\"move\",\"unitId\":16,\"target\":{\"x\":4}}", false, 409, "invalid turn command: target out of move range\n"},
		{"target extra key", "{\"type\":\"move\",\"unitId\":16,\"target\":{\"x\":4,\"y\":7,\"z\":1}}", false, 200, "[{\"from\":{\"x\":4,\"y\":8},\"to\":{\"x\":4,\"y\":7},\"type\":\"unitMoved\",\"unitId\":16}]\n"},
		{"extra key top", "{\"type\":\"move\",\"unitId\":16,\"extra\":1,\"target\":{\"x\":4,\"y\":7}}", false, 200, "[{\"from\":{\"x\":4,\"y\":8},\"to\":{\"x\":4,\"y\":7},\"type\":\"unitMoved\",\"unitId\":16}]\n"},
		{"type missing", "{\"unitId\":16,\"target\":{\"x\":4,\"y\":7}}", false, 409, "invalid turn command: unsupported command type: \n"},
		{"type null", "{\"type\":null,\"unitId\":16,\"target\":{\"x\":4,\"y\":7}}", false, 409, "invalid turn command: unsupported command type: \n"},
		{"type number", "{\"type\":1,\"unitId\":16,\"target\":{\"x\":4,\"y\":7}}", false, 400, "Invalid turnCommand format\n"},
		{"type empty", "{\"type\":\"\",\"unitId\":16,\"target\":{\"x\":4,\"y\":7}}", false, 409, "invalid turn command: unsupported command type: \n"},
		{"type fly no target", "{\"type\":\"fly\",\"unitId\":16}", false, 409, "invalid turn command: unsupported command type: fly\n"},
		{"type fly unitId string", "{\"type\":\"fly\",\"unitId\":\"16\",\"target\":{\"x\":4,\"y\":7}}", false, 400, "Invalid turnCommand format\n"},
		{"type fly target string", "{\"type\":\"fly\",\"unitId\":16,\"target\":\"x\"}", false, 400, "Invalid turnCommand format\n"},
		{"type fly target fraction", "{\"type\":\"fly\",\"unitId\":16,\"target\":{\"x\":4.5,\"y\":7}}", false, 400, "Invalid turnCommand format\n"},
		{"type missing no token", "{\"unitId\":16,\"target\":{\"x\":4,\"y\":7}}", true, 409, "invalid turn command: unsupported command type: \n"},
		{"move no target no token", "{\"type\":\"move\",\"unitId\":16}", true, 401, "invalid player token\n"},
		{"move unitId string no token", "{\"type\":\"move\",\"unitId\":\"16\",\"target\":{\"x\":4,\"y\":7}}", true, 400, "Invalid turnCommand format\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, playerTokens, _, h := createTestRoomWithMatch(t)
			header := bearerHeader(playerTokens[0])
			if tt.noToken {
				header = nil
			}

			rr := serveRequest(t, h, "POST", "/api/match-rooms/"+roomID+turnCommandsPath, tt.body, header)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
			if got := rr.Body.String(); got != tt.want {
				t.Errorf("body = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTurnCommandKindsHaveRoundTrip(t *testing.T) {
	doc := loadOpenAPISpec(t)
	mapping := unionMapping(t, doc, "TurnCommand")
	tests := map[string]struct {
		body      string
		wantEvent string
	}{
		"move":      {`{"type":"move","unitId":16,"target":{"x":4,"y":7}}`, "unitMoved"},
		"placeBomb": {`{"type":"placeBomb","unitId":16,"target":{"x":4,"y":6}}`, "bombPlaced"},
	}

	if got, want := slices.Sorted(maps.Keys(mapping)), slices.Sorted(maps.Keys(tests)); !slices.Equal(got, want) {
		t.Fatalf("TurnCommand mapping kinds = %v, want %v", got, want)
	}
	inMapping := make(map[string]bool, len(mapping))
	for _, ref := range mapping {
		inMapping[ref] = true
	}
	for _, member := range doc.Components.Schemas["TurnCommand"].Value.OneOf {
		if !inMapping[member.Ref] {
			t.Errorf("TurnCommand oneOf member %s is missing from the discriminator mapping", member.Ref)
		}
	}

	for kind, tt := range tests {
		t.Run(kind, func(t *testing.T) {
			roomID, playerTokens, _, h := createTestRoomWithMatch(t)

			rr := serveRequest(t, h, "POST", "/api/match-rooms/"+roomID+turnCommandsPath, tt.body, bearerHeader(playerTokens[0]))

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200, body %q", rr.Code, rr.Body.String())
			}
			if want := `"type":"` + tt.wantEvent + `"`; !strings.Contains(rr.Body.String(), want) {
				t.Errorf("body = %q, want it to contain %s", rr.Body.String(), want)
			}
		})
	}
}

func TestOpenAPITurnCommandsUnion(t *testing.T) {
	const statePath = "/api/match-rooms/abc/match/state"
	state := func(commands string) []byte {
		return []byte(`{"turn":1,"inSuddenDeath":false,"activeTeam":1,` +
			`"grid":[[{"type":"TerrainPlain","occupantType":"OccupantNone","occupantId":0}]],` +
			`"units":[],"bombs":[],"softBlocks":[],"turnCommands":[` + commands + `],"turnStartEvents":[]}`)
	}
	const (
		move      = `{"target":{"x":1,"y":2},"type":"move","unitId":17}`
		placeBomb = `{"target":{"x":1,"y":3},"type":"placeBomb","unitId":17}`
	)
	tests := []struct {
		name     string
		commands string
		wantErr  bool
	}{
		{"a move and a placeBomb", move + "," + placeBomb, false},
		{"an unknown type", `{"target":{"x":1,"y":2},"type":"fly","unitId":17}`, true},
		{"an extra key", `{"target":{"x":1,"y":2},"type":"move","unitId":17,"extra":1}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newOpenAPIRig(t).check("GET", statePath, 200, state(tt.commands))

			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Errorf("check error = %v, want error %v", err, tt.wantErr)
			}
		})
	}
}
