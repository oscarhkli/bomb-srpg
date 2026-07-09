// Depth bands local to blast/fire rendering — between DEPTH_GRID(0) and DEPTH_OCCUPANT(10)
// in the shared web/src/constants.ts (blast under occupants; fire above occupants, since it
// renders "on top of the blast and unit/softblock").
export const DEPTH_BLAST = 5;
export const DEPTH_FIRE = 15;

// Blast timing/visuals (placeholder values — tunable later without touching sequencing rules)
export const BLAST_SPEED_MS_PER_TILE = 60;
export const BLAST_DURATION_MS = 3000;
export const BLAST_BEAM_WIDTH = 32;
export const BLAST_COLOR_OUTER = 0xf58e27;
export const BLAST_COLOR_MID = 0xf5ee27;
export const BLAST_COLOR_INNER = 0xfcfabb;
export const BLAST_ALPHA = 0.6;

// Fire shape (unitDamaged / softBlockDestroyed)
export const FIRE_SHAPE_SIZE = 42;
export const FIRE_ALPHA = 0.7;
export const FIRE_DURATION_MS = 5000;
