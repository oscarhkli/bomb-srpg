import { describe, it, expect, beforeEach, vi, type MockInstance } from 'vitest';
import { mockScene } from '../test/setup';
import {
  firstGraphics as terrainGraphics,
  occupantGraphics,
  pointerDownOf,
  errorTextByMessage,
  flush,
  tweenConfigAt,
  fireShutdown,
  fireCameraFadeOutComplete,
} from '../test/sceneHelpers';
import { makeCfg, makeState, plainTile, tileOf, makeUnit, makeBomb } from '../test/fixtures';
import {
  initRoom,
  initToken,
  getMatchState,
  getMatchConfig,
  getAllowedTiles,
  submitTurnCommand,
  resolveTurn,
  startTurn,
  rematch,
  deleteMatch,
} from '../engine/api';
import { playResolveTurnEvents } from '../rendering/resolveTurnPlayer';
import TurnBanner from '../ui/TurnBanner';
import SuddenDeathCutscene from '../ui/SuddenDeathCutscene';
import VictoryCutscene from '../ui/VictoryCutscene';
import MatchScene, { type MatchSceneData } from './MatchScene';
import type { Coordinate, GameCfg, GameEvent, GameState, Tile } from '../types/api';
import {
  DEPTH_SUDDEN_DEATH_BOMB,
  DEPTH_OCCUPANT,
  SUDDEN_DEATH_BOMB_DROP_DURATION_MS,
} from '../constants';

vi.mock('../engine/api');
vi.mock('../rendering/resolveTurnPlayer');
vi.mock('../ui/TurnBanner');
vi.mock('../ui/SuddenDeathCutscene');
vi.mock('../ui/VictoryCutscene');

// Queues getMatchState() resolutions across the whole per-turn lifecycle: call 1 is the
// initial board render in create(), call 2 is beginTurn()'s own per-turn refresh (fires
// automatically right after create(), before any test interaction) — both get `states[0]`
// unless a distinct value is passed. Any further explicit `states` cover calls made by a
// test's own action (a Move/Bomb refresh, a resolve-turn refresh, etc.), and the LAST state
// given becomes the persistent fallback for every call beyond that (e.g. the beginTurn() that
// automatically follows a successful resolve).
function queueMatchStates(...states: GameState[]): void {
  const mocked = vi.mocked(getMatchState);
  const [first, ...rest] = states;
  if (!first) {
    throw new Error('queueMatchStates requires at least one state');
  }
  mocked.mockResolvedValueOnce(first).mockResolvedValueOnce(first);
  for (const state of rest) {
    mocked.mockResolvedValueOnce(state);
  }
  mocked.mockResolvedValue(rest.length > 0 ? rest[rest.length - 1]! : first);
}

// flush() (from sceneHelpers) drains the microtask queue enough times for the full
// create() -> beginTurn() chain (getMatchState -> initToken -> startTurn ->
// [SuddenDeathCutscene] -> TurnBanner) to settle, regardless of how many `.then`/`await` hops
// are in it — cheap since everything here is an already-resolved mock promise, not a real timer.
async function bootScene(overrides: Partial<MatchSceneData> = {}): Promise<MatchScene> {
  const scene = new MatchScene();
  scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'], ...overrides });
  await flush();
  return scene;
}

// Also created synchronously within create()'s initial .then(), right after the grid/occupants
// — same index formula as occupantGraphics (results[i+1]), kept as a distinctly-named alias here
// since "resolve button" reads clearer than "occupant" at its call sites.
function resolveButtonGraphics(unitCount: number): ReturnType<typeof mockScene.add.graphics> {
  return occupantGraphics(unitCount);
}

// Drives the full UI path (click unit -> click Move/Bomb -> click allowed tile -> click Yes)
// assuming getAllowedTiles is mocked to resolve with exactly one tile.
async function submitViaUI(
  unitGraphics: ReturnType<typeof mockScene.add.graphics>,
  buttonIndex: 0 | 1 // 0 = Move, 1 = Bomb
): Promise<void> {
  pointerDownOf(unitGraphics)(); // opens TurnCommandPanel
  const actionButtonGraphics = mockScene.add.graphics.mock.results
    .slice(-3)
    .map(r => r.value as ReturnType<typeof mockScene.add.graphics>)[buttonIndex];
  pointerDownOf(actionButtonGraphics!)();
  await flush();
  const [overlayTileGraphics] = mockScene.add.graphics.mock.results
    .slice(-1)
    .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
  pointerDownOf(overlayTileGraphics!)();
  const [, yesButtonGraphics] = mockScene.add.graphics.mock.results
    .slice(-3)
    .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
  pointerDownOf(yesButtonGraphics!)();
}

// vi.spyOn(Class.prototype, 'method') avoids referencing `Class.prototype.method` as a bare
// value (which trips @typescript-eslint/unbound-method) while still giving a MockInstance for
// assertions/call-order checks on the automocked TurnBanner/SuddenDeathCutscene/VictoryCutscene.
let turnBannerPlay: MockInstance<TurnBanner['play']>;
let suddenDeathCutscenePlay: MockInstance<SuddenDeathCutscene['play']>;
let victoryCutscenePlay: MockInstance<VictoryCutscene['play']>;

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(getMatchConfig).mockResolvedValue(makeCfg());
  vi.mocked(playResolveTurnEvents).mockReturnValue({ ok: true, done: Promise.resolve() });
  vi.mocked(startTurn).mockResolvedValue({ inSuddenDeath: false, gameEvents: [] });
  turnBannerPlay = vi.spyOn(TurnBanner.prototype, 'play').mockResolvedValue(undefined);
  suddenDeathCutscenePlay = vi
    .spyOn(SuddenDeathCutscene.prototype, 'play')
    .mockResolvedValue(undefined);
  victoryCutscenePlay = vi.spyOn(VictoryCutscene.prototype, 'play').mockReturnValue(undefined);
});

describe('MatchScene', () => {
  it('calls initRoom with data.roomId then getMatchState on create', async () => {
    queueMatchStates(makeState({ grid: [[plainTile()]] }));

    await bootScene({ roomId: 'room-abc' });

    expect(initRoom).toHaveBeenCalledWith('room-abc');
    // Once for the initial board render, once more for beginTurn()'s per-turn refresh.
    expect(getMatchState).toHaveBeenCalledTimes(2);
  });

  it('never logs roomId or playerTokens to the console', async () => {
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => undefined);
    queueMatchStates(makeState({ grid: [[plainTile()]] }));

    await bootScene({ roomId: 'room-xyz', playerTokens: ['tok1', 'tok2'] });

    expect(consoleSpy).not.toHaveBeenCalledWith(
      expect.stringContaining('roomId'),
      expect.anything(),
      expect.anything(),
      expect.anything()
    );
    consoleSpy.mockRestore();
  });

  it('centers camera on a 3x2 grid', async () => {
    const row = (): Tile[] => [plainTile(), plainTile(), plainTile()];
    queueMatchStates(makeState({ grid: [row(), row()] }));

    await bootScene();

    expect(mockScene.cameras.main.centerOn).toHaveBeenCalledWith(72, 48);
  });

  it('centers camera on a differently-sized 5x7 grid', async () => {
    const row = (): Tile[] => Array.from({ length: 5 }, plainTile);
    queueMatchStates(makeState({ grid: Array.from({ length: 7 }, row) }));

    await bootScene();

    expect(mockScene.cameras.main.centerOn).toHaveBeenCalledWith(120, 168);
  });

  it('calls initToken exactly once at turn start (beginTurn), not on unit click', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1, units: [unit] }));

    await bootScene({ playerTokens: ['team1-token', 'team2-token'] });
    expect(initToken).toHaveBeenCalledWith('team1-token');
    expect(initToken).toHaveBeenCalledOnce();

    const graphicsBefore = mockScene.add.graphics.mock.calls.length;
    pointerDownOf(occupantGraphics(0))();

    // TurnCommandPanel.openFor draws 3 new button Graphics (Move/Bomb/Back), but initToken is
    // not called again — it already fired once for this turn in beginTurn().
    expect(mockScene.add.graphics.mock.calls.length).toBe(graphicsBefore + 3);
    expect(initToken).toHaveBeenCalledOnce();
  });

  it('does not open TurnCommandPanel or call initToken again when clicking a unit off the active team', async () => {
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => undefined);
    const unit = makeUnit({ id: 7, team: 2, position: { x: 0, y: 0 } });
    queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1, units: [unit] }));

    await bootScene();
    expect(initToken).toHaveBeenCalledOnce();

    const graphicsBefore = mockScene.add.graphics.mock.calls.length;
    pointerDownOf(occupantGraphics(0))();

    expect(initToken).toHaveBeenCalledOnce();
    expect(consoleSpy).toHaveBeenCalledWith('Unit 7 is clicked', unit);
    expect(mockScene.add.graphics.mock.calls.length).toBe(graphicsBefore);
    consoleSpy.mockRestore();
  });

  it('caches getAllowedTiles per (unitId, turnCmdType): repeat moveButton clicks skip the network call', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1, units: [unit] }));
    vi.mocked(getAllowedTiles).mockResolvedValue([{ x: 1, y: 0 }]);

    await bootScene();

    const onUnitPointerDown = pointerDownOf(occupantGraphics(0));

    // Open the panel, click Move, close it, reopen, click Move again.
    onUnitPointerDown();
    const [moveButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    const clickMove = (): void => pointerDownOf(moveButtonGraphics!)();
    clickMove();
    await flush();

    onUnitPointerDown(); // closes and reopens the panel for the same unit
    onUnitPointerDown();
    clickMove();
    await flush();

    expect(getAllowedTiles).toHaveBeenCalledTimes(1);
    expect(getAllowedTiles).toHaveBeenCalledWith({ unitId: 7, turnCmdType: 'move' });
  });

  it('shows an error when getAllowedTiles rejects', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1, units: [unit] }));
    vi.mocked(getAllowedTiles).mockRejectedValue(new Error('tiles unavailable'));

    await bootScene();

    const onUnitPointerDown = pointerDownOf(occupantGraphics(0));
    onUnitPointerDown();

    const [moveButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(moveButtonGraphics!)();
    await flush();

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'tiles unavailable',
      expect.objectContaining({})
    );
  });

  it('does not render a Unit with hp 0', async () => {
    const dead = makeUnit({ hp: 0 });
    queueMatchStates(makeState({ grid: [[plainTile()]], units: [dead] }));

    await bootScene();

    // No occupant Graphics exists for the dead unit: only the grid, ResolveTurnButton, and
    // TurnPanel header(s) — TurnPanel renders once when gameCfg resolves and again inside
    // beginTurn()'s per-turn refresh, so exactly 2 "header" graphics beyond grid+button.
    expect(mockScene.add.graphics).toHaveBeenCalledTimes(4);
  });

  it('shows an error message in the error panel when getMatchState rejects, without rendering the board', async () => {
    vi.mocked(getMatchState).mockRejectedValue(new Error('network fail'));

    await bootScene();

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Failed to load match state',
      expect.objectContaining({})
    );
    // Only the error panel's own background Graphics exists — no board (grid) was rendered,
    // and beginTurn() never ran since the initial getMatchState() rejected.
    expect(mockScene.add.graphics).toHaveBeenCalledTimes(1);
    expect(mockScene.cameras.main.centerOn).not.toHaveBeenCalled();
    expect(startTurn).not.toHaveBeenCalled();
  });

  it('handles a confirmed Move: submits the command, tweens the unit, and refreshes state', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [
      [{ type: 'TerrainPlain', occupantType: 'OccupantUnit', occupantId: 7 }, plainTile()],
    ];
    const target: Coordinate = { x: 1, y: 0 };
    const movedUnit = makeUnit({ id: 7, team: 1, position: target });
    const refreshedGrid: Tile[][] = [
      [
        tileOf('TerrainPlain'),
        { type: 'TerrainPlain', occupantType: 'OccupantUnit', occupantId: 7 },
      ],
    ];

    queueMatchStates(
      makeState({ grid: initialGrid, activeTeam: 1, units: [unit] }),
      makeState({ grid: refreshedGrid, activeTeam: 2, units: [movedUnit] })
    );
    vi.mocked(getAllowedTiles).mockResolvedValue([target]);
    const events: GameEvent[] = [
      { type: 'unitMoved', unitId: 7, from: { x: 0, y: 0 }, to: target },
    ];
    vi.mocked(submitTurnCommand).mockResolvedValue(events);

    await bootScene();

    const unitGraphics = occupantGraphics(0);
    await submitViaUI(unitGraphics, 0);
    await flush();

    expect(submitTurnCommand).toHaveBeenCalledWith({ type: 'move', unitId: 7, target });
    expect(mockScene.tweens.add).toHaveBeenCalledWith(
      expect.objectContaining({ targets: unitGraphics, x: 48, y: 0, ease: 'Linear' })
    );
  });

  it('handles a confirmed PlaceBomb: submits the command and renders the new bomb', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [[plainTile(), plainTile()]];
    const target: Coordinate = { x: 1, y: 0 };
    const refreshedGrid: Tile[][] = [
      [
        tileOf('TerrainPlain'),
        { type: 'TerrainPlain', occupantType: 'OccupantBomb', occupantId: 42 },
      ],
    ];

    queueMatchStates(
      makeState({ grid: initialGrid, activeTeam: 1, units: [unit] }),
      makeState({
        grid: refreshedGrid,
        activeTeam: 2,
        units: [unit],
        bombs: [makeBomb({ id: 42, ownerId: 7, position: target, countdown: 3 })],
      })
    );
    vi.mocked(getAllowedTiles).mockResolvedValue([target]);
    const events: GameEvent[] = [
      { type: 'bombPlaced', unitId: 7, bombId: 42, position: target, range: 2, countdown: 3 },
    ];
    vi.mocked(submitTurnCommand).mockResolvedValue(events);

    await bootScene();

    const unitGraphics = occupantGraphics(0);
    await submitViaUI(unitGraphics, 1);
    await flush();

    expect(submitTurnCommand).toHaveBeenCalledWith({ type: 'placeBomb', unitId: 7, target });
    // In-place render (renderBomb), not a wholesale swap: a bomb container is drawn at the
    // target tile center (tileCenter of {x:1,y:0} = 72,24) and the initial unit graphics survives.
    expect(mockScene.add.container).toHaveBeenCalledWith(72, 24, expect.any(Array));
    expect(unitGraphics.destroy).not.toHaveBeenCalled();
  });

  // AC4: a rejected command surfaces the *actual* server error and resyncs by rebuilding the
  // occupant layer from truth, while the terrain layer survives the swap.
  it('shows the actual error and resyncs by rebuilding occupants (terrain untouched) when submitTurnCommand rejects', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [[plainTile(), plainTile()]];
    const target: Coordinate = { x: 1, y: 0 };

    queueMatchStates(makeState({ grid: initialGrid, activeTeam: 1, units: [unit] }));
    vi.mocked(getAllowedTiles).mockResolvedValue([target]);
    vi.mocked(submitTurnCommand).mockRejectedValue(new Error('stale token'));

    await bootScene();

    const gridGraphics = terrainGraphics();
    const unitGraphics = occupantGraphics(0);
    await submitViaUI(unitGraphics, 0);
    await flush();

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'stale token',
      expect.objectContaining({})
    );
    // Occupant layer rebuilt from the recovery refetch (old unit graphics destroyed)...
    expect(unitGraphics.destroy).toHaveBeenCalled();
    // ...but the terrain layer is never in scope of an occupant swap.
    expect(gridGraphics.destroy).not.toHaveBeenCalled();
  });

  it('shows an error and stops processing when the unitMoved event is malformed (out-of-bounds target)', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [[plainTile(), plainTile()]];
    const target: Coordinate = { x: 1, y: 0 };

    queueMatchStates(makeState({ grid: initialGrid, activeTeam: 1, units: [unit] }));
    vi.mocked(getAllowedTiles).mockResolvedValue([target]);
    // `to` is out-of-bounds for a 1x2 grid — structurally malformed, regardless of legality.
    const events: GameEvent[] = [
      { type: 'unitMoved', unitId: 7, from: { x: 0, y: 0 }, to: { x: 99, y: 99 } },
    ];
    vi.mocked(submitTurnCommand).mockResolvedValue(events);

    await bootScene();

    const unitGraphics = occupantGraphics(0);
    await submitViaUI(unitGraphics, 0);
    await flush();

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Invalid unitMoved event received from server',
      expect.objectContaining({})
    );
    expect(mockScene.tweens.add).not.toHaveBeenCalledWith(
      expect.objectContaining({ targets: unitGraphics })
    );
  });

  it('accepts a unitMoved event even when the client grid did not show the unit at `from` (no client-side legality re-derivation)', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    // Deliberately inconsistent with `unit`'s own position: the grid tile at `from` shows
    // no occupant, even though the units array (the source boardRenderer used) has unit 7
    // there. Previously this tripped the "mover was at from" tile-based re-derivation.
    const initialGrid: Tile[][] = [[plainTile(), plainTile()]];
    const target: Coordinate = { x: 1, y: 0 };
    const movedUnit = makeUnit({ id: 7, team: 1, position: target });

    queueMatchStates(
      makeState({ grid: initialGrid, activeTeam: 1, units: [unit] }),
      makeState({ grid: initialGrid, activeTeam: 2, units: [movedUnit] })
    );
    vi.mocked(getAllowedTiles).mockResolvedValue([target]);
    const events: GameEvent[] = [
      { type: 'unitMoved', unitId: 7, from: { x: 0, y: 0 }, to: target },
    ];
    vi.mocked(submitTurnCommand).mockResolvedValue(events);

    await bootScene();

    const unitGraphics = occupantGraphics(0);
    await submitViaUI(unitGraphics, 0);
    await flush();

    expect(mockScene.tweens.add).toHaveBeenCalledWith(
      expect.objectContaining({ targets: unitGraphics, x: 48, y: 0, ease: 'Linear' })
    );
    expect(mockScene.add.text).not.toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Invalid unitMoved event received from server',
      expect.objectContaining({})
    );
  });

  it('tweens the unit to the server-reported tile, not the requested one, when they differ (e.g. a future push/swap move)', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [[plainTile(), plainTile(), plainTile()]];
    const requestedTarget: Coordinate = { x: 1, y: 0 };
    const actualTo: Coordinate = { x: 2, y: 0 };
    const pushedUnit = makeUnit({ id: 7, team: 1, position: actualTo });

    queueMatchStates(
      makeState({ grid: initialGrid, activeTeam: 1, units: [unit] }),
      makeState({ grid: initialGrid, activeTeam: 2, units: [pushedUnit] })
    );
    vi.mocked(getAllowedTiles).mockResolvedValue([requestedTarget]);
    // Server reports the unit actually landed one tile further than the client requested.
    const events: GameEvent[] = [
      { type: 'unitMoved', unitId: 7, from: { x: 0, y: 0 }, to: actualTo },
    ];
    vi.mocked(submitTurnCommand).mockResolvedValue(events);

    await bootScene();

    const unitGraphics = occupantGraphics(0);
    await submitViaUI(unitGraphics, 0);
    await flush();

    // The tween follows the server's `to` (x:2 → 96px), not the requested x:1 (48px), and no
    // error is raised — the client renders the server's authoritative event verbatim.
    expect(mockScene.tweens.add).toHaveBeenCalledWith(
      expect.objectContaining({ targets: unitGraphics, x: 96, y: 0, ease: 'Linear' })
    );
    expect(mockScene.add.text).not.toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Failed to refresh match state',
      expect.objectContaining({})
    );
  });

  // AC1 + AC2: a successful move updates the board in place — no wholesale occupant swap follows,
  // the terrain layer survives, and the per-op gameState refetch still runs. (The former
  // proactive-diff "out of sync" flag on the happy path was removed in spec007.)
  it('updates in place on a successful move: no occupant rebuild, terrain survives, state refetched', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [[plainTile(), plainTile()]];
    const target: Coordinate = { x: 1, y: 0 };
    const movedUnit = makeUnit({ id: 7, team: 1, position: target });

    queueMatchStates(
      makeState({ grid: initialGrid, activeTeam: 1, units: [unit] }),
      makeState({ grid: initialGrid, activeTeam: 2, units: [movedUnit] })
    );
    vi.mocked(getAllowedTiles).mockResolvedValue([target]);
    vi.mocked(submitTurnCommand).mockResolvedValue([
      { type: 'unitMoved', unitId: 7, from: { x: 0, y: 0 }, to: target },
    ]);

    await bootScene();

    const gridGraphics = terrainGraphics(); // painted once at scene entry
    const unitGraphics = occupantGraphics(0);
    const refetchesBefore = vi.mocked(getMatchState).mock.calls.length;

    await submitViaUI(unitGraphics, 0);
    await flush();

    // In-place tween — the initial unit graphics is never destroyed (no occupant rebuild)...
    expect(unitGraphics.destroy).not.toHaveBeenCalled();
    // ...and the terrain layer is untouched by the happy path.
    expect(gridGraphics.destroy).not.toHaveBeenCalled();
    // The per-op gameState refetch still ran.
    expect(vi.mocked(getMatchState).mock.calls.length).toBeGreaterThan(refetchesBefore);
  });

  it('ignores a unit click while a confirm dialog is already open', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [
      [{ type: 'TerrainPlain', occupantType: 'OccupantUnit', occupantId: 7 }, plainTile()],
    ];
    const target: Coordinate = { x: 1, y: 0 };

    queueMatchStates(makeState({ grid: initialGrid, activeTeam: 1, units: [unit] }));
    vi.mocked(getAllowedTiles).mockResolvedValue([target]);

    await bootScene();

    const unitGraphics = occupantGraphics(0);
    pointerDownOf(unitGraphics)();
    const [moveButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(moveButtonGraphics!)();
    await flush();
    const [overlayTileGraphics] = mockScene.add.graphics.mock.results
      .slice(-1)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(overlayTileGraphics!)();

    const [, yesButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    const graphicsCallCountBeforeClick = mockScene.add.graphics.mock.calls.length;

    pointerDownOf(unitGraphics)(); // clicking a unit while confirm is pending must be a no-op

    expect(yesButtonGraphics!.destroy).not.toHaveBeenCalled();
    expect(mockScene.add.graphics.mock.calls.length).toBe(graphicsCallCountBeforeClick);
  });

  it('ignores a second confirm-Yes click while the first submission is still in flight', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [
      [{ type: 'TerrainPlain', occupantType: 'OccupantUnit', occupantId: 7 }, plainTile()],
    ];
    const target: Coordinate = { x: 1, y: 0 };

    queueMatchStates(makeState({ grid: initialGrid, activeTeam: 1, units: [unit] }));
    vi.mocked(getAllowedTiles).mockResolvedValue([target]);
    vi.mocked(submitTurnCommand).mockReturnValue(new Promise<GameEvent[]>(() => undefined));

    await bootScene();

    const unitGraphics = occupantGraphics(0);
    pointerDownOf(unitGraphics)();
    const [moveButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(moveButtonGraphics!)();
    await flush();
    const [overlayTileGraphics] = mockScene.add.graphics.mock.results
      .slice(-1)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(overlayTileGraphics!)();
    const [, firstYesButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);

    pointerDownOf(firstYesButtonGraphics!)(); // submitTurnCommand now pending, never resolves in this test
    pointerDownOf(overlayTileGraphics!)(); // re-click the still-visible tile
    const [, secondYesButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(secondYesButtonGraphics!)(); // second Yes click while the first is still in flight

    expect(submitTurnCommand).toHaveBeenCalledTimes(1);
  });

  it('tweens a unit cumulatively across two sequential moves', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [
      [
        { type: 'TerrainPlain', occupantType: 'OccupantUnit', occupantId: 7 },
        plainTile(),
        plainTile(),
      ],
    ];
    const firstTarget: Coordinate = { x: 1, y: 0 };
    const secondTarget: Coordinate = { x: 2, y: 0 };
    const gridAfterFirstMove: Tile[][] = [
      [
        plainTile(),
        { type: 'TerrainPlain', occupantType: 'OccupantUnit', occupantId: 7 },
        plainTile(),
      ],
    ];
    const gridAfterSecondMove: Tile[][] = [
      [
        plainTile(),
        plainTile(),
        { type: 'TerrainPlain', occupantType: 'OccupantUnit', occupantId: 7 },
      ],
    ];

    queueMatchStates(
      makeState({ grid: initialGrid, activeTeam: 1, units: [unit] }),
      makeState({
        grid: gridAfterFirstMove,
        activeTeam: 1,
        units: [makeUnit({ id: 7, team: 1, position: firstTarget })],
      }),
      makeState({
        grid: gridAfterSecondMove,
        activeTeam: 1,
        units: [makeUnit({ id: 7, team: 1, position: secondTarget })],
      })
    );
    vi.mocked(getAllowedTiles)
      .mockResolvedValueOnce([firstTarget])
      .mockResolvedValueOnce([secondTarget]);
    vi.mocked(submitTurnCommand)
      .mockResolvedValueOnce([
        { type: 'unitMoved', unitId: 7, from: { x: 0, y: 0 }, to: firstTarget },
      ])
      .mockResolvedValueOnce([
        { type: 'unitMoved', unitId: 7, from: firstTarget, to: secondTarget },
      ]);

    await bootScene();

    const unitGraphics = occupantGraphics(0);
    await submitViaUI(unitGraphics, 0);
    await flush();

    // The mock Graphics object never actually animates (tweens.add is a no-op spy), so simulate
    // the first tween having completed by advancing the object's position ourselves before the
    // second move — this is what the real Phaser tween would have left it at.
    unitGraphics.x = 48;
    unitGraphics.y = 0;

    pointerDownOf(unitGraphics)();
    const [moveButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(moveButtonGraphics!)();
    await flush();
    const [overlayTileGraphics] = mockScene.add.graphics.mock.results
      .slice(-1)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(overlayTileGraphics!)();
    const [, yesButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(yesButtonGraphics!)();
    await flush();

    expect(mockScene.tweens.add).toHaveBeenLastCalledWith(
      expect.objectContaining({ targets: unitGraphics, x: 96, y: 0 })
    );
  });

  it('fetches gameCfg alongside match state and renders TurnPanel with turn/maxTurns/activeTeam', async () => {
    queueMatchStates(makeState({ grid: [[plainTile()]], turn: 4, activeTeam: 2 }));
    vi.mocked(getMatchConfig).mockResolvedValue(makeCfg({ maxTurns: 12 }));

    await bootScene();

    expect(getMatchConfig).toHaveBeenCalledOnce();
    // TurnPanel renders "Turn" label text among the added texts.
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Turn',
      expect.objectContaining({})
    );
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      '4',
      expect.objectContaining({})
    );
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      '12',
      expect.objectContaining({})
    );
  });

  async function setUpEmptyBoardAndClickResolve(activeTeam = 1): Promise<void> {
    queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam }));

    await bootScene({ playerTokens: ['team1-token', 'team2-token'] });

    pointerDownOf(resolveButtonGraphics(0))();
  }

  it('pins ResolveTurnButton to the camera viewport (scrollFactor 0) instead of the grid/world', async () => {
    queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1 }));

    await bootScene();

    expect(resolveButtonGraphics(0).setScrollFactor).toHaveBeenCalledWith(0);
  });

  it('shows an error instead of crashing when ResolveTurnButton is clicked before match config has loaded', async () => {
    queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1 }));
    vi.mocked(getMatchConfig).mockReturnValue(new Promise<GameCfg>(() => undefined));

    await bootScene();

    expect(() => pointerDownOf(resolveButtonGraphics(0))()).not.toThrow();
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      expect.stringContaining('loading'),
      expect.objectContaining({})
    );
    expect(resolveTurn).not.toHaveBeenCalled();
  });

  it('clears the TurnCommandPanel action stack (treating it as nothing selected) when ResolveTurnButton opens the confirm dialog', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1, units: [unit] }));
    vi.mocked(getAllowedTiles).mockResolvedValue([{ x: 0, y: 0 }]);

    await bootScene();

    // Open the unit's TurnCommandPanel (draws Move/Bomb/Back as the next 3 Graphics).
    pointerDownOf(occupantGraphics(0))();
    const [moveButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);

    pointerDownOf(resolveButtonGraphics(1))();

    expect(moveButtonGraphics!.destroy).toHaveBeenCalled();
  });

  it('opens the resolve confirm even when a TurnCommandPanel confirm is already open, so a stale action never blocks it', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [
      [{ type: 'TerrainPlain', occupantType: 'OccupantUnit', occupantId: 7 }, plainTile()],
    ];
    queueMatchStates(makeState({ grid: initialGrid, activeTeam: 1, units: [unit] }));
    vi.mocked(getAllowedTiles).mockResolvedValue([{ x: 1, y: 0 }]);

    await bootScene();

    // Drive the Move flow until the shared ConfirmDialog is open (click unit -> Move ->
    // allowed tile). At this point confirmDialog.isOpen is true.
    pointerDownOf(occupantGraphics(0))();
    const [moveButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(moveButtonGraphics!)();
    await flush();
    const [overlayTile] = mockScene.add.graphics.mock.results
      .slice(-1)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(overlayTile!)();

    // Clicking Resolve must still surface the resolve prompt (previously the open confirm
    // caused an early return and nothing happened).
    pointerDownOf(resolveButtonGraphics(1))();

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Confirm to end this turn?',
      expect.objectContaining({})
    );
  });

  it('opens ConfirmDialog with the resolve-turn prompt when ResolveTurnButton is clicked', async () => {
    await setUpEmptyBoardAndClickResolve();

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Confirm to end this turn?',
      expect.objectContaining({})
    );
  });

  it('on confirmed resolve: inits the active team token, calls resolveTurn, hands events to the player, then loops back into beginTurn for the next turn', async () => {
    await setUpEmptyBoardAndClickResolve(2);
    expect(initToken).toHaveBeenCalledWith('team2-token');
    vi.mocked(getMatchState).mockResolvedValue(makeState({ grid: [[plainTile()]], activeTeam: 1 }));
    const events: GameEvent[] = [{ type: 'bombCountdownUpdated', bombId: 1, countdown: 2 }];
    vi.mocked(resolveTurn).mockResolvedValue(events);

    // ConfirmDialog's Yes button is the most-recently-created graphics among the last 3.
    const [, yesButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(yesButtonGraphics!)();
    await flush();

    expect(resolveTurn).toHaveBeenCalledOnce();
    expect(playResolveTurnEvents).toHaveBeenCalledOnce();
    const [calledEvents, deps] = vi.mocked(playResolveTurnEvents).mock.calls[0]!;
    expect(calledEvents).toBe(events);
    expect(deps.gameStateSnapshot.activeTeam).toBe(2);

    // A fresh beginTurn() ran for the next turn: startTurn() called again, and initToken
    // re-fired for the new active team (1) — but NOT from inside handleResolveTurn itself,
    // only from beginTurn().
    expect(startTurn).toHaveBeenCalledTimes(2);
    expect(initToken).toHaveBeenCalledWith('team1-token');
  });

  it('always refreshes state even when resolveTurn() itself rejects', async () => {
    await setUpEmptyBoardAndClickResolve();
    vi.mocked(resolveTurn).mockRejectedValue(new Error('resolve failed'));

    const [, yesButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(yesButtonGraphics!)();
    await flush();

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'resolve failed',
      expect.objectContaining({})
    );
    expect(playResolveTurnEvents).not.toHaveBeenCalled();
  });

  // AC3: after a successful resolve, the animated end-state left by playResolveTurnEvents stands —
  // no wholesale occupant swap follows (which would snap sprites out of their finished animation).
  // The turn-advance UI (turnPanel + resolve button) still refreshes. (The former runtime
  // occupantsMatch "out of sync" check was removed in spec007 and relocated to a test oracle.)
  it('keeps the animated end-state on a successful resolve (no occupant rebuild) but refreshes turnPanel and the resolve button', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    queueMatchStates(
      makeState({ grid: [[plainTile()]], activeTeam: 1, units: [unit] }),
      // The post-resolve refetch reports a new turn number so turnPanel.update is observable.
      makeState({ grid: [[plainTile()]], turn: 5, activeTeam: 2, units: [unit] })
    );
    vi.mocked(resolveTurn).mockResolvedValue([]);

    await bootScene({ playerTokens: ['team1-token', 'team2-token'] });

    const unitGraphics = occupantGraphics(0);
    const resolveButton = resolveButtonGraphics(1); // grid + 1 unit + resolve button

    pointerDownOf(resolveButtonGraphics(1))();
    const [, yesButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(yesButtonGraphics!)();
    await flush();

    // No wholesale occupant swap after playback — the unit graphics is never destroyed.
    expect(unitGraphics.destroy).not.toHaveBeenCalled();
    // turnPanel refreshed with the post-resolve turn number...
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      '5',
      expect.objectContaining({})
    );
    // ...and the resolve button was re-rendered (old button graphics destroyed).
    expect(resolveButton.destroy).toHaveBeenCalled();
  });

  it('clears previous error messages once a new turn command begins, so they do not accumulate across turns', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [
      [{ type: 'TerrainPlain', occupantType: 'OccupantUnit', occupantId: 7 }, plainTile()],
    ];
    const target: Coordinate = { x: 1, y: 0 };

    queueMatchStates(makeState({ grid: initialGrid, activeTeam: 1, units: [unit] }));
    vi.mocked(getAllowedTiles).mockRejectedValueOnce(new Error('tiles unavailable'));

    await bootScene();

    const unitGraphics = occupantGraphics(0);
    pointerDownOf(unitGraphics)();
    const [moveButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(moveButtonGraphics!)();
    await flush();

    const staleErrorText = errorTextByMessage('tiles unavailable');
    expect(staleErrorText.destroy).not.toHaveBeenCalled();

    // Now successfully submit a Move for the same unit — a fresh turn-command action.
    vi.mocked(getAllowedTiles).mockResolvedValue([target]);
    vi.mocked(submitTurnCommand).mockResolvedValue([
      { type: 'unitMoved', unitId: 7, from: { x: 0, y: 0 }, to: target },
    ]);
    await submitViaUI(occupantGraphics(0), 0);
    await flush();

    expect(staleErrorText.destroy).toHaveBeenCalled();
  });

  describe('startTurn sequence', () => {
    it('calls getMatchState -> initToken -> startTurn -> TurnBanner.play in order for a normal turn', async () => {
      queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1 }));

      await bootScene();

      expect(startTurn).toHaveBeenCalledOnce();
      expect(suddenDeathCutscenePlay).not.toHaveBeenCalled();
      expect(turnBannerPlay).toHaveBeenCalledWith(1);
      const initTokenOrder = vi.mocked(initToken).mock.invocationCallOrder[0]!;
      const startTurnOrder = vi.mocked(startTurn).mock.invocationCallOrder[0]!;
      const bannerOrder = turnBannerPlay.mock.invocationCallOrder[0]!;
      expect(initTokenOrder).toBeLessThan(startTurnOrder);
      expect(startTurnOrder).toBeLessThan(bannerOrder);
    });

    it('plays the SuddenDeathCutscene with only the bombPlaced-typed gameEvents, then the TurnBanner, when inSuddenDeath is true', async () => {
      queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1 }));
      const bombEvent: GameEvent = {
        type: 'bombPlaced',
        unitId: 0,
        bombId: 1,
        position: { x: 0, y: 0 },
        range: 2,
        countdown: 3,
      };
      const otherEvent: GameEvent = { type: 'bombCountdownUpdated', bombId: 1, countdown: 2 };
      vi.mocked(startTurn).mockResolvedValue({
        inSuddenDeath: true,
        gameEvents: [bombEvent, otherEvent],
      });

      await bootScene();

      expect(suddenDeathCutscenePlay).toHaveBeenCalledWith([bombEvent], expect.any(Function));
      expect(turnBannerPlay).toHaveBeenCalledWith(1);
      const cutsceneOrder = suddenDeathCutscenePlay.mock.invocationCallOrder[0]!;
      const bannerOrder = turnBannerPlay.mock.invocationCallOrder[0]!;
      expect(cutsceneOrder).toBeLessThan(bannerOrder);
    });

    it('drops a sudden-death bomb as a single tweened container, not two independently-positioned objects', async () => {
      queueMatchStates(makeState({ grid: [[plainTile(), plainTile()]], activeTeam: 1 }));
      const bombEvent: GameEvent = {
        type: 'bombPlaced',
        unitId: 0,
        bombId: 1,
        position: { x: 1, y: 0 },
        range: 2,
        countdown: 3,
      };
      vi.mocked(startTurn).mockResolvedValue({ inSuddenDeath: true, gameEvents: [bombEvent] });

      await bootScene();

      const dropSuddenDeathBomb = suddenDeathCutscenePlay.mock.calls[0]![1];
      const dropPromise = dropSuddenDeathBomb(bombEvent);

      const bombContainer = mockScene.add.container.mock.results[0]!.value as ReturnType<
        typeof mockScene.add.container
      >;

      // Single target, single rest value — no second object that can drift out of sync.
      expect(mockScene.tweens.add).toHaveBeenCalledOnce();
      const tweenCfg = tweenConfigAt(0) as {
        targets: unknown;
        y: number;
        duration: number;
        ease: string;
        onComplete: () => void;
      };
      expect(tweenCfg.targets).toBe(bombContainer);
      expect(tweenCfg.y).toBe(24); // tileCenter of {x:1,y:0}: cy = 0*48 + 24
      expect(tweenCfg.duration).toBe(SUDDEN_DEATH_BOMB_DROP_DURATION_MS);
      expect(bombContainer.setDepth).toHaveBeenCalledWith(DEPTH_SUDDEN_DEATH_BOMB);

      tweenCfg.onComplete();
      await dropPromise;

      expect(bombContainer.setDepth).toHaveBeenLastCalledWith(DEPTH_OCCUPANT);
    });

    it('starts the sudden-death bomb drop a fixed margin above its own rest tile, not a fixed camera-height offset, so it starts off-screen regardless of the tile', async () => {
      queueMatchStates(makeState({ grid: [[plainTile(), plainTile()]], activeTeam: 1 }));
      const bombEvent: GameEvent = {
        type: 'bombPlaced',
        unitId: 0,
        bombId: 1,
        position: { x: 1, y: 0 },
        range: 2,
        countdown: 3,
      };
      vi.mocked(startTurn).mockResolvedValue({ inSuddenDeath: true, gameEvents: [bombEvent] });

      await bootScene();

      const dropSuddenDeathBomb = suddenDeathCutscenePlay.mock.calls[0]![1];
      void dropSuddenDeathBomb(bombEvent);

      const bombContainer = mockScene.add.container.mock.results[0]!.value as ReturnType<
        typeof mockScene.add.container
      >;

      // restY (tileCenter of {x:1,y:0}) is 24; the start position must sit BOMB_SIZE above it
      // (-24), not `restY - cameras.main.height` (-696 for the default 720px-tall mock camera) —
      // a fixed camera-height offset only clears the screen for tiles near the top of the board.
      expect(bombContainer.y).toBe(-24);
    });

    it('refreshes gameState (picking up the injected bomb) when inSuddenDeath is true, so a later resolveTurn validates that bombId', async () => {
      const bombEvent: GameEvent = {
        type: 'bombPlaced',
        unitId: 0,
        bombId: 999,
        position: { x: 0, y: 0 },
        range: 2,
        countdown: 3,
      };
      queueMatchStates(
        makeState({ grid: [[plainTile()]], activeTeam: 1 }),
        makeState({
          grid: [[plainTile()]],
          activeTeam: 1,
          bombs: [
            makeBomb({ id: 999, ownerId: 0, position: { x: 0, y: 0 }, range: 2, countdown: 3 }),
          ],
        })
      );
      vi.mocked(startTurn).mockResolvedValue({ inSuddenDeath: true, gameEvents: [bombEvent] });

      await bootScene();

      // getMatchState: initial board render, beginTurn()'s per-turn refresh, then a 3rd refresh
      // triggered by inSuddenDeath so gameState.bombs picks up the server-injected bomb.
      expect(getMatchState).toHaveBeenCalledTimes(3);

      vi.mocked(resolveTurn).mockResolvedValue([
        { type: 'bombCountdownUpdated', bombId: 999, countdown: 2 },
      ]);
      pointerDownOf(resolveButtonGraphics(0))();
      const [, yesButtonGraphics] = mockScene.add.graphics.mock.results
        .slice(-3)
        .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
      pointerDownOf(yesButtonGraphics!)();
      await flush();

      expect(playResolveTurnEvents).toHaveBeenCalledOnce();
      const [, deps] = vi.mocked(playResolveTurnEvents).mock.calls[0]!;
      expect(deps.gameStateSnapshot.bombs.map(b => b.id)).toContain(999);
    });

    it('disables unit-click and resolve-button interactions while startTurn is in progress, and re-enables them once TurnBanner finishes', async () => {
      const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
      queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1, units: [unit] }));
      let resolveBanner: () => void = () => undefined;
      turnBannerPlay.mockReturnValue(
        new Promise<void>(resolve => {
          resolveBanner = resolve;
        })
      );

      const scene = new MatchScene();
      scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
      await flush();

      const graphicsBefore = mockScene.add.graphics.mock.calls.length;
      pointerDownOf(occupantGraphics(0))();
      pointerDownOf(resolveButtonGraphics(1))();
      expect(mockScene.add.graphics.mock.calls.length).toBe(graphicsBefore);

      resolveBanner();
      await flush();

      pointerDownOf(occupantGraphics(0))();
      expect(mockScene.add.graphics.mock.calls.length).toBeGreaterThan(graphicsBefore);
    });

    it('does not resume beginTurn (play the TurnBanner or touch gameState) if the scene is shut down while startTurn is still in flight', async () => {
      queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1 }));
      let resolveStartTurn: (resp: {
        inSuddenDeath: boolean;
        gameEvents: GameEvent[];
      }) => void = () => undefined;
      vi.mocked(startTurn).mockReturnValue(
        new Promise(resolve => {
          resolveStartTurn = resolve;
        })
      );

      const scene = new MatchScene();
      scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
      await flush();

      fireShutdown();
      resolveStartTurn({ inSuddenDeath: false, gameEvents: [] });
      await flush();

      expect(turnBannerPlay).not.toHaveBeenCalled();
    });
  });

  describe('victory cutscene and rematch', () => {
    // winnerTeamId omitted entirely simulates the server sending a matchEnded event with no
    // winnerTeamId field at all (missing, not just out-of-range).
    async function resolveWithMatchEnded(winnerTeamId?: number): Promise<void> {
      await setUpEmptyBoardAndClickResolve();
      const event: GameEvent =
        winnerTeamId === undefined ? { type: 'matchEnded' } : { type: 'matchEnded', winnerTeamId };
      vi.mocked(resolveTurn).mockResolvedValue([event]);

      const [, yesButtonGraphics] = mockScene.add.graphics.mock.results
        .slice(-3)
        .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
      pointerDownOf(yesButtonGraphics!)();
      await flush();
    }

    it.each([1, 2, -1])(
      'renders VictoryCutscene(%i) instead of looping into another beginTurn when resolveTurn reports matchEnded',
      async winnerTeamId => {
        await resolveWithMatchEnded(winnerTeamId);

        expect(victoryCutscenePlay).toHaveBeenCalledWith(winnerTeamId, expect.any(Object));
        // Only one startTurn() call (the very first turn) — no second beginTurn() ran.
        expect(startTurn).toHaveBeenCalledTimes(1);
      }
    );

    it.each([
      ['out of range', 3],
      ['missing', undefined],
    ])(
      'shows an error and does not render VictoryCutscene when winnerTeamId is %s',
      async (_label, winnerTeamId) => {
        await resolveWithMatchEnded(winnerTeamId);

        expect(victoryCutscenePlay).not.toHaveBeenCalled();
        expect(mockScene.add.text).toHaveBeenCalledWith(
          expect.any(Number),
          expect.any(Number),
          'Invalid matchEnded event received from server',
          expect.objectContaining({})
        );
      }
    );

    it('permanently disables unit-click and resolve-button interactions once the match has ended', async () => {
      const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
      queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1, units: [unit] }));
      await bootScene();
      vi.mocked(resolveTurn).mockResolvedValue([{ type: 'matchEnded', winnerTeamId: 1 }]);

      pointerDownOf(resolveButtonGraphics(1))();
      const [, yesButtonGraphics] = mockScene.add.graphics.mock.results
        .slice(-3)
        .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
      pointerDownOf(yesButtonGraphics!)();
      await flush();

      const graphicsBefore = mockScene.add.graphics.mock.calls.length;
      pointerDownOf(occupantGraphics(0))();
      expect(mockScene.add.graphics.mock.calls.length).toBe(graphicsBefore);
    });

    it('rematch: fades out, then restarts the scene with the same roomId/playerTokens and isRematch=true', async () => {
      await resolveWithMatchEnded(1);

      const onRematch = victoryCutscenePlay.mock.calls[0]![1].onRematch;
      onRematch();

      expect(mockScene.cameras.main.fadeOut).toHaveBeenCalledWith(200, 0, 0, 0);
      expect(mockScene.scene.restart).not.toHaveBeenCalled();

      fireCameraFadeOutComplete();

      expect(mockScene.scene.restart).toHaveBeenCalledWith({
        roomId: 'room-abc',
        playerTokens: ['team1-token', 'team2-token'],
        isRematch: true,
      });
    });

    it('create() calls rematch() before getMatchState() and fades back in when isRematch is true', async () => {
      queueMatchStates(makeState({ grid: [[plainTile()]] }));
      vi.mocked(rematch).mockResolvedValue({ success: true, playerTokens: ['t1', 't2'] });

      await bootScene({ isRematch: true });

      expect(rematch).toHaveBeenCalledOnce();
      expect(getMatchState).toHaveBeenCalled();
      expect(mockScene.cameras.main.fadeIn).toHaveBeenCalledWith(200);
    });

    it('does not call rematch() on a normal (non-rematch) create()', async () => {
      queueMatchStates(makeState({ grid: [[plainTile()]] }));

      await bootScene();

      expect(rematch).not.toHaveBeenCalled();
      expect(mockScene.cameras.main.fadeIn).not.toHaveBeenCalled();
    });

    it('return to settings: fades out and calls deleteMatch() concurrently, then starts MatchSettingsScene once both settle', async () => {
      let resolveDelete: () => void = () => undefined;
      vi.mocked(deleteMatch).mockReturnValue(
        new Promise<void>(resolve => {
          resolveDelete = resolve;
        })
      );
      await resolveWithMatchEnded(1);

      const onReturnToSettings = victoryCutscenePlay.mock.calls[0]![1].onReturnToSettings;
      onReturnToSettings();

      expect(mockScene.cameras.main.fadeOut).toHaveBeenCalledWith(200, 0, 0, 0);
      expect(deleteMatch).toHaveBeenCalledOnce();
      expect(mockScene.scene.start).not.toHaveBeenCalled();

      fireCameraFadeOutComplete();
      await flush();
      expect(mockScene.scene.start).not.toHaveBeenCalled(); // fade done, but deleteMatch() hasn't settled yet

      resolveDelete();
      await flush();

      expect(mockScene.scene.start).toHaveBeenCalledWith('MatchSettingsScene');
    });

    it('logs the failure reason via console.error and still starts MatchSettingsScene when deleteMatch() rejects', async () => {
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => undefined);
      vi.mocked(deleteMatch).mockRejectedValue(new Error('delete failed'));
      await resolveWithMatchEnded(1);

      const onReturnToSettings = victoryCutscenePlay.mock.calls[0]![1].onReturnToSettings;
      onReturnToSettings();
      fireCameraFadeOutComplete();
      await flush();

      expect(consoleSpy).toHaveBeenCalledWith('Failed to delete match:', 'delete failed');
      expect(mockScene.scene.start).toHaveBeenCalledWith('MatchSettingsScene');
      consoleSpy.mockRestore();
    });

    it('ignores a second Rematch click while the first fade-out/restart is already in progress', async () => {
      await resolveWithMatchEnded(1);

      const onRematch = victoryCutscenePlay.mock.calls[0]![1].onRematch;
      onRematch();
      onRematch();

      expect(mockScene.cameras.main.fadeOut).toHaveBeenCalledTimes(1);
      fireCameraFadeOutComplete();
      expect(mockScene.scene.restart).toHaveBeenCalledTimes(1);
    });

    it('ignores a second Return-to-Settings click while the first fade-out/delete is already in progress', async () => {
      await resolveWithMatchEnded(1);

      const onReturnToSettings = victoryCutscenePlay.mock.calls[0]![1].onReturnToSettings;
      onReturnToSettings();
      onReturnToSettings();

      expect(mockScene.cameras.main.fadeOut).toHaveBeenCalledTimes(1);
      expect(deleteMatch).toHaveBeenCalledTimes(1);
      fireCameraFadeOutComplete();
      await flush();
      expect(mockScene.scene.start).toHaveBeenCalledTimes(1);
    });

    it('a stale getMatchState() fetch from before a rematch restart does not touch the new scene once it resolves', async () => {
      let resolveStaleState: (state: GameState) => void = () => undefined;
      vi.mocked(getMatchState).mockReturnValueOnce(
        new Promise<GameState>(resolve => {
          resolveStaleState = resolve;
        })
      );

      const scene = new MatchScene();
      scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
      await flush();

      // Simulate scene.restart(): shutdown fires (bumping generation), then create() runs again.
      fireShutdown();
      queueMatchStates(makeState({ grid: [[plainTile()]] }));
      scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'], isRematch: true });
      await flush();

      const centerOnCallsBeforeStaleResolve = mockScene.cameras.main.centerOn.mock.calls.length;
      resolveStaleState(makeState({ grid: [[plainTile(), plainTile()]] }));
      await flush();

      // The stale fetch's own board dimensions (2 cols) never reach centerOn — no new call at all
      // from it, since it bails out on the generation mismatch.
      expect(mockScene.cameras.main.centerOn.mock.calls.length).toBe(
        centerOnCallsBeforeStaleResolve
      );
    });
  });
});
