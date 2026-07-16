import Phaser from 'phaser';
import { destroyAll, colorToCss } from '../ui/gameObjectUtils';
import { BOMB_COUNTDOWN_TEXT_COLOR } from './constants';
import type { BombGraphics } from './resolveTurnPlayer';
import {
  TILE_SIZE,
  TERRAIN_COLORS,
  TERRAIN_BORDER_COLOR,
  TEAM_COLOR_FALLBACK,
  TEAM_COLORS,
  UNIT_SIZE,
  OCCUPANT_STROKE_COLOR,
  OCCUPANT_ICON_RADIUS,
  OCCUPANT_ICON_STROKE_WIDTH,
  SOFTBLOCK_COLOR,
  SOFTBLOCK_SIZE,
  SOFTBLOCK_CORNER_RADIUS,
  BOMB_COLOR,
  BOMB_SIZE,
  DEPTH_GRID,
  DEPTH_OCCUPANT,
} from '../constants';
import type { Bomb, Coordinate, GameState, SoftBlock, Tile, Unit } from '../types/api';

// Static board rendering: draws the grid, units, softBlocks, and bombs into a scene and tracks
// the created graphics in the shared per-occupant maps. This is the "initial/static" twin of
// resolveTurnPlayer's animated playback — the maps are the shared contract between them.
// The scene owns the state and the click semantics; the renderer only wires each occupant's
// pointerdown to the provided callbacks.
//
// Two layers, two lifetimes: `terrainObjects` holds the immutable grid, painted once per scene
// entry (renderTerrain) and never swapped; `occupantObjects` holds units/softBlocks/bombs, which
// renderOccupants destroys-and-rebuilds on a wholesale swap. Keeping them apart lets an occupant
// swap (error recovery, future Reset) leave the terrain untouched.
export interface BoardRenderContext {
  scene: Phaser.Scene;
  terrainObjects: Phaser.GameObjects.GameObject[];
  occupantObjects: Phaser.GameObjects.GameObject[];
  unitGraphicsById: Map<number, Phaser.GameObjects.Graphics>;
  bombGraphicsById: Map<number, BombGraphics>;
  softBlockGraphicsById: Map<number, Phaser.GameObjects.Graphics>;
  onUnitClicked: (unit: Unit) => void;
}

// Paints the immutable terrain (grid) layer and returns the grid dimensions so the caller can
// size dependent UI (grid bounds, camera centering) from a single source. Called once per scene
// entry. Idempotent: destroys any prior terrain first, so a scene re-entry (rematch) that re-runs
// create() replaces the grid instead of leaking stale graphics.
export function renderTerrain(
  ctx: BoardRenderContext,
  grid: Tile[][]
): { cols: number; rows: number } {
  destroyAll(ctx.terrainObjects);
  renderGrid(ctx, grid);
  return { cols: grid[0]?.length ?? 0, rows: grid.length };
}

// Destroy-and-rebuild the occupant layer (units, softBlocks, bombs) from truth. This is the
// wholesale occupant swap; it leaves the terrain layer untouched.
export function renderOccupants(ctx: BoardRenderContext, state: GameState): void {
  destroyAll(ctx.occupantObjects);
  ctx.unitGraphicsById.clear();
  ctx.bombGraphicsById.clear();
  ctx.softBlockGraphicsById.clear();

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
      const x = col * TILE_SIZE;
      const y = row * TILE_SIZE;
      g.fillStyle(TERRAIN_COLORS[tile.type]);
      g.fillRect(x, y, TILE_SIZE, TILE_SIZE);
      g.strokeRect(x, y, TILE_SIZE, TILE_SIZE);
    }
  }
}

function renderUnits(ctx: BoardRenderContext, units: Unit[]): void {
  units
    .filter(unit => unit.hp > 0)
    .forEach(unit => {
      const { cx, cy } = tileCenter(unit.position);
      const g = ctx.scene.add.graphics();
      g.setDepth(DEPTH_OCCUPANT);
      ctx.occupantObjects.push(g);
      ctx.unitGraphicsById.set(unit.id, g);
      const teamColor = TEAM_COLORS[unit.team];
      if (teamColor === undefined) {
        console.warn(
          `Unit ${unit.id} has unconfigured team ${unit.team}, rendering as fallback grey`
        );
      }
      g.fillStyle(teamColor ?? TEAM_COLOR_FALLBACK);
      g.fillRect(cx - UNIT_SIZE / 2, cy - UNIT_SIZE / 2, UNIT_SIZE, UNIT_SIZE);
      drawArchetypeIcon(g, unit.type, cx, cy);
      attachUnitClickHandler(ctx, g, unit);
    });
}

function attachUnitClickHandler(
  ctx: BoardRenderContext,
  g: Phaser.GameObjects.Graphics,
  unit: Unit
): void {
  setTileHitArea(g, unit.position);
  g.on('pointerdown', () => ctx.onUnitClicked(unit));
}

function renderSoftBlocks(ctx: BoardRenderContext, softBlocks: SoftBlock[]): void {
  softBlocks.forEach(block => {
    const { cx, cy } = tileCenter(block.position);
    const g = ctx.scene.add.graphics();
    g.setDepth(DEPTH_OCCUPANT);
    ctx.occupantObjects.push(g);
    g.fillStyle(SOFTBLOCK_COLOR);
    g.fillRoundedRect(
      cx - SOFTBLOCK_SIZE / 2,
      cy - SOFTBLOCK_SIZE / 2,
      SOFTBLOCK_SIZE,
      SOFTBLOCK_SIZE,
      SOFTBLOCK_CORNER_RADIUS
    );
    attachClickLogger(g, block.position, `SoftBlock ${block.id}`, block);
    ctx.softBlockGraphicsById.set(block.id, g);
  });
}

// Renders a single bomb; used both by renderOccupants and by MatchScene's optimistic bomb placement.
// The circle and countdown text are drawn at local (0,0) and parented in a Container placed at
// the tile center, so the bomb is one positionable/tweenable unit (see dropSuddenDeathBomb in
// MatchScene) instead of two independently-positioned objects that can drift apart.
export function renderBomb(ctx: BoardRenderContext, bomb: Bomb): void {
  const { cx, cy } = tileCenter(bomb.position);
  const g = ctx.scene.add.graphics();
  g.fillStyle(BOMB_COLOR);
  g.fillCircle(0, 0, BOMB_SIZE / 2);
  const text = ctx.scene.add.text(0, 0, String(bomb.countdown), {
    color: colorToCss(BOMB_COUNTDOWN_TEXT_COLOR),
  });
  text.setOrigin(0.5);

  const container = ctx.scene.add.container(cx, cy, [g, text]);
  container.setDepth(DEPTH_OCCUPANT);
  ctx.occupantObjects.push(container);

  attachContainerClickLogger(container, `Bomb ${bomb.id}`, bomb);
  ctx.bombGraphicsById.set(bomb.id, { container, circle: g, countdownText: text });
}

// Containers always have local origin (0,0) — unlike attachClickLogger's world-space tile rect
// (valid only because the ungrouped Graphics it targets never moves from (0,0)), the hit area
// here must be centered on the container's own origin instead.
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
  g: Phaser.GameObjects.Graphics,
  position: Coordinate,
  label: string,
  details: unknown
): void {
  setTileHitArea(g, position);
  g.on('pointerdown', () => console.log(`${label} is clicked`, details));
}

// Makes a graphics object clickable over the full 48x48 tile at the given grid position.
function setTileHitArea(g: Phaser.GameObjects.Graphics, position: Coordinate): void {
  const hitArea = new Phaser.Geom.Rectangle(
    position.x * TILE_SIZE,
    position.y * TILE_SIZE,
    TILE_SIZE,
    TILE_SIZE
  );
  g.setInteractive(hitArea, (shape: Phaser.Geom.Rectangle, x: number, y: number) =>
    Phaser.Geom.Rectangle.Contains(shape, x, y)
  );
}

function drawArchetypeIcon(
  g: Phaser.GameObjects.Graphics,
  archetype: string,
  cx: number,
  cy: number
): void {
  g.lineStyle(OCCUPANT_ICON_STROKE_WIDTH, OCCUPANT_STROKE_COLOR);
  switch (archetype) {
    case 'Bandit':
      g.strokeCircle(cx, cy, OCCUPANT_ICON_RADIUS);
      break;
    case 'Witch':
      g.strokePoints(regularPolygonPoints(cx, cy, 3, OCCUPANT_ICON_RADIUS), true);
      break;
    case 'Fighter':
      g.strokePoints(regularPolygonPoints(cx, cy, 5, OCCUPANT_ICON_RADIUS), true);
      break;
    case 'King':
      g.strokePoints(starPoints(cx, cy, 5, OCCUPANT_ICON_RADIUS, 4), true);
      break;
    default:
      console.warn(`Unrecognized archetype "${archetype}", drawing no icon`);
  }
}

export function tileCenter(position: Coordinate): { cx: number; cy: number } {
  return {
    cx: position.x * TILE_SIZE + TILE_SIZE / 2,
    cy: position.y * TILE_SIZE + TILE_SIZE / 2,
  };
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
