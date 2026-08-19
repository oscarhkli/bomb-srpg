// Pure, Phaser-free formation logic for UnitPage: slot (de)serialization, display order, and
// the click-placement queries that drive FormationPanel/ArchetypesPanel.
import type { TeamSlot } from '../../types/api';

export const NO_UNIT = 'NO_UNIT';

// Left-to-right render order so 1-based order numbers read 4, 2, 1, 3, 5 (mirrors
// engine/presets.go's P1StartingPositions center-out ordering). Index 0 is always King.
export const SLOT_DISPLAY_ORDER_P1 = [3, 1, 0, 2, 4] as const;

// Team 2 faces Team 1, so its order numbers mirror P1's: 5, 3, 1, 2, 4 (P2StartingPositions).
export const SLOT_DISPLAY_ORDER_P2 = [4, 2, 0, 1, 3] as const;

// Builds the 5-slot array from gameCfg.p{X}Slots, defaulting missing slots to NO_UNIT;
// index 0 is always forced to 'King'.
export function deserializeTeams(teams: TeamSlot[]): string[] {
  const slots = Array.from({ length: 5 }, (_, i) => teams[i]?.archetype ?? NO_UNIT);
  slots[0] = 'King';
  return slots;
}

// Inverse of deserializeTeams: trims trailing NO_UNIT entries (a lone King serializes to
// ['King']); interior gaps are kept as explicit NO_UNIT. Slot 0 is always King, every other
// slot (including NO_UNIT ones) is Normal — Boss-mode formations aren't built through this UI.
export function serializeTeams(slots: string[]): TeamSlot[] {
  const result = [...slots];
  while (result.length > 1 && result[result.length - 1] === NO_UNIT) {
    result.pop();
  }
  return result.map((archetype, i) => ({ archetype, role: i === 0 ? 'King' : 'Normal' }));
}

// Lowest non-King slot index that's NO_UNIT, or null if the formation is full.
export function lowestFreeSlot(slots: string[]): number | null {
  for (let i = 1; i < slots.length; i++) {
    if (slots[i] === NO_UNIT) {
      return i;
    }
  }
  return null;
}

// Count of occupied (non-NO_UNIT) slots, King included.
export function occupiedCount(slots: string[]): number {
  return slots.filter(s => s !== NO_UNIT).length;
}
