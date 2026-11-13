import Phaser from 'phaser';
import { destroyAll, colorToCss } from '../ui/gameObjectUtils';
import { BOMB_COUNTDOWN_TEXT_COLOR } from './constants';
import { firstNonBaseFrame } from './spriteFrames';
import { boardOffset, setBoardOffset } from './boardOffset';
import type { BombGraphics } from './resolveTurnPlayer';
import {
  TILE_SIZE,
  SPRITE_GROUND_MARGIN,
  DEPTH_STAGE_BACKGROUND,
  DEPTH_OCCUPANT,
} from '../constants';
import type { Bomb, Coordinate, GameState, SoftBlock, Tile, Unit } from '../api/types';

export interface BoardRenderContext {
  scene: Phaser.Scene;
  terrainObjects: Phaser.GameObjects.GameObject[];
  occupantObjects: Phaser.GameObjects.GameObject[];
  unitSpritesById: Map<number, Phaser.GameObjects.Sprite>;
  bombGraphicsById: Map<number, BombGraphics>;
  softBlockSpritesById: Map<number, Phaser.GameObjects.Sprite>;
  onUnitClicked: (unit: Unit) => void;
}

const UNIT_TEXTURE_KEYS: Record<string, Record<number, string>> = {
  Fighter: { 1: 'unit_fighter_blue', 2: 'unit_fighter_red' },
  King: { 1: 'unit_king_blue', 2: 'unit_king_red' },
  Bandit: { 1: 'unit_bandit_blue', 2: 'unit_bandit_red' },
  Witch: { 1: 'unit_witch_blue', 2: 'unit_witch_red' },
  Prologue: { 2: 'unit_prologue_red' },
};

const STAGE_TEXTURE_KEYS: Record<string, string> = {
  Plain: 'stage_plain',
  Standard: 'stage_standard',
  Divided: 'stage_divided',
};

// Must run before renderStageBackground().
export function establishBoardLayout(grid: Tile[][]): { cols: number; rows: number } {
  const cols = grid[0]?.length ?? 0;
  const rows = grid.length;
  setBoardOffset(cols, rows);
  return { cols, rows };
}

function trimmedContentOrigin(scene: Phaser.Scene, key: string, frame: string): [number, number] {
  const f = scene.textures.getFrame(key, frame);
  return [(f.x + f.width / 2) / f.realWidth, (f.y + f.height / 2) / f.realHeight];
}

// Requires establishBoardLayout() to have run for this grid.
export function renderStageBackground(
  ctx: BoardRenderContext,
  grid: Tile[][],
  stagePresetName: string
): void {
  destroyAll(ctx.terrainObjects);
  const cols = grid[0]?.length ?? 0;
  const rows = grid.length;
  const key = STAGE_TEXTURE_KEYS[stagePresetName];
  if (!key) {
    console.warn(`No Stage background texture for preset "${stagePresetName}"`);
    return;
  }
  const frame = firstNonBaseFrame(ctx.scene, key);
  const cx = boardOffset.x + (cols * TILE_SIZE) / 2;
  const cy = boardOffset.y + (rows * TILE_SIZE) / 2;
  const bg = ctx.scene.add.image(cx, cy, key, frame);
  bg.setOrigin(...trimmedContentOrigin(ctx.scene, key, frame));
  bg.setDepth(DEPTH_STAGE_BACKGROUND);
  ctx.terrainObjects.push(bg);
}

export function hazardsUnplayed(turn: number, hazardsPlayedTurn: number): boolean {
  return turn > hazardsPlayedTurn;
}

export function bombsToDraw(state: GameState, hazardsPlayedTurn: number): Bomb[] {
  if (!hazardsUnplayed(state.turn, hazardsPlayedTurn)) {
    return state.bombs;
  }
  const hidden = new Set(
    state.turnStartEvents.filter(evt => evt.type === 'bombPlaced').map(evt => evt.bombId)
  );
  return state.bombs.filter(bomb => !hidden.has(bomb.id));
}

export function renderOccupants(ctx: BoardRenderContext, state: GameState): void {
  destroyAll(ctx.occupantObjects);
  ctx.unitSpritesById.clear();
  ctx.bombGraphicsById.clear();
  ctx.softBlockSpritesById.clear();

  renderUnits(ctx, state.units);
  renderSoftBlocks(ctx, state.softBlocks);
  state.bombs.forEach(bomb => renderBomb(ctx, bomb));
}

function groundY(position: Coordinate): number {
  return (position.y + 1) * TILE_SIZE + SPRITE_GROUND_MARGIN + boardOffset.y;
}

export function occupantDepth(position: Coordinate): number {
  return DEPTH_OCCUPANT + position.y * 0.01;
}

export function resolveUnitTextureKey(archetype: string, team: number, context = ''): string {
  const textureKey = UNIT_TEXTURE_KEYS[archetype]?.[team];
  if (!textureKey) {
    console.warn(
      `${context}No sprite for archetype "${archetype}"/team ${team}, falling back to blue`
    );
  }
  return textureKey ?? UNIT_TEXTURE_KEYS[archetype]?.[1] ?? 'unit_fighter_blue';
}

function renderUnits(ctx: BoardRenderContext, units: Unit[]): void {
  units
    .filter(unit => unit.hp > 0)
    .forEach(unit => {
      const { cx } = tileCenter(unit.position);
      const key = resolveUnitTextureKey(unit.type, unit.team, `Unit ${unit.id} `);
      const frame = firstNonBaseFrame(ctx.scene, key);
      const sprite = ctx.scene.add.sprite(cx, groundY(unit.position), key, frame);
      sprite.setOrigin(0.5, 1);
      sprite.setDepth(occupantDepth(unit.position));
      ctx.occupantObjects.push(sprite);
      ctx.unitSpritesById.set(unit.id, sprite);
      attachUnitClickHandler(ctx, sprite, unit);
    });
}

function attachUnitClickHandler(
  ctx: BoardRenderContext,
  sprite: Phaser.GameObjects.Sprite,
  unit: Unit
): void {
  setTileHitArea(sprite);
  sprite.on('pointerdown', () => ctx.onUnitClicked(unit));
}

function renderSoftBlocks(ctx: BoardRenderContext, softBlocks: SoftBlock[]): void {
  softBlocks.forEach(block => {
    const { cx } = tileCenter(block.position);
    const frame = firstNonBaseFrame(ctx.scene, 'soft_block');
    const sprite = ctx.scene.add.sprite(cx, groundY(block.position), 'soft_block', frame);
    sprite.setOrigin(0.5, 1);
    sprite.setDepth(occupantDepth(block.position));
    ctx.occupantObjects.push(sprite);
    attachClickLogger(sprite, `SoftBlock ${block.id}`, block);
    ctx.softBlockSpritesById.set(block.id, sprite);
  });
}

export function renderBomb(ctx: BoardRenderContext, bomb: Bomb): void {
  const { cx, cy } = tileCenter(bomb.position);
  const frame = firstNonBaseFrame(ctx.scene, 'bomb');
  // The container sits at tile center, so reach down to the tile's bottom edge first.
  const glyph = ctx.scene.add.sprite(0, TILE_SIZE / 2 + SPRITE_GROUND_MARGIN, 'bomb', frame);
  glyph.setOrigin(0.5, 1);
  const text = ctx.scene.add.text(0, 0, String(bomb.countdown), {
    color: colorToCss(BOMB_COUNTDOWN_TEXT_COLOR),
  });
  text.setOrigin(0.5);

  const container = ctx.scene.add.container(cx, cy, [glyph, text]);
  container.setDepth(occupantDepth(bomb.position));
  ctx.occupantObjects.push(container);

  attachContainerClickLogger(container, `Bomb ${bomb.id}`, bomb);
  ctx.bombGraphicsById.set(bomb.id, { container, countdownText: text });
}

function attachContainerClickLogger(
  container: Phaser.GameObjects.Container,
  label: string,
  details: unknown
): void {
  const hitArea = new Phaser.Geom.Rectangle(-TILE_SIZE / 2, -TILE_SIZE / 2, TILE_SIZE, TILE_SIZE);
  container.setInteractive(hitArea, (shape: Phaser.Geom.Rectangle, x: number, y: number) =>
    Phaser.Geom.Rectangle.Contains(shape, x, y)
  );
  container.on('pointerdown', () => console.log(`${label} is clicked`, details));
}

function attachClickLogger(
  sprite: Phaser.GameObjects.Sprite,
  label: string,
  details: unknown
): void {
  setTileHitArea(sprite);
  sprite.on('pointerdown', () => console.log(`${label} is clicked`, details));
}

// Clickable over just the tile footprint. Hit areas are in the sprite's local space.
function setTileHitArea(sprite: Phaser.GameObjects.Sprite): void {
  const localX = sprite.width / 2 - TILE_SIZE / 2;
  const localY = sprite.height - SPRITE_GROUND_MARGIN - TILE_SIZE;
  const hitArea = new Phaser.Geom.Rectangle(localX, localY, TILE_SIZE, TILE_SIZE);
  sprite.setInteractive(hitArea, (shape: Phaser.Geom.Rectangle, x: number, y: number) =>
    Phaser.Geom.Rectangle.Contains(shape, x, y)
  );
}

export function tileCenter(position: Coordinate): { cx: number; cy: number } {
  return {
    cx: position.x * TILE_SIZE + TILE_SIZE / 2 + boardOffset.x,
    cy: position.y * TILE_SIZE + TILE_SIZE / 2 + boardOffset.y,
  };
}
