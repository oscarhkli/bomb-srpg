package server

import (
	"bomb-srpg/engine"
	"bytes"
	"encoding/json/v2"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy"
)

type openAPIRig struct {
	t      *testing.T
	doc    *openapi3.T
	router routers.Router
	mux    *http.ServeMux
	hits   map[string]bool
	kinds  map[string]bool
}

func loadOpenAPISpec(t *testing.T) *openapi3.T {
	t.Helper()
	doc, err := openapi3.NewLoader().LoadFromFile("api/openapi.yaml")
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if err := doc.Validate(t.Context()); err != nil {
		t.Fatalf("validate spec: %v", err)
	}
	return doc
}

func newOpenAPIRig(t *testing.T) *openAPIRig {
	t.Helper()
	doc := loadOpenAPISpec(t)
	router, err := legacy.NewRouter(doc)
	if err != nil {
		t.Fatalf("build router: %v", err)
	}
	return &openAPIRig{t: t, doc: doc, router: router, hits: map[string]bool{}, kinds: map[string]bool{}}
}

func (r *openAPIRig) serve(opts ...Option) {
	r.t.Helper()
	r.mux = http.NewServeMux()
	RegisterRoutes(r.mux, NewHandler(NewServerStateManager(opts...)))
}

func (r *openAPIRig) check(method, path string, status int, body []byte) error {
	header := http.Header{}
	switch {
	case status == http.StatusNoContent:
	case status >= http.StatusBadRequest:
		header.Set("Content-Type", "text/plain")
	default:
		header.Set("Content-Type", "application/json")
	}
	return r.checkResponse(method, path, status, header, body)
}

func (r *openAPIRig) checkResponse(method, path string, status int, header http.Header, body []byte) error {
	req := httptest.NewRequest(method, path, nil)
	where := fmt.Sprintf("%s %s", method, req.URL.Path)
	route, params, err := r.router.FindRoute(req)
	if err != nil {
		return fmt.Errorf("%s: %w", where, err)
	}
	in := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req, PathParams: params, Route: route},
		Status:                 status,
		Header:                 header,
		Options:                &openapi3filter.Options{IncludeResponseStatus: true},
	}
	in.SetBodyBytes(body)
	if err := openapi3filter.ValidateResponse(r.t.Context(), in); err != nil {
		return fmt.Errorf("%s %d: %w", where, status, err)
	}
	if strings.HasPrefix(header.Get("Content-Type"), "application/json") {
		var decoded any
		if err := json.Unmarshal(body, &decoded); err != nil {
			return fmt.Errorf("%s %d: decode body: %w", where, status, err)
		}
		if err := r.walk("$", decoded); err != nil {
			return fmt.Errorf("%s %d: %w", where, status, err)
		}
	}
	r.hits[fmt.Sprintf("%s %s %d", method, route.Path, status)] = true
	return nil
}

func (r *openAPIRig) walk(at string, v any) error {
	switch v := v.(type) {
	case nil:
		return fmt.Errorf("JSON null at %s", at)
	case map[string]any:
		if _, isEvent := v["newHp"]; isEvent {
			if kind, ok := v["type"].(string); ok {
				r.kinds[kind] = true
			}
		}
		for _, key := range slices.Sorted(maps.Keys(v)) {
			if err := r.walk(at+"."+key, v[key]); err != nil {
				return err
			}
		}
	case []any:
		for i, item := range v {
			if err := r.walk(fmt.Sprintf("%s[%d]", at, i), item); err != nil {
				return err
			}
		}
	}
	return nil
}

func TestOpenAPIValidatorRejects(t *testing.T) {
	validState := `{"turn":1,"inSuddenDeath":false,"activeTeam":1,` +
		`"grid":[[{"type":"TerrainPlain","occupantType":"OccupantNone","occupantId":0}]],` +
		`"units":[],"bombs":[],"softBlocks":[],"turnCommands":[],"turnStartEvents":[]}`
	state := func(old, replacement string) string {
		if !strings.Contains(validState, old) {
			t.Fatalf("valid state lacks %q", old)
		}
		return strings.Replace(validState, old, replacement, 1)
	}
	const statePath = "/api/match-rooms/abc/match/state"

	rig := newOpenAPIRig(t)
	if err := rig.check("GET", statePath, 200, []byte(validState)); err != nil {
		t.Fatalf("valid state rejected: %v", err)
	}

	tests := []struct {
		name    string
		method  string
		path    string
		status  int
		body    string
		wantErr string
	}{
		{"null grid", "GET", statePath, 200, state(`"grid":[[{"type":"TerrainPlain","occupantType":"OccupantNone","occupantId":0}]]`, `"grid":null`), "/grid"},
		{"null units", "GET", statePath, 200, state(`"units":[]`, `"units":null`), "/units"},
		{"wrong enum value", "GET", statePath, 200, state(`"TerrainPlain"`, `"TerrainNope"`), "TerrainNope"},
		{"missing required property", "GET", statePath, 200, state(`"turn":1,`, ``), `"turn"`},
		{"unknown extra property", "GET", statePath, 200, state(`"turn":1,`, `"turn":1,"extra":true,`), `"extra"`},
		{"wrong integer type", "GET", statePath, 200, state(`"turn":1`, `"turn":"1"`), "/turn"},
		{"undeclared status", "GET", "/api/health", 404, "match not found", "status is not supported"},
		{"route absent from the spec", "GET", "/api/nope", 200, "{}", "no matching operation"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rig.check(tt.method, tt.path, tt.status, []byte(tt.body))
			if err == nil {
				t.Fatal("check() accepted the body, want an error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("check() error = %v, want it to contain %q", err, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.method+" ") || !strings.Contains(err.Error(), strings.SplitN(tt.path, "?", 2)[0]) {
				t.Errorf("check() error = %v, want it to name the method and path", err)
			}
		})
	}
}

func TestOpenAPIOperationsMatchRoutes(t *testing.T) {
	src, err := os.ReadFile("routes.go")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`mux\.HandleFunc\("([A-Z]+) (/[^"]*)"`)
	registered := map[string]bool{}
	for _, m := range re.FindAllSubmatch(src, -1) {
		registered[string(m[1])+" "+string(m[2])] = true
	}

	doc := loadOpenAPISpec(t)
	described := map[string]bool{}
	for path, item := range doc.Paths.Map() {
		for method := range item.Operations() {
			described[method+" "+path] = true
		}
	}

	for _, op := range slices.Sorted(maps.Keys(registered)) {
		if !described[op] {
			t.Errorf("%s is registered in routes.go but missing from the spec", op)
		}
	}
	for _, op := range slices.Sorted(maps.Keys(described)) {
		if !registered[op] {
			t.Errorf("%s is in the spec but not registered in routes.go", op)
		}
	}
	if len(registered) == 0 {
		t.Fatal("no routes parsed from routes.go")
	}
}

type call struct {
	method    string
	path      string
	auth      string
	body      []byte
	malformed bool
}

func (r *openAPIRig) do(c call, want int) []byte {
	r.t.Helper()
	return r.exec(c, want).Body.Bytes()
}

func (r *openAPIRig) exec(c call, want int) *httptest.ResponseRecorder {
	r.t.Helper()
	if !c.malformed {
		r.validateRequest(c)
	}
	req := httptest.NewRequest(c.method, c.path, bytes.NewReader(c.body))
	if c.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.auth != "" {
		req.Header.Set("Authorization", c.auth)
	}
	rec := httptest.NewRecorder()
	r.mux.ServeHTTP(rec, req)
	if rec.Code != want {
		r.t.Errorf("%s %s: status = %d, want %d, body %q", c.method, c.path, rec.Code, want, rec.Body.String())
	}
	if err := r.checkResponse(c.method, c.path, rec.Code, rec.Header(), rec.Body.Bytes()); err != nil {
		r.t.Errorf("response: %v", err)
	}
	return rec
}

func (r *openAPIRig) validateRequest(c call) {
	r.t.Helper()
	req := httptest.NewRequest(c.method, c.path, bytes.NewReader(c.body))
	if c.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	route, params, err := r.router.FindRoute(req)
	if err != nil {
		r.t.Errorf("%s %s: %v", c.method, c.path, err)
		return
	}
	in := &openapi3filter.RequestValidationInput{
		Request:    req,
		PathParams: params,
		Route:      route,
		Options:    &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc},
	}
	if err := openapi3filter.ValidateRequest(r.t.Context(), in); err != nil {
		r.t.Errorf("%s %s: request: %v", c.method, c.path, err)
	}
}

func marshalBody(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func decodeBody(t *testing.T, raw []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatalf("decode %q: %v", raw, err)
	}
}

func bearer(token string) string { return "Bearer " + token }

func (r *openAPIRig) newRoom() string {
	r.t.Helper()
	rec := r.exec(call{method: http.MethodPost, path: "/api/match-rooms"}, http.StatusCreated)
	var res struct {
		ID string `json:"id"`
	}
	decodeBody(r.t, rec.Body.Bytes(), &res)
	if got, want := rec.Header().Get("Location"), "/api/match-rooms/"+res.ID; got != want {
		r.t.Errorf("Location = %q, want %q", got, want)
	}
	return "/api/match-rooms/" + res.ID
}

func (r *openAPIRig) startMatch(room string, cfg engine.GameCfg) [2]string {
	r.t.Helper()
	body := marshalBody(r.t, struct {
		GameCfg engine.GameCfg `json:"gameCfg"`
	}{cfg})
	raw := r.do(call{method: http.MethodPost, path: room + "/match", body: body}, http.StatusCreated)
	var res struct {
		PlayerTokens [2]string `json:"playerTokens"`
	}
	decodeBody(r.t, raw, &res)
	return res.PlayerTokens
}

type openAPIState struct {
	ActiveTeam int   `json:"activeTeam"`
	Bombs      []any `json:"bombs"`
}

func (r *openAPIRig) state(room string) openAPIState {
	r.t.Helper()
	var gs openAPIState
	decodeBody(r.t, r.do(call{method: http.MethodGet, path: room + "/match/state"}, http.StatusOK), &gs)
	return gs
}

func (r *openAPIRig) resolveActiveTeam(room string, tokens [2]string) {
	r.t.Helper()
	token := tokens[r.state(room).ActiveTeam-1]
	r.do(call{method: http.MethodPost, path: room + "/match/resolve", auth: bearer(token)}, http.StatusOK)
}

func (r *openAPIRig) submit(room, token string, cmd engine.TurnCommand, want int) {
	r.t.Helper()
	r.do(call{method: http.MethodPost, path: room + "/match/turn-commands", auth: bearer(token), body: marshalBody(r.t, cmd)}, want)
}

func (r *openAPIRig) staticCases() {
	t := r.t
	t.Helper()
	room := r.newRoom()
	tokens := r.startMatch(room, validGameCfg())
	fighter := engine.NewUnitID(1, 1)
	move := marshalBody(t, engine.NewMoveCommand(fighter, engine.Coordinate{X: 2, Y: 8}))
	surrender := marshalBody(t, struct {
		TeamID int `json:"teamId"`
	}{1})
	createBody := marshalBody(t, struct {
		GameCfg engine.GameCfg `json:"gameCfg"`
	}{validGameCfg()})

	tokenRoutes := []struct {
		method    string
		suffix    string
		body      []byte
		wrongTeam bool
	}{
		{http.MethodPost, "/rematch", nil, false},
		{http.MethodDelete, "/match", nil, false},
		{http.MethodPost, "/match/turn-commands", move, true},
		{http.MethodPost, "/match/cpu-status/consume", nil, false},
		{http.MethodPost, "/match/reset", nil, true},
		{http.MethodPost, "/match/resolve", nil, true},
		{http.MethodPost, "/match/surrender", surrender, true},
	}
	for _, tr := range tokenRoutes {
		path := room + tr.suffix
		auths := []string{"", bearer("wrong"), "Token abc"}
		if tr.wrongTeam {
			auths = append(auths, bearer(tokens[1]))
		}
		for _, auth := range auths {
			r.do(call{method: tr.method, path: path, auth: auth, body: tr.body}, http.StatusUnauthorized)
		}
	}

	const gone = "/api/match-rooms/NOPE"
	allowedQuery := fmt.Sprintf("?unitId=%d&turnCmdType=move", fighter)
	notFound := []call{
		{method: http.MethodPost, path: gone + "/match", body: createBody},
		{method: http.MethodPost, path: gone + "/rematch", auth: bearer("x")},
		{method: http.MethodDelete, path: gone + "/match", auth: bearer("x")},
		{method: http.MethodGet, path: gone + "/match/state"},
		{method: http.MethodPost, path: gone + "/match/turn-commands", auth: bearer("x"), body: move},
		{method: http.MethodPost, path: gone + "/match/cpu-status/consume", auth: bearer("x")},
		{method: http.MethodPost, path: gone + "/match/reset", auth: bearer("x")},
		{method: http.MethodPost, path: gone + "/match/resolve", auth: bearer("x")},
		{method: http.MethodPost, path: gone + "/match/surrender", auth: bearer("x"), body: surrender},
		{method: http.MethodGet, path: gone + "/match/config"},
		{method: http.MethodGet, path: gone + "/match/allowed-tiles" + allowedQuery},
	}
	for _, c := range notFound {
		r.do(c, http.StatusNotFound)
	}

	empty := r.newRoom()
	badRequests := []call{
		{method: http.MethodPost, path: empty + "/match", body: []byte("{"), malformed: true},
		{method: http.MethodPost, path: empty + "/match", body: marshalBody(t, struct {
			GameCfg engine.GameCfg `json:"gameCfg"`
		}{engine.GameCfg{StagePreset: "Nope", P1Slots: []engine.TeamSlot{}, P2Slots: []engine.TeamSlot{}}})},
		{method: http.MethodPost, path: room + "/match/turn-commands", auth: bearer(tokens[0]), body: []byte("{"), malformed: true},
		{method: http.MethodPost, path: room + "/match/turn-commands", auth: bearer(tokens[0]), body: marshalBody(t, engine.NewMoveCommand(0, engine.Coordinate{X: 2, Y: 8}))},
		{method: http.MethodPost, path: room + "/match/turn-commands", auth: bearer(tokens[0]), body: marshalBody(t, engine.NewMoveCommand(engine.NewUnitID(3, 1), engine.Coordinate{X: 2, Y: 8}))},
		{method: http.MethodPost, path: room + "/match/surrender", auth: bearer(tokens[0]), body: []byte("{"), malformed: true},
		{method: http.MethodPost, path: room + "/match/surrender", auth: bearer(tokens[0]), body: []byte(`{"teamId":3}`)},
		{method: http.MethodGet, path: room + "/match/allowed-tiles?turnCmdType=move", malformed: true},
		{method: http.MethodGet, path: room + "/match/allowed-tiles?unitId=abc&turnCmdType=move", malformed: true},
	}
	for _, c := range badRequests {
		r.do(c, http.StatusBadRequest)
	}

	conflicts := []call{
		{method: http.MethodPost, path: room + "/match", body: createBody},
		{method: http.MethodPost, path: room + "/match/turn-commands", auth: bearer(tokens[0]), body: marshalBody(t, engine.NewMoveCommand(fighter, engine.Coordinate{X: 0, Y: 0}))},
		{method: http.MethodDelete, path: room + "/match", auth: bearer(tokens[0])},
		{method: http.MethodGet, path: room + "/match/allowed-tiles?unitId=200&turnCmdType=move"},
	}
	for _, c := range conflicts {
		r.do(c, http.StatusConflict)
	}
}

func (r *openAPIRig) lifecycle() {
	t := r.t
	t.Helper()
	r.do(call{method: http.MethodGet, path: "/api/health"}, http.StatusOK)
	r.do(call{method: http.MethodGet, path: "/api/catalog"}, http.StatusOK)

	room := r.newRoom()
	cfg := validGameCfg()
	cfg.StagePreset = "Divided"
	cfg.MaxTurns = 20
	cfg.AllowResetTurn = true
	tokens := r.startMatch(room, cfg)
	fighter := engine.NewUnitID(1, 1)

	r.do(call{method: http.MethodGet, path: room + "/match/config"}, http.StatusOK)
	r.do(call{method: http.MethodGet, path: fmt.Sprintf("%s/match/allowed-tiles?unitId=%d&turnCmdType=move", room, fighter)}, http.StatusOK)
	r.do(call{method: http.MethodPost, path: room + "/rematch", auth: bearer(tokens[0])}, http.StatusCreated)

	r.submit(room, tokens[0], engine.NewMoveCommand(fighter, engine.Coordinate{X: 2, Y: 8}), http.StatusOK)
	r.submit(room, tokens[0], engine.NewMoveCommand(fighter, engine.Coordinate{X: 3, Y: 8}), http.StatusConflict)
	r.do(call{method: http.MethodPost, path: room + "/match/reset", auth: bearer(tokens[0])}, http.StatusNoContent)
	r.submit(room, tokens[0], engine.NewMoveCommand(fighter, engine.Coordinate{X: 2, Y: 8}), http.StatusOK)
	r.resolveActiveTeam(room, tokens)
	r.resolveActiveTeam(room, tokens)

	r.submit(room, tokens[0], engine.NewMoveCommand(fighter, engine.Coordinate{X: 2, Y: 6}), http.StatusOK)
	r.submit(room, tokens[0], engine.NewPlaceBombCommand(fighter, engine.Coordinate{X: 2, Y: 5}), http.StatusOK)
	if len(r.state(room).Bombs) == 0 {
		t.Error("bombs is empty after a bomb was placed")
	}
	r.resolveActiveTeam(room, tokens)
	for range 3 {
		r.resolveActiveTeam(room, tokens)
	}
	r.do(call{method: http.MethodDelete, path: room + "/match", auth: bearer(tokens[0])}, http.StatusConflict)
	r.resolveActiveTeam(room, tokens)

	r.do(call{method: http.MethodPost, path: room + "/match/surrender", auth: bearer(tokens[1]), body: []byte(`{"teamId":2}`)}, http.StatusOK)
	r.do(call{method: http.MethodDelete, path: room + "/match", auth: bearer(tokens[0])}, http.StatusNoContent)
	for _, suffix := range []string{"/match/state", "/match/config", fmt.Sprintf("/match/allowed-tiles?unitId=%d&turnCmdType=move", fighter)} {
		r.do(call{method: http.MethodGet, path: room + suffix}, http.StatusNotFound)
	}
	r.do(call{method: http.MethodDelete, path: room + "/match", auth: bearer(tokens[0])}, http.StatusNoContent)
	r.do(call{method: http.MethodPost, path: room + "/rematch", auth: bearer(tokens[0])}, http.StatusCreated)
	r.state(room)
}

func (r *openAPIRig) vsCPU() {
	t := r.t
	t.Helper()
	cpuUnit := engine.NewUnitID(2, 1)
	hold := holdCPUTurn(t)
	r.serve(
		WithDecider(func(*engine.GameState) ([]engine.TurnCommand, error) {
			return []engine.TurnCommand{engine.NewMoveCommand(cpuUnit, engine.Coordinate{X: 5, Y: 1})}, nil
		}),
		WithSpawner(hold.spawn),
	)
	room := r.newRoom()
	tokens := r.startMatch(room, vsCpuGameCfg())
	r.do(call{method: http.MethodPost, path: room + "/match/resolve", auth: bearer(tokens[0])}, http.StatusOK)

	consume := call{method: http.MethodPost, path: room + "/match/cpu-status/consume", auth: bearer(tokens[0])}
	var status struct {
		TurnPhase string `json:"turnPhase"`
	}
	phase := func() string {
		decodeBody(t, r.do(consume, http.StatusOK), &status)
		return status.TurnPhase
	}
	if got := phase(); got != "TurnPhasePlanning" {
		t.Errorf("first consume phase = %q, want TurnPhasePlanning", got)
	}
	r.submit(room, tokens[1], engine.NewMoveCommand(cpuUnit, engine.Coordinate{X: 5, Y: 1}), http.StatusConflict)
	r.do(call{method: http.MethodPost, path: room + "/match/reset", auth: bearer(tokens[1])}, http.StatusConflict)
	r.do(call{method: http.MethodPost, path: room + "/match/resolve", auth: bearer(tokens[1])}, http.StatusConflict)

	hold.runOutsideRoomLock()
	if got := phase(); got != "TurnPhaseReady" {
		t.Errorf("consume after the CPU turn phase = %q, want TurnPhaseReady", got)
	}
	if got := phase(); got != "TurnPhaseIdle" {
		t.Errorf("second consume phase = %q, want TurnPhaseIdle", got)
	}
}

func (r *openAPIRig) assertComplete() {
	t := r.t
	t.Helper()
	for _, path := range r.doc.Paths.Keys() {
		for method, op := range r.doc.Paths.Value(path).Operations() {
			for _, code := range op.Responses.Keys() {
				if !r.hits[method+" "+path+" "+code] {
					t.Errorf("%s %s declares %s but no run produced it", method, path, code)
				}
			}
		}
	}

	var declared []string
	for _, v := range r.doc.Components.Schemas["GameEvtType"].Value.Enum {
		if kind, ok := v.(string); ok {
			declared = append(declared, kind)
		}
	}
	seen := slices.Sorted(maps.Keys(r.kinds))
	if !slices.Equal(slices.Sorted(slices.Values(declared)), seen) {
		t.Errorf("event kinds seen = %v, want the spec enum %v", seen, declared)
	}

	for _, name := range slices.Sorted(maps.Keys(r.doc.Components.Schemas)) {
		schema := r.doc.Components.Schemas[name].Value
		if !schema.Type.Is("object") {
			continue
		}
		want := slices.Sorted(maps.Keys(schema.Properties))
		if name == "GameEvent" {
			want = []string{"countdown", "newHp", "type", "unitId"}
		}
		if got := slices.Sorted(slices.Values(schema.Required)); !slices.Equal(got, want) {
			t.Errorf("schema %s required = %v, want %v", name, got, want)
		}
	}
}

func TestOpenAPIWire(t *testing.T) {
	rig := newOpenAPIRig(t)
	rig.serve()
	rig.staticCases()
	rig.lifecycle()
	rig.vsCPU()
	rig.assertComplete()
}
