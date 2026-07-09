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
import {
  TERRAIN_COLORS,
  TERRAIN_BORDER_COLOR,
  TEAM_COLORS,
  OCCUPANT_STROKE_COLOR,
  SOFTBLOCK_COLOR,
  SOFTBLOCK_CORNER_RADIUS,
  BOMB_COLOR,
} from '../constants';
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
  SoftBlock,
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

function makeSoftBlock(overrides: Partial<SoftBlock> = {}): SoftBlock {
  return { id: 1, position: { x: 0, y: 0 }, ...overrides };
}

function makeBomb(overrides: Partial<Bomb> = {}): Bomb {
  return { id: 1, ownerId: 1, position: { x: 0, y: 0 }, range: 2, countdown: 3, ...overrides };
}

// The grid is always the first Graphics instance created in create().
function gridGraphics(): ReturnType<typeof mockScene.add.graphics> {
  return mockScene.add.graphics.mock.results[0]!.value as ReturnType<typeof mockScene.add.graphics>;
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

  it('renders a 3x2 grid as 6 tiles at correct world positions', async () => {
    const row = (): Tile[] => [plainTile(), plainTile(), plainTile()];
    vi.mocked(getMatchState).mockResolvedValue(makeState([row(), row()]));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const grid = gridGraphics();
    expect(grid.fillRect).toHaveBeenCalledTimes(6);
    expect(grid.fillRect).toHaveBeenNthCalledWith(1, 0, 0, 48, 48);
    expect(grid.fillRect).toHaveBeenNthCalledWith(2, 48, 0, 48, 48);
    expect(grid.fillRect).toHaveBeenNthCalledWith(3, 96, 0, 48, 48);
    expect(grid.fillRect).toHaveBeenNthCalledWith(4, 0, 48, 48, 48);
    expect(grid.fillRect).toHaveBeenNthCalledWith(5, 48, 48, 48, 48);
    expect(grid.fillRect).toHaveBeenNthCalledWith(6, 96, 48, 48, 48);
  });

  it('fills each terrain type with its TERRAIN_COLORS value', async () => {
    const types: TerrainType[] = [
      'TerrainPlain',
      'TerrainBlock',
      'TerrainTower',
      'TerrainWater',
      'TerrainLava',
    ];
    vi.mocked(getMatchState).mockResolvedValue(makeState([types.map(tileOf)]));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const grid = gridGraphics();
    types.forEach((type, i) => {
      expect(grid.fillStyle).toHaveBeenNthCalledWith(i + 1, TERRAIN_COLORS[type]);
    });
  });

  it('draws a 1px black border around every tile', async () => {
    const row = (): Tile[] => [plainTile(), plainTile(), plainTile()];
    vi.mocked(getMatchState).mockResolvedValue(makeState([row(), row()]));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const grid = gridGraphics();
    expect(grid.lineStyle).toHaveBeenCalledWith(1, TERRAIN_BORDER_COLOR);
    expect(grid.strokeRect).toHaveBeenCalledTimes(6);
    expect(grid.strokeRect).toHaveBeenNthCalledWith(1, 0, 0, 48, 48);
    expect(grid.strokeRect).toHaveBeenNthCalledWith(6, 96, 48, 48, 48);
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

  it('renders a Unit as a team-colored 32x32 square centered on its tile', async () => {
    const unit = makeUnit({ position: { x: 1, y: 0 }, team: 1 });
    vi.mocked(getMatchState).mockResolvedValue(
      makeState([[plainTile(), plainTile()]], { units: [unit] })
    );

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const g = occupantGraphics(0);
    // tile (1,0) center = (48 + 24, 24) = (72, 24); 32x32 square top-left = (56, 8)
    expect(g.fillStyle).toHaveBeenCalledWith(TEAM_COLORS[1]);
    expect(g.fillRect).toHaveBeenCalledWith(56, 8, 32, 32);
  });

  it('warns and falls back to white when a Unit has an unconfigured team color', async () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => undefined);
    const unit = makeUnit({ team: 99 });
    vi.mocked(getMatchState).mockResolvedValue(makeState([[plainTile()]], { units: [unit] }));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const g = occupantGraphics(0);
    expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining('99'));
    expect(g.fillStyle).toHaveBeenCalledWith(0xffffff);
    warnSpy.mockRestore();
  });

  it('draws a white-stroked circle icon inside a Bandit unit', async () => {
    const unit = makeUnit({ type: 'Bandit', position: { x: 0, y: 0 } });
    vi.mocked(getMatchState).mockResolvedValue(makeState([[plainTile()]], { units: [unit] }));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const g = occupantGraphics(0);
    expect(g.lineStyle).toHaveBeenCalledWith(2, OCCUPANT_STROKE_COLOR);
    expect(g.strokeCircle).toHaveBeenCalledWith(24, 24, 10);
  });

  it('draws a white-stroked 3-point triangle icon inside a Witch unit, apex centered', async () => {
    const unit = makeUnit({ type: 'Witch', position: { x: 0, y: 0 } });
    vi.mocked(getMatchState).mockResolvedValue(makeState([[plainTile()]], { units: [unit] }));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const g = occupantGraphics(0);
    expect(g.lineStyle).toHaveBeenCalledWith(2, OCCUPANT_STROKE_COLOR);
    expect(g.strokePoints).toHaveBeenCalledTimes(1);
    const [points, closed] = g.strokePoints.mock.calls[0] as [{ x: number; y: number }[], boolean];
    expect(points).toHaveLength(3);
    expect(closed).toBe(true);
    expect(points[0]!.x).toBe(24); // apex centered on tile (0,0) -> cx=24
  });

  it('draws a white-stroked 5-point pentagon icon inside a Fighter unit, apex centered', async () => {
    const unit = makeUnit({ type: 'Fighter', position: { x: 0, y: 0 } });
    vi.mocked(getMatchState).mockResolvedValue(makeState([[plainTile()]], { units: [unit] }));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const g = occupantGraphics(0);
    expect(g.lineStyle).toHaveBeenCalledWith(2, OCCUPANT_STROKE_COLOR);
    expect(g.strokePoints).toHaveBeenCalledTimes(1);
    const [points, closed] = g.strokePoints.mock.calls[0] as [{ x: number; y: number }[], boolean];
    expect(points).toHaveLength(5);
    expect(closed).toBe(true);
    expect(points[0]!.x).toBe(24);
  });

  it('draws a white-stroked 10-point star icon inside a King unit, apex centered', async () => {
    const unit = makeUnit({ type: 'King', position: { x: 0, y: 0 } });
    vi.mocked(getMatchState).mockResolvedValue(makeState([[plainTile()]], { units: [unit] }));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const g = occupantGraphics(0);
    expect(g.lineStyle).toHaveBeenCalledWith(2, OCCUPANT_STROKE_COLOR);
    expect(g.strokePoints).toHaveBeenCalledTimes(1);
    const [points, closed] = g.strokePoints.mock.calls[0] as [{ x: number; y: number }[], boolean];
    expect(points).toHaveLength(10);
    expect(closed).toBe(true);
    expect(points[0]!.x).toBe(24);
  });

  it('warns and draws no icon for an unrecognized archetype', async () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => undefined);
    const unit = makeUnit({ type: 'Mystic' });
    vi.mocked(getMatchState).mockResolvedValue(makeState([[plainTile()]], { units: [unit] }));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const g = occupantGraphics(0);
    expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining('Mystic'));
    expect(g.strokeCircle).not.toHaveBeenCalled();
    expect(g.strokePoints).not.toHaveBeenCalled();
    warnSpy.mockRestore();
  });

  it('makes a Unit clickable over its full 48x48 tile and logs on click', async () => {
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => undefined);
    const unit = makeUnit({ id: 7, position: { x: 1, y: 0 } });
    vi.mocked(getMatchState).mockResolvedValue(
      makeState([[plainTile(), plainTile()]], { units: [unit] })
    );

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const g = occupantGraphics(0);
    expect(g.setInteractive).toHaveBeenCalledWith(
      expect.objectContaining({ x: 48, y: 0, width: 48, height: 48 }),
      expect.any(Function)
    );
    const onPointerDown = g.on.mock.calls.find(
      call => call[0] === 'pointerdown'
    )?.[1] as () => void;
    expect(onPointerDown).toBeInstanceOf(Function);

    onPointerDown();
    expect(consoleSpy).toHaveBeenCalledWith('Unit 7 is clicked', unit);
    consoleSpy.mockRestore();
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

  it('renders a SoftBlock as a light-grey 42x42 rounded rect centered on its tile', async () => {
    const block = makeSoftBlock({ position: { x: 1, y: 0 } });
    vi.mocked(getMatchState).mockResolvedValue(
      makeState([[plainTile(), plainTile()]], { softBlocks: [block] })
    );

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const g = occupantGraphics(0);
    // tile (1,0) center = (72, 24); 42x42 rect top-left = (51, 3)
    expect(g.fillStyle).toHaveBeenCalledWith(SOFTBLOCK_COLOR);
    expect(g.fillRoundedRect).toHaveBeenCalledWith(51, 3, 42, 42, SOFTBLOCK_CORNER_RADIUS);
  });

  it('makes a SoftBlock clickable over its full tile and logs on click', async () => {
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => undefined);
    const block = makeSoftBlock({ id: 3, position: { x: 0, y: 1 } });
    vi.mocked(getMatchState).mockResolvedValue(
      makeState([[plainTile()], [plainTile()]], { softBlocks: [block] })
    );

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const g = occupantGraphics(0);
    expect(g.setInteractive).toHaveBeenCalledWith(
      expect.objectContaining({ x: 0, y: 48, width: 48, height: 48 }),
      expect.any(Function)
    );
    const onPointerDown = g.on.mock.calls.find(
      call => call[0] === 'pointerdown'
    )?.[1] as () => void;
    onPointerDown();
    expect(consoleSpy).toHaveBeenCalledWith('SoftBlock 3 is clicked', block);
    consoleSpy.mockRestore();
  });

  it('renders a Bomb as a 24x24 dark circle with white countdown text centered on its tile', async () => {
    const bomb = makeBomb({ position: { x: 1, y: 0 }, countdown: 5 });
    vi.mocked(getMatchState).mockResolvedValue(
      makeState([[plainTile(), plainTile()]], { bombs: [bomb] })
    );

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const g = occupantGraphics(0);
    expect(g.fillStyle).toHaveBeenCalledWith(BOMB_COLOR);
    expect(g.fillCircle).toHaveBeenCalledWith(72, 24, 12);

    expect(mockScene.add.text).toHaveBeenCalledWith(
      72,
      24,
      '5',
      expect.objectContaining({ color: '#ffffff' })
    );
    const text = mockScene.add.text.mock.results[0]!.value as ReturnType<typeof mockScene.add.text>;
    expect(text.setOrigin).toHaveBeenCalledWith(0.5);
  });

  it('makes a Bomb clickable over its full tile and logs on click', async () => {
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => undefined);
    const bomb = makeBomb({ id: 9, position: { x: 0, y: 0 } });
    vi.mocked(getMatchState).mockResolvedValue(makeState([[plainTile()]], { bombs: [bomb] }));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();

    const g = occupantGraphics(0);
    expect(g.setInteractive).toHaveBeenCalledWith(
      expect.objectContaining({ x: 0, y: 0, width: 48, height: 48 }),
      expect.any(Function)
    );
    const onPointerDown = g.on.mock.calls.find(
      call => call[0] === 'pointerdown'
    )?.[1] as () => void;
    onPointerDown();
    expect(consoleSpy).toHaveBeenCalledWith('Bomb 9 is clicked', bomb);
    consoleSpy.mockRestore();
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

  it('shows an error and stops processing when the unitMoved event is invalid', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [[plainTile(), plainTile()]];
    const target: Coordinate = { x: 1, y: 0 };

    vi.mocked(getMatchState)
      .mockResolvedValueOnce(makeState(initialGrid, { activeTeam: 1, units: [unit] }))
      .mockResolvedValueOnce(makeState(initialGrid, { activeTeam: 1, units: [unit] }));
    vi.mocked(getAllowedTiles).mockResolvedValue([target]);
    // unitId mismatches the actor (7) — invalid per spec's unitMoved validation.
    const events: GameEvent[] = [
      { type: 'unitMoved', unitId: 999, from: { x: 0, y: 0 }, to: target },
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

  it('flags a mismatch when the refreshed grid differs from the predicted grid', async () => {
    const unit = makeUnit({ id: 7, team: 1, position: { x: 0, y: 0 } });
    const initialGrid: Tile[][] = [
      [{ type: 'TerrainPlain', occupantType: 'OccupantUnit', occupantId: 7 }, plainTile()],
    ];
    const target: Coordinate = { x: 1, y: 0 };
    // Refresh returns a grid that does NOT match the predicted post-move grid (unit missing).
    const unexpectedGrid: Tile[][] = [[plainTile(), plainTile()]];

    vi.mocked(getMatchState)
      .mockResolvedValueOnce(makeState(initialGrid, { activeTeam: 1, units: [unit] }))
      .mockResolvedValueOnce(makeState(unexpectedGrid, { activeTeam: 1, units: [unit] }));
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

  it('stacks multiple error messages within one action instead of overlapping at the same position', async () => {
    await setUpEmptyBoardAndClickResolve();
    vi.mocked(resolveTurn).mockRejectedValue(new Error('resolve failed'));
    vi.mocked(getMatchState).mockRejectedValue(new Error('network down'));

    const [, yesButtonGraphics] = mockScene.add.graphics.mock.results
      .slice(-3)
      .map(r => r.value as ReturnType<typeof mockScene.add.graphics>);
    pointerDownOf(yesButtonGraphics!)();
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();

    const firstCall = textCalls().find(c => c[2] === 'resolve failed');
    const secondCall = textCalls().find(c => c[2] === 'Failed to refresh match state');
    expect(firstCall).toBeDefined();
    expect(secondCall).toBeDefined();
    expect(secondCall![1]).toBeGreaterThan(firstCall![1]);
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
