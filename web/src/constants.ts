export const TEAM_COLOR_FALLBACK = 0x4c4c4c;
export const FADE_MS = 200;

export const TILE_SIZE = 32;

// Bottom-padding band every sprite's untrimmed 64x64 canvas reserves below the tile edge.
export const SPRITE_GROUND_MARGIN = 16;

export const GAME_BOARD_REGION_WIDTH = 480;
export const GAME_BOARD_REGION_HEIGHT = 360;
export const GAME_CONTROL_REGION_X = GAME_BOARD_REGION_WIDTH;
export const GAME_CONTROL_REGION_WIDTH = 160;
export const GAME_CONTROL_REGION_HEIGHT = 360;

export const BOMB_SIZE = 24;

export const CANVAS_ZOOM_BREAKPOINT_WIDTH = 1360;
export const CANVAS_ZOOM_BREAKPOINT_HEIGHT = 848;

export const TEAM_COLORS: Record<number, number> = {
  1: 0x212df3,
  2: 0xf32d21,
};

export const DEPTH_STAGE_BACKGROUND = 0;
export const DEPTH_OCCUPANT = 10;
export const DEPTH_ALLOWED_TILE_OVERLAY = 20;
export const DEPTH_TURN_COMMAND_PANEL = 30;
export const DEPTH_CONFIRM_DIALOG = 40;
export const DEPTH_ERROR_PANEL = 50;

export const GAME_FONT_FAMILY = "'Roboto', sans-serif";

export const DISABLED_BUTTON_COLOR = 0x999999;

export const PANEL_BUTTON_FILL_COLOR = 0x583f0e;
export const PANEL_BUTTON_FILL_ALPHA = 0.2;
export const PANEL_BUTTON_BORDER_COLOR = 0xdc9e23;
export const PANEL_BUTTON_BORDER_WIDTH = 2;
export const PANEL_BUTTON_SPACING = 12;

export const BUTTON_TEXTURE_NEUTRAL = 'button_neutral';
export const BUTTON_TEXTURE_SELECTED = 'button_selected';
export const BUTTON_TEXTURE_CLICKED = 'button_clicked';
export const BUTTON_LABEL_MOVE = 'button_label_move';
export const BUTTON_LABEL_BOMB = 'button_label_bomb';
export const BUTTON_LABEL_BACK = 'button_label_back';
export const BUTTON_LABEL_YES = 'button_label_yes';
export const BUTTON_LABEL_NO = 'button_label_no';
export const SPRITE_BUTTON_CLICKED_LABEL_OFFSET_Y = 2;
export const SPRITE_BUTTON_ROW_SPACING = 4;

export const TURN_COMMAND_PANEL_BOTTOM_MARGIN = 40;

export const ALLOWED_TILE_MOVE_COLOR = 0x86c64f;
export const ALLOWED_TILE_MOVE_ALPHA = 0.65;
export const ALLOWED_TILE_MOVE_SELECTED_COLOR = 0xdaedca;

export const ALLOWED_TILE_BOMB_COLOR = 0xe69138;
export const ALLOWED_TILE_BOMB_ALPHA = 0.65;
export const ALLOWED_TILE_BOMB_SELECTED_COLOR = 0xf7dec3;

export const CONFIRM_DIALOG_WIDTH = 240;
export const CONFIRM_DIALOG_HEIGHT = 144;
export const CONFIRM_DIALOG_DIM_COLOR = 0x1a1a1a;
export const CONFIRM_DIALOG_DIM_ALPHA = 0.6;

export const UNIT_MOVE_TWEEN_DURATION = 500;

export const TURN_PANEL_WIDTH = 96;
export const TURN_PANEL_HEIGHT = 48;
export const TURN_PANEL_TOP_MARGIN = 2;
export const TURN_PANEL_MARGIN = 48;
export const TURN_PANEL_PADDING = 8;
export const TURN_PANEL_TEXT_COLOR = 0xeeeeee;
export const SUDDEN_DEATH_COLOR = 0xff0000;

export const LIFECYCLE_BUTTON_WIDTH = 320;
export const LIFECYCLE_BUTTON_HEIGHT = 72;
export const LIFECYCLE_BUTTON_HEIGHT_SMALL = 44;
export const RESOLVE_BUTTON_LABEL = 'End this turn';

export const ERROR_PANEL_X = TURN_PANEL_MARGIN;
export const ERROR_PANEL_Y = TURN_PANEL_MARGIN + TURN_PANEL_HEIGHT + 16;
export const ERROR_PANEL_WIDTH = 240;
export const ERROR_PANEL_HEIGHT = 400;
export const ERROR_PANEL_PADDING = 8;
export const ERROR_PANEL_BG_COLOR = 0x1a1a1a;
export const ERROR_PANEL_BG_ALPHA = 0.75;

export const TURN_BANNER_HEIGHT = 144;
export const TURN_BANNER_FONT_SIZE = 48;
export const TURN_BANNER_TEXT_COLOR = 0xffffff;
export const TURN_BANNER_HOLD_MS = 2000;

export const CPU_POLL_BACKOFF_MS = [250, 500, 1000, 2000] as const;
export const CPU_POLL_BUDGET_MS = 30000;
export const CPU_PLAN_RESOLVE_HOLD_MS = 600;
export const CPU_PLAN_EVENT_STEP_MS = 500;

export const SUDDEN_DEATH_CUTSCENE_DURATION_MS = 3000;
export const SUDDEN_DEATH_PULSE_HALF_MS = 250;
export const SUDDEN_DEATH_PULSE_PEAK_ALPHA = 0.9;
export const SUDDEN_DEATH_BOMB_DROP_DELAY_MS = 2000;
export const SUDDEN_DEATH_BOMB_DROP_DURATION_MS = 2000;

export const DEPTH_SUDDEN_DEATH_OVERLAY = 60;
export const DEPTH_SUDDEN_DEATH_BOMB = 65;
export const DEPTH_TURN_BANNER = 70;

export const DEPTH_MATCH_SUMMARY_PANEL = 35;
export const MATCH_SUMMARY_BUTTON_SIZE = 48;
export const MATCH_SUMMARY_BUTTON_LABEL = '≡';
export const MATCH_SUMMARY_BUTTON_TEXT_COLOR = 0xffffff;
export const MATCH_SUMMARY_BUTTON_ICON_FONT_SIZE = 48;
export const MATCH_SUMMARY_PANEL_WIDTH = 576;
export const MATCH_SUMMARY_PANEL_HEIGHT = 288;
export const MATCH_SUMMARY_TEXT_FONT_SIZE = 32;
export const MATCH_SUMMARY_TEXT_COLOR = 0xffffff;
export const MATCH_SUMMARY_TOP_SECTION_RATIO = 0.15;
export const MATCH_SUMMARY_MID_SECTION_RATIO = 0.35;
export const MATCH_SUMMARY_SECTION_GAP = 12;
export const MATCH_SUMMARY_TEAM_BADGE_WIDTH = 76;
export const MATCH_SUMMARY_TEAM_BADGE_HEIGHT = 20;
export const MATCH_SUMMARY_TEAM_BADGE_CORNER_RADIUS = 8;
export const MATCH_SUMMARY_BUTTON_WIDTH = 256;
export const MATCH_SUMMARY_BUTTON_HEIGHT = 20;
export const RESET_BUTTON_LABEL = 'Reset this turn';
export const SURRENDER_BUTTON_LABEL = 'Surrender';
export const BACK_BUTTON_LABEL = 'Back';
export const CONFIRM_TEXT_RESOLVE = 'Confirm to end this turn?';
export const CONFIRM_TEXT_RESET = 'All turn actions will reset. Confirm?';
export const CONFIRM_TEXT_SURRENDER = 'Confirm to surrender?';

export const SETTINGS_SCENE_MARGIN = 12;
export const SETTINGS_REGION_HEIGHT = 42;
export const SETTINGS_HEADER_SPACER = 24;
export const SETTINGS_TEXT_FONT_SIZE = 20;
export const SETTINGS_CORNER_RADIUS = 4;

export const BACK_BUTTON_SIZE = 32;
export const BACK_BUTTON_COLOR = 0x4c4c4c;
// U+2B90, a symbol not an emoji — can tofu on sparse font coverage.
export const BACK_BUTTON_GLYPH = '⮐';
export const BACK_BUTTON_GLYPH_FONT_SIZE = 32;

export const UNIT_PAGE_TEAM_BADGE_WIDTH = 48;
export const UNIT_PAGE_TEAM_BADGE_HEIGHT = 24;
export const UNIT_PAGE_TEAM_BADGE_CORNER_RADIUS = 4;
export const UNIT_PAGE_TITLE_GAP = 4;

export const FORMATION_PANEL_HEIGHT_RATIO = 0.35;
export const UNIT_FORMATION_HEADER_FONT_SIZE = 32;
export const UNIT_SLOT_SIZE = 48;
export const UNIT_SLOT_SPACING = 6;
export const UNIT_SLOT_ORDER_LABEL_INSET = 2;
export const UNIT_SLOT_ORDER_LABEL_FONT_SIZE = 20;
// Renders above the slot's unit sprite, which stays at Phaser's default depth (0).
export const DEPTH_UNIT_SLOT_LABEL = 1;

export const UNIT_CARD_WIDTH = 90;
export const UNIT_CARD_HEIGHT = 100;
export const UNIT_CARD_PADDING = 6;
export const UNIT_CARD_SPACING = 6;
export const UNIT_CARD_SPRITE_SIZE = 48;
export const UNIT_CARD_NAME_GAP = 8;
export const UNIT_CARD_LINE_GAP = 16;
export const UNIT_CARD_STAT_GLYPH_GAP = 6;
export const ARCHETYPES_PER_ROW = 4;

export const SETTINGS_NAV_BUTTON_WIDTH = 72;
export const SETTINGS_NAV_BUTTON_HEIGHT = 48;
export const NEXT_BUTTON_LABEL = 'NEXT →';
export const START_MATCH_BUTTON_LABEL = 'Start Match';

export const STAGES_PANEL_WIDTH_RATIO = 0.6;
export const STAGE_DETAIL_PANEL_WIDTH_RATIO = 0.4;
export const STAGE_PANEL_PADDING = 6;

export const STAGE_CARD_SIZE = 80;
export const STAGE_CARD_PADDING = 6;
export const STAGE_CARD_SPACING = 6;
export const STAGE_CARD_NAME_FONT_SIZE = 16;
// Rounded up from 2 so it stays distinguishable from PANEL_BUTTON_BORDER_WIDTH.
export const STAGE_CARD_SELECTED_BORDER_WIDTH = 3;

export const STAGE_DETAIL_INNER_PANEL_SIZE_RATIO = 0.8;
export const STAGE_DETAIL_INNER_PANEL_PADDING = 6;
export const STAGE_DETAIL_ROW_FONT_SIZE = 20;
export const STAGE_DETAIL_ROW_GAP = 6;
export const STAGE_DETAIL_DESCRIPTION_LINES = 2;
export const STAGE_DETAIL_WIDTH_HEIGHT_GAP = 6;

export const MAX_TURNS_ARROW_LEFT_LABEL = '❰';
export const MAX_TURNS_ARROW_RIGHT_LABEL = '❱';
export const MAX_TURNS_ARROW_INSET = 12;
export const MAX_TURNS_RECOMMENDED_GLYPH = '🌟';
export const MAX_TURNS_RECOMMENDED_FONT_SIZE = 12;
// Fixed slot for the recommended glyph so toggling it never shifts the layout.
export const MAX_TURNS_RECOMMENDED_GLYPH_GAP = 10;

export const TITLE_TOP_MARGIN = 24;
export const TITLE_FONT_SIZE = 44;
export const TITLE_GAME_MODE_FONT_SIZE = 20;
export const TITLE_SUBMENU_LINE_GAP = 36;
export const TITLE_COPYRIGHT_BOTTOM_MARGIN = 6;
export const TITLE_COPYRIGHT_FONT_SIZE = 12;
export const TITLE_HOVER_BOMB_GAP = 12;

export const DEPTH_VICTORY_CUTSCENE = 80;
export const VICTORY_SUBTITLE_FONT_SIZE = 36;
export const VICTORY_TITLE_FONT_SIZE = 48;
export const VICTORY_SUBTITLE_OFFSET_Y = -24;
export const VICTORY_TITLE_OFFSET_Y = 24;
export const VICTORY_SUBTITLE_X_SHIFT_RATIO = 0.05;
export const VICTORY_BUTTON_DELAY_MS = 2000;
export const VICTORY_BUTTON_GAP = 16;
