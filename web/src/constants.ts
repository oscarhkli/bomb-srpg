import type { TerrainType } from './types/api';

export const TILE_SIZE = 48;

export const TERRAIN_COLORS: Record<TerrainType, number> = {
  TerrainPlain: 0x4caf50, // green
  TerrainBlock: 0x9e9e9e, // grey
  TerrainTower: 0x795548, // brown
  TerrainWater: 0x2196f3, // blue
  TerrainLava: 0xff9800, // orange
};

export const TERRAIN_BORDER_COLOR = 0x000000;

export const UNIT_SIZE = 32;
export const SOFTBLOCK_SIZE = 42;
export const BOMB_SIZE = 24;

export const TEAM_COLORS: Record<number, number> = {
  1: 0x212df3, // blue
  2: 0xf32d21, // red
};

export const SOFTBLOCK_COLOR = 0xe6e6e6;
export const SOFTBLOCK_CORNER_RADIUS = 4;
export const BOMB_COLOR = 0x222222;
export const OCCUPANT_STROKE_COLOR = 0xffffff;
export const OCCUPANT_ICON_RADIUS = 10;
export const OCCUPANT_ICON_STROKE_WIDTH = 2;
