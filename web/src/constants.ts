import type { TerrainType } from './types/api';

// Fallback color for an unconfigured/unknown team ID.
export const TEAM_COLOR_FALLBACK = 0x4c4c4c;
// Default fade in/out duration used across this UI.
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
// Button labels/prompts, unprefixed like RESOLVE_BUTTON_LABEL.
export const RESET_BUTTON_LABEL = 'Reset this turn';
export const SURRENDER_BUTTON_LABEL = 'Surrender';
export const BACK_BUTTON_LABEL = 'Back';
export const CONFIRM_TEXT_RESOLVE = 'Confirm to end this turn?';
export const CONFIRM_TEXT_RESET = 'All turn actions will reset. Confirm?';
export const CONFIRM_TEXT_SURRENDER = 'Confirm to surrender?';

// MatchSettingsScene (p3-spec009-stage) — chrome shared by every Page.
export const SETTINGS_SCENE_MARGIN = 24;
// Both HeaderRegion and NavRegion are this tall.
export const SETTINGS_REGION_HEIGHT = 84;
// Gap between BackButton and the active Page's title in HeaderRegion.
export const SETTINGS_HEADER_SPACER = 48;
// Default text size for this scene (page titles, UnitCard stats/name).
export const SETTINGS_TEXT_FONT_SIZE = 24;
// Corner radius shared by this scene's rounded shapes (BackButton, UnitSlot, UnitCard).
export const SETTINGS_CORNER_RADIUS = 8;

export const BACK_BUTTON_SIZE = 64;
export const BACK_BUTTON_COLOR = 0x4c4c4c;
// U+2B90, a symbol not an emoji — can tofu on sparse font coverage.
export const BACK_BUTTON_GLYPH = '⮐';
export const BACK_BUTTON_GLYPH_FONT_SIZE = 36;

// UnitPage's TeamBadge (header): P{X} on a TeamColor rounded-rect.
export const UNIT_PAGE_TEAM_BADGE_WIDTH = 96;
export const UNIT_PAGE_TEAM_BADGE_HEIGHT = 48;
export const UNIT_PAGE_TEAM_BADGE_CORNER_RADIUS = 8;
// Gap between the TeamBadge and the "Unit Selection" title text.
export const UNIT_PAGE_TITLE_GAP = 8;

// FormationPanel — the top band of UnitPage's body, full width.
export const FORMATION_PANEL_HEIGHT_RATIO = 0.35;
export const UNIT_FORMATION_HEADER_FONT_SIZE = 36;
export const UNIT_SLOT_SIZE = 96;
export const UNIT_SLOT_SPACING = 12;
export const UNIT_SLOT_ORDER_LABEL_INSET = 4;
// Matches SETTINGS_TEXT_FONT_SIZE.
export const UNIT_SLOT_ORDER_LABEL_FONT_SIZE = 24;
// The order-number label must render above the slot's unit sprite (they overlap); the sprite
// itself is left at Phaser's default depth (0).
export const DEPTH_UNIT_SLOT_LABEL = 1;

// ArchetypesPanel / UnitCard
export const UNIT_CARD_WIDTH = 180;
export const UNIT_CARD_HEIGHT = 200;
export const UNIT_CARD_PADDING = 12;
export const UNIT_CARD_SPACING = 12;
export const UNIT_CARD_SPRITE_SIZE = 96;
// Vertical gap between the sprite's bottom edge and the name text below it.
export const UNIT_CARD_NAME_GAP = 16;
// Vertical gap between each subsequent text line (name -> stats -> skill).
export const UNIT_CARD_LINE_GAP = 32;
// Extra gap between archetype.speed and the 💣 glyph on UnitCard's stat line.
export const UNIT_CARD_STAT_GLYPH_GAP = 12;
export const ARCHETYPES_PER_ROW = 4;

// NavRegion buttons (NextButton / StartMatchButton) — pill(144, 96), overriding
// PANEL_BUTTON_WIDTH/HEIGHT while keeping PANEL_BUTTON_* fill/border colors.
export const SETTINGS_NAV_BUTTON_WIDTH = 144;
export const SETTINGS_NAV_BUTTON_HEIGHT = 96;
export const NEXT_BUTTON_LABEL = 'NEXT →';
export const START_MATCH_BUTTON_LABEL = 'Start Match';

// StagePage — StagesPanel / StageDetailPanel split the body region 60/40.
export const STAGES_PANEL_WIDTH_RATIO = 0.6;
export const STAGE_DETAIL_PANEL_WIDTH_RATIO = 0.4;
export const STAGE_PANEL_PADDING = 12;

// StagesPanel / StageCard
export const STAGE_CARD_SIZE = 160;
export const STAGE_CARD_PADDING = 12;
export const STAGE_CARD_SPACING = 12;
export const STAGE_CARD_NAME_FONT_SIZE = 36;
// Reuses PANEL_BUTTON_BORDER_COLOR (0xdc9e23) for the selected-card border.
export const STAGE_CARD_SELECTED_BORDER_WIDTH = 4;

// StageDetailPanel / InnerPanel
export const STAGE_DETAIL_INNER_PANEL_SIZE_RATIO = 0.8;
export const STAGE_DETAIL_INNER_PANEL_PADDING = 12;
// Matches SETTINGS_TEXT_FONT_SIZE.
export const STAGE_DETAIL_ROW_FONT_SIZE = 24;
export const STAGE_DETAIL_ROW_GAP = 12;
// Description row reserves 2 lines of height (1 extra for wrapping), regardless of whether the
// current description actually wraps.
export const STAGE_DETAIL_DESCRIPTION_LINES = 2;
// Horizontal offset of the width/height numbers from the centered "x" glyph on that row.
export const STAGE_DETAIL_WIDTH_HEIGHT_GAP = 12;

// MaxTurnsSelector — inside StageDetailPanel's InnerPanel, 4th row.
export const MAX_TURNS_ARROW_LEFT_LABEL = '❰';
export const MAX_TURNS_ARROW_RIGHT_LABEL = '❱';
// Fixed inset of each arrow from InnerPanel's nearest edge.
export const MAX_TURNS_ARROW_INSET = 24;
export const MAX_TURNS_RECOMMENDED_GLYPH = '🌟';
export const MAX_TURNS_RECOMMENDED_FONT_SIZE = 16;
// Reserved gap between the maxTurns value and the (possibly-hidden) recommended-glyph slot, so
// the glyph's fixed slot never shifts other elements when toggled visible/hidden.
export const MAX_TURNS_RECOMMENDED_GLYPH_GAP = 20;

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
