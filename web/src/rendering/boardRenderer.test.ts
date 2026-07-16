import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import {
  firstGraphics as terrainGraphics,
  occupantGraphics,
  pointerDownOf,
} from '../test/sceneHelpers';
import {
  tileOf,
  plainTile,
  makeUnit as unit,
  makeSoftBlock as softBlock,
  makeBomb as bomb,
} from '../test/fixtures';
import {
  TERRAIN_COLORS,
  TERRAIN_BORDER_COLOR,
  TEAM_COLOR_FALLBACK,
  TEAM_COLORS,
  OCCUPANT_STROKE_COLOR,
  SOFTBLOCK_COLOR,
  SOFTBLOCK_CORNER_RADIUS,
  BOMB_COLOR,
} from '../constants';
import type { Bomb, GameState, SoftBlock, Tile, TerrainType, Unit } from '../types/api';
import type { BombGraphics } from './resolveTurnPlayer';
import {
  renderTerrain,
  renderOccupants,
  renderBomb,
  tileCenter,
  type BoardRenderContext,
} from './boardRenderer';

beforeEach(() => {
  vi.clearAllMocks();
});

function ctx(overrides: Partial<BoardRenderContext> = {}): BoardRenderContext {
  return {
    scene: mockScene as never,
    terrainObjects: [],
    occupantObjects: [],
    unitGraphicsById: new Map(),
    bombGraphicsById: new Map<number, BombGraphics>(),
    softBlockGraphicsById: new Map(),
    onUnitClicked: vi.fn(),
    ...overrides,
  };
}

function state(
  grid: Tile[][],
  parts: { units?: Unit[]; softBlocks?: SoftBlock[]; bombs?: Bomb[] } = {}
): GameState {
  return {
    turn: 1,
    inSuddenDeath: false,
    activeTeam: 1,
    grid,
    units: parts.units ?? [],
    bombs: parts.bombs ?? [],
    softBlocks: parts.softBlocks ?? [],
    turnCommands: [],
  };
}

// Entry-order paint: terrain (grid = first Graphics) then occupants, matching create()'s sequence
// so the occupantGraphics(i) = graphicsAt(i+1) indexing holds for the occupant assertions below.
function renderAll(c: BoardRenderContext, s: GameState): void {
  renderTerrain(c, s.grid);
  renderOccupants(c, s);
}

describe('tileCenter', () => {
  it('returns the pixel center of a tile', () => {
    expect(tileCenter({ x: 1, y: 0 })).toEqual({ cx: 72, cy: 24 });
  });
});

describe('renderTerrain', () => {
  it('returns the grid dimensions', () => {
    const dims = renderTerrain(ctx(), [
      [plainTile(), plainTile(), plainTile()],
      [plainTile(), plainTile(), plainTile()],
    ]);
    expect(dims).toEqual({ cols: 3, rows: 2 });
  });

  it('draws every tile at its world position with terrain fill and a border', () => {
    renderTerrain(ctx(), [
      [plainTile(), plainTile(), plainTile()],
      [plainTile(), plainTile(), plainTile()],
    ]);

    const grid = terrainGraphics();
    expect(grid.lineStyle).toHaveBeenCalledWith(1, TERRAIN_BORDER_COLOR);
    expect(grid.fillRect).toHaveBeenCalledTimes(6);
    expect(grid.fillRect).toHaveBeenNthCalledWith(1, 0, 0, 48, 48);
    expect(grid.fillRect).toHaveBeenNthCalledWith(6, 96, 48, 48, 48);
    expect(grid.strokeRect).toHaveBeenCalledTimes(6);
  });

  it('fills each terrain type with its TERRAIN_COLORS value', () => {
    const types: TerrainType[] = [
      'TerrainPlain',
      'TerrainBlock',
      'TerrainTower',
      'TerrainWater',
      'TerrainLava',
    ];
    renderTerrain(ctx(), [types.map(tileOf)]);

    const grid = terrainGraphics();
    types.forEach((type, i) => {
      expect(grid.fillStyle).toHaveBeenNthCalledWith(i + 1, TERRAIN_COLORS[type]);
    });
  });

  it('tracks the grid graphics in terrainObjects, not occupantObjects', () => {
    const c = ctx();
    renderTerrain(c, [[plainTile()]]);

    expect(c.terrainObjects).toHaveLength(1);
    expect(c.terrainObjects[0]).toBe(terrainGraphics());
    expect(c.occupantObjects).toHaveLength(0);
  });

  it('destroys the prior terrain on a repeat entry so re-running create() does not leak grids', () => {
    const c = ctx();
    renderTerrain(c, [[plainTile()]]);
    const firstGrid = terrainGraphics();

    renderTerrain(c, [[plainTile()]]);

    expect(firstGrid.destroy).toHaveBeenCalled();
    expect(c.terrainObjects).toHaveLength(1);
  });
});

describe('renderOccupants — units', () => {
  it('renders a live unit as a team-colored 32x32 square centered on its tile', () => {
    renderAll(
      ctx(),
      state([[plainTile(), plainTile()]], { units: [unit({ position: { x: 1, y: 0 }, team: 1 })] })
    );

    const g = occupantGraphics(0);
    expect(g.fillStyle).toHaveBeenCalledWith(TEAM_COLORS[1]);
    expect(g.fillRect).toHaveBeenCalledWith(56, 8, 32, 32);
  });

  it('does not render a dead unit (hp 0)', () => {
    renderAll(ctx(), state([[plainTile()]], { units: [unit({ hp: 0 })] }));
    // Only the grid Graphics was created.
    expect(mockScene.add.graphics).toHaveBeenCalledTimes(1);
  });

  it('warns and falls back to TEAM_COLOR_FALLBACK for an unconfigured team color', () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => undefined);
    renderAll(ctx(), state([[plainTile()]], { units: [unit({ team: 99 })] }));

    expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining('99'));
    expect(occupantGraphics(0).fillStyle).toHaveBeenCalledWith(TEAM_COLOR_FALLBACK);
    warnSpy.mockRestore();
  });

  it('registers the unit graphics in the map', () => {
    const c = ctx();
    renderAll(c, state([[plainTile()]], { units: [unit({ id: 7 })] }));
    expect(c.unitGraphicsById.has(7)).toBe(true);
  });

  it('draws a circle icon for Bandit', () => {
    renderAll(ctx(), state([[plainTile()]], { units: [unit({ type: 'Bandit' })] }));
    const g = occupantGraphics(0);
    expect(g.lineStyle).toHaveBeenCalledWith(2, OCCUPANT_STROKE_COLOR);
    expect(g.strokeCircle).toHaveBeenCalledWith(24, 24, 10);
  });

  it('draws an apex-centered 3-point polygon for Witch', () => {
    renderAll(ctx(), state([[plainTile()]], { units: [unit({ type: 'Witch' })] }));
    const [points, closed] = occupantGraphics(0).strokePoints.mock.calls[0] as [
      { x: number; y: number }[],
      boolean,
    ];
    expect(points).toHaveLength(3);
    expect(closed).toBe(true);
    expect(points[0]!.x).toBe(24);
  });

  it('draws a 10-vertex star for King', () => {
    renderAll(ctx(), state([[plainTile()]], { units: [unit({ type: 'King' })] }));
    const [points] = occupantGraphics(0).strokePoints.mock.calls[0] as [{ x: number; y: number }[]];
    expect(points).toHaveLength(10);
  });

  it('warns and draws no icon for an unrecognized archetype', () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => undefined);
    renderAll(ctx(), state([[plainTile()]], { units: [unit({ type: 'Mystic' })] }));

    const g = occupantGraphics(0);
    expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining('Mystic'));
    expect(g.strokeCircle).not.toHaveBeenCalled();
    expect(g.strokePoints).not.toHaveBeenCalled();
    warnSpy.mockRestore();
  });

  it('makes a unit clickable over its full tile and invokes onUnitClicked on pointerdown', () => {
    const onUnitClicked = vi.fn();
    const u = unit({ id: 7, position: { x: 1, y: 0 } });
    renderAll(ctx({ onUnitClicked }), state([[plainTile(), plainTile()]], { units: [u] }));

    const g = occupantGraphics(0);
    expect(g.setInteractive).toHaveBeenCalledWith(
      expect.objectContaining({ x: 48, y: 0, width: 48, height: 48 }),
      expect.any(Function)
    );
    pointerDownOf(g)();
    expect(onUnitClicked).toHaveBeenCalledWith(u);
  });
});

describe('renderOccupants — softBlocks & bombs', () => {
  it('renders a softBlock as a rounded rect and registers it in the map', () => {
    const c = ctx();
    renderAll(
      c,
      state([[plainTile(), plainTile()]], {
        softBlocks: [softBlock({ id: 3, position: { x: 1, y: 0 } })],
      })
    );

    const g = occupantGraphics(0);
    expect(g.fillStyle).toHaveBeenCalledWith(SOFTBLOCK_COLOR);
    expect(g.fillRoundedRect).toHaveBeenCalledWith(51, 3, 42, 42, SOFTBLOCK_CORNER_RADIUS);
    expect(c.softBlockGraphicsById.has(3)).toBe(true);
  });

  it('logs on softBlock click', () => {
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => undefined);
    const block = softBlock({ id: 3 });
    renderAll(ctx(), state([[plainTile()]], { softBlocks: [block] }));
    pointerDownOf(occupantGraphics(0))();
    expect(consoleSpy).toHaveBeenCalledWith('SoftBlock 3 is clicked', block);
    consoleSpy.mockRestore();
  });

  it('renders a bomb as a circle with countdown text parented in a single container and registers both in the map', () => {
    const c = ctx();
    renderAll(
      c,
      state([[plainTile(), plainTile()]], {
        bombs: [bomb({ id: 9, position: { x: 1, y: 0 }, countdown: 5 })],
      })
    );

    const g = occupantGraphics(0);
    expect(g.fillStyle).toHaveBeenCalledWith(BOMB_COLOR);
    expect(g.fillCircle).toHaveBeenCalledWith(0, 0, 12);
    expect(mockScene.add.text).toHaveBeenCalledWith(0, 0, '5', expect.objectContaining({}));
    expect(mockScene.add.container).toHaveBeenCalledWith(72, 24, [g, expect.anything()]);
    expect(c.bombGraphicsById.has(9)).toBe(true);
  });
});

describe('renderBomb', () => {
  it('adds a single bomb without touching the rest of the board and registers it', () => {
    const c = ctx();
    renderBomb(c, bomb({ id: 42, position: { x: 0, y: 0 }, countdown: 2 }));

    expect(mockScene.add.graphics).toHaveBeenCalledOnce();
    const g = terrainGraphics();
    expect(g.fillCircle).toHaveBeenCalledWith(0, 0, 12);
    expect(mockScene.add.container).toHaveBeenCalledWith(24, 24, [g, expect.anything()]);
    expect(c.bombGraphicsById.has(42)).toBe(true);
    expect(c.occupantObjects.length).toBeGreaterThan(0);
  });
});

describe('renderOccupants — teardown', () => {
  it('destroys prior occupant objects and clears the graphics maps on re-render', () => {
    const c = ctx();
    renderAll(c, state([[plainTile()]], { units: [unit({ id: 7 })] }));
    const firstUnitGraphics = occupantGraphics(0);
    expect(c.unitGraphicsById.has(7)).toBe(true);

    renderOccupants(c, state([[plainTile()]]));

    expect(firstUnitGraphics.destroy).toHaveBeenCalled();
    expect(c.unitGraphicsById.has(7)).toBe(false);
  });

  it('leaves the terrain layer untouched on an occupant swap', () => {
    const c = ctx();
    renderTerrain(c, [[plainTile()]]);
    const grid = terrainGraphics();
    renderOccupants(c, state([[plainTile()]], { units: [unit({ id: 7 })] }));

    // Swap the occupants again — the grid must survive both rebuilds.
    renderOccupants(c, state([[plainTile()]]));

    expect(grid.destroy).not.toHaveBeenCalled();
    expect(c.terrainObjects).toEqual([grid]);
  });
});
