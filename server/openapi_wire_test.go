package server

import (
	"bomb-srpg/engine"
	"bomb-srpg/room"
	"bomb-srpg/server/api"
	"bytes"
	"encoding/json/v2"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"reflect"
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
	events map[string]string
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
	return &openAPIRig{t: t, doc: doc, router: router, hits: map[string]bool{}, kinds: map[string]bool{}, events: unionMapping(t, doc, "GameEvent")}
}

func unionMapping(t *testing.T, doc *openapi3.T, schema string) map[string]string {
	t.Helper()
	union := doc.Components.Schemas[schema]
	if union == nil || union.Value.Discriminator == nil {
		t.Fatalf("%s has no discriminator", schema)
	}
	mapping := make(map[string]string, len(union.Value.Discriminator.Mapping))
	for kind, ref := range union.Value.Discriminator.Mapping {
		mapping[kind] = ref.Ref
	}
	return mapping
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
		if kind, ok := v["type"].(string); ok {
			if _, isEvent := r.events[kind]; isEvent {
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
		{"event with another kind's key", "GET", statePath, 200, state(`"turnStartEvents":[]`, `"turnStartEvents":[{"type":"matchEnded","winnerTeamId":1,"newHp":3}]`), `"newHp"`},
		{"event with an unknown type", "GET", statePath, 200, state(`"turnStartEvents":[]`, `"turnStartEvents":[{"type":"mystery"}]`), "mystery"},
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

func (r *openAPIRig) startMatch(room string, cfg api.GameCfg) [2]string {
	r.t.Helper()
	body := marshalBody(r.t, api.CreateMatchRequest{GameCfg: cfg})
	raw := r.do(call{method: http.MethodPost, path: room + "/match", body: body}, http.StatusCreated)
	var res api.CreateMatchResponse
	decodeBody(r.t, raw, &res)
	return [2]string(res.PlayerTokens)
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

func (r *openAPIRig) submit(room, token string, cmd api.TurnCommand, want int) {
	r.t.Helper()
	r.do(call{method: http.MethodPost, path: room + "/match/turn-commands", auth: bearer(token), body: marshalBody(r.t, cmd)}, want)
}

func (r *openAPIRig) staticCases() {
	t := r.t
	t.Helper()
	room := r.newRoom()
	tokens := r.startMatch(room, validWireGameCfg())
	fighter := engine.NewUnitID(1, 1)
	move := marshalBody(t, wireMoveCmd(fighter, 2, 8))
	surrender := marshalBody(t, api.SurrenderRequest{TeamID: 1})
	createBody := marshalBody(t, api.CreateMatchRequest{GameCfg: validWireGameCfg()})

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
	unknownArchetype := validWireGameCfg()
	unknownArchetype.P1Slots[1].Archetype = "Nope"
	badRequests := []call{
		{method: http.MethodPost, path: empty + "/match", body: marshalBody(t, api.CreateMatchRequest{GameCfg: unknownArchetype})},
		{method: http.MethodPost, path: empty + "/match", body: []byte("{"), malformed: true},
		{method: http.MethodPost, path: empty + "/match", body: marshalBody(t, api.CreateMatchRequest{GameCfg: api.GameCfg{StagePreset: "Nope", P1Slots: []api.TeamSlot{}, P2Slots: []api.TeamSlot{}}})},
		{method: http.MethodPost, path: room + "/match/turn-commands", auth: bearer(tokens[0]), body: []byte("{"), malformed: true},
		{method: http.MethodPost, path: room + "/match/turn-commands", auth: bearer(tokens[0]), body: marshalBody(t, wireMoveCmd(0, 2, 8))},
		{method: http.MethodPost, path: room + "/match/turn-commands", auth: bearer(tokens[0]), body: marshalBody(t, wireMoveCmd(engine.NewUnitID(3, 1), 2, 8))},
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
		{method: http.MethodPost, path: room + "/match/turn-commands", auth: bearer(tokens[0]), body: marshalBody(t, wireMoveCmd(fighter, 0, 0))},
		{method: http.MethodDelete, path: room + "/match", auth: bearer(tokens[0])},
		{method: http.MethodGet, path: room + "/match/allowed-tiles?unitId=200&turnCmdType=move"},
		{method: http.MethodGet, path: room + "/match/allowed-tiles?unitId=17&turnCmdType=nope", malformed: true},
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
	cfg := validWireGameCfg()
	cfg.StagePreset = "Divided"
	cfg.MaxTurns = 20
	cfg.AllowResetTurn = true
	tokens := r.startMatch(room, cfg)
	fighter := engine.NewUnitID(1, 1)

	r.do(call{method: http.MethodGet, path: room + "/match/config"}, http.StatusOK)
	r.do(call{method: http.MethodGet, path: fmt.Sprintf("%s/match/allowed-tiles?unitId=%d&turnCmdType=move", room, fighter)}, http.StatusOK)
	r.do(call{method: http.MethodPost, path: room + "/rematch", auth: bearer(tokens[0])}, http.StatusCreated)

	r.submit(room, tokens[0], wireMoveCmd(fighter, 2, 8), http.StatusOK)
	r.submit(room, tokens[0], wireMoveCmd(fighter, 3, 8), http.StatusConflict)
	r.do(call{method: http.MethodPost, path: room + "/match/reset", auth: bearer(tokens[0])}, http.StatusNoContent)
	r.submit(room, tokens[0], wireMoveCmd(fighter, 2, 8), http.StatusOK)
	r.resolveActiveTeam(room, tokens)
	r.resolveActiveTeam(room, tokens)

	r.submit(room, tokens[0], wireMoveCmd(fighter, 2, 6), http.StatusOK)
	r.submit(room, tokens[0], wirePlaceBombCmd(fighter, 2, 5), http.StatusOK)
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
	cfg := validWireGameCfg()
	cfg.VsCPU = true
	tokens := r.startMatch(room, cfg)
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
	r.submit(room, tokens[1], wireMoveCmd(cpuUnit, 5, 1), http.StatusConflict)
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

	r.assertEventKinds()
	r.assertEnums()
	r.assertClosedAndNonNullable()
	r.assertGuards()

	for _, name := range slices.Sorted(maps.Keys(r.doc.Components.Schemas)) {
		schema := r.doc.Components.Schemas[name].Value
		if !schema.Type.Is("object") {
			continue
		}
		want := slices.Sorted(maps.Keys(schema.Properties))
		if got := slices.Sorted(slices.Values(schema.Required)); !slices.Equal(got, want) {
			t.Errorf("schema %s required = %v, want %v", name, got, want)
		}
	}
}

func (r *openAPIRig) assertEventKinds() {
	t := r.t
	t.Helper()
	if got, want := slices.Sorted(maps.Keys(r.kinds)), slices.Sorted(maps.Keys(r.events)); !slices.Equal(got, want) {
		t.Errorf("event kinds in real responses = %v, want the discriminator mapping %v", got, want)
	}
	inMapping := make(map[string]bool, len(r.events))
	for _, ref := range r.events {
		inMapping[ref] = true
	}
	for _, member := range r.doc.Components.Schemas["GameEvent"].Value.OneOf {
		if !inMapping[member.Ref] {
			t.Errorf("GameEvent oneOf member %s is missing from the discriminator mapping", member.Ref)
		}
	}
}

func textEnumNames[T interface {
	~int
	fmt.Stringer
}]() []string {
	var names []string
	for v := T(0); ; v++ {
		name := v.String()
		if strings.HasSuffix(name, "Unknown") {
			return names
		}
		names = append(names, name)
	}
}

func engineSkillNames(t *testing.T) []string {
	t.Helper()
	field, ok := reflect.TypeFor[engine.Archetype]().FieldByName("PresetSkills")
	if !ok {
		t.Fatal("engine.Archetype has no PresetSkills field")
	}
	all := reflect.New(field.Type).Elem()
	all.SetUint(1<<32 - 1)
	names, ok := reflect.TypeAssert[[]string](all.MethodByName("Names").Call(nil)[0])
	if !ok {
		t.Fatal("PresetSkills.Names() does not return []string")
	}
	return names
}

func (r *openAPIRig) specEnum(name string) []string {
	r.t.Helper()
	ref := r.doc.Components.Schemas[name]
	if ref == nil {
		r.t.Fatalf("spec has no schema %s", name)
	}
	var values []string
	for _, v := range ref.Value.Enum {
		s, ok := v.(string)
		if !ok {
			r.t.Fatalf("schema %s enum value %v is not a string", name, v)
		}
		values = append(values, s)
	}
	return values
}

func (r *openAPIRig) assertEnums() {
	t := r.t
	t.Helper()
	want := map[string][]string{
		"TerrainType":  textEnumNames[engine.TerrainType](),
		"OccupantType": textEnumNames[engine.OccupantType](),
		"UnitRole":     textEnumNames[engine.UnitRole](),
		"CPUTurnPhase": textEnumNames[room.CPUTurnPhase](),
		"SkillType":    engineSkillNames(t),
		"TurnCmdType":  {string(engine.TurnCmdMove), string(engine.TurnCmdPlaceBomb)},
	}
	for _, name := range slices.Sorted(maps.Keys(r.doc.Components.Schemas)) {
		if len(r.doc.Components.Schemas[name].Value.Enum) > 0 && want[name] == nil {
			t.Errorf("schema %s has an enum with no engine list to compare against", name)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(want)) {
		got := slices.Sorted(slices.Values(r.specEnum(name)))
		if w := slices.Sorted(slices.Values(want[name])); !slices.Equal(got, w) {
			t.Errorf("spec enum %s = %v, want the engine values %v", name, got, w)
		}
	}
}

func (r *openAPIRig) assertClosedAndNonNullable() {
	t := r.t
	t.Helper()
	var visit func(at string, ref *openapi3.SchemaRef)
	visit = func(at string, ref *openapi3.SchemaRef) {
		if ref == nil || ref.Ref != "" {
			return
		}
		s := ref.Value
		if s.Nullable {
			t.Errorf("%s is nullable", at)
		}
		if s.Type.Is("object") && (s.AdditionalProperties.Has == nil || *s.AdditionalProperties.Has) {
			t.Errorf("%s is an object without additionalProperties: false", at)
		}
		for _, name := range slices.Sorted(maps.Keys(s.Properties)) {
			visit(at+"."+name, s.Properties[name])
		}
		visit(at+"[]", s.Items)
		visit(at+"{}", s.AdditionalProperties.Schema)
		for _, group := range [][]*openapi3.SchemaRef{s.AllOf, s.OneOf, s.AnyOf} {
			for i, sub := range group {
				visit(fmt.Sprintf("%s(%d)", at, i), sub)
			}
		}
		visit(at+".not", s.Not)
	}
	content := func(at string, c openapi3.Content) {
		for mediaType, media := range c {
			visit(at+" "+mediaType, media.Schema)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(r.doc.Components.Schemas)) {
		visit("schema "+name, r.doc.Components.Schemas[name])
	}
	for _, name := range slices.Sorted(maps.Keys(r.doc.Components.Responses)) {
		content("response "+name, r.doc.Components.Responses[name].Value.Content)
	}
	for _, path := range r.doc.Paths.Keys() {
		item := r.doc.Paths.Value(path)
		for method, op := range item.Operations() {
			at := method + " " + path
			for _, p := range slices.Concat(item.Parameters, op.Parameters) {
				visit(at+" parameter "+p.Value.Name, p.Value.Schema)
			}
			if op.RequestBody != nil {
				content(at+" request", op.RequestBody.Value.Content)
			}
			for _, code := range op.Responses.Keys() {
				resp := op.Responses.Value(code).Value
				content(at+" "+code, resp.Content)
				for name, h := range resp.Headers {
					visit(at+" "+code+" header "+name, h.Value.Schema)
				}
			}
		}
	}
}

func (r *openAPIRig) assertGuards() {
	t := r.t
	t.Helper()
	post := r.doc.Paths.Value("/api/match-rooms").Post
	if post == nil {
		t.Fatal("spec lacks POST /api/match-rooms")
	}
	created := post.Responses.Value("201")
	if created == nil || created.Value.Headers["Location"] == nil || !created.Value.Headers["Location"].Value.Required {
		t.Error("POST /api/match-rooms 201 must declare the Location header as required")
	}

	tokens := r.doc.Components.Schemas["CreateMatchResponse"].Value.Properties["playerTokens"].Value
	if tokens.MinItems != 2 {
		t.Errorf("CreateMatchResponse.playerTokens minItems = %d, want 2", tokens.MinItems)
	}
	if tokens.MaxItems == nil || *tokens.MaxItems != 2 {
		t.Error("CreateMatchResponse.playerTokens maxItems must be 2")
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
