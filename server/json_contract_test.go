package server

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"bomb-srpg/server/api"
)

func TestSurrenderRequestDecoding(t *testing.T) {
	const (
		formatMsg = "Invalid surrenderRequest format"
		teamMsg   = "invalid game config: team must be 1 or 2"
	)
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantMsg    string
	}{
		{"exact keys", `{"teamId":1}`, http.StatusOK, ""},
		{"keys match case-sensitively", `{"TeamID":1}`, http.StatusBadRequest, teamMsg},
		{"unknown member ignored", `{"teamId":1,"extra":[true]}`, http.StatusOK, ""},
		{"duplicate key", `{"teamId":3,"teamId":1}`, http.StatusBadRequest, formatMsg},
		{"invalid UTF-8 in unknown member", "{\"teamId\":1,\"x\":\"\xff\"}", http.StatusBadRequest, formatMsg},
		{"invalid UTF-8 body", "\xff", http.StatusBadRequest, formatMsg},
		{"byte order mark before value", "\xef\xbb\xbf{\"teamId\":1}", http.StatusBadRequest, formatMsg},
		{"trailing data after value", `{"teamId":1} garbage`, http.StatusBadRequest, formatMsg},
		{"second value after value", `{"teamId":1}{"teamId":3}`, http.StatusBadRequest, formatMsg},
		{"truncated", `{"teamId":`, http.StatusBadRequest, formatMsg},
		{"wrong member type", `{"teamId":"1"}`, http.StatusBadRequest, formatMsg},
		{"fractional number", `{"teamId":1.5}`, http.StatusBadRequest, formatMsg},
		{"integer written with a fraction", `{"teamId":1.0}`, http.StatusBadRequest, formatMsg},
		{"integer written with an exponent", `{"teamId":1e0}`, http.StatusBadRequest, formatMsg},
		{"integer written as 0.1e1", `{"teamId":0.1e1}`, http.StatusBadRequest, formatMsg},
		{"exponent that is out of range", `{"teamId":1e2}`, http.StatusBadRequest, formatMsg},
		{"exponent beyond float64", `{"teamId":1e400}`, http.StatusBadRequest, formatMsg},
		{"integer of one million", `{"teamId":1000000}`, http.StatusBadRequest, teamMsg},
		{"integer above 2^53", `{"teamId":9007199254740993}`, http.StatusBadRequest, teamMsg},
		{"negative number", `{"teamId":-1}`, http.StatusBadRequest, teamMsg},
		{"null member", `{"teamId":null}`, http.StatusBadRequest, teamMsg},
		{"null body", `null`, http.StatusBadRequest, teamMsg},
		{"array body", `[1]`, http.StatusBadRequest, formatMsg},
		{"empty body", ``, http.StatusBadRequest, formatMsg},
		{"whitespace-only body", " \n\t", http.StatusBadRequest, formatMsg},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID, playerTokens, _, h := createTestRoomWithMatch(t)
			rr := doRequest(t, h, "POST", "/api/match-rooms/"+roomID+"/match/surrender", tt.body, playerTokens[0])

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body %q", rr.Code, tt.wantStatus, rr.Body.String())
			}
			if tt.wantStatus != http.StatusOK {
				if !strings.Contains(rr.Body.String(), tt.wantMsg) {
					t.Errorf("body = %q, want it to contain %q", rr.Body.String(), tt.wantMsg)
				}
				return
			}
			var gameEvents []api.GameEvent
			if err := json.UnmarshalRead(rr.Body, &gameEvents); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if len(gameEvents) != 1 || wireWinner(gameEvents[0]) != 2 {
				t.Errorf("gameEvents = %+v, want one event with WinnerTeamID 2", gameEvents)
			}
		})
	}
}

func TestRequestBodyContentTypeIgnored(t *testing.T) {
	const cfg = `{"gameCfg":{"stagePreset":"Plain","maxTurns":10,` +
		`"p1Slots":[{"archetype":"King","role":"King"},{"archetype":"Fighter","role":"Normal"}],` +
		`"p2Slots":[{"archetype":"King","role":"King"},{"archetype":"Witch","role":"Normal"}]}}`
	tests := []struct {
		name       string
		path       string
		body       string
		wantStatus int
	}{
		{"create match", "/match", cfg, http.StatusCreated},
		{"submit turn command", "/match/turn-commands", `{"type":"move","unitId":16,"target":{"x":4,"y":7}}`, http.StatusOK},
		{"surrender", "/match/surrender", `{"teamId":1}`, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				roomID string
				h      *Handler
				header http.Header
			)
			if tt.path == "/match" {
				roomID, _, h = newTestRoom(t)
				header = http.Header{}
			} else {
				var tokens [2]string
				roomID, tokens, _, h = createTestRoomWithMatch(t)
				header = bearerHeader(tokens[0])
			}
			header.Set("Content-Type", "text/plain")

			rr := serveRequest(t, h, "POST", "/api/match-rooms/"+roomID+tt.path, tt.body, header)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body %q", rr.Code, tt.wantStatus, rr.Body.String())
			}
		})
	}
}

func TestMatchStateWireShape(t *testing.T) {
	roomID, _, _, h := createTestRoomWithMatch(t)
	rr := doRequest(t, h, "GET", "/api/match-rooms/"+roomID+"/match/state", "", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	var state map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	tests := []struct {
		key  string
		want string
	}{
		{"turn", "number"},
		{"activeTeam", "number"},
		{"inSuddenDeath", "bool"},
		{"grid", "array"},
		{"units", "array"},
		{"bombs", "array"},
		{"softBlocks", "array"},
		{"turnCommands", "array"},
		{"turnStartEvents", "array"},
	}
	if len(state) != len(tests) {
		t.Errorf("state has %d keys, want %d", len(state), len(tests))
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			var got string
			switch state[tt.key].(type) {
			case []any:
				got = "array"
			case float64:
				got = "number"
			case bool:
				got = "bool"
			default:
				got = fmt.Sprintf("%T", state[tt.key])
			}
			if got != tt.want {
				t.Errorf("%s is %s, want %s", tt.key, got, tt.want)
			}
		})
	}
}
