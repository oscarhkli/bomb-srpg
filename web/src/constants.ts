import type { TerrainType } from './types/api';

// Shared across multiple components (not scoped to any single one below).
// Fallback color whenever a team ID isn't in TEAM_COLORS — used both for a genuine
// "unconfigured team" bug guard (TurnBanner, TurnPanel, boardRenderer) and for VictoryCutscene's
// draw case (winnerTeamId -1, which is never a TEAM_COLORS key by construction).
export const TEAM_COLOR_FALLBACK = 0x4c4c4c;
// Duration for every fade in/out in this UI (TurnBanner, VictoryCutscene, MatchScene's
// rematch/return-to-settings camera transitions) — 200ms unless a specific effect calls for
// something else.
export const FADE_MS = 200;

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
// since overlays/panels/dialogs must render above the board contents.
export const DEPTH_GRID = 0;
export const DEPTH_OCCUPANT = 10;
export const DEPTH_ALLOWED_TILE_OVERLAY = 20;
export const DEPTH_TURN_COMMAND_PANEL = 30;
export const DEPTH_CONFIRM_DIALOG = 40;
export const DEPTH_ERROR_PANEL = 50;

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

export const CONFIRM_DIALOG_WIDTH = 240;
export const CONFIRM_DIALOG_HEIGHT = 144;
export const CONFIRM_DIALOG_DIM_COLOR = 0x1a1a1a;
export const CONFIRM_DIALOG_DIM_ALPHA = 0.6;

export const UNIT_MOVE_TWEEN_DURATION = 500;

// TurnPanel
export const TURN_PANEL_WIDTH = 96;
export const TURN_PANEL_HEIGHT = 48;
export const TURN_PANEL_MARGIN = 48;
export const TURN_PANEL_PADDING = 8;
export const TURN_PANEL_TEXT_COLOR = 0xeeeeee;
export const SUDDEN_DEATH_COLOR = 0xff0000;

// Shared pill-button size for the "big" lifecycle-style buttons: ResolveTurnButton,
// ResetTurnButton, SurrenderButton, MatchSummaryPanelBackButton, and VictoryCutscene's
// Rematch/Return-to-Settings — distinct from TurnCommandPanel's smaller PANEL_BUTTON_* size.
export const LIFECYCLE_BUTTON_WIDTH = 320;
export const LIFECYCLE_BUTTON_HEIGHT = 72;
export const RESOLVE_BUTTON_LABEL = 'End this turn';

// Error panel — fixed left-side panel so error text is always legible instead of overlapping
// at screen-center.
export const ERROR_PANEL_X = TURN_PANEL_MARGIN;
export const ERROR_PANEL_Y = TURN_PANEL_MARGIN + TURN_PANEL_HEIGHT + 16;
export const ERROR_PANEL_WIDTH = 240;
export const ERROR_PANEL_HEIGHT = 400;
export const ERROR_PANEL_PADDING = 8;
export const ERROR_PANEL_BG_COLOR = 0x1a1a1a;
export const ERROR_PANEL_BG_ALPHA = 0.75;

// TurnBanner
export const TURN_BANNER_HEIGHT = 144;
export const TURN_BANNER_FONT_SIZE = 48;
export const TURN_BANNER_TEXT_COLOR = 0xffffff;
export const TURN_BANNER_HOLD_MS = 2000;

// SuddenDeathCutscene
export const SUDDEN_DEATH_CUTSCENE_DURATION_MS = 3000;
export const SUDDEN_DEATH_PULSE_HALF_MS = 250;
export const SUDDEN_DEATH_PULSE_PEAK_ALPHA = 0.9;
export const SUDDEN_DEATH_BOMB_DROP_DELAY_MS = 2000;
export const SUDDEN_DEATH_BOMB_DROP_DURATION_MS = 2000;

// Depth bands above DEPTH_ERROR_PANEL
export const DEPTH_SUDDEN_DEATH_OVERLAY = 60;
export const DEPTH_SUDDEN_DEATH_BOMB = 65;
export const DEPTH_TURN_BANNER = 70;

// MatchSummaryButton / MatchSummaryPanel
export const DEPTH_MATCH_SUMMARY_PANEL = 35; // above DEPTH_TURN_COMMAND_PANEL, below DEPTH_CONFIRM_DIALOG
export const MATCH_SUMMARY_BUTTON_SIZE = 48;
export const MATCH_SUMMARY_BUTTON_LABEL = '≡';
export const MATCH_SUMMARY_BUTTON_TEXT_COLOR = 0xffffff;
export const MATCH_SUMMARY_BUTTON_ICON_FONT_SIZE = 48;
export const MATCH_SUMMARY_PANEL_WIDTH = 720;
export const MATCH_SUMMARY_PANEL_HEIGHT = 640;
export const MATCH_SUMMARY_TEXT_FONT_SIZE = 36;
export const MATCH_SUMMARY_TEXT_COLOR = 0xffffff;
export const MATCH_SUMMARY_TOP_SECTION_RATIO = 0.15;
export const MATCH_SUMMARY_MID_SECTION_RATIO = 0.35;
// Vertical breathing room between the top (Stage/Max Turns) and mid (team stats) sections.
export const MATCH_SUMMARY_SECTION_GAP = 24;
// Smaller pill-button height shared by MatchSummaryPanel's TurnLifeCycleButtons and
// VictoryCutscene's Rematch/Return-to-Settings buttons — down from the full LIFECYCLE_BUTTON_HEIGHT
// (72px), same width, unscaled font.
export const LIFECYCLE_BUTTON_HEIGHT_SMALL = 44;
// P1/P2 badge behind the mid-section's team headers, filled with that team's TEAM_COLORS entry.
export const MATCH_SUMMARY_TEAM_BADGE_WIDTH = 96;
export const MATCH_SUMMARY_TEAM_BADGE_HEIGHT = 48;
export const MATCH_SUMMARY_TEAM_BADGE_CORNER_RADIUS = 8;
// Button label/prompt text, following RESOLVE_BUTTON_LABEL's un-prefixed naming — these describe
// the ResetTurnButton/SurrenderButton/BackButton concepts themselves, not MatchSummaryPanel's own
// layout (which is what the MATCH_SUMMARY_ prefix above is reserved for).
export const RESET_BUTTON_LABEL = 'Reset this turn';
export const SURRENDER_BUTTON_LABEL = 'Surrender';
export const BACK_BUTTON_LABEL = 'Back';
export const CONFIRM_TEXT_RESOLVE = 'Confirm to end this turn?';
export const CONFIRM_TEXT_RESET = 'All turn actions will reset. Confirm?';
export const CONFIRM_TEXT_SURRENDER = 'Confirm to surrender?';

// VictoryCutscene — above every other overlay, since it's the terminal screen.
export const DEPTH_VICTORY_CUTSCENE = 80;
// Named by size tier, not line position: the draw case's single "Draw Game" line uses the
// TITLE size, matching the non-draw case's 2nd (bigger) line, not a literal "line 2".
export const VICTORY_SUBTITLE_FONT_SIZE = 36;
export const VICTORY_TITLE_FONT_SIZE = 48;
// Vertical offset of each text line's center from the banner's own vertical center.
export const VICTORY_SUBTITLE_OFFSET_Y = -24;
export const VICTORY_TITLE_OFFSET_Y = 24;
// "Winner..." sits slightly left of dead-center, as a fraction of the canvas width;
// "Player {X}!" stays fully centered.
export const VICTORY_SUBTITLE_X_SHIFT_RATIO = 0.05;
export const VICTORY_BUTTON_DELAY_MS = 2000;
export const VICTORY_BUTTON_GAP = 16;
