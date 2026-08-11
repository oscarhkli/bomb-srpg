import { TILE_SIZE } from '../constants';

// Depth bands local to blast/fire rendering — between DEPTH_GRID(0) and DEPTH_OCCUPANT(10)
// in the shared web/src/constants.ts (blast under occupants; fire above occupants, since it
// renders "on top of the blast and unit/softblock").
export const DEPTH_BLAST = 5;
export const DEPTH_FIRE = 15;

// Blast/fire visuals were tuned at Phase 3's 48px tile; both scale with TILE_SIZE so they stay
// proportionate if it changes again, rather than needing a separate manual rescale each time.
const PHASE_3_TILE_SIZE = 48;
const PHASE_3_BLAST_BEAM_WIDTH = 32;
const PHASE_3_FIRE_SHAPE_SIZE = 42;

// Blast timing/visuals (placeholder values — tunable later without touching sequencing rules)
export const BLAST_SPEED_MS_PER_TILE = 60;
export const BLAST_DURATION_MS = 3000;
export const BLAST_BEAM_WIDTH = Math.round(
  (PHASE_3_BLAST_BEAM_WIDTH * TILE_SIZE) / PHASE_3_TILE_SIZE
);
export const BLAST_COLOR_OUTER = 0xf58e27;
export const BLAST_COLOR_MID = 0xf5ee27;
export const BLAST_COLOR_INNER = 0xfcfabb;
export const BLAST_ALPHA = 0.6;

export const FIRE_GLYPH = '🔥';
export const FIRE_SHAPE_SIZE = Math.round(
  (PHASE_3_FIRE_SHAPE_SIZE * TILE_SIZE) / PHASE_3_TILE_SIZE
);
export const FIRE_DURATION_MS = 5000;

export const BOMB_COUNTDOWN_ZERO_COLOR = 0xff0000;
export const BOMB_COUNTDOWN_TEXT_COLOR = 0xffffff;
