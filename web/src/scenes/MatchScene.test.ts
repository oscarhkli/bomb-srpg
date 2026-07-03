import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import { initRoom, getMatchState } from '../engine/api';
import {
  TERRAIN_COLORS,
  TERRAIN_BORDER_COLOR,
  TEAM_COLORS,
  OCCUPANT_STROKE_COLOR,
  SOFTBLOCK_COLOR,
  SOFTBLOCK_CORNER_RADIUS,
  BOMB_COLOR,
} from '../constants';
import MatchScene from './MatchScene';
import type { GameState, Tile, TerrainType, Unit, SoftBlock, Bomb } from '../types/api';

vi.mock('../engine/api');

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

// The error text (when getMatchState rejects) is always the first Text instance created.
function errorText(): ReturnType<typeof mockScene.add.text> {
  return mockScene.add.text.mock.results[0]!.value as ReturnType<typeof mockScene.add.text>;
}

beforeEach(() => {
  vi.clearAllMocks();
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

    // Only the grid Graphics instance should exist — no occupant Graphics for the dead unit.
    expect(mockScene.add.graphics).toHaveBeenCalledTimes(1);
  });

  it('shows an error message when getMatchState rejects, without rendering', async () => {
    vi.mocked(getMatchState).mockRejectedValue(new Error('network fail'));

    const scene = new MatchScene();
    scene.create({ roomId: 'room-abc', playerTokens: ['t1', 't2'] });
    await Promise.resolve();
    await Promise.resolve();

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Failed to load match state'
    );
    expect(errorText().setOrigin).toHaveBeenCalledWith(0.5);
    expect(mockScene.add.graphics).not.toHaveBeenCalled();
    expect(mockScene.cameras.main.centerOn).not.toHaveBeenCalled();
  });
});
