import type { TeamSlot } from '../../api/types';

export const NO_UNIT = 'NO_UNIT';

export const SLOT_DISPLAY_ORDER_P1 = [3, 1, 0, 2, 4] as const;

export const SLOT_DISPLAY_ORDER_P2 = [4, 2, 0, 1, 3] as const;

// Builds the 5-slot array from a team's slots; missing slots are NO_UNIT and slot 0 is King.
export function deserializeTeams(teams: TeamSlot[]): string[] {
  const slots = Array.from({ length: 5 }, (_, i) => teams[i]?.archetype ?? NO_UNIT);
  slots[0] = 'King';
  return slots;
}

export function serializeTeams(slots: string[]): TeamSlot[] {
  const result = [...slots];
  while (result.length > 1 && result[result.length - 1] === NO_UNIT) {
    result.pop();
  }
  return result.map((archetype, i) => ({ archetype, role: i === 0 ? 'King' : 'Normal' }));
}

export function lowestFreeSlot(slots: string[]): number | null {
  for (let i = 1; i < slots.length; i++) {
    if (slots[i] === NO_UNIT) {
      return i;
    }
  }
  return null;
}

export function occupiedCount(slots: string[]): number {
  return slots.filter(s => s !== NO_UNIT).length;
}
