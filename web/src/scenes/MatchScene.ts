import Phaser from 'phaser';
import { initRoom, getMatchState } from '../engine/api';
import { TILE_SIZE, TERRAIN_COLORS, TERRAIN_BORDER_COLOR } from '../constants';
import type { Tile } from '../types/api';

export interface MatchSceneData {
  roomId: string;
  playerTokens: [string, string];
}

export default class MatchScene extends Phaser.Scene {
  private roomId!: string;
  private playerTokens!: [string, string];
  private graphics!: Phaser.GameObjects.Graphics;

  constructor() {
    super('MatchScene');
  }

  create(data: MatchSceneData): void {
    this.roomId = data.roomId;
    this.playerTokens = data.playerTokens;
    initRoom(data.roomId);

    getMatchState()
      .then(state => {
        this.renderGrid(state.grid);
        this.centerCamera(state.grid);
      })
      .catch(() => {
        this.showError('Failed to load match state');
      });
  }

  private renderGrid(grid: Tile[][]): void {
    this.graphics = this.add.graphics();
    this.graphics.lineStyle(1, TERRAIN_BORDER_COLOR);
    for (let row = 0; row < grid.length; row++) {
      const rowTiles = grid[row];
      if (!rowTiles) continue;
      for (let col = 0; col < rowTiles.length; col++) {
        const tile = rowTiles[col];
        if (!tile) continue;
        const x = col * TILE_SIZE;
        const y = row * TILE_SIZE;
        this.graphics.fillStyle(TERRAIN_COLORS[tile.type]);
        this.graphics.fillRect(x, y, TILE_SIZE, TILE_SIZE);
        this.graphics.strokeRect(x, y, TILE_SIZE, TILE_SIZE);
      }
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
