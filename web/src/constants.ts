import type { TerrainType } from './types/api'

export const TILE_SIZE = 48

export const TERRAIN_COLORS: Record<TerrainType, number> = {
  TerrainPlain: 0x4caf50, // green
  TerrainBlock: 0x9e9e9e, // grey
  TerrainTower: 0x795548, // brown
  TerrainWater: 0x2196f3, // blue
  TerrainLava: 0xff9800, // orange
}

export const TERRAIN_BORDER_COLOR = 0x000000
