import type { TeamSlot } from '../../api/types';

export const NO_UNIT = 'NO_UNIT';

const KING_SLOT = 0;

export const SLOT_DISPLAY_ORDER_P1 = [3, 1, 0, 2, 4] as const;

export const SLOT_DISPLAY_ORDER_P2 = [4, 2, 0, 1, 3] as const;

export function deserializeTeams(teams: TeamSlot[]): string[] {
  const slots = Array.from({ length: 5 }, (_, i) => teams[i]?.archetype ?? NO_UNIT);
  slots[KING_SLOT] = 'King';
  return slots;
}

export function serializeTeams(slots: string[]): TeamSlot[] {
  const result = [...slots];
  while (result.length > 1 && result[result.length - 1] === NO_UNIT) {
    result.pop();
  }
  return result.map((archetype, i) => ({ archetype, role: i === KING_SLOT ? 'King' : 'Normal' }));
}

export function lowestFreeSlot(slots: string[]): number | null {
  for (let i = KING_SLOT + 1; i < slots.length; i++) {
    if (slots[i] === NO_UNIT) {
      return i;
    }
  }
  return null;
}

export function occupiedCount(slots: string[]): number {
  return slots.filter(s => s !== NO_UNIT).length;
}
