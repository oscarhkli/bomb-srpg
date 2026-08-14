import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import { occupantSprite, stageBackgroundImage, pointerDownOf } from '../test/sceneHelpers';
import {
  plainTile,
  makeUnit as unit,
  makeSoftBlock as softBlock,
  makeBomb as bomb,
} from '../test/fixtures';
import { TILE_SIZE, SPRITE_GROUND_MARGIN, DEPTH_STAGE_BACKGROUND } from '../constants';
import type { Bomb, GameState, SoftBlock, Tile, Unit } from '../types/api';
import type { BombGraphics } from './resolveTurnPlayer';
import {
  establishBoardLayout,
  renderStageBackground,
  renderOccupants,
  renderBomb,
  tileCenter,
  resolveUnitTextureKey,
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

// Entry-order paint: board layout, then occupants.
function renderAll(c: BoardRenderContext, s: GameState): void {
  establishBoardLayout(s.grid);
  renderOccupants(c, s);
}

describe('tileCenter', () => {
  it('returns the pixel center of a tile', () => {
    expect(tileCenter({ x: 1, y: 0 })).toEqual({ cx: 48, cy: 16 });
  });
});

describe('establishBoardLayout', () => {
  it('returns the grid dimensions', () => {
    const dims = establishBoardLayout([
      [plainTile(), plainTile(), plainTile()],
      [plainTile(), plainTile(), plainTile()],
    ]);
    expect(dims).toEqual({ cols: 3, rows: 2 });
  });
});

describe('renderStageBackground', () => {
  it('renders the texture key matching the stage preset name', () => {
    establishBoardLayout([[plainTile()]]);
    renderStageBackground(ctx(), [[plainTile()]], 'Standard');

    expect(mockScene.add.image).toHaveBeenLastCalledWith(
      expect.any(Number),
      expect.any(Number),
      'stage_standard',
      expect.any(String)
    );
  });

  it('centers on the grid regardless of grid size, using boardOffset', () => {
    const c = ctx();
    // 3x2 grid, centered in GameBoardRegion (offsetX=192, offsetY=148).
    establishBoardLayout([
      [plainTile(), plainTile(), plainTile()],
      [plainTile(), plainTile(), plainTile()],
    ]);
    renderStageBackground(
      c,
      [
        [plainTile(), plainTile(), plainTile()],
        [plainTile(), plainTile(), plainTile()],
      ],
      'Plain'
    );

    const cx = 192 + (3 * TILE_SIZE) / 2;
    const cy = 148 + (2 * TILE_SIZE) / 2;
    expect(mockScene.add.image).toHaveBeenLastCalledWith(cx, cy, 'stage_plain', expect.any(String));
  });

  it('sets origin to the frame center and depth DEPTH_STAGE_BACKGROUND when the trim is symmetric', () => {
    renderStageBackground(ctx(), [[plainTile()]], 'Plain');

    const bg = stageBackgroundImage();
    expect(bg.setOrigin).toHaveBeenCalledWith(0.5, 0.5);
    expect(bg.setDepth).toHaveBeenCalledWith(DEPTH_STAGE_BACKGROUND);
  });

  it('computes origin from the trimmed content center, not the untrimmed canvas center', () => {
    // Actual Stage-Plain.json trim data: a 320x320 visible crop offset (64, 64) within a
    // 640x640 untrimmed canvas — not centered, so origin must not default to (0.5, 0.5).
    mockScene.textures.getFrame.mockReturnValueOnce({
      x: 64,
      y: 64,
      width: 320,
      height: 320,
      realWidth: 640,
      realHeight: 640,
    });

    renderStageBackground(ctx(), [[plainTile()]], 'Plain');

    const bg = stageBackgroundImage();
    expect(bg.setOrigin).toHaveBeenCalledWith(0.35, 0.35);
  });

  it('tracks the background image in terrainObjects, not occupantObjects', () => {
    const c = ctx();
    renderStageBackground(c, [[plainTile()]], 'Plain');

    expect(c.terrainObjects).toHaveLength(1);
    expect(c.terrainObjects[0]).toBe(stageBackgroundImage());
    expect(c.occupantObjects).toHaveLength(0);
  });

  it('destroys the prior background on a repeat entry so re-running create() does not stack', () => {
    const c = ctx();
    renderStageBackground(c, [[plainTile()]], 'Plain');
    const firstBg = stageBackgroundImage();

    renderStageBackground(c, [[plainTile()]], 'Plain');

    expect(firstBg.destroy).toHaveBeenCalled();
    expect(c.terrainObjects).toHaveLength(1);
  });

  it('warns and skips rendering for an unrecognized stage preset name', () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => undefined);
    const c = ctx();

    renderStageBackground(c, [[plainTile()]], 'Nonexistent');

    expect(mockScene.add.image).not.toHaveBeenCalled();
    expect(c.terrainObjects).toHaveLength(0);
    expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining('Nonexistent'));
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

describe('resolveUnitTextureKey', () => {
  it('resolves the texture key matching archetype + team', () => {
    expect(resolveUnitTextureKey('Bandit', 2)).toBe('unit_bandit_red');
  });

  it('warns and falls back to the blue texture for an unconfigured team', () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => undefined);

    expect(resolveUnitTextureKey('Bandit', 99)).toBe('unit_bandit_blue');

    expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining('99'));
    warnSpy.mockRestore();
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
    renderStageBackground(c, [[plainTile()]], 'Plain');
    const bg = stageBackgroundImage();
    renderOccupants(c, state([[plainTile()]], { units: [unit({ id: 7 })] }));

    // Swap the occupants again — the background must survive both rebuilds.
    renderOccupants(c, state([[plainTile()]]));

    expect(bg.destroy).not.toHaveBeenCalled();
    expect(c.terrainObjects).toEqual([bg]);
  });
});
