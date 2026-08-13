import Phaser from 'phaser';
import { destroyAll, colorToCss } from '../ui/gameObjectUtils';
import { BOMB_COUNTDOWN_TEXT_COLOR } from './constants';
import { firstNonBaseFrame } from './spriteFrames';
import { boardOffset, setBoardOffset } from './boardOffset';
import type { BombGraphics } from './resolveTurnPlayer';
import {
  TILE_SIZE,
  SPRITE_GROUND_MARGIN,
  TERRAIN_COLORS,
  TERRAIN_BORDER_COLOR,
  DEPTH_GRID,
  DEPTH_OCCUPANT,
  UNIT_SIZE,
  OCCUPANT_STROKE_COLOR,
  OCCUPANT_ICON_RADIUS,
  OCCUPANT_ICON_STROKE_WIDTH,
} from '../constants';
import type { Bomb, Coordinate, GameState, SoftBlock, Tile, Unit } from '../types/api';

// Static board rendering: draws grid/units/softBlocks/bombs and tracks each one's Sprite/Graphics
// in the shared per-occupant maps. terrainObjects and occupantObjects have separate lifetimes so
// an occupant swap (renderOccupants) never touches the terrain layer.
export interface BoardRenderContext {
  scene: Phaser.Scene;
  terrainObjects: Phaser.GameObjects.GameObject[];
  occupantObjects: Phaser.GameObjects.GameObject[];
  unitSpritesById: Map<number, Phaser.GameObjects.Sprite>;
  bombGraphicsById: Map<number, BombGraphics>;
  softBlockSpritesById: Map<number, Phaser.GameObjects.Sprite>;
  onUnitClicked: (unit: Unit) => void;
}

// Archetype + team -> texture key
const UNIT_TEXTURE_KEYS: Record<string, Record<number, string>> = {
  Fighter: { 1: 'unit_fighter_blue', 2: 'unit_fighter_red' },
  King: { 1: 'unit_king_blue', 2: 'unit_king_red' },
  Bandit: { 1: 'unit_bandit_blue', 2: 'unit_bandit_red' },
  Witch: { 1: 'unit_witch_blue', 2: 'unit_witch_red' },
};

// Paints the immutable terrain (grid) layer; returns its dimensions for sizing dependent UI.
// Idempotent — destroys any prior terrain first so scene re-entry doesn't leak graphics.
export function renderTerrain(
  ctx: BoardRenderContext,
  grid: Tile[][]
): { cols: number; rows: number } {
  destroyAll(ctx.terrainObjects);
  const cols = grid[0]?.length ?? 0;
  const rows = grid.length;
  setBoardOffset(cols, rows);
  renderGrid(ctx, grid);
  return { cols, rows };
}

// Destroy-and-rebuild the occupant layer (units, softBlocks, bombs) from truth. This is the
// wholesale occupant swap; it leaves the terrain layer untouched.
export function renderOccupants(ctx: BoardRenderContext, state: GameState): void {
  destroyAll(ctx.occupantObjects);
  ctx.unitSpritesById.clear();
  ctx.bombGraphicsById.clear();
  ctx.softBlockSpritesById.clear();

  renderUnits(ctx, state.units);
  renderSoftBlocks(ctx, state.softBlocks);
  state.bombs.forEach(bomb => renderBomb(ctx, bomb));
}

function renderGrid(ctx: BoardRenderContext, grid: Tile[][]): void {
  const g = ctx.scene.add.graphics();
  g.setDepth(DEPTH_GRID);
  ctx.terrainObjects.push(g);
  g.lineStyle(1, TERRAIN_BORDER_COLOR);
  for (let row = 0; row < grid.length; row++) {
    const rowTiles = grid[row];
    if (!rowTiles) {
      continue;
    }
    for (let col = 0; col < rowTiles.length; col++) {
      const tile = rowTiles[col];
      if (!tile) {
        continue;
      }
      const x = col * TILE_SIZE + boardOffset.x;
      const y = row * TILE_SIZE + boardOffset.y;
      g.fillStyle(TERRAIN_COLORS[tile.type]);
      g.fillRect(x, y, TILE_SIZE, TILE_SIZE);
      g.strokeRect(x, y, TILE_SIZE, TILE_SIZE);
    }
  }
}

// Ground-anchored world Y for a sprite on the given tile: the tile's bottom edge, plus the
// fixed bottom-band every untrimmed 64x64 sprite canvas reserves (SPRITE_GROUND_MARGIN).
function groundY(position: Coordinate): number {
  return (position.y + 1) * TILE_SIZE + SPRITE_GROUND_MARGIN + boardOffset.y;
}

// occupantDepth returns a small per-row offset so occupants sort visually by row within the
// DEPTH_OCCUPANT band. Exported for movement tweens and sudden-death bomb drops that re-anchor
// an occupant's depth after initial render.
export function occupantDepth(position: Coordinate): number {
  return DEPTH_OCCUPANT + position.y * 0.01;
}

function renderUnits(ctx: BoardRenderContext, units: Unit[]): void {
  units
    .filter(unit => unit.hp > 0)
    .forEach(unit => {
      const { cx } = tileCenter(unit.position);
      const textureKey = UNIT_TEXTURE_KEYS[unit.type]?.[unit.team];
      if (!textureKey) {
        console.warn(
          `Unit ${unit.id} has no sprite for archetype "${unit.type}"/team ${unit.team}, falling back to blue`
        );
      }
      const key = textureKey ?? UNIT_TEXTURE_KEYS[unit.type]?.[1] ?? 'unit_fighter_blue';
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

// Renders a single bomb; used both by renderOccupants and by MatchScene's optimistic bomb placement.
export function renderBomb(ctx: BoardRenderContext, bomb: Bomb): void {
  const { cx, cy } = tileCenter(bomb.position);
  const frame = firstNonBaseFrame(ctx.scene, 'bomb');
  // Container sits at tile center, so the glyph's local Y must reach down to the tile's bottom
  // edge (+TILE_SIZE/2) before adding the ground margin, mirroring groundY()'s world-space math.
  const glyph = ctx.scene.add.sprite(0, TILE_SIZE / 2 + SPRITE_GROUND_MARGIN, 'bomb', frame);
  glyph.setOrigin(0.5, 1);
  // Countdown text is added last, so it renders on top of the bomb sprite within the container.
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

// Containers use local origin (0,0), not world space.
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

// Makes a sprite clickable over just its tile footprint, not the full untrimmed sprite canvas.
// Hit-area shapes are in the sprite's LOCAL space (origin-relative), not world/tile coordinates
// — must be derived from sprite.width/height, not the occupant's grid position.
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

// Paints a unit sprite (team-colored square + archetype icon) into a caller-owned Graphics.
// Kept for MatchSettingsScene's UnitPage, which still uses vector unit icons.
export function drawUnitSprite(
  g: Phaser.GameObjects.Graphics,
  cx: number,
  cy: number,
  size: number,
  archetype: string,
  teamColor: number,
  cornerRadius = 0
): void {
  g.fillStyle(teamColor);
  if (cornerRadius > 0) {
    g.fillRoundedRect(cx - size / 2, cy - size / 2, size, size, cornerRadius);
  } else {
    g.fillRect(cx - size / 2, cy - size / 2, size, size);
  }
  drawArchetypeIcon(g, archetype, cx, cy, (size * OCCUPANT_ICON_RADIUS) / UNIT_SIZE);
}

export function drawArchetypeIcon(
  g: Phaser.GameObjects.Graphics,
  archetype: string,
  cx: number,
  cy: number,
  radius: number = OCCUPANT_ICON_RADIUS
): void {
  g.lineStyle(OCCUPANT_ICON_STROKE_WIDTH, OCCUPANT_STROKE_COLOR);
  switch (archetype) {
    case 'Bandit':
      g.strokeCircle(cx, cy, radius);
      break;
    case 'Witch':
      g.strokePoints(regularPolygonPoints(cx, cy, 3, radius), true);
      break;
    case 'Fighter':
      g.strokePoints(regularPolygonPoints(cx, cy, 5, radius), true);
      break;
    case 'King':
      g.strokePoints(starPoints(cx, cy, 5, radius, radius * 0.4), true);
      break;
    default:
      console.warn(`Unrecognized archetype "${archetype}", drawing no icon`);
  }
}

// Vertices of a regular polygon centered at (cx, cy), first vertex pointing straight up.
function regularPolygonPoints(
  cx: number,
  cy: number,
  sides: number,
  radius: number
): Phaser.Math.Vector2[] {
  return Array.from({ length: sides }, (_, i) => {
    const angle = -Math.PI / 2 + (i * 2 * Math.PI) / sides;
    return new Phaser.Math.Vector2(cx + radius * Math.cos(angle), cy + radius * Math.sin(angle));
  });
}

// Vertices of a 5-pointed star centered at (cx, cy), alternating outer/inner radius,
// first vertex pointing straight up.
function starPoints(
  cx: number,
  cy: number,
  points: number,
  outerRadius: number,
  innerRadius: number
): Phaser.Math.Vector2[] {
  const vertices: Phaser.Math.Vector2[] = [];
  for (let i = 0; i < points * 2; i++) {
    const radius = i % 2 === 0 ? outerRadius : innerRadius;
    const angle = -Math.PI / 2 + (i * Math.PI) / points;
    vertices.push(
      new Phaser.Math.Vector2(cx + radius * Math.cos(angle), cy + radius * Math.sin(angle))
    );
  }
  return vertices;
}
