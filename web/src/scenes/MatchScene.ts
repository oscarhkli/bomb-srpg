import Phaser from 'phaser';
import { initRoom, getMatchState } from '../engine/api';
import {
  TILE_SIZE,
  TERRAIN_COLORS,
  TERRAIN_BORDER_COLOR,
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
} from '../constants';
import type { Coordinate, Tile, Unit, SoftBlock, Bomb } from '../types/api';

export interface MatchSceneData {
  roomId: string;
  playerTokens: [string, string];
}

export default class MatchScene extends Phaser.Scene {
  private roomId!: string;
  private playerTokens!: [string, string];

  constructor() {
    super('MatchScene');
  }

  create(data: MatchSceneData): void {
    this.roomId = data.roomId;
    this.playerTokens = data.playerTokens;
    console.log('roomId:', this.roomId, 'playerTokens:', this.playerTokens);
    initRoom(data.roomId);

    getMatchState()
      .then(state => {
        this.renderGrid(state.grid);
        this.renderUnits(state.units);
        this.renderSoftBlocks(state.softBlocks);
        this.renderBombs(state.bombs);
        this.centerCamera(state.grid);
      })
      .catch(() => {
        this.showError('Failed to load match state');
      });
  }

  private renderGrid(grid: Tile[][]): void {
    const g = this.add.graphics();
    g.lineStyle(1, TERRAIN_BORDER_COLOR);
    for (let row = 0; row < grid.length; row++) {
      const rowTiles = grid[row];
      if (!rowTiles) continue;
      for (let col = 0; col < rowTiles.length; col++) {
        const tile = rowTiles[col];
        if (!tile) continue;
        const x = col * TILE_SIZE;
        const y = row * TILE_SIZE;
        g.fillStyle(TERRAIN_COLORS[tile.type]);
        g.fillRect(x, y, TILE_SIZE, TILE_SIZE);
        g.strokeRect(x, y, TILE_SIZE, TILE_SIZE);
      }
    }
  }

  private renderUnits(units: Unit[]): void {
    units
      .filter(unit => unit.hp > 0)
      .forEach(unit => {
        const { cx, cy } = tileCenter(unit.position);
        const g = this.add.graphics();
        const teamColor = TEAM_COLORS[unit.team];
        if (teamColor === undefined) {
          console.warn(`Unit ${unit.id} has unconfigured team ${unit.team}, rendering as white`);
        }
        g.fillStyle(teamColor ?? 0xffffff);
        g.fillRect(cx - UNIT_SIZE / 2, cy - UNIT_SIZE / 2, UNIT_SIZE, UNIT_SIZE);
        this.drawArchetypeIcon(g, unit.type, cx, cy);
        this.attachClickLogger(g, unit.position, `Unit ${unit.id}`, unit);
      });
  }

  private renderSoftBlocks(softBlocks: SoftBlock[]): void {
    softBlocks.forEach(block => {
      const { cx, cy } = tileCenter(block.position);
      const g = this.add.graphics();
      g.fillStyle(SOFTBLOCK_COLOR);
      g.fillRoundedRect(
        cx - SOFTBLOCK_SIZE / 2,
        cy - SOFTBLOCK_SIZE / 2,
        SOFTBLOCK_SIZE,
        SOFTBLOCK_SIZE,
        SOFTBLOCK_CORNER_RADIUS
      );
      this.attachClickLogger(g, block.position, `SoftBlock ${block.id}`, block);
    });
  }

  private renderBombs(bombs: Bomb[]): void {
    bombs.forEach(bomb => {
      const { cx, cy } = tileCenter(bomb.position);
      const g = this.add.graphics();
      g.fillStyle(BOMB_COLOR);
      g.fillCircle(cx, cy, BOMB_SIZE / 2);
      this.add.text(cx, cy, String(bomb.countdown), { color: '#ffffff' }).setOrigin(0.5);
      this.attachClickLogger(g, bomb.position, `Bomb ${bomb.id}`, bomb);
    });
  }

  private attachClickLogger(
    g: Phaser.GameObjects.Graphics,
    position: Coordinate,
    label: string,
    details: unknown
  ): void {
    const hitArea = new Phaser.Geom.Rectangle(
      position.x * TILE_SIZE,
      position.y * TILE_SIZE,
      TILE_SIZE,
      TILE_SIZE
    );
    g.setInteractive(hitArea, (shape: Phaser.Geom.Rectangle, x: number, y: number) =>
      Phaser.Geom.Rectangle.Contains(shape, x, y)
    );
    g.on('pointerdown', () => console.log(`${label} is clicked`, details));
  }

  private drawArchetypeIcon(
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

  private centerCamera(grid: Tile[][]): void {
    const cols = grid[0]?.length ?? 0;
    const rows = grid.length;
    this.cameras.main.centerOn((cols * TILE_SIZE) / 2, (rows * TILE_SIZE) / 2);
  }

  private showError(message: string): void {
    const { width, height } = this.cameras.main;
    this.add.text(width / 2, height / 2, message).setOrigin(0.5);
  }
}

function tileCenter(position: Coordinate): { cx: number; cy: number } {
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
