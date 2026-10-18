import { describe, it, expect, beforeEach, vi, type MockInstance } from 'vitest';
import { mockScene } from '../test/setup';
import {
  stageBackgroundImage,
  occupantGraphics,
  occupantSprite,
  lastGraphics,
  lastContainers,
  pointerDownOf,
  textByContent,
  flush,
  tweenConfigAt,
  fireShutdown,
  fireLastShutdownListener,
  fireCameraFadeOutComplete,
  delayedCallAt,
  drainDelayedCalls,
} from '../test/sceneHelpers';
import {
  makeCfg,
  makeState,
  plainTile,
  tileOf,
  makeUnit,
  makeBomb,
  makeBombPlacedEvent,
} from '../test/fixtures';
import {
  initRoom,
  initToken,
  getMatchState,
  getMatchConfig,
  getAllowedTiles,
  submitTurnCommand,
  resolveTurn,
  resetTurn,
  surrender,
  startTurn,
  rematch,
  deleteMatch,
  consumeCpuStatus,
} from '../api/client';
import { playResolveTurnEvents } from '../rendering/resolveTurnPlayer';
import TurnBanner from '../ui/TurnBanner';
import SuddenDeathCutscene from '../ui/SuddenDeathCutscene';
import VictoryCutscene from '../ui/VictoryCutscene';
import MatchScene, { type MatchSceneData } from './MatchScene';
import type {
  Coordinate,
  CpuStatusResponse,
  GameCfg,
  GameEvent,
  GameState,
  Tile,
} from '../api/types';
import {
  DEPTH_SUDDEN_DEATH_BOMB,
  DEPTH_OCCUPANT,
  SUDDEN_DEATH_BOMB_DROP_DURATION_MS,
  CONFIRM_TEXT_RESET,
  CPU_POLL_BACKOFF_MS,
  CPU_PLAN_RESOLVE_HOLD_MS,
  CPU_PLAN_EVENT_STEP_MS,
} from '../constants';

vi.mock('../api/client');
vi.mock('../rendering/resolveTurnPlayer');
vi.mock('../ui/TurnBanner');
vi.mock('../ui/SuddenDeathCutscene');
vi.mock('../ui/VictoryCutscene');

// create() and beginTurn() both consume states[0]; the last state is the persistent fallback.
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

async function bootScene(overrides: Partial<MatchSceneData> = {}): Promise<MatchScene> {
  const scene = new MatchScene();
  scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'], ...overrides });
  await flush();
  return scene;
}

function matchSummaryButtonGraphics(): ReturnType<typeof mockScene.add.graphics> {
  return occupantGraphics(0);
}

function clickSummaryPanelButton(button: 'resolve' | 'reset' | 'surrender' | 'back'): void {
  pointerDownOf(matchSummaryButtonGraphics())();
  const offset = { resolve: 1, reset: 2, surrender: 3, back: 4 }[button];
  const buttonGraphics = lastGraphics(5)[offset]!;
  pointerDownOf(buttonGraphics)();
}

async function expectActionStackClearedBy(
  button: 'resolve' | 'reset' | 'surrender'
): Promise<void> {
  const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
  queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1, units: [unit] }));
  vi.mocked(getAllowedTiles).mockResolvedValue([{ x: 0, y: 0 }]);

  await bootScene();

  pointerDownOf(occupantSprite(0))();
  const [moveButtonContainer] = lastContainers(3);

  clickSummaryPanelButton(button);

  expect(moveButtonContainer!.destroy).toHaveBeenCalled();
}

// Drives click unit -> Move/Bomb -> allowed tile -> Yes, with getAllowedTiles mocked to one tile.
async function submitViaUI(
  unitGraphics: ReturnType<typeof mockScene.add.sprite>,
  buttonIndex: 0 | 1 // 0 = Move, 1 = Bomb
): Promise<void> {
  pointerDownOf(unitGraphics)();
  const actionButtonContainer = lastContainers(3)[buttonIndex];
  pointerDownOf(actionButtonContainer!)();
  await flush();
  const [overlayTileGraphics] = lastGraphics(1);
  pointerDownOf(overlayTileGraphics!)();
  const [yesButtonContainer] = lastContainers(2);
  pointerDownOf(yesButtonContainer!)();
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

  describe('camera setup', () => {
    it('renders through the single default main camera, with no cameras.add call', async () => {
      const defaultCamera = mockScene.cameras.main;

      await bootScene();

      expect(mockScene.cameras.add).not.toHaveBeenCalled();
      expect(mockScene.cameras.remove).not.toHaveBeenCalled();
      expect(mockScene.cameras.main).toBe(defaultCamera);
    });
  });

  it('never centers the camera on the grid — it stays scrolled to world origin, anchoring the grid top-left of GameBoardRegion', async () => {
    const row = (): Tile[] => [plainTile(), plainTile(), plainTile()];
    queueMatchStates(makeState({ grid: [row(), row()] }));

    await bootScene();

    expect(mockScene.cameras.main.centerOn).not.toHaveBeenCalled();
  });

  it('calls initToken exactly once at turn start (beginTurn), not on unit click', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1, units: [unit] }));

    await bootScene({ playerTokens: ['team1-token', 'team2-token'] });
    expect(initToken).toHaveBeenCalledWith('team1-token');
    expect(initToken).toHaveBeenCalledOnce();

    const containersBefore = mockScene.add.container.mock.calls.length;
    pointerDownOf(occupantSprite(0))();

    expect(mockScene.add.container.mock.calls.length).toBe(containersBefore + 3);
    expect(initToken).toHaveBeenCalledOnce();
  });

  it('does not open TurnCommandPanel or call initToken again when clicking a unit off the active team', async () => {
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => undefined);
    const unit = makeUnit({ id: 7, team: 2, position: { x: 0, y: 0 } });
    queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1, units: [unit] }));

    await bootScene();
    expect(initToken).toHaveBeenCalledOnce();

    const graphicsBefore = mockScene.add.graphics.mock.calls.length;
    pointerDownOf(occupantSprite(0))();

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

    const onUnitPointerDown = pointerDownOf(occupantSprite(0));

    onUnitPointerDown();
    const [moveButtonContainer] = lastContainers(3);
    const clickMove = (): void => pointerDownOf(moveButtonContainer!)();
    clickMove();
    await flush();

    onUnitPointerDown();
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

    const onUnitPointerDown = pointerDownOf(occupantSprite(0));
    onUnitPointerDown();

    const [moveButtonContainer] = lastContainers(3);
    pointerDownOf(moveButtonContainer!)();
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

    expect(mockScene.add.graphics).toHaveBeenCalledTimes(3);
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

    const unitGraphics = occupantSprite(0);
    await submitViaUI(unitGraphics, 0);
    await flush();

    expect(submitTurnCommand).toHaveBeenCalledWith({ type: 'move', unitId: 7, target });
    expect(mockScene.tweens.add).toHaveBeenCalledWith(
      expect.objectContaining({ targets: unitGraphics, x: 256, y: 212, ease: 'Linear' })
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

    const unitGraphics = occupantSprite(0);
    await submitViaUI(unitGraphics, 1);
    await flush();

    expect(submitTurnCommand).toHaveBeenCalledWith({ type: 'placeBomb', unitId: 7, target });
    expect(mockScene.add.container).toHaveBeenCalledWith(256, 180, expect.any(Array));
    expect(unitGraphics.destroy).not.toHaveBeenCalled();
  });

  it('disables the Bomb button on a second click after a successful bomb placement, even without an occupant rebuild', async () => {
    const unit = makeUnit({
      id: 7,
      team: 1,
      position: { x: 0, y: 0 },
      maxBombCount: 2,
      hasUsedSkill: false,
    });
    const updatedUnit = makeUnit({
      id: 7,
      team: 1,
      position: { x: 0, y: 0 },
      maxBombCount: 2,
      bombUsed: 1,
      hasUsedSkill: true,
    });
    const initialGrid: Tile[][] = [[plainTile(), plainTile()]];
    const target: Coordinate = { x: 1, y: 0 };

    queueMatchStates(
      makeState({ grid: initialGrid, activeTeam: 1, units: [unit] }),
      makeState({
        grid: initialGrid,
        activeTeam: 1,
        units: [updatedUnit],
        bombs: [makeBomb({ id: 42, ownerId: 7, position: target, countdown: 3 })],
      })
    );
    vi.mocked(getAllowedTiles).mockResolvedValue([target]);
    vi.mocked(submitTurnCommand).mockResolvedValue([
      { type: 'bombPlaced', unitId: 7, bombId: 42, position: target, range: 2, countdown: 3 },
    ]);

    await bootScene();

    const unitGraphics = occupantSprite(0);
    await submitViaUI(unitGraphics, 1);
    await flush();

    pointerDownOf(unitGraphics)();
    const [, placeBombButtonContainer] = lastContainers(3);

    expect(placeBombButtonContainer!.setInteractive).not.toHaveBeenCalled();
  });

  it('shows the actual error and resyncs by rebuilding occupants (terrain untouched) when submitTurnCommand rejects', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [[plainTile(), plainTile()]];
    const target: Coordinate = { x: 1, y: 0 };

    queueMatchStates(makeState({ grid: initialGrid, activeTeam: 1, units: [unit] }));
    vi.mocked(getAllowedTiles).mockResolvedValue([target]);
    vi.mocked(submitTurnCommand).mockRejectedValue(new Error('stale token'));

    await bootScene();

    const gridGraphics = stageBackgroundImage();
    const unitGraphics = occupantSprite(0);
    await submitViaUI(unitGraphics, 0);
    await flush();

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'stale token',
      expect.objectContaining({})
    );
    expect(unitGraphics.destroy).toHaveBeenCalled();
    expect(gridGraphics.destroy).not.toHaveBeenCalled();
  });

  it('shows an error and stops processing when the unitMoved event is malformed (out-of-bounds target)', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [[plainTile(), plainTile()]];
    const target: Coordinate = { x: 1, y: 0 };

    queueMatchStates(makeState({ grid: initialGrid, activeTeam: 1, units: [unit] }));
    vi.mocked(getAllowedTiles).mockResolvedValue([target]);
    const events: GameEvent[] = [
      { type: 'unitMoved', unitId: 7, from: { x: 0, y: 0 }, to: { x: 99, y: 99 } },
    ];
    vi.mocked(submitTurnCommand).mockResolvedValue(events);

    await bootScene();

    const unitGraphics = occupantSprite(0);
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

    const unitGraphics = occupantSprite(0);
    await submitViaUI(unitGraphics, 0);
    await flush();

    expect(mockScene.tweens.add).toHaveBeenCalledWith(
      expect.objectContaining({ targets: unitGraphics, x: 256, y: 212, ease: 'Linear' })
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
    const events: GameEvent[] = [
      { type: 'unitMoved', unitId: 7, from: { x: 0, y: 0 }, to: actualTo },
    ];
    vi.mocked(submitTurnCommand).mockResolvedValue(events);

    await bootScene();

    const unitGraphics = occupantSprite(0);
    await submitViaUI(unitGraphics, 0);
    await flush();

    expect(mockScene.tweens.add).toHaveBeenCalledWith(
      expect.objectContaining({ targets: unitGraphics, x: 272, y: 212, ease: 'Linear' })
    );
    expect(mockScene.add.text).not.toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Failed to refresh match state',
      expect.objectContaining({})
    );
  });

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

    const gridGraphics = stageBackgroundImage();
    const unitGraphics = occupantSprite(0);
    const refetchesBefore = vi.mocked(getMatchState).mock.calls.length;

    await submitViaUI(unitGraphics, 0);
    await flush();

    expect(unitGraphics.destroy).not.toHaveBeenCalled();
    expect(gridGraphics.destroy).not.toHaveBeenCalled();
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

    const unitGraphics = occupantSprite(0);
    pointerDownOf(unitGraphics)();
    const [moveButtonContainer] = lastContainers(3);
    pointerDownOf(moveButtonContainer!)();
    await flush();
    const [overlayTileGraphics] = lastGraphics(1);
    pointerDownOf(overlayTileGraphics!)();

    const [yesButtonContainer] = lastContainers(2);
    const graphicsCallCountBeforeClick = mockScene.add.graphics.mock.calls.length;

    pointerDownOf(unitGraphics)();

    expect(yesButtonContainer!.destroy).not.toHaveBeenCalled();
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

    const unitGraphics = occupantSprite(0);
    pointerDownOf(unitGraphics)();
    const [moveButtonContainer] = lastContainers(3);
    pointerDownOf(moveButtonContainer!)();
    await flush();
    const [overlayTileGraphics] = lastGraphics(1);
    pointerDownOf(overlayTileGraphics!)();
    const [firstYesButtonContainer] = lastContainers(2);

    pointerDownOf(firstYesButtonContainer!)();
    pointerDownOf(overlayTileGraphics!)();
    const [secondYesButtonContainer] = lastContainers(2);
    pointerDownOf(secondYesButtonContainer!)();

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

    const unitGraphics = occupantSprite(0);
    await submitViaUI(unitGraphics, 0);
    await flush();

    // The mock tween never animates, so advance the sprite to where the first move left it.
    unitGraphics.x = 48;
    unitGraphics.y = 48;

    pointerDownOf(unitGraphics)();
    const [moveButtonContainer] = lastContainers(3);
    pointerDownOf(moveButtonContainer!)();
    await flush();
    const [overlayTileGraphics] = lastGraphics(1);
    pointerDownOf(overlayTileGraphics!)();
    const [yesButtonContainer] = lastContainers(2);
    pointerDownOf(yesButtonContainer!)();
    await flush();

    expect(mockScene.tweens.add).toHaveBeenLastCalledWith(
      expect.objectContaining({ targets: unitGraphics, x: 80, y: 48 })
    );
  });

  it('fetches gameCfg alongside match state and renders TurnPanel with turn/maxTurns/activeTeam', async () => {
    queueMatchStates(makeState({ grid: [[plainTile()]], turn: 4, activeTeam: 2 }));
    vi.mocked(getMatchConfig).mockResolvedValue(makeCfg({ maxTurns: 12 }));

    await bootScene();

    expect(getMatchConfig).toHaveBeenCalledOnce();
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

  it('renders the Stage background matching gameCfg.stagePreset once both state and config have loaded', async () => {
    queueMatchStates(makeState({ grid: [[plainTile()]] }));
    vi.mocked(getMatchConfig).mockResolvedValue(makeCfg({ stagePreset: 'Divided' }));

    await bootScene();

    expect(mockScene.add.image).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'stage_divided',
      expect.any(String)
    );
  });

  it('does not render a Stage background while gameCfg is still loading', async () => {
    queueMatchStates(makeState({ grid: [[plainTile()]] }));
    vi.mocked(getMatchConfig).mockReturnValue(new Promise<GameCfg>(() => undefined));

    await bootScene();

    expect(mockScene.add.image).not.toHaveBeenCalled();
  });

  async function setUpEmptyBoardAndClickResolve(activeTeam = 1): Promise<void> {
    queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam }));

    await bootScene({ playerTokens: ['team1-token', 'team2-token'] });

    clickSummaryPanelButton('resolve');
  }

  it('pins MatchSummaryButton to the camera viewport (scrollFactor 0) instead of the grid/world', async () => {
    queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1 }));

    await bootScene();

    expect(matchSummaryButtonGraphics().setScrollFactor).toHaveBeenCalledWith(0);
  });

  it('shows an error instead of crashing when MatchSummaryButton is clicked before match config has loaded', async () => {
    queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1 }));
    vi.mocked(getMatchConfig).mockReturnValue(new Promise<GameCfg>(() => undefined));

    await bootScene();

    expect(() => pointerDownOf(matchSummaryButtonGraphics())()).not.toThrow();
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      expect.stringContaining('loading'),
      expect.objectContaining({})
    );
    expect(resolveTurn).not.toHaveBeenCalled();
  });

  it('clears the TurnCommandPanel action stack (treating it as nothing selected) when ResolveTurnButton opens the confirm dialog', async () => {
    await expectActionStackClearedBy('resolve');
  });

  it('opens the resolve confirm even when a TurnCommandPanel confirm is already open, so a stale action never blocks it', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [
      [{ type: 'TerrainPlain', occupantType: 'OccupantUnit', occupantId: 7 }, plainTile()],
    ];
    queueMatchStates(makeState({ grid: initialGrid, activeTeam: 1, units: [unit] }));
    vi.mocked(getAllowedTiles).mockResolvedValue([{ x: 1, y: 0 }]);

    await bootScene();

    pointerDownOf(occupantSprite(0))();
    const [moveButtonContainer] = lastContainers(3);
    pointerDownOf(moveButtonContainer!)();
    await flush();
    const [overlayTile] = lastGraphics(1);
    pointerDownOf(overlayTile!)();

    clickSummaryPanelButton('resolve');

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

  it('on confirmed resolve: inits the active team token, calls resolveTurn, hands resolveTurnGameEvents to the player, then loops back into beginTurn for the next turn', async () => {
    await setUpEmptyBoardAndClickResolve(2);
    expect(initToken).toHaveBeenCalledWith('team2-token');
    vi.mocked(getMatchState).mockResolvedValue(makeState({ grid: [[plainTile()]], activeTeam: 1 }));
    const resolveTurnGameEvents: GameEvent[] = [
      { type: 'bombCountdownUpdated', bombId: 1, countdown: 2 },
    ];
    vi.mocked(resolveTurn).mockResolvedValue({ planGameEvents: [], resolveTurnGameEvents });

    const [yesButtonContainer] = lastContainers(2);
    pointerDownOf(yesButtonContainer!)();
    await flush();

    expect(resolveTurn).toHaveBeenCalledOnce();
    expect(playResolveTurnEvents).toHaveBeenCalledOnce();
    const [calledEvents, deps] = vi.mocked(playResolveTurnEvents).mock.calls[0]!;
    expect(calledEvents).toBe(resolveTurnGameEvents);
    expect(deps.gameStateSnapshot.activeTeam).toBe(2);

    expect(startTurn).toHaveBeenCalledTimes(2);
    expect(initToken).toHaveBeenCalledWith('team1-token');
  });

  it('discards planGameEvents from the resolve response instead of replaying them', async () => {
    await setUpEmptyBoardAndClickResolve();
    const planGameEvents: GameEvent[] = [
      { type: 'unitMoved', unitId: 1, from: { x: 0, y: 0 }, to: { x: 1, y: 0 } },
    ];
    const resolveTurnGameEvents: GameEvent[] = [
      { type: 'bombExploded', bombId: 1, position: { x: 1, y: 0 }, affectedPositions: [] },
    ];
    vi.mocked(resolveTurn).mockResolvedValue({ planGameEvents, resolveTurnGameEvents });

    const [yesButtonContainer] = lastContainers(2);
    pointerDownOf(yesButtonContainer!)();
    await flush();

    expect(playResolveTurnEvents).toHaveBeenCalledOnce();
    const [calledEvents] = vi.mocked(playResolveTurnEvents).mock.calls[0]!;
    expect(calledEvents).toBe(resolveTurnGameEvents);
    expect(calledEvents).not.toContain(planGameEvents[0]);
  });

  it('always refreshes state even when resolveTurn() itself rejects', async () => {
    await setUpEmptyBoardAndClickResolve();
    vi.mocked(resolveTurn).mockRejectedValue(new Error('resolve failed'));

    const [yesButtonContainer] = lastContainers(2);
    pointerDownOf(yesButtonContainer!)();
    await flush();

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'resolve failed',
      expect.objectContaining({})
    );
    expect(playResolveTurnEvents).not.toHaveBeenCalled();
  });

  it('keeps the animated end-state on a successful resolve (no occupant rebuild) but refreshes turnPanel', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    queueMatchStates(
      makeState({ grid: [[plainTile()]], activeTeam: 1, units: [unit] }),
      makeState({ grid: [[plainTile()]], turn: 5, activeTeam: 2, units: [unit] })
    );
    vi.mocked(resolveTurn).mockResolvedValue({ planGameEvents: [], resolveTurnGameEvents: [] });

    await bootScene({ playerTokens: ['team1-token', 'team2-token'] });

    const unitGraphics = occupantSprite(0);

    clickSummaryPanelButton('resolve');
    const [yesButtonContainer] = lastContainers(2);
    pointerDownOf(yesButtonContainer!)();
    await flush();

    expect(unitGraphics.destroy).not.toHaveBeenCalled();
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      '5',
      expect.objectContaining({})
    );
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

    const unitGraphics = occupantSprite(0);
    pointerDownOf(unitGraphics)();
    const [moveButtonContainer] = lastContainers(3);
    pointerDownOf(moveButtonContainer!)();
    await flush();

    const staleErrorText = textByContent('tiles unavailable');
    expect(staleErrorText.destroy).not.toHaveBeenCalled();

    vi.mocked(getAllowedTiles).mockResolvedValue([target]);
    vi.mocked(submitTurnCommand).mockResolvedValue([
      { type: 'unitMoved', unitId: 7, from: { x: 0, y: 0 }, to: target },
    ]);
    await submitViaUI(occupantSprite(0), 0);
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

      expect(mockScene.tweens.add).toHaveBeenCalledOnce();
      const tweenCfg = tweenConfigAt(0) as {
        targets: unknown;
        y: number;
        duration: number;
        ease: string;
        onComplete: () => void;
      };
      expect(tweenCfg.targets).toBe(bombContainer);
      expect(tweenCfg.y).toBe(180);
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

      expect(getMatchState).toHaveBeenCalledTimes(3);

      vi.mocked(resolveTurn).mockResolvedValue({
        planGameEvents: [],
        resolveTurnGameEvents: [{ type: 'bombCountdownUpdated', bombId: 999, countdown: 2 }],
      });
      clickSummaryPanelButton('resolve');
      const [yesButtonContainer] = lastContainers(2);
      pointerDownOf(yesButtonContainer!)();
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

      const containersBefore = mockScene.add.container.mock.calls.length;
      pointerDownOf(occupantSprite(0))();
      pointerDownOf(matchSummaryButtonGraphics())();
      expect(mockScene.add.container.mock.calls.length).toBe(containersBefore);

      resolveBanner();
      await flush();

      pointerDownOf(occupantSprite(0))();
      expect(mockScene.add.container.mock.calls.length).toBeGreaterThan(containersBefore);
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

  describe('VS CPU turn', () => {
    function readyStatus(overrides: Partial<CpuStatusResponse> = {}): CpuStatusResponse {
      return {
        turnPhase: 'TurnPhaseReady',
        planGameEvents: [],
        resolveTurnGameEvents: [],
        ...overrides,
      };
    }

    beforeEach(() => {
      vi.mocked(getMatchConfig).mockResolvedValue(makeCfg({ vsCpu: true }));
    });

    it('does not animate the CPU turn until its own TurnBanner has finished, even when polling resolves instantly (AC 6, 7)', async () => {
      queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 2, turn: 1 }));
      // Ends the match so the CPU turn terminates instead of recursing into another beginTurn().
      vi.mocked(consumeCpuStatus).mockResolvedValue(
        readyStatus({ resolveTurnGameEvents: [{ type: 'matchEnded', winnerTeamId: 2 }] })
      );
      let resolveCpuBanner: () => void = () => undefined;
      turnBannerPlay.mockReturnValue(
        new Promise<void>(resolve => {
          resolveCpuBanner = resolve;
        })
      );

      await bootScene();
      await flush();

      expect(playResolveTurnEvents).not.toHaveBeenCalled();

      resolveCpuBanner();
      await flush();

      expect(playResolveTurnEvents).toHaveBeenCalled();
    });

    it('polls consumeCpuStatus once startTurn resolves, animates plan -> hold -> resolve, then opens the next turn (AC 6, 7, 11)', async () => {
      queueMatchStates(
        makeState({ grid: [[plainTile()]], activeTeam: 2, turn: 1 }),
        makeState({ grid: [[plainTile()]], activeTeam: 1, turn: 2 })
      );
      const moveEvent: GameEvent = {
        type: 'unitMoved',
        unitId: 0x21,
        from: { x: 0, y: 0 },
        to: { x: 0, y: 0 },
      };
      const damagedEvent: GameEvent = {
        type: 'unitDamaged',
        unitId: 0x21,
        newHp: 1,
        position: { x: 0, y: 0 },
      };
      vi.mocked(consumeCpuStatus).mockResolvedValue(
        readyStatus({ planGameEvents: [moveEvent], resolveTurnGameEvents: [damagedEvent] })
      );

      await bootScene();
      await flush();

      expect(consumeCpuStatus).toHaveBeenCalled();
      delayedCallAt(CPU_PLAN_RESOLVE_HOLD_MS)();
      await flush();

      expect(playResolveTurnEvents).toHaveBeenCalledWith(
        [damagedEvent],
        expect.objectContaining({})
      );
      expect(startTurn).toHaveBeenCalledTimes(2);
      expect(turnBannerPlay).toHaveBeenCalledWith(1);
    });

    it("locally reflects the CPU's own bombPlaced plan event in gameStateSnapshot, without a server round trip (AC 1, 3)", async () => {
      const preState = makeState({ grid: [[plainTile()]], activeTeam: 2, turn: 1 });
      const nextTurnState = makeState({ grid: [[plainTile()]], activeTeam: 1, turn: 2 });
      queueMatchStates(preState, nextTurnState);
      const bombPlacedEvent = makeBombPlacedEvent({ bombId: 5 });
      const countdownEvent: GameEvent = { type: 'bombCountdownUpdated', bombId: 5, countdown: 2 };
      vi.mocked(consumeCpuStatus).mockResolvedValue(
        readyStatus({
          planGameEvents: [bombPlacedEvent],
          resolveTurnGameEvents: [countdownEvent],
        })
      );

      await bootScene();
      await flush();
      delayedCallAt(CPU_PLAN_RESOLVE_HOLD_MS)();
      await flush();

      const [events, options] = vi.mocked(playResolveTurnEvents).mock.calls.at(-1)!;
      expect(events).toEqual([countdownEvent]);
      expect(options.gameStateSnapshot.bombs).toContainEqual({
        id: 5,
        ownerId: 0,
        position: { x: 0, y: 0 },
        range: 2,
        countdown: 3,
      });
    });

    it("keeps a bomb from an earlier turn in gameStateSnapshot even though it explodes during this turn's resolution", async () => {
      const oldBomb = makeBomb({ id: 9 });
      const preState = makeState({
        grid: [[plainTile()]],
        activeTeam: 2,
        turn: 6,
        bombs: [oldBomb],
      });
      const nextTurnState = makeState({ grid: [[plainTile()]], activeTeam: 1, turn: 7 });
      queueMatchStates(preState, nextTurnState);
      const countdownEvent: GameEvent = { type: 'bombCountdownUpdated', bombId: 9, countdown: 0 };
      const explodedEvent: GameEvent = {
        type: 'bombExploded',
        bombId: 9,
        position: { x: 0, y: 0 },
        affectedPositions: [],
      };
      vi.mocked(consumeCpuStatus).mockResolvedValue(
        readyStatus({ resolveTurnGameEvents: [countdownEvent, explodedEvent] })
      );

      await bootScene();
      await flush();

      const [events, options] = vi.mocked(playResolveTurnEvents).mock.calls.at(-1)!;
      expect(events).toEqual([countdownEvent, explodedEvent]);
      expect(options.gameStateSnapshot.bombs).toContainEqual(oldBomb);
    });

    it('animates planGameEvents one at a time, holding CPU_PLAN_EVENT_STEP_MS between each (AC 2)', async () => {
      const unitA = makeUnit({ id: 0x21, team: 2, position: { x: 0, y: 0 } });
      const unitB = makeUnit({ id: 0x22, team: 2, position: { x: 0, y: 0 } });
      queueMatchStates(
        makeState({
          grid: [[plainTile()]],
          activeTeam: 2,
          turn: 1,
          units: [unitA, unitB],
        }),
        makeState({ grid: [[plainTile()]], activeTeam: 1, turn: 2, units: [unitA, unitB] })
      );
      const firstMove: GameEvent = {
        type: 'unitMoved',
        unitId: 0x21,
        from: { x: 0, y: 0 },
        to: { x: 0, y: 0 },
      };
      const secondMove: GameEvent = {
        type: 'unitMoved',
        unitId: 0x22,
        from: { x: 0, y: 0 },
        to: { x: 0, y: 0 },
      };
      vi.mocked(consumeCpuStatus).mockResolvedValue(
        readyStatus({ planGameEvents: [firstMove, secondMove] })
      );

      await bootScene();
      await flush();
      await flush();

      expect(mockScene.tweens.add).toHaveBeenCalledTimes(1);

      delayedCallAt(CPU_PLAN_EVENT_STEP_MS)();
      await flush();

      expect(mockScene.tweens.add).toHaveBeenCalledTimes(2);
    });

    it('skips the 600ms hold when planGameEvents is empty (AC 10)', async () => {
      queueMatchStates(
        makeState({ grid: [[plainTile()]], activeTeam: 2, turn: 1 }),
        makeState({ grid: [[plainTile()]], activeTeam: 1, turn: 2 })
      );
      vi.mocked(consumeCpuStatus).mockResolvedValue(readyStatus());

      await bootScene();

      expect(mockScene.time.delayedCall).not.toHaveBeenCalledWith(
        CPU_PLAN_RESOLVE_HOLD_MS,
        expect.any(Function)
      );
      expect(startTurn).toHaveBeenCalledTimes(2);
    });

    it('retries TurnPhasePlanning with backoff 250/500/1000/2000ms before succeeding', async () => {
      queueMatchStates(
        makeState({ grid: [[plainTile()]], activeTeam: 2, turn: 1 }),
        makeState({ grid: [[plainTile()]], activeTeam: 1, turn: 2 })
      );
      vi.mocked(consumeCpuStatus)
        .mockResolvedValueOnce({
          turnPhase: 'TurnPhasePlanning',
          planGameEvents: [],
          resolveTurnGameEvents: [],
        })
        .mockResolvedValueOnce({
          turnPhase: 'TurnPhasePlanning',
          planGameEvents: [],
          resolveTurnGameEvents: [],
        })
        .mockResolvedValue(readyStatus());

      await bootScene();
      await flush();
      delayedCallAt(CPU_POLL_BACKOFF_MS[0])();
      await flush();
      delayedCallAt(CPU_POLL_BACKOFF_MS[1])();
      await flush();

      expect(consumeCpuStatus).toHaveBeenCalledTimes(3);
      expect(startTurn).toHaveBeenCalledTimes(2);
    });

    it('locks unit-click interactions through the whole CPU turn and only releases after the Player TurnBanner finishes (AC 6)', async () => {
      const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
      queueMatchStates(
        makeState({ grid: [[plainTile()]], activeTeam: 2, turn: 1, units: [unit] }),
        makeState({ grid: [[plainTile()]], activeTeam: 1, turn: 2, units: [unit] })
      );
      vi.mocked(consumeCpuStatus).mockResolvedValue(readyStatus());
      let resolveHumanBanner: () => void = () => undefined;
      turnBannerPlay.mockImplementation(activeTeam => {
        if (activeTeam === 1) {
          return new Promise<void>(resolve => {
            resolveHumanBanner = resolve;
          });
        }
        return Promise.resolve();
      });

      await bootScene();
      await flush();

      const containersBefore = mockScene.add.container.mock.calls.length;
      pointerDownOf(occupantSprite(0))();
      expect(mockScene.add.container.mock.calls.length).toBe(containersBefore);

      resolveHumanBanner();
      await flush();

      pointerDownOf(occupantSprite(0))();
      expect(mockScene.add.container.mock.calls.length).toBeGreaterThan(containersBefore);
    });

    it('opens VictoryCutscene instead of beginTurn() when resolveTurnGameEvents ends with matchEnded (AC 11)', async () => {
      queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 2, turn: 1 }));
      vi.mocked(consumeCpuStatus).mockResolvedValue(
        readyStatus({ resolveTurnGameEvents: [{ type: 'matchEnded', winnerTeamId: 2 }] })
      );

      await bootScene();

      expect(victoryCutscenePlay).toHaveBeenCalledWith(2, expect.any(String), expect.any(Object));
      expect(startTurn).toHaveBeenCalledTimes(1);
    });

    it('re-renders silently from getMatchState() when polling times out but the turn advanced (AC 12)', async () => {
      queueMatchStates(
        makeState({ grid: [[plainTile()]], activeTeam: 2, turn: 1 }),
        makeState({ grid: [[plainTile()]], activeTeam: 1, turn: 2 })
      );
      vi.mocked(consumeCpuStatus).mockResolvedValue({
        turnPhase: 'TurnPhasePlanning',
        planGameEvents: [],
        resolveTurnGameEvents: [],
      });
      const dateNowSpy = vi.spyOn(Date, 'now').mockReturnValueOnce(0).mockReturnValue(31000);

      await bootScene();
      await flush();
      fireCameraFadeOutComplete();
      await flush();
      dateNowSpy.mockRestore();

      expect(mockScene.add.text).not.toHaveBeenCalledWith(
        expect.any(Number),
        expect.any(Number),
        expect.stringContaining('did not complete'),
        expect.anything()
      );
      expect(startTurn).toHaveBeenCalledTimes(2);
    });

    it('surfaces an error and never force-ends the match when polling times out and the turn is unchanged (AC 12)', async () => {
      queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 2, turn: 1 }));
      vi.mocked(consumeCpuStatus).mockResolvedValue({
        turnPhase: 'TurnPhasePlanning',
        planGameEvents: [],
        resolveTurnGameEvents: [],
      });
      const dateNowSpy = vi.spyOn(Date, 'now').mockReturnValueOnce(0).mockReturnValue(31000);

      await bootScene();
      await flush();
      dateNowSpy.mockRestore();

      expect(victoryCutscenePlay).not.toHaveBeenCalled();
      expect(startTurn).toHaveBeenCalledTimes(1);
    });

    it('retries via the backoff schedule while still within budget (wall-clock never exceeds 30s)', async () => {
      queueMatchStates(
        makeState({ grid: [[plainTile()]], activeTeam: 2, turn: 1 }),
        makeState({ grid: [[plainTile()]], activeTeam: 1, turn: 2 })
      );
      vi.mocked(consumeCpuStatus)
        .mockResolvedValueOnce({
          turnPhase: 'TurnPhasePlanning',
          planGameEvents: [],
          resolveTurnGameEvents: [],
        })
        .mockResolvedValue(readyStatus());

      await bootScene();
      await flush();
      drainDelayedCalls();
      await flush();

      expect(consumeCpuStatus).toHaveBeenCalledTimes(2);
      expect(startTurn).toHaveBeenCalledTimes(2);
    });

    it('treats TurnPhaseIdle as a bug signal: stops polling, renders none of its events, and runs the same recovery as AC 12 (AC 13)', async () => {
      queueMatchStates(
        makeState({ grid: [[plainTile()]], activeTeam: 2, turn: 1 }),
        makeState({ grid: [[plainTile()]], activeTeam: 1, turn: 2 })
      );
      vi.mocked(consumeCpuStatus).mockResolvedValue({
        turnPhase: 'TurnPhaseIdle',
        planGameEvents: [
          { type: 'unitMoved', unitId: 1, from: { x: 0, y: 0 }, to: { x: 0, y: 0 } },
        ],
        resolveTurnGameEvents: [],
      });

      await bootScene();
      fireCameraFadeOutComplete();
      await flush();

      expect(consumeCpuStatus).toHaveBeenCalledTimes(1);
      expect(playResolveTurnEvents).not.toHaveBeenCalled();
      expect(startTurn).toHaveBeenCalledTimes(2);
    });

    it('resolves a backoff wait via its own shutdown listener, not just the scheduled timer', async () => {
      queueMatchStates(
        makeState({ grid: [[plainTile()]], activeTeam: 2, turn: 1 }),
        makeState({ grid: [[plainTile()]], activeTeam: 1, turn: 2 })
      );
      vi.mocked(consumeCpuStatus)
        .mockResolvedValueOnce({
          turnPhase: 'TurnPhasePlanning',
          planGameEvents: [],
          resolveTurnGameEvents: [],
        })
        .mockResolvedValue(readyStatus());

      await bootScene();
      await flush();
      expect(consumeCpuStatus).toHaveBeenCalledTimes(1);

      fireLastShutdownListener();
      await flush();

      expect(consumeCpuStatus).toHaveBeenCalledTimes(2);
      expect(startTurn).toHaveBeenCalledTimes(2);
    });

    it('aborts the CPU turn without rendering resolveTurnGameEvents when a planGameEvents entry is malformed', async () => {
      queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 2, turn: 1 }));
      const malformedMove: GameEvent = {
        type: 'unitMoved',
        unitId: 1,
        from: { x: 0, y: 0 },
      };
      vi.mocked(consumeCpuStatus).mockResolvedValue(
        readyStatus({
          planGameEvents: [malformedMove],
          resolveTurnGameEvents: [{ type: 'matchEnded', winnerTeamId: 2 }],
        })
      );

      await bootScene();
      await flush();

      expect(mockScene.add.text).toHaveBeenCalledWith(
        expect.any(Number),
        expect.any(Number),
        expect.stringContaining('Invalid unitMoved event'),
        expect.anything()
      );
      expect(playResolveTurnEvents).not.toHaveBeenCalled();
      expect(victoryCutscenePlay).not.toHaveBeenCalled();
    });
  });

  describe('victory cutscene and rematch', () => {
    async function resolveWithMatchEnded(winnerTeamId?: number): Promise<void> {
      await setUpEmptyBoardAndClickResolve();
      const event: GameEvent =
        winnerTeamId === undefined ? { type: 'matchEnded' } : { type: 'matchEnded', winnerTeamId };
      vi.mocked(resolveTurn).mockResolvedValue({
        planGameEvents: [],
        resolveTurnGameEvents: [event],
      });

      const [yesButtonContainer] = lastContainers(2);
      pointerDownOf(yesButtonContainer!)();
      await flush();
    }

    it.each([1, 2, -1])(
      'renders VictoryCutscene(%i) instead of looping into another beginTurn when resolveTurn reports matchEnded',
      async winnerTeamId => {
        await resolveWithMatchEnded(winnerTeamId);

        expect(victoryCutscenePlay).toHaveBeenCalledWith(
          winnerTeamId,
          expect.any(String),
          expect.any(Object)
        );
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
      vi.mocked(resolveTurn).mockResolvedValue({
        planGameEvents: [],
        resolveTurnGameEvents: [{ type: 'matchEnded', winnerTeamId: 1 }],
      });

      clickSummaryPanelButton('resolve');
      const [yesButtonContainer] = lastContainers(2);
      pointerDownOf(yesButtonContainer!)();
      await flush();

      const graphicsBefore = mockScene.add.graphics.mock.calls.length;
      pointerDownOf(occupantSprite(0))();
      expect(mockScene.add.graphics.mock.calls.length).toBe(graphicsBefore);
    });

    it('rematch: fades out, then restarts the scene with the same roomId/playerTokens and isRematch=true', async () => {
      await resolveWithMatchEnded(1);

      const onRematch = victoryCutscenePlay.mock.calls[0]![2].onRematch;
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

    it('does not call rematch() on a normal (non-rematch) create(), but still fades in (fadeTransition)', async () => {
      queueMatchStates(makeState({ grid: [[plainTile()]] }));

      await bootScene();

      expect(rematch).not.toHaveBeenCalled();
      expect(mockScene.cameras.main.fadeIn).toHaveBeenCalledWith(200);
    });

    it('a Prologue Match (vsCpu) shows Return to Title and fadeTransitions to TitleScene, deleting the match concurrently (p4-spec009 AC 14)', async () => {
      vi.mocked(getMatchConfig).mockResolvedValue(makeCfg({ vsCpu: true }));
      let resolveDelete: () => void = () => undefined;
      vi.mocked(deleteMatch).mockReturnValue(
        new Promise<void>(resolve => {
          resolveDelete = resolve;
        })
      );
      await resolveWithMatchEnded(1);

      expect(victoryCutscenePlay).toHaveBeenCalledWith(1, 'Return to Title', expect.any(Object));
      const onReturnClicked = victoryCutscenePlay.mock.calls[0]![2].onReturnClicked;
      onReturnClicked();

      expect(mockScene.cameras.main.fadeOut).toHaveBeenCalledWith(200, 0, 0, 0);
      expect(deleteMatch).toHaveBeenCalledOnce();

      fireCameraFadeOutComplete();
      resolveDelete();
      await flush();

      expect(mockScene.scene.start).toHaveBeenCalledWith('TitleScene');
    });

    it('return to settings: fades out and calls deleteMatch() concurrently, then starts MatchSettingsScene once both settle', async () => {
      let resolveDelete: () => void = () => undefined;
      vi.mocked(deleteMatch).mockReturnValue(
        new Promise<void>(resolve => {
          resolveDelete = resolve;
        })
      );
      await resolveWithMatchEnded(1);

      const onReturnClicked = victoryCutscenePlay.mock.calls[0]![2].onReturnClicked;
      onReturnClicked();

      expect(mockScene.cameras.main.fadeOut).toHaveBeenCalledWith(200, 0, 0, 0);
      expect(deleteMatch).toHaveBeenCalledOnce();
      expect(mockScene.scene.start).not.toHaveBeenCalled();

      fireCameraFadeOutComplete();
      await flush();
      expect(mockScene.scene.start).not.toHaveBeenCalled();

      resolveDelete();
      await flush();

      expect(mockScene.scene.start).toHaveBeenCalledWith('MatchSettingsScene', {
        gameCfg: makeCfg(),
      });
    });

    it('logs the failure reason via console.error and still starts MatchSettingsScene when deleteMatch() rejects', async () => {
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => undefined);
      vi.mocked(deleteMatch).mockRejectedValue(new Error('delete failed'));
      await resolveWithMatchEnded(1);

      const onReturnClicked = victoryCutscenePlay.mock.calls[0]![2].onReturnClicked;
      onReturnClicked();
      fireCameraFadeOutComplete();
      await flush();

      expect(consoleSpy).toHaveBeenCalledWith('Failed to delete match:', 'delete failed');
      expect(mockScene.scene.start).toHaveBeenCalledWith('MatchSettingsScene', {
        gameCfg: makeCfg(),
      });
      consoleSpy.mockRestore();
    });

    it('ignores a second Rematch click while the first fade-out/restart is already in progress', async () => {
      await resolveWithMatchEnded(1);

      const onRematch = victoryCutscenePlay.mock.calls[0]![2].onRematch;
      onRematch();
      onRematch();

      expect(mockScene.cameras.main.fadeOut).toHaveBeenCalledTimes(1);
      fireCameraFadeOutComplete();
      expect(mockScene.scene.restart).toHaveBeenCalledTimes(1);
    });

    it('ignores a second Return-to-Settings click while the first fade-out/delete is already in progress', async () => {
      await resolveWithMatchEnded(1);

      const onReturnClicked = victoryCutscenePlay.mock.calls[0]![2].onReturnClicked;
      onReturnClicked();
      onReturnClicked();

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

      fireShutdown();
      queueMatchStates(makeState({ grid: [[plainTile()]] }));
      scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'], isRematch: true });
      await flush();

      const centerOnCallsBeforeStaleResolve = mockScene.cameras.main.centerOn.mock.calls.length;
      resolveStaleState(makeState({ grid: [[plainTile(), plainTile()]] }));
      await flush();

      expect(mockScene.cameras.main.centerOn.mock.calls.length).toBe(
        centerOnCallsBeforeStaleResolve
      );
    });

    it('does not flash the previous match Stage background while the rematch gameCfg is still loading', async () => {
      vi.mocked(getMatchConfig).mockResolvedValueOnce(makeCfg({ stagePreset: 'Plain' }));
      queueMatchStates(makeState({ grid: [[plainTile()]] }));

      const scene = new MatchScene();
      scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
      await flush();

      expect(mockScene.add.image).toHaveBeenLastCalledWith(
        expect.any(Number),
        expect.any(Number),
        'stage_plain',
        expect.any(String)
      );
      const imageCallsBeforeRematch = mockScene.add.image.mock.calls.length;

      fireShutdown();
      let resolveFreshCfg: (cfg: GameCfg) => void = () => undefined;
      vi.mocked(getMatchConfig).mockReturnValueOnce(
        new Promise<GameCfg>(resolve => {
          resolveFreshCfg = resolve;
        })
      );
      queueMatchStates(makeState({ grid: [[plainTile()]] }));
      scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'], isRematch: true });
      await flush();

      expect(mockScene.add.image.mock.calls.length).toBe(imageCallsBeforeRematch);

      resolveFreshCfg(makeCfg({ stagePreset: 'Divided' }));
      await flush();

      expect(mockScene.add.image).toHaveBeenLastCalledWith(
        expect.any(Number),
        expect.any(Number),
        'stage_divided',
        expect.any(String)
      );
    });

    it('does not repaint using the previous match state while a rematch entry gameState fetch is still loading', async () => {
      vi.mocked(getMatchConfig).mockResolvedValueOnce(makeCfg({ stagePreset: 'Plain' }));
      queueMatchStates(makeState({ grid: [[plainTile()]] }));

      const scene = new MatchScene();
      scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
      await flush();

      const imageCallsBeforeRematch = mockScene.add.image.mock.calls.length;
      expect(imageCallsBeforeRematch).toBeGreaterThan(0);

      fireShutdown();
      let resolveFreshState: (state: GameState) => void = () => undefined;
      vi.mocked(getMatchState).mockReturnValueOnce(
        new Promise<GameState>(resolve => {
          resolveFreshState = resolve;
        })
      );
      vi.mocked(getMatchConfig).mockResolvedValueOnce(makeCfg({ stagePreset: 'Divided' }));
      scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'], isRematch: true });
      await flush();

      expect(mockScene.add.image.mock.calls.length).toBe(imageCallsBeforeRematch);

      resolveFreshState(makeState({ grid: [[plainTile()]] }));
      await flush();

      expect(mockScene.add.image).toHaveBeenLastCalledWith(
        expect.any(Number),
        expect.any(Number),
        'stage_divided',
        expect.any(String)
      );
    });

    it('does not display the previous match maxTurns while a rematch entry gameCfg is still loading', async () => {
      vi.mocked(getMatchConfig).mockResolvedValueOnce(makeCfg({ maxTurns: 12 }));
      queueMatchStates(makeState({ grid: [[plainTile()]] }));

      const scene = new MatchScene();
      scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
      await flush();

      fireShutdown();
      let resolveFreshCfg: (cfg: GameCfg) => void = () => undefined;
      vi.mocked(getMatchConfig).mockReturnValueOnce(
        new Promise<GameCfg>(resolve => {
          resolveFreshCfg = resolve;
        })
      );
      queueMatchStates(makeState({ grid: [[plainTile()]] }));
      const textCallsBeforeRematch = mockScene.add.text.mock.calls.length;
      scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'], isRematch: true });
      await flush();

      const textCallsDuringRematch = mockScene.add.text.mock.calls.slice(textCallsBeforeRematch);
      expect(textCallsDuringRematch).not.toContainEqual([
        expect.any(Number),
        expect.any(Number),
        '12',
        expect.objectContaining({}),
      ]);

      resolveFreshCfg(makeCfg({ maxTurns: 20 }));
      await flush();

      expect(mockScene.add.text).toHaveBeenCalledWith(
        expect.any(Number),
        expect.any(Number),
        '20',
        expect.objectContaining({})
      );
    });
  });

  describe('Reset flow (ResetTurnButton)', () => {
    it('fades the camera out then in (like Rematch), calls resetTurn() then getMatchState(), and rebuilds only the occupant layer', async () => {
      const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
      queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1, units: [unit] }));
      vi.mocked(resetTurn).mockResolvedValue(undefined);

      await bootScene({ playerTokens: ['team1-token', 'team2-token'] });
      mockScene.cameras.main.fadeIn.mockClear();

      const unitGraphics = occupantSprite(0);
      const grid = stageBackgroundImage();

      clickSummaryPanelButton('reset');
      const [yesButtonContainer] = lastContainers(2);
      pointerDownOf(yesButtonContainer!)();

      expect(mockScene.cameras.main.fadeOut).toHaveBeenCalledWith(200, 0, 0, 0);
      expect(mockScene.cameras.main.fadeIn).not.toHaveBeenCalled();

      await flush();

      expect(resetTurn).toHaveBeenCalledOnce();
      expect(getMatchState).toHaveBeenCalledTimes(3);
      expect(grid.destroy).not.toHaveBeenCalled();
      expect(unitGraphics.destroy).toHaveBeenCalled();
      expect(mockScene.cameras.main.fadeIn).toHaveBeenCalledWith(200);
    });

    it('re-enables interactions once resetTurn()+getMatchState() settle, independent of the undim tween finishing', async () => {
      const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
      queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1, units: [unit] }));
      vi.mocked(resetTurn).mockResolvedValue(undefined);

      await bootScene();

      clickSummaryPanelButton('reset');
      const [yesButtonContainer] = lastContainers(2);
      pointerDownOf(yesButtonContainer!)();
      await flush();

      const freshUnitSprite = mockScene.add.sprite.mock.results.at(-1)!.value as ReturnType<
        typeof mockScene.add.sprite
      >;
      const containersBefore = mockScene.add.container.mock.calls.length;
      pointerDownOf(freshUnitSprite)();

      expect(mockScene.add.container.mock.calls.length).toBeGreaterThan(containersBefore);
    });

    it('shows an error and still re-enables interactions when resetTurn() rejects', async () => {
      const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
      queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1, units: [unit] }));
      vi.mocked(resetTurn).mockRejectedValue(new Error('reset failed'));

      await bootScene();

      clickSummaryPanelButton('reset');
      const [yesButtonContainer] = lastContainers(2);
      pointerDownOf(yesButtonContainer!)();
      await flush();

      expect(mockScene.add.text).toHaveBeenCalledWith(
        expect.any(Number),
        expect.any(Number),
        'reset failed',
        expect.objectContaining({})
      );
      expect(getMatchState).toHaveBeenCalledTimes(2);

      const containersBefore = mockScene.add.container.mock.calls.length;
      pointerDownOf(occupantSprite(0))();
      expect(mockScene.add.container.mock.calls.length).toBeGreaterThan(containersBefore);
    });

    it('still schedules the dim overlay to fade out and destroy itself if the scene is shut down while resetTurn() is in flight', async () => {
      const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
      queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1, units: [unit] }));
      let resolveReset: () => void = () => undefined;
      vi.mocked(resetTurn).mockReturnValue(
        new Promise<void>(resolve => {
          resolveReset = resolve;
        })
      );

      await bootScene();

      clickSummaryPanelButton('reset');
      const [yesButtonContainer] = lastContainers(2);
      pointerDownOf(yesButtonContainer!)();

      fireShutdown();
      resolveReset();
      await flush();

      expect(mockScene.cameras.main.fadeIn).toHaveBeenCalledWith(200);
    });

    it('clears the TurnCommandPanel action stack when ResetTurnButton opens the confirm dialog', async () => {
      await expectActionStackClearedBy('reset');
    });
  });

  describe('Surrender flow (SurrenderButton)', () => {
    it("calls surrender() with the currently active team's id and renders VictoryCutscene when matchEnded is returned", async () => {
      queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 2 }));
      vi.mocked(surrender).mockResolvedValue([{ type: 'matchEnded', winnerTeamId: 1 }]);

      await bootScene();
      clickSummaryPanelButton('surrender');
      const [yesButtonContainer] = lastContainers(2);
      pointerDownOf(yesButtonContainer!)();
      await flush();

      expect(surrender).toHaveBeenCalledWith({ teamId: 2 });
      expect(victoryCutscenePlay).toHaveBeenCalledWith(1, expect.any(String), expect.any(Object));
    });

    it('shows an error and does not render VictoryCutscene when the surrender response has no matchEnded event', async () => {
      queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1 }));
      vi.mocked(surrender).mockResolvedValue([]);

      await bootScene();
      clickSummaryPanelButton('surrender');
      const [yesButtonContainer] = lastContainers(2);
      pointerDownOf(yesButtonContainer!)();
      await flush();

      expect(victoryCutscenePlay).not.toHaveBeenCalled();
      expect(mockScene.add.text).toHaveBeenCalledWith(
        expect.any(Number),
        expect.any(Number),
        'Invalid response from surrender',
        expect.objectContaining({})
      );
    });

    it('shows an error and does not render VictoryCutscene when surrender() rejects', async () => {
      queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1 }));
      vi.mocked(surrender).mockRejectedValue(new Error('network down'));

      await bootScene();
      clickSummaryPanelButton('surrender');
      const [yesButtonContainer] = lastContainers(2);
      pointerDownOf(yesButtonContainer!)();
      await flush();

      expect(mockScene.add.text).toHaveBeenCalledWith(
        expect.any(Number),
        expect.any(Number),
        'network down',
        expect.objectContaining({})
      );
      expect(victoryCutscenePlay).not.toHaveBeenCalled();
    });

    it('clears the TurnCommandPanel action stack when SurrenderButton opens the confirm dialog', async () => {
      await expectActionStackClearedBy('surrender');
    });
  });

  describe('lifecycle-button confirm re-entrancy (MatchSummaryPanel)', () => {
    it('ignores a second lifecycle button click while another lifecycle confirm is already pending', async () => {
      queueMatchStates(makeState({ grid: [[plainTile()]], activeTeam: 1 }));
      await bootScene();

      pointerDownOf(matchSummaryButtonGraphics())();
      const [, , resetGraphics, surrenderGraphics] = lastGraphics(5);
      pointerDownOf(resetGraphics!)();

      const textCallsBefore = mockScene.add.text.mock.calls.length;
      pointerDownOf(surrenderGraphics!)();

      expect(mockScene.add.text.mock.calls.length).toBe(textCallsBefore);
    });

    it('does not block a lifecycle button click caused only by a stale TurnCommandPanel confirm', async () => {
      const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
      const initialGrid: Tile[][] = [
        [{ type: 'TerrainPlain', occupantType: 'OccupantUnit', occupantId: 7 }, plainTile()],
      ];
      queueMatchStates(makeState({ grid: initialGrid, activeTeam: 1, units: [unit] }));
      vi.mocked(getAllowedTiles).mockResolvedValue([{ x: 1, y: 0 }]);

      await bootScene();

      pointerDownOf(occupantSprite(0))();
      const [moveButtonContainer] = lastContainers(3);
      pointerDownOf(moveButtonContainer!)();
      await flush();
      const [overlayTile] = lastGraphics(1);
      pointerDownOf(overlayTile!)();

      clickSummaryPanelButton('reset');

      expect(mockScene.add.text).toHaveBeenCalledWith(
        expect.any(Number),
        expect.any(Number),
        CONFIRM_TEXT_RESET,
        expect.objectContaining({})
      );
    });
  });
});
