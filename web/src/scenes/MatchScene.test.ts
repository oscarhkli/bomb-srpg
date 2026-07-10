import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import {
  initRoom,
  initToken,
  getMatchState,
  getMatchConfig,
  getAllowedTiles,
  submitTurnCommand,
  resolveTurn,
} from '../engine/api';
import { playResolveTurnEvents } from '../rendering/resolveTurnPlayer';
import MatchScene from './MatchScene';
import type {
  Coordinate,
  GameCfg,
  GameEvent,
  GameState,
  Tile,
  TerrainType,
  Unit,
  Bomb,
} from '../types/api';

vi.mock('../engine/api');
vi.mock('../rendering/resolveTurnPlayer');

function makeCfg(overrides: Partial<GameCfg> = {}): GameCfg {
  return {
    stagePreset: 'default',
    p1Teams: [],
    p2Teams: [],
    maxTurns: 30,
    allowResetTurn: true,
    suddenDeath: false,
    ...overrides,
  };
}

function makeState(grid: Tile[][], overrides: Partial<GameState> = {}): GameState {
  return {
    turn: 1,
    activeTeam: 0,
    grid,
    units: [],
    bombs: [],
    softBlocks: [],
    turnCommands: [],
    ...overrides,
  };
}

function plainTile(): Tile {
  return tileOf('TerrainPlain');
}

function tileOf(type: TerrainType): Tile {
  return { type, occupantType: 'OccupantNone', occupantId: 0 };
}

function makeUnit(overrides: Partial<Unit> = {}): Unit {
  return {
    id: 1,
    type: 'Fighter',
    position: { x: 0, y: 0 },
    speed: 2,
    bombMaxRange: 2,
    bombPower: 1,
    maxBombCount: 3,
    bombUsed: 0,
    team: 1,
    hp: 1,
    skills: [],
    hasMoved: false,
    hasUsedSkill: false,
    ...overrides,
  };
}

function makeBomb(overrides: Partial<Bomb> = {}): Bomb {
  return { id: 1, ownerId: 1, position: { x: 0, y: 0 }, range: 2, countdown: 3, ...overrides };
}

// Occupant Graphics instances are created after the grid's, in array order
// (units, then softBlocks, then bombs) — index 0 here means "first occupant", i.e.
// mock.results[1] overall (results[0] is the grid).
function occupantGraphics(index: number): ReturnType<typeof mockScene.add.graphics> {
  return mockScene.add.graphics.mock.results[index + 1]!.value as ReturnType<
    typeof mockScene.add.graphics
  >;
}

// mockScene.add.text's mock.calls infer a `[]` call-signature (from its no-arg factory
// implementation), so raw index access needs a cast — this helper centralizes it.
function textCalls(): [number, number, string, ...unknown[]][] {
  return mockScene.add.text.mock.calls as unknown as [number, number, string, ...unknown[]][];
}

function errorTextByMessage(message: string): ReturnType<typeof mockScene.add.text> {
  const index = textCalls().findIndex(c => c[2] === message);
  return mockScene.add.text.mock.results[index]!.value as ReturnType<typeof mockScene.add.text>;
}

function pointerDownOf(g: ReturnType<typeof mockScene.add.graphics>): () => void {
  return g.on.mock.calls.find(call => call[0] === 'pointerdown')?.[1] as () => void;
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
  await Promise.resolve();
  await Promise.resolve();
  const [overlayTileGraphics] = mockScene.add.graphics.mock.results
    .slice(-1)
    .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
  pointerDownOf(overlayTileGraphics!)();
  const [, yesButtonGraphics] = mockScene.add.graphics.mock.results
    .slice(-3)
    .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
  pointerDownOf(yesButtonGraphics!)();
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(getMatchConfig).mockResolvedValue(makeCfg());
  vi.mocked(playResolveTurnEvents).mockReturnValue({ ok: true, done: Promise.resolve() });
});

describe('MatchScene', () => {
  it('logs roomId and playerTokens on create', async () => {
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => undefined);
    vi.mocked(getMatchState).mockResolvedValue(makeState([[plainTile()]]));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-xyz', playerTokens: ['tok1', 'tok2'] });
    await Promise.resolve();

    expect(consoleSpy).toHaveBeenCalledWith('roomId:', 'room-xyz', 'playerTokens:', [
      'tok1',
      'tok2',
    ]);
    consoleSpy.mockRestore();
  });

  it('calls initRoom with data.roomId then getMatchState on create', async () => {
    vi.mocked(getMatchState).mockResolvedValue(makeState([[plainTile()]]));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    expect(initRoom).toHaveBeenCalledWith('room-abc');
    expect(getMatchState).toHaveBeenCalledOnce();
  });

  it('centers camera on a 3x2 grid', async () => {
    const row = (): Tile[] => [plainTile(), plainTile(), plainTile()];
    vi.mocked(getMatchState).mockResolvedValue(makeState([row(), row()]));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    expect(mockScene.cameras.main.centerOn).toHaveBeenCalledWith(72, 48);
  });

  it('centers camera on a differently-sized 5x7 grid', async () => {
    const row = (): Tile[] => Array.from({ length: 5 }, plainTile);
    vi.mocked(getMatchState).mockResolvedValue(makeState(Array.from({ length: 7 }, row)));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    expect(mockScene.cameras.main.centerOn).toHaveBeenCalledWith(120, 168);
  });

  it('opens TurnCommandPanel and calls initToken when clicking a unit on the active team', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    vi.mocked(getMatchState).mockResolvedValue(
      makeState([[plainTile()]], { activeTeam: 1, units: [unit] })
    );

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['team1-token', 'team2-token'] });
    await Promise.resolve();

    const graphicsBefore = mockScene.add.graphics.mock.calls.length;
    const g = occupantGraphics(0);
    const onPointerDown = g.on.mock.calls.find(
      call => call[0] === 'pointerdown'
    )?.[1] as () => void;

    onPointerDown();

    expect(initToken).toHaveBeenCalledWith('team1-token');
    // TurnCommandPanel.openFor draws 3 new button Graphics (Move/Bomb/Back)
    expect(mockScene.add.graphics.mock.calls.length).toBe(graphicsBefore + 3);
  });

  it('does not open TurnCommandPanel or call initToken when clicking a unit off the active team', async () => {
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => undefined);
    const unit = makeUnit({ id: 7, team: 2, position: { x: 0, y: 0 } });
    vi.mocked(getMatchState).mockResolvedValue(
      makeState([[plainTile()]], { activeTeam: 1, units: [unit] })
    );

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['team1-token', 'team2-token'] });
    await Promise.resolve();

    const graphicsBefore = mockScene.add.graphics.mock.calls.length;
    const g = occupantGraphics(0);
    const onPointerDown = g.on.mock.calls.find(
      call => call[0] === 'pointerdown'
    )?.[1] as () => void;

    onPointerDown();

    expect(initToken).not.toHaveBeenCalled();
    expect(consoleSpy).toHaveBeenCalledWith('Unit 7 is clicked', unit);
    expect(mockScene.add.graphics.mock.calls.length).toBe(graphicsBefore);
    consoleSpy.mockRestore();
  });

  it('caches getAllowedTiles per (unitId, turnCmdType): repeat moveButton clicks skip the network call', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    vi.mocked(getMatchState).mockResolvedValue(
      makeState([[plainTile()]], { activeTeam: 1, units: [unit] })
    );
    vi.mocked(getAllowedTiles).mockResolvedValue([{ x: 1, y: 0 }]);

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const g = occupantGraphics(0);
    const onUnitPointerDown = g.on.mock.calls.find(
      call => call[0] === 'pointerdown'
    )?.[1] as () => void;

    // Open the panel, click Move, close it, reopen, click Move again.
    onUnitPointerDown();
    const [moveButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    const clickMove = () =>
      (
        moveButtonGraphics!.on.mock.calls.find(call => call[0] === 'pointerdown')?.[1] as () => void
      )();
    clickMove();
    await Promise.resolve();
    await Promise.resolve();

    onUnitPointerDown(); // closes and reopens the panel for the same unit
    onUnitPointerDown();
    clickMove();
    await Promise.resolve();
    await Promise.resolve();

    expect(getAllowedTiles).toHaveBeenCalledTimes(1);
    expect(getAllowedTiles).toHaveBeenCalledWith({ unitId: 7, turnCmdType: 'move' });
  });

  it('shows an error when getAllowedTiles rejects', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    vi.mocked(getMatchState).mockResolvedValue(
      makeState([[plainTile()]], { activeTeam: 1, units: [unit] })
    );
    vi.mocked(getAllowedTiles).mockRejectedValue(new Error('tiles unavailable'));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const g = occupantGraphics(0);
    const onUnitPointerDown = g.on.mock.calls.find(
      call => call[0] === 'pointerdown'
    )?.[1] as () => void;
    onUnitPointerDown();

    const [moveButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    (
      moveButtonGraphics!.on.mock.calls.find(call => call[0] === 'pointerdown')?.[1] as () => void
    )();
    await Promise.resolve();
    await Promise.resolve();

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'tiles unavailable',
      expect.objectContaining({})
    );
  });

  it('does not render a Unit with hp 0', async () => {
    const dead = makeUnit({ hp: 0 });
    vi.mocked(getMatchState).mockResolvedValue(makeState([[plainTile()]], { units: [dead] }));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    // Baseline is grid(1) + ResolveTurnButton(1) + TurnPanel header(1) — no occupant
    // Graphics for the dead unit on top of that.
    expect(mockScene.add.graphics).toHaveBeenCalledTimes(3);
  });

  it('shows an error message in the error panel when getMatchState rejects, without rendering the board', async () => {
    vi.mocked(getMatchState).mockRejectedValue(new Error('network fail'));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();
    await Promise.resolve();

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Failed to load match state',
      expect.objectContaining({})
    );
    // Only the error panel's own background Graphics exists — no board (grid) was rendered.
    expect(mockScene.add.graphics).toHaveBeenCalledTimes(1);
    expect(mockScene.cameras.main.centerOn).not.toHaveBeenCalled();
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

    vi.mocked(getMatchState)
      .mockResolvedValueOnce(makeState(initialGrid, { activeTeam: 1, units: [unit] }))
      .mockResolvedValueOnce(makeState(refreshedGrid, { activeTeam: 2, units: [movedUnit] }));
    vi.mocked(getAllowedTiles).mockResolvedValue([target]);
    const events: GameEvent[] = [
      { type: 'unitMoved', unitId: 7, from: { x: 0, y: 0 }, to: target },
    ];
    vi.mocked(submitTurnCommand).mockResolvedValue(events);

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const unitGraphics = occupantGraphics(0);
    await submitViaUI(unitGraphics, 0);
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();

    expect(submitTurnCommand).toHaveBeenCalledWith({ type: 'move', unitId: 7, target });
    expect(mockScene.tweens.add).toHaveBeenCalledWith(
      expect.objectContaining({ targets: unitGraphics, x: 48, y: 0, ease: 'Linear' })
    );
    expect(getMatchState).toHaveBeenCalledTimes(2);
    expect(mockScene.add.text).not.toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      expect.stringContaining('out of sync'),
      expect.objectContaining({})
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

    vi.mocked(getMatchState)
      .mockResolvedValueOnce(makeState(initialGrid, { activeTeam: 1, units: [unit] }))
      .mockResolvedValueOnce(
        makeState(refreshedGrid, {
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

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const unitGraphics = occupantGraphics(0);
    await submitViaUI(unitGraphics, 1);
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();

    expect(submitTurnCommand).toHaveBeenCalledWith({ type: 'placeBomb', unitId: 7, target });
    expect(getMatchState).toHaveBeenCalledTimes(2);
    expect(mockScene.add.text).not.toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      expect.stringContaining('out of sync'),
      expect.objectContaining({})
    );
  });

  it('shows an error and still refreshes state when submitTurnCommand rejects', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [[plainTile(), plainTile()]];
    const target: Coordinate = { x: 1, y: 0 };

    vi.mocked(getMatchState)
      .mockResolvedValueOnce(makeState(initialGrid, { activeTeam: 1, units: [unit] }))
      .mockResolvedValueOnce(makeState(initialGrid, { activeTeam: 1, units: [unit] }));
    vi.mocked(getAllowedTiles).mockResolvedValue([target]);
    vi.mocked(submitTurnCommand).mockRejectedValue(new Error('stale token'));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const unitGraphics = occupantGraphics(0);
    await submitViaUI(unitGraphics, 0);
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'stale token',
      expect.objectContaining({})
    );
    expect(getMatchState).toHaveBeenCalledTimes(2);
  });

  it('shows an error and stops processing when the unitMoved event is malformed (out-of-bounds target)', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [[plainTile(), plainTile()]];
    const target: Coordinate = { x: 1, y: 0 };

    vi.mocked(getMatchState)
      .mockResolvedValueOnce(makeState(initialGrid, { activeTeam: 1, units: [unit] }))
      .mockResolvedValueOnce(makeState(initialGrid, { activeTeam: 1, units: [unit] }));
    vi.mocked(getAllowedTiles).mockResolvedValue([target]);
    // `to` is out-of-bounds for a 1x2 grid — structurally malformed, regardless of legality.
    const events: GameEvent[] = [
      { type: 'unitMoved', unitId: 7, from: { x: 0, y: 0 }, to: { x: 99, y: 99 } },
    ];
    vi.mocked(submitTurnCommand).mockResolvedValue(events);

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const unitGraphics = occupantGraphics(0);
    await submitViaUI(unitGraphics, 0);
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Invalid unitMoved event received from server',
      expect.objectContaining({})
    );
    expect(mockScene.tweens.add).not.toHaveBeenCalled();
    expect(getMatchState).toHaveBeenCalledTimes(2); // refresh still runs
  });

  it('accepts a unitMoved event even when the client grid did not show the unit at `from` (no client-side legality re-derivation)', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    // Deliberately inconsistent with `unit`'s own position: the grid tile at `from` shows
    // no occupant, even though the units array (the source boardRenderer used) has unit 7
    // there. Previously this tripped the "mover was at from" tile-based re-derivation.
    const initialGrid: Tile[][] = [[plainTile(), plainTile()]];
    const target: Coordinate = { x: 1, y: 0 };
    const movedUnit = makeUnit({ id: 7, team: 1, position: target });

    vi.mocked(getMatchState)
      .mockResolvedValueOnce(makeState(initialGrid, { activeTeam: 1, units: [unit] }))
      .mockResolvedValueOnce(makeState(initialGrid, { activeTeam: 2, units: [movedUnit] }));
    vi.mocked(getAllowedTiles).mockResolvedValue([target]);
    const events: GameEvent[] = [
      { type: 'unitMoved', unitId: 7, from: { x: 0, y: 0 }, to: target },
    ];
    vi.mocked(submitTurnCommand).mockResolvedValue(events);

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const unitGraphics = occupantGraphics(0);
    await submitViaUI(unitGraphics, 0);
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();

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

  it('does not flag a mismatch when the server lands the unit on a different tile than requested (e.g. a future push/swap move)', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [[plainTile(), plainTile(), plainTile()]];
    const requestedTarget: Coordinate = { x: 1, y: 0 };
    const actualTo: Coordinate = { x: 2, y: 0 };
    const pushedUnit = makeUnit({ id: 7, team: 1, position: actualTo });

    vi.mocked(getMatchState)
      .mockResolvedValueOnce(makeState(initialGrid, { activeTeam: 1, units: [unit] }))
      .mockResolvedValueOnce(makeState(initialGrid, { activeTeam: 2, units: [pushedUnit] }));
    vi.mocked(getAllowedTiles).mockResolvedValue([requestedTarget]);
    // Server reports the unit actually landed one tile further than the client requested.
    const events: GameEvent[] = [
      { type: 'unitMoved', unitId: 7, from: { x: 0, y: 0 }, to: actualTo },
    ];
    vi.mocked(submitTurnCommand).mockResolvedValue(events);

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const unitGraphics = occupantGraphics(0);
    await submitViaUI(unitGraphics, 0);
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();

    expect(mockScene.add.text).not.toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      expect.stringContaining('out of sync'),
      expect.objectContaining({})
    );
    expect(mockScene.add.text).not.toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Failed to refresh match state',
      expect.objectContaining({})
    );
  });

  it('flags a mismatch when the post-move refreshed state has the moved unit at the wrong position', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [
      [{ type: 'TerrainPlain', occupantType: 'OccupantUnit', occupantId: 7 }, plainTile()],
    ];
    const target: Coordinate = { x: 1, y: 0 };
    // Refresh returns the actor unit still at its pre-move position (0,0), not the event's `to`.
    const staleUnit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });

    vi.mocked(getMatchState)
      .mockResolvedValueOnce(makeState(initialGrid, { activeTeam: 1, units: [unit] }))
      .mockResolvedValueOnce(makeState(initialGrid, { activeTeam: 1, units: [staleUnit] }));
    vi.mocked(getAllowedTiles).mockResolvedValue([target]);
    const events: GameEvent[] = [
      { type: 'unitMoved', unitId: 7, from: { x: 0, y: 0 }, to: target },
    ];
    vi.mocked(submitTurnCommand).mockResolvedValue(events);

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const unitGraphics = occupantGraphics(0);
    await submitViaUI(unitGraphics, 0);
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Match state is out of sync with the server',
      expect.objectContaining({})
    );
  });

  it('ignores a unit click while a confirm dialog is already open', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [
      [{ type: 'TerrainPlain', occupantType: 'OccupantUnit', occupantId: 7 }, plainTile()],
    ];
    const target: Coordinate = { x: 1, y: 0 };

    vi.mocked(getMatchState).mockResolvedValueOnce(
      makeState(initialGrid, { activeTeam: 1, units: [unit] })
    );
    vi.mocked(getAllowedTiles).mockResolvedValue([target]);

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const unitGraphics = occupantGraphics(0);
    pointerDownOf(unitGraphics)();
    const [moveButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(moveButtonGraphics!)();
    await Promise.resolve();
    await Promise.resolve();
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

    vi.mocked(getMatchState).mockResolvedValueOnce(
      makeState(initialGrid, { activeTeam: 1, units: [unit] })
    );
    vi.mocked(getAllowedTiles).mockResolvedValue([target]);
    vi.mocked(submitTurnCommand).mockReturnValue(new Promise<GameEvent[]>(() => undefined));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const unitGraphics = occupantGraphics(0);
    pointerDownOf(unitGraphics)();
    const [moveButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(moveButtonGraphics!)();
    await Promise.resolve();
    await Promise.resolve();
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

    vi.mocked(getMatchState)
      .mockResolvedValueOnce(makeState(initialGrid, { activeTeam: 1, units: [unit] }))
      .mockResolvedValueOnce(
        makeState(gridAfterFirstMove, {
          activeTeam: 1,
          units: [makeUnit({ id: 7, team: 1, position: firstTarget })],
        })
      )
      .mockResolvedValueOnce(
        makeState(gridAfterSecondMove, {
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

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const unitGraphics = occupantGraphics(0);
    await submitViaUI(unitGraphics, 0);
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();

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
    await Promise.resolve();
    await Promise.resolve();
    const [overlayTileGraphics] = mockScene.add.graphics.mock.results
      .slice(-1)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(overlayTileGraphics!)();
    const [, yesButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(yesButtonGraphics!)();
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();

    expect(mockScene.tweens.add).toHaveBeenLastCalledWith(
      expect.objectContaining({ targets: unitGraphics, x: 96, y: 0 })
    );
  });

  it('fetches gameCfg alongside match state and renders TurnPanel with turn/maxTurns/activeTeam', async () => {
    vi.mocked(getMatchState).mockResolvedValue(
      makeState([[plainTile()]], { turn: 4, activeTeam: 2 })
    );
    vi.mocked(getMatchConfig).mockResolvedValue(makeCfg({ maxTurns: 12 }));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();
    await Promise.resolve();

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

  // With no units/softBlocks/bombs, Graphics creation order is: grid(0), ResolveTurnButton(1),
  // TurnPanel header(2) — see the "does not render a Unit with hp 0" baseline above.
  function resolveButtonGraphics(): ReturnType<typeof mockScene.add.graphics> {
    return mockScene.add.graphics.mock.results[1]!.value as ReturnType<
      typeof mockScene.add.graphics
    >;
  }

  async function setUpEmptyBoardAndClickResolve(activeTeam = 1): Promise<void> {
    vi.mocked(getMatchState).mockResolvedValue(makeState([[plainTile()]], { activeTeam }));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['team1-token', 'team2-token'] });
    await Promise.resolve();

    pointerDownOf(resolveButtonGraphics())();
  }

  it('pins ResolveTurnButton to the camera viewport (scrollFactor 0) instead of the grid/world', async () => {
    vi.mocked(getMatchState).mockResolvedValue(makeState([[plainTile()]], { activeTeam: 1 }));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    expect(resolveButtonGraphics().setScrollFactor).toHaveBeenCalledWith(0);
  });

  it('shows an error instead of crashing when ResolveTurnButton is clicked before match config has loaded', async () => {
    vi.mocked(getMatchState).mockResolvedValue(makeState([[plainTile()]], { activeTeam: 1 }));
    vi.mocked(getMatchConfig).mockReturnValue(new Promise<GameCfg>(() => undefined));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    expect(() => pointerDownOf(resolveButtonGraphics())()).not.toThrow();
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
    vi.mocked(getMatchState).mockResolvedValue(
      makeState([[plainTile()]], { activeTeam: 1, units: [unit] })
    );
    vi.mocked(getAllowedTiles).mockResolvedValue([{ x: 0, y: 0 }]);

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    // Graphics creation order on create(): grid(0), unit(1), ResolveTurnButton(2), TurnPanel(3).
    const resolveButton = mockScene.add.graphics.mock.results[2]!.value as ReturnType<
      typeof mockScene.add.graphics
    >;

    // Open the unit's TurnCommandPanel (draws Move/Bomb/Back as the next 3 Graphics).
    pointerDownOf(occupantGraphics(0))();
    const [moveButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);

    pointerDownOf(resolveButton)();

    expect(moveButtonGraphics!.destroy).toHaveBeenCalled();
  });

  it('opens the resolve confirm even when a TurnCommandPanel confirm is already open, so a stale action never blocks it', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [
      [{ type: 'TerrainPlain', occupantType: 'OccupantUnit', occupantId: 7 }, plainTile()],
    ];
    vi.mocked(getMatchState).mockResolvedValue(
      makeState(initialGrid, { activeTeam: 1, units: [unit] })
    );
    vi.mocked(getAllowedTiles).mockResolvedValue([{ x: 1, y: 0 }]);

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    // Graphics order: grid(0), unit(1), ResolveTurnButton(2), TurnPanel(3).
    const resolveButton = mockScene.add.graphics.mock.results[2]!.value as ReturnType<
      typeof mockScene.add.graphics
    >;

    // Drive the Move flow until the shared ConfirmDialog is open (click unit -> Move ->
    // allowed tile). At this point confirmDialog.isOpen is true.
    pointerDownOf(occupantGraphics(0))();
    const [moveButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(moveButtonGraphics!)();
    await Promise.resolve();
    await Promise.resolve();
    const [overlayTile] = mockScene.add.graphics.mock.results
      .slice(-1)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(overlayTile!)();

    // Clicking Resolve must still surface the resolve prompt (previously the open confirm
    // caused an early return and nothing happened).
    pointerDownOf(resolveButton)();

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

  it('on confirmed resolve: inits the active team token, calls resolveTurn, hands events to the player, then refreshes state', async () => {
    await setUpEmptyBoardAndClickResolve(2);
    vi.mocked(getMatchState).mockResolvedValue(makeState([[plainTile()]], { activeTeam: 1 }));
    const events: GameEvent[] = [{ type: 'bombCountdownUpdated', bombId: 1, countdown: 2 }];
    vi.mocked(resolveTurn).mockResolvedValue(events);

    // ConfirmDialog's Yes button is the most-recently-created graphics among the last 3.
    const [, yesButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(yesButtonGraphics!)();
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();

    expect(initToken).toHaveBeenCalledWith('team2-token');
    expect(resolveTurn).toHaveBeenCalledOnce();
    expect(playResolveTurnEvents).toHaveBeenCalledOnce();
    const [calledEvents, deps] = vi.mocked(playResolveTurnEvents).mock.calls[0]!;
    expect(calledEvents).toBe(events);
    expect(deps.gameStateSnapshot.activeTeam).toBe(2);
    expect(getMatchState).toHaveBeenCalledTimes(2); // initial load + final sanity-check refresh
  });

  it('always refreshes state even when resolveTurn() itself rejects', async () => {
    await setUpEmptyBoardAndClickResolve();
    vi.mocked(resolveTurn).mockRejectedValue(new Error('resolve failed'));

    const [, yesButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(yesButtonGraphics!)();
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'resolve failed',
      expect.objectContaining({})
    );
    expect(getMatchState).toHaveBeenCalledTimes(2);
    expect(playResolveTurnEvents).not.toHaveBeenCalled();
  });

  it('flags a mismatch when the post-resolve state has a bomb the client no longer tracks', async () => {
    await setUpEmptyBoardAndClickResolve();
    vi.mocked(resolveTurn).mockResolvedValue([]);
    vi.mocked(getMatchState).mockResolvedValue(
      makeState([[plainTile()]], { bombs: [makeBomb({ id: 1 })] })
    );

    const [, yesButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(yesButtonGraphics!)();
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Match state is out of sync with the server',
      expect.objectContaining({})
    );
  });

  it('clears previous error messages once a new turn command begins, so they do not accumulate across turns', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [
      [{ type: 'TerrainPlain', occupantType: 'OccupantUnit', occupantId: 7 }, plainTile()],
    ];
    const target: Coordinate = { x: 1, y: 0 };

    vi.mocked(getMatchState).mockResolvedValueOnce(
      makeState(initialGrid, { activeTeam: 1, units: [unit] })
    );
    vi.mocked(getAllowedTiles).mockRejectedValueOnce(new Error('tiles unavailable'));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const unitGraphics = occupantGraphics(0);
    pointerDownOf(unitGraphics)();
    const [moveButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(moveButtonGraphics!)();
    await Promise.resolve();
    await Promise.resolve();

    const staleErrorText = errorTextByMessage('tiles unavailable');
    expect(staleErrorText.destroy).not.toHaveBeenCalled();

    // Now successfully submit a Move for the same unit — a fresh turn-command action.
    vi.mocked(getMatchState)
      .mockResolvedValueOnce(makeState(initialGrid, { activeTeam: 1, units: [unit] }))
      .mockResolvedValueOnce(makeState(initialGrid, { activeTeam: 1, units: [unit] }));
    vi.mocked(getAllowedTiles).mockResolvedValue([target]);
    vi.mocked(submitTurnCommand).mockResolvedValue([
      { type: 'unitMoved', unitId: 7, from: { x: 0, y: 0 }, to: target },
    ]);
    await submitViaUI(occupantGraphics(0), 0);
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();

    expect(staleErrorText.destroy).toHaveBeenCalled();
  });
});
