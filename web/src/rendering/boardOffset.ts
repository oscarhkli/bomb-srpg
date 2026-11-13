import { GAME_BOARD_REGION_WIDTH, GAME_BOARD_REGION_HEIGHT, TILE_SIZE } from '../constants';

export const boardOffset = { x: 0, y: 0 };

export function setBoardOffset(cols: number, rows: number): void {
  boardOffset.x = (GAME_BOARD_REGION_WIDTH - cols * TILE_SIZE) / 2;
  boardOffset.y = (GAME_BOARD_REGION_HEIGHT - rows * TILE_SIZE) / 2;
}
