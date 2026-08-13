import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import {
  firstGraphics as terrainGraphics,
  occupantSprite,
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
  TEAM_COLORS,
  OCCUPANT_STROKE_COLOR,
  TILE_SIZE,
  SPRITE_GROUND_MARGIN,
} from '../constants';
import type { Bomb, GameState, SoftBlock, Tile, TerrainType, Unit } from '../types/api';
import type { BombGraphics } from './resolveTurnPlayer';
import {
  renderTerrain,
  renderOccupants,
  renderBomb,
  tileCenter,
  drawUnitSprite,
  drawArchetypeIcon,
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
    unitSpritesById: new Map(),
    bombGraphicsById: new Map<number, BombGraphics>(),
    softBlockSpritesById: new Map(),
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

// Entry-order paint: terrain (grid = first Graphics) then occupants.
function renderAll(c: BoardRenderContext, s: GameState): void {
  renderTerrain(c, s.grid);
  renderOccupants(c, s);
}

describe('tileCenter', () => {
  it('returns the pixel center of a tile', () => {
    expect(tileCenter({ x: 1, y: 0 })).toEqual({ cx: 48, cy: 16 });
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

    // On this 3x2 grid, centered in GameBoardRegion (offsetX=192, offsetY=148).
    const grid = terrainGraphics();
    expect(grid.lineStyle).toHaveBeenCalledWith(1, TERRAIN_BORDER_COLOR);
    expect(grid.fillRect).toHaveBeenCalledTimes(6);
    expect(grid.fillRect).toHaveBeenNthCalledWith(1, 0 + 192, 0 + 148, TILE_SIZE, TILE_SIZE);
    expect(grid.fillRect).toHaveBeenNthCalledWith(
      6,
      2 * TILE_SIZE + 192,
      TILE_SIZE + 148,
      TILE_SIZE,
      TILE_SIZE
    );
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
  it('renders a live unit as a sprite ground-anchored on its tile with origin (0.5, 1)', () => {
    renderAll(
      ctx(),
      state([[plainTile(), plainTile()]], { units: [unit({ position: { x: 1, y: 0 }, team: 1 })] })
    );

    // On this 1x2 grid, centered in GameBoardRegion (offsetX=208, offsetY=164).
    const sprite = occupantSprite(0);
    expect(sprite.x).toBe(1 * TILE_SIZE + TILE_SIZE / 2 + 208);
    expect(sprite.y).toBe((0 + 1) * TILE_SIZE + SPRITE_GROUND_MARGIN + 164);
    expect(sprite.setOrigin).toHaveBeenCalledWith(0.5, 1);
  });

  it('picks the texture key matching archetype + team', () => {
    renderAll(ctx(), state([[plainTile()]], { units: [unit({ type: 'Bandit', team: 2 })] }));
    expect(mockScene.add.sprite).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'unit_bandit_red',
      'unit_bandit_red-frame'
    );
  });

  it('does not render a dead unit (hp 0)', () => {
    renderAll(ctx(), state([[plainTile()]], { units: [unit({ hp: 0 })] }));
    expect(mockScene.add.sprite).not.toHaveBeenCalled();
  });

  it('warns and falls back to the blue texture for an unconfigured team', () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => undefined);
    renderAll(ctx(), state([[plainTile()]], { units: [unit({ type: 'Bandit', team: 99 })] }));

    expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining('99'));
    expect(mockScene.add.sprite).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'unit_bandit_blue',
      'unit_bandit_blue-frame'
    );
    warnSpy.mockRestore();
  });

  it('registers the unit sprite in the map', () => {
    const c = ctx();
    renderAll(c, state([[plainTile()]], { units: [unit({ id: 7 })] }));
    expect(c.unitSpritesById.has(7)).toBe(true);
  });

  it('makes a unit clickable over its tile footprint (local space, independent of grid position) and invokes onUnitClicked on pointerdown', () => {
    const onUnitClicked = vi.fn();
    const u = unit({ id: 7, position: { x: 1, y: 0 } });
    renderAll(ctx({ onUnitClicked }), state([[plainTile(), plainTile()]], { units: [u] }));

    const sprite = occupantSprite(0);
    // Local-space hit rect: sprite is 64x64 (untrimmed canvas), origin (0.5, 1) — the tile
    // footprint sits at x=[16,48), y=[16,48) regardless of the unit's grid position.
    expect(sprite.setInteractive).toHaveBeenCalledWith(
      expect.objectContaining({ x: 16, y: 16, width: TILE_SIZE, height: TILE_SIZE }),
      expect.any(Function)
    );
    pointerDownOf(sprite)();
    expect(onUnitClicked).toHaveBeenCalledWith(u);
  });
});

describe('renderOccupants — softBlocks & bombs', () => {
  it('renders a softBlock as a ground-anchored sprite and registers it in the map', () => {
    const c = ctx();
    renderAll(
      c,
      state([[plainTile(), plainTile()]], {
        softBlocks: [softBlock({ id: 3, position: { x: 1, y: 0 } })],
      })
    );

    // On this 1x2 grid, centered in GameBoardRegion (offsetX=208, offsetY=164).
    const sprite = occupantSprite(0);
    expect(mockScene.add.sprite).toHaveBeenCalledWith(
      1 * TILE_SIZE + TILE_SIZE / 2 + 208,
      (0 + 1) * TILE_SIZE + SPRITE_GROUND_MARGIN + 164,
      'soft_block',
      'soft_block-frame'
    );
    expect(sprite.setOrigin).toHaveBeenCalledWith(0.5, 1);
    expect(c.softBlockSpritesById.has(3)).toBe(true);
  });

  it('logs on softBlock click', () => {
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => undefined);
    const block = softBlock({ id: 3 });
    renderAll(ctx(), state([[plainTile()]], { softBlocks: [block] }));
    pointerDownOf(occupantSprite(0))();
    expect(consoleSpy).toHaveBeenCalledWith('SoftBlock 3 is clicked', block);
    consoleSpy.mockRestore();
  });

  it('renders a bomb as a sprite with countdown text parented in a single container and registers both in the map', () => {
    const c = ctx();
    renderAll(
      c,
      state([[plainTile(), plainTile()]], {
        bombs: [bomb({ id: 9, position: { x: 1, y: 0 }, countdown: 5 })],
      })
    );

    expect(mockScene.add.sprite).toHaveBeenCalledWith(
      0,
      TILE_SIZE / 2 + SPRITE_GROUND_MARGIN,
      'bomb',
      'bomb-frame'
    );
    // Countdown text is added last, so it renders on top of the bomb sprite.
    expect(mockScene.add.text).toHaveBeenCalledWith(0, 0, '5', expect.objectContaining({}));
    // tileCenter of {x:1,y:0} on this 1x2 grid, centered in GameBoardRegion: (256, 180).
    expect(mockScene.add.container).toHaveBeenCalledWith(256, 180, [
      expect.anything(),
      expect.anything(),
    ]);
    expect(c.bombGraphicsById.has(9)).toBe(true);
  });
});

describe('renderBomb', () => {
  it('adds a single bomb without touching the rest of the board and registers it', () => {
    const c = ctx();
    renderBomb(c, bomb({ id: 42, position: { x: 0, y: 0 }, countdown: 2 }));

    // No Graphics allocated for the bomb itself — glyph is a Sprite, countdown a Text.
    expect(mockScene.add.graphics).not.toHaveBeenCalled();
    expect(mockScene.add.sprite).toHaveBeenCalledWith(
      0,
      TILE_SIZE / 2 + SPRITE_GROUND_MARGIN,
      'bomb',
      'bomb-frame'
    );
    expect(mockScene.add.container).toHaveBeenCalledWith(TILE_SIZE / 2, TILE_SIZE / 2, [
      expect.anything(),
      expect.anything(),
    ]);
    expect(c.bombGraphicsById.has(42)).toBe(true);
    expect(c.occupantObjects.length).toBeGreaterThan(0);
  });
});

// Retained for MatchSettingsScene's UnitPage, which still draws vector unit icons.
describe('drawUnitSprite', () => {
  it('fills a team-colored square of the given size and scales the archetype icon radius', () => {
    const c = ctx();
    renderTerrain(c, [[plainTile()]]);
    const g = terrainGraphics();

    drawUnitSprite(g as never, 48, 48, 96, 'Bandit', TEAM_COLORS[1]!);

    expect(g.fillStyle).toHaveBeenCalledWith(TEAM_COLORS[1]);
    expect(g.fillRect).toHaveBeenCalledWith(0, 0, 96, 96);
    // radius scales with size: 96 * (10/32) = 30
    expect(g.strokeCircle).toHaveBeenCalledWith(48, 48, 30);
  });

  it('fills a rounded-corner square when cornerRadius is given', () => {
    const c = ctx();
    renderTerrain(c, [[plainTile()]]);
    const g = terrainGraphics();

    drawUnitSprite(g as never, 48, 48, 96, 'Bandit', TEAM_COLORS[1]!, 8);

    expect(g.fillRoundedRect).toHaveBeenCalledWith(0, 0, 96, 96, 8);
  });
});

describe('drawArchetypeIcon', () => {
  it('defaults to OCCUPANT_ICON_RADIUS when no radius is given', () => {
    const c = ctx();
    renderTerrain(c, [[plainTile()]]);
    const g = terrainGraphics();

    drawArchetypeIcon(g as never, 'Bandit', 10, 10);

    expect(g.strokeCircle).toHaveBeenCalledWith(10, 10, 10);
  });

  it('honors an explicit radius', () => {
    const c = ctx();
    renderTerrain(c, [[plainTile()]]);
    const g = terrainGraphics();

    drawArchetypeIcon(g as never, 'Bandit', 10, 10, 40);

    expect(g.strokeCircle).toHaveBeenCalledWith(10, 10, 40);
  });

  it('draws icons using OCCUPANT_STROKE_COLOR', () => {
    const c = ctx();
    renderTerrain(c, [[plainTile()]]);
    const g = terrainGraphics();

    drawArchetypeIcon(g as never, 'Bandit', 10, 10);

    expect(g.lineStyle).toHaveBeenCalledWith(2, OCCUPANT_STROKE_COLOR);
  });

  it("scales the King star's inner radius proportionally with an explicit radius", () => {
    const c = ctx();
    renderTerrain(c, [[plainTile()]]);
    const g = terrainGraphics();

    drawArchetypeIcon(g as never, 'King', 0, 0, 40);

    // Outer/inner ratio must stay 10:4 (OCCUPANT_ICON_RADIUS's default) at any size.
    const [points] = g.strokePoints.mock.calls[0] as [{ x: number; y: number }[]];
    const distances = points.map(p => Math.hypot(p.x, p.y));
    const outer = Math.max(...distances);
    const inner = Math.min(...distances);
    expect(outer).toBeCloseTo(40);
    expect(inner).toBeCloseTo(16);
  });
});

describe('renderOccupants — teardown', () => {
  it('destroys prior occupant objects and clears the graphics maps on re-render', () => {
    const c = ctx();
    renderAll(c, state([[plainTile()]], { units: [unit({ id: 7 })] }));
    const firstUnitSprite = occupantSprite(0);
    expect(c.unitSpritesById.has(7)).toBe(true);

    renderOccupants(c, state([[plainTile()]]));

    expect(firstUnitSprite.destroy).toHaveBeenCalled();
    expect(c.unitSpritesById.has(7)).toBe(false);
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
