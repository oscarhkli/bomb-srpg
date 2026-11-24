package server

import (
	"bomb-srpg/engine"
	"bomb-srpg/room"
	"bomb-srpg/server/api"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"math"
	"net/http"
	"slices"
	"strings"
)

type Handler struct {
	Manager *ServerStateManager
	Logger  *slog.Logger
}

type HandlerOption func(*Handler)

func WithHandlerLogger(logger *slog.Logger) HandlerOption {
	return func(h *Handler) {
		h.Logger = logger
	}
}

func NewHandler(m *ServerStateManager, opts ...HandlerOption) *Handler {
	h := &Handler{
		Manager: m,
		Logger:  slog.Default(),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func (h *Handler) extractBearerToken(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", room.ErrInvalidToken
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == auth {
		return "", room.ErrInvalidToken
	}
	return token, nil
}

type statusError struct {
	status int
	text   string
}

func (e *statusError) Error() string { return e.text }

func badRequest(text string) error {
	return &statusError{status: http.StatusBadRequest, text: text}
}

type tokenKey struct{}

type tokenResult struct {
	token string
	err   error
}

func (h *Handler) withToken(next api.StrictHandlerFunc, _ string) api.StrictHandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
		token, err := h.extractBearerToken(r)
		return next(context.WithValue(ctx, tokenKey{}, tokenResult{token: token, err: err}), w, r, request)
	}
}

func tokenFromContext(ctx context.Context) (string, error) {
	res, ok := ctx.Value(tokenKey{}).(tokenResult)
	if !ok {
		return "", room.ErrInvalidToken
	}
	return res.token, res.err
}

func (h *Handler) responseError(w http.ResponseWriter, r *http.Request, err error) {
	status, text := mapError(err)
	if statusErr, ok := errors.AsType[*statusError](err); ok {
		status, text = statusErr.status, statusErr.text
	}
	h.Logger.Warn("request failed", "method", r.Method, "pattern", r.Pattern, "status", status, "error", err)
	http.Error(w, text, status)
}

func (h *Handler) requestError(w http.ResponseWriter, r *http.Request, err error) {
	switch r.Pattern {
	case createMatchPattern:
		h.responseError(w, r, badRequest(invalidConfigText))
	case turnCommandsPattern:
		h.responseError(w, r, badRequest(invalidTurnCmdText))
	case surrenderPattern:
		h.responseError(w, r, badRequest(invalidSurrenderText))
	default:
		h.responseError(w, r, badRequest(err.Error()))
	}
}

const (
	missingQueryText     = "missing required query parameters: unitId and turnCmdType are required"
	invalidUnitIDText    = "Invalid unitId parameter"
	invalidConfigText    = "Invalid configuration format"
	invalidTurnCmdText   = "Invalid turnCommand format"
	invalidSurrenderText = "Invalid surrenderRequest format"
	maxWireUnitID        = math.MaxUint8
)

func (h *Handler) bindError(w http.ResponseWriter, r *http.Request, err error) {
	var required *api.RequiredParamError
	var invalid *api.InvalidParamFormatError
	switch {
	case errors.As(err, &required):
		h.responseError(w, r, badRequest(missingQueryText))
	case errors.As(err, &invalid) && invalid.ParamName == "unitId":
		query := r.URL.Query()
		if query.Get("unitId") == "" || query.Get("turnCmdType") == "" {
			h.responseError(w, r, badRequest(missingQueryText))
			return
		}
		h.responseError(w, r, badRequest(invalidUnitIDText))
	default:
		h.responseError(w, r, badRequest(err.Error()))
	}
}

func (h *Handler) GetHealth(_ context.Context, _ api.GetHealthRequestObject) (api.GetHealthResponseObject, error) {
	return api.GetHealth200JSONResponse{Status: "ok"}, nil
}

func (h *Handler) GetCatalog(_ context.Context, _ api.GetCatalogRequestObject) (api.GetCatalogResponseObject, error) {
	return api.GetCatalog200JSONResponse(wireCatalog(engine.GetAllArchetypes(), engine.GetAllStagePresets())), nil
}

func (h *Handler) CreateMatchRoom(_ context.Context, _ api.CreateMatchRoomRequestObject) (api.CreateMatchRoomResponseObject, error) {
	id, err := h.Manager.createMatchRoom()
	if err != nil {
		h.Logger.Error("create match room failed", "error", err)
		return nil, &statusError{status: http.StatusInternalServerError, text: "Failed to create new MatchRoom"}
	}
	h.Logger.Info("match room created", "roomID", id)
	return api.CreateMatchRoom201JSONResponse{
		Body:    api.CreateMatchRoomResponse{ID: id},
		Headers: api.CreateMatchRoom201ResponseHeaders{Location: "/api/match-rooms/" + id},
	}, nil
}

func (h *Handler) CreateMatch(_ context.Context, request api.CreateMatchRequestObject) (api.CreateMatchResponseObject, error) {
	gameCfg, err := engineGameCfg(request.Body.GameCfg)
	if err != nil {
		return nil, err
	}
	rm, err := h.Manager.Room(request.RoomID)
	if err != nil {
		return nil, err
	}
	tokens, err := rm.Start(gameCfg)
	if err != nil {
		return nil, err
	}
	h.Logger.Info("match created", "roomID", request.RoomID)
	return api.CreateMatch201JSONResponse(wireMatchCreated(tokens)), nil
}

func (h *Handler) Rematch(ctx context.Context, request api.RematchRequestObject) (api.RematchResponseObject, error) {
	token, err := tokenFromContext(ctx)
	if err != nil {
		return nil, err
	}
	rm, err := h.Manager.Room(request.RoomID)
	if err != nil {
		return nil, err
	}
	tokens, err := rm.Rematch(token)
	if err != nil {
		return nil, err
	}
	h.Logger.Info("rematch created", "roomID", request.RoomID)
	return api.Rematch201JSONResponse(wireMatchCreated(tokens)), nil
}

func (h *Handler) DeleteMatch(ctx context.Context, request api.DeleteMatchRequestObject) (api.DeleteMatchResponseObject, error) {
	token, err := tokenFromContext(ctx)
	if err != nil {
		return nil, err
	}
	rm, err := h.Manager.Room(request.RoomID)
	if err != nil {
		return nil, err
	}
	if err := rm.Delete(token); err != nil {
		return nil, err
	}
	h.Logger.Info("match deleted", "roomID", request.RoomID)
	return api.DeleteMatch204Response{}, nil
}

func (h *Handler) GetMatchState(_ context.Context, request api.GetMatchStateRequestObject) (api.GetMatchStateResponseObject, error) {
	rm, err := h.Manager.Room(request.RoomID)
	if err != nil {
		return nil, err
	}
	gs, err := rm.State()
	if err != nil {
		return nil, err
	}
	return api.GetMatchState200JSONResponse(wireGameState(gs)), nil
}

func (h *Handler) SubmitTurnCommand(ctx context.Context, request api.SubmitTurnCommandRequestObject) (api.SubmitTurnCommandResponseObject, error) {
	cmd, err := engineTurnCommand(*request.Body)
	if err != nil {
		return nil, err
	}
	token, err := tokenFromContext(ctx)
	if err != nil {
		return nil, err
	}
	rm, err := h.Manager.Room(request.RoomID)
	if err != nil {
		return nil, err
	}
	gameEvents, err := rm.Submit(cmd, token)
	if err != nil {
		return nil, err
	}
	return api.SubmitTurnCommand200JSONResponse(mapSlice(gameEvents, wireGameEvent)), nil
}

func (h *Handler) ConsumeCPUStatus(ctx context.Context, request api.ConsumeCPUStatusRequestObject) (api.ConsumeCPUStatusResponseObject, error) {
	token, err := tokenFromContext(ctx)
	if err != nil {
		return nil, err
	}
	rm, err := h.Manager.Room(request.RoomID)
	if err != nil {
		return nil, err
	}
	status, err := rm.ConsumeCPU(token)
	if err != nil {
		return nil, err
	}
	h.Logger.Debug("consume cpu status", "roomID", request.RoomID, "turnPhase", status.Phase, "planGameEvents", status.Plan, "resolveTurnGameEvents", status.Resolve)
	return api.ConsumeCPUStatus200JSONResponse{
		TurnPhase:             api.CPUTurnPhase(status.Phase.String()),
		PlanGameEvents:        mapSlice(status.Plan, wireGameEvent),
		ResolveTurnGameEvents: mapSlice(status.Resolve, wireGameEvent),
	}, nil
}

func (h *Handler) ResetTurn(ctx context.Context, request api.ResetTurnRequestObject) (api.ResetTurnResponseObject, error) {
	token, err := tokenFromContext(ctx)
	if err != nil {
		return nil, err
	}
	rm, err := h.Manager.Room(request.RoomID)
	if err != nil {
		return nil, err
	}
	if err := rm.Reset(token); err != nil {
		return nil, err
	}
	return api.ResetTurn204Response{}, nil
}

func (h *Handler) ResolveTurn(ctx context.Context, request api.ResolveTurnRequestObject) (api.ResolveTurnResponseObject, error) {
	token, err := tokenFromContext(ctx)
	if err != nil {
		return nil, err
	}
	rm, err := h.Manager.Room(request.RoomID)
	if err != nil {
		return nil, err
	}
	planGameEvents, resolveTurnGameEvents, err := rm.Resolve(token)
	if err != nil {
		return nil, err
	}
	h.Logger.Debug("resolve turn", "roomID", request.RoomID, "planGameEvents", planGameEvents, "resolveTurnGameEvents", resolveTurnGameEvents)
	return api.ResolveTurn200JSONResponse{
		PlanGameEvents:        mapSlice(planGameEvents, wireGameEvent),
		ResolveTurnGameEvents: mapSlice(resolveTurnGameEvents, wireGameEvent),
	}, nil
}

func (h *Handler) Surrender(ctx context.Context, request api.SurrenderRequestObject) (api.SurrenderResponseObject, error) {
	token, err := tokenFromContext(ctx)
	if err != nil {
		return nil, err
	}
	teamID := request.Body.TeamID
	if teamID != engine.Team1 && teamID != engine.Team2 {
		return nil, fmt.Errorf("%w: team must be 1 or 2", room.ErrInvalidConfig)
	}
	rm, err := h.Manager.Room(request.RoomID)
	if err != nil {
		return nil, err
	}
	gameEvents, err := rm.Surrender(teamID, token)
	if err != nil {
		return nil, err
	}
	return api.Surrender200JSONResponse(mapSlice(gameEvents, wireGameEvent)), nil
}

func (h *Handler) GetMatchConfig(_ context.Context, request api.GetMatchConfigRequestObject) (api.GetMatchConfigResponseObject, error) {
	rm, err := h.Manager.Room(request.RoomID)
	if err != nil {
		return nil, err
	}
	gameCfg, err := rm.Config()
	if err != nil {
		return nil, err
	}
	return api.GetMatchConfig200JSONResponse(wireGameCfg(gameCfg)), nil
}

func (h *Handler) GetAllowedTiles(_ context.Context, request api.GetAllowedTilesRequestObject) (api.GetAllowedTilesResponseObject, error) {
	if request.Params.TurnCmdType == "" {
		return nil, badRequest(missingQueryText)
	}
	unitID, ok := engineUnitID(request.Params.UnitID)
	if !ok {
		return nil, badRequest(invalidUnitIDText)
	}
	rm, err := h.Manager.Room(request.RoomID)
	if err != nil {
		return nil, err
	}
	allowed, err := rm.AllowedTiles(unitID, engine.TurnCmdType(request.Params.TurnCmdType))
	if err != nil {
		return nil, err
	}
	return api.GetAllowedTiles200JSONResponse(wireCoordinates(allowed)), nil
}

func mapSlice[S, D any](src []S, convert func(S) D) []D {
	out := make([]D, 0, len(src))
	for _, v := range src {
		out = append(out, convert(v))
	}
	return out
}

func nonZeroPtr[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}

func engineUnitID(n int) (engine.UnitID, bool) {
	if n < 0 || n > maxWireUnitID {
		return 0, false
	}
	return engine.UnitID(n & maxWireUnitID), true
}

func wireCoordinate(c engine.Coordinate) api.Coordinate {
	return api.Coordinate{X: c.X, Y: c.Y}
}

func wireCoordinates(cs []engine.Coordinate) []api.Coordinate {
	return mapSlice(cs, wireCoordinate)
}

func wireCoordinatePtr(c *engine.Coordinate) *api.Coordinate {
	if c == nil {
		return nil
	}
	w := wireCoordinate(*c)
	return &w
}

func wireSkills(names []string) []api.SkillType {
	return mapSlice(names, func(name string) api.SkillType { return api.SkillType(name) })
}

func wireGameEvent(evt engine.GameEvent) api.GameEvent {
	out := api.GameEvent{
		Type:         api.GameEvtType(evt.Type),
		UnitID:       int(evt.UnitID),
		NewHp:        evt.NewHP,
		Countdown:    evt.Countdown,
		Position:     wireCoordinatePtr(evt.Position),
		From:         wireCoordinatePtr(evt.From),
		To:           wireCoordinatePtr(evt.To),
		BombID:       nonZeroPtr(int64(evt.BombID)),
		SoftBlockID:  nonZeroPtr(evt.SoftBlockID),
		ItemID:       nonZeroPtr(evt.ItemID),
		Range:        nonZeroPtr(evt.Range),
		WinnerTeamID: nonZeroPtr(evt.WinnerTeamID),
	}
	if len(evt.AffectedPositions) > 0 {
		affected := wireCoordinates(evt.AffectedPositions)
		out.AffectedPositions = &affected
	}
	return out
}

func wireTile(t engine.Tile) api.Tile {
	return api.Tile{
		Type:         api.TerrainType(t.Type.String()),
		OccupantType: api.OccupantType(t.OccupantType.String()),
		OccupantID:   t.OccupantID,
	}
}

func wireUnit(u *engine.Unit) api.Unit {
	return api.Unit{
		ID:           int(u.ID),
		Type:         u.Type.Name,
		Position:     wireCoordinate(u.Position),
		Speed:        u.Speed,
		BombMaxRange: u.BombMaxRange,
		BombPower:    u.BombPower,
		MaxBombCount: u.MaxBombCount,
		BombUsed:     u.BombUsed,
		Team:         u.Team,
		Hp:           u.HP,
		Skills:       wireSkills(u.Skills.Names()),
		HasMoved:     u.HasMoved,
		HasUsedSkill: u.HasUsedSkill,
	}
}

func wireBomb(b *engine.Bomb) api.Bomb {
	return api.Bomb{
		ID:        int64(b.ID),
		OwnerID:   int(b.OwnerID),
		Position:  wireCoordinate(b.Position),
		Range:     b.Range,
		Countdown: b.Countdown,
	}
}

func wireSoftBlock(sb *engine.SoftBlock) api.SoftBlock {
	return api.SoftBlock{ID: sb.ID, Position: wireCoordinate(sb.Position)}
}

func wireTurnCommand(cmd engine.TurnCommand) api.TurnCommand {
	return api.TurnCommand{Type: api.TurnCmdType(cmd.Type), UnitID: int(cmd.UnitID), Target: wireCoordinate(cmd.Target)}
}

func wireGameState(gs *engine.GameState) api.GameState {
	return api.GameState{
		Turn:            gs.Turn,
		InSuddenDeath:   gs.InSuddenDeath,
		ActiveTeam:      gs.ActiveTeam,
		Grid:            mapSlice(gs.Grid, func(row []engine.Tile) []api.Tile { return mapSlice(row, wireTile) }),
		Units:           mapSlice(slices.Sorted(maps.Keys(gs.Units)), func(id engine.UnitID) api.Unit { return wireUnit(gs.Units[id]) }),
		Bombs:           mapSlice(slices.Sorted(maps.Keys(gs.Bombs)), func(id engine.BombID) api.Bomb { return wireBomb(gs.Bombs[id]) }),
		SoftBlocks:      mapSlice(slices.Sorted(maps.Keys(gs.SoftBlocks)), func(id int) api.SoftBlock { return wireSoftBlock(gs.SoftBlocks[id]) }),
		TurnCommands:    mapSlice(gs.TurnCommands, wireTurnCommand),
		TurnStartEvents: mapSlice(gs.TurnStartEvents, wireGameEvent),
	}
}

func wireTeamSlot(slot engine.TeamSlot) api.TeamSlot {
	return api.TeamSlot{Archetype: slot.Archetype, Role: api.UnitRole(slot.Role.String())}
}

func wireGameCfg(cfg engine.GameCfg) api.GameCfg {
	return api.GameCfg{
		VsCPU:          cfg.VSCpu,
		StagePreset:    cfg.StagePreset,
		P1Slots:        mapSlice(cfg.P1Slots, wireTeamSlot),
		P2Slots:        mapSlice(cfg.P2Slots, wireTeamSlot),
		MaxTurns:       cfg.MaxTurns,
		AllowResetTurn: cfg.AllowResetTurn,
	}
}

func wireCatalog(archetypes []engine.Archetype, stagePresets []engine.StagePreset) api.Catalog {
	return api.Catalog{
		Archetypes: mapSlice(archetypes, func(a engine.Archetype) api.Archetype {
			return api.Archetype{Name: a.Name, Speed: a.BaseSpeed, BombMaxRange: a.BombMaxRange, Skills: wireSkills(a.PresetSkills.Names())}
		}),
		StagePresets: mapSlice(stagePresets, func(p engine.StagePreset) api.StagePreset {
			return api.StagePreset{Name: p.Name, Description: p.Description, Width: p.Width, Height: p.Height, MaxTurns: p.MaxTurns}
		}),
	}
}

func wireMatchCreated(tokens [2]string) api.CreateMatchResponse {
	return api.CreateMatchResponse{Success: true, PlayerTokens: tokens[:]}
}

func engineTurnCommand(cmd api.TurnCommand) (engine.TurnCommand, error) {
	unitID, ok := engineUnitID(cmd.UnitID)
	if !ok {
		return engine.TurnCommand{}, badRequest(invalidTurnCmdText)
	}
	return engine.TurnCommand{
		Type:   engine.TurnCmdType(cmd.Type),
		UnitID: unitID,
		Target: engine.Coordinate{X: cmd.Target.X, Y: cmd.Target.Y},
	}, nil
}

func engineTeamSlots(slots []api.TeamSlot) ([]engine.TeamSlot, error) {
	out := make([]engine.TeamSlot, 0, len(slots))
	for _, slot := range slots {
		var role engine.UnitRole
		if err := role.UnmarshalText([]byte(slot.Role)); err != nil {
			return nil, badRequest(invalidConfigText)
		}
		out = append(out, engine.TeamSlot{Archetype: slot.Archetype, Role: role})
	}
	return out, nil
}

func engineGameCfg(cfg api.GameCfg) (engine.GameCfg, error) {
	p1Slots, err := engineTeamSlots(cfg.P1Slots)
	if err != nil {
		return engine.GameCfg{}, err
	}
	p2Slots, err := engineTeamSlots(cfg.P2Slots)
	if err != nil {
		return engine.GameCfg{}, err
	}
	return engine.GameCfg{
		VSCpu:          cfg.VsCPU,
		StagePreset:    cfg.StagePreset,
		P1Slots:        p1Slots,
		P2Slots:        p2Slots,
		MaxTurns:       cfg.MaxTurns,
		AllowResetTurn: cfg.AllowResetTurn,
	}, nil
}
