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

// Depth (z-order) bands — explicit rather than relying on Phaser's creation-order default,
// since spec003 introduces overlays/panels/dialogs that must render above the board contents.
export const DEPTH_GRID = 0;
export const DEPTH_OCCUPANT = 10;
export const DEPTH_ALLOWED_TILE_OVERLAY = 20;
export const DEPTH_TURN_COMMAND_PANEL = 30;
export const DEPTH_CONFIRM_DIALOG = 40;

// Phase 3.3's first UI text — single font referenced everywhere via this constant so it can
// be swapped later without touching call sites. Loaded via a <link> tag in index.html.
export const GAME_FONT_FAMILY = "'Roboto', sans-serif";

export const DISABLED_BUTTON_COLOR = 0x999999;

export const PANEL_BUTTON_FILL_COLOR = 0x583f0e;
export const PANEL_BUTTON_FILL_ALPHA = 0.2;
export const PANEL_BUTTON_BORDER_COLOR = 0xdc9e23;
export const PANEL_BUTTON_WIDTH = 46;
export const PANEL_BUTTON_HEIGHT = 32;
export const PANEL_BUTTON_BORDER_WIDTH = 2;
export const PANEL_BUTTON_SPACING = 12;

export const TURN_COMMAND_PANEL_WIDTH = 192;
export const TURN_COMMAND_PANEL_HEIGHT = 144;
export const TURN_COMMAND_PANEL_GUTTER = 16;

export const ALLOWED_TILE_MOVE_COLOR = 0x86c64f;
export const ALLOWED_TILE_MOVE_ALPHA = 0.65;
export const ALLOWED_TILE_MOVE_SELECTED_COLOR = 0xdaedca;

export const ALLOWED_TILE_BOMB_COLOR = 0xe69138;
export const ALLOWED_TILE_BOMB_ALPHA = 0.65;
export const ALLOWED_TILE_BOMB_SELECTED_COLOR = 0xf7dec3;

export const CONFIRM_DIALOG_WIDTH = 160;
export const CONFIRM_DIALOG_HEIGHT = 100;
export const CONFIRM_DIALOG_DIM_COLOR = 0x1a1a1a;
export const CONFIRM_DIALOG_DIM_ALPHA = 0.6;

export const UNIT_MOVE_TWEEN_DURATION = 500;
