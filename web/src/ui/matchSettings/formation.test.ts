import { describe, it, expect } from 'vitest';
import type { TeamSlot, UnitRole } from '../../types/api';
import {
  NO_UNIT,
  SLOT_DISPLAY_ORDER_P1,
  SLOT_DISPLAY_ORDER_P2,
  deserializeTeams,
  serializeTeams,
  lowestFreeSlot,
  occupiedCount,
} from './formation';

function slot(archetype: string, role: UnitRole = 'Normal'): TeamSlot {
  return { archetype, role };
}

describe('SLOT_DISPLAY_ORDER_P1', () => {
  it('maps left-to-right rendering to order numbers 4,2,1,3,5', () => {
    expect(SLOT_DISPLAY_ORDER_P1).toEqual([3, 1, 0, 2, 4]);
    const orderNumbers = SLOT_DISPLAY_ORDER_P1.map(i => i + 1);
    expect(orderNumbers).toEqual([4, 2, 1, 3, 5]);
  });
});

describe('SLOT_DISPLAY_ORDER_P2', () => {
  it('maps left-to-right rendering to order numbers 5,3,1,2,4 (mirrors P1)', () => {
    expect(SLOT_DISPLAY_ORDER_P2).toEqual([4, 2, 0, 1, 3]);
    const orderNumbers = SLOT_DISPLAY_ORDER_P2.map(i => i + 1);
    expect(orderNumbers).toEqual([5, 3, 1, 2, 4]);
  });
});

describe('deserializeTeams', () => {
  it.each<[string, TeamSlot[], string[]]>([
    ['lone King', [slot('King', 'King')], ['King', NO_UNIT, NO_UNIT, NO_UNIT, NO_UNIT]],
    [
      'full team',
      [slot('King', 'King'), slot('Fighter'), slot('Witch'), slot('Witch'), slot('Bandit')],
      ['King', 'Fighter', 'Witch', 'Witch', 'Bandit'],
    ],
    [
      'interior gap preserved',
      [slot('King', 'King'), slot('Fighter'), slot(NO_UNIT), slot('Witch'), slot('Witch')],
      ['King', 'Fighter', NO_UNIT, 'Witch', 'Witch'],
    ],
    // index 0 is always forced to King even if the input says otherwise (defensive; backend
    // never sends this, but deserialize shouldn't trust it blindly).
    ['index 0 forced to King', [slot('Fighter')], ['King', NO_UNIT, NO_UNIT, NO_UNIT, NO_UNIT]],
  ])('%s', (_name, input, expected) => {
    expect(deserializeTeams(input)).toEqual(expected);
  });
});

describe('serializeTeams', () => {
  it.each<[string, string[], TeamSlot[]]>([
    [
      'lone King trims to length 1',
      ['King', NO_UNIT, NO_UNIT, NO_UNIT, NO_UNIT],
      [slot('King', 'King')],
    ],
    [
      'full team unchanged',
      ['King', 'Fighter', 'Witch', 'Witch', 'Bandit'],
      [slot('King', 'King'), slot('Fighter'), slot('Witch'), slot('Witch'), slot('Bandit')],
    ],
    [
      'interior gap kept, no trailing NO_UNIT to trim',
      ['King', 'Fighter', NO_UNIT, 'Witch', 'Witch'],
      [slot('King', 'King'), slot('Fighter'), slot(NO_UNIT), slot('Witch'), slot('Witch')],
    ],
    [
      'trailing NO_UNIT trimmed even after an interior gap',
      ['King', 'Fighter', NO_UNIT, 'Witch', NO_UNIT],
      [slot('King', 'King'), slot('Fighter'), slot(NO_UNIT), slot('Witch')],
    ],
    [
      'multiple trailing NO_UNIT trimmed',
      ['King', 'Fighter', NO_UNIT, NO_UNIT, NO_UNIT],
      [slot('King', 'King'), slot('Fighter')],
    ],
  ])('%s', (_name, input, expected) => {
    expect(serializeTeams(input)).toEqual(expected);
  });
});

describe('lowestFreeSlot', () => {
  it.each<[string, string[], number | null]>([
    ['all free', ['King', NO_UNIT, NO_UNIT, NO_UNIT, NO_UNIT], 1],
    [
      'first non-King slot occupied, next is free',
      ['King', 'Fighter', NO_UNIT, NO_UNIT, NO_UNIT],
      2,
    ],
    [
      'skips occupied slots to find the lowest free one',
      ['King', 'Fighter', NO_UNIT, 'Witch', NO_UNIT],
      2,
    ],
    ['full team has no free slot', ['King', 'Fighter', 'Witch', 'Witch', 'Bandit'], null],
  ])('%s', (_name, slots, expected) => {
    expect(lowestFreeSlot(slots)).toBe(expected);
  });
});

describe('occupiedCount', () => {
  it.each<[string, string[], number]>([
    ['lone King', ['King', NO_UNIT, NO_UNIT, NO_UNIT, NO_UNIT], 1],
    ['King + 1', ['King', 'Fighter', NO_UNIT, NO_UNIT, NO_UNIT], 2],
    ['full team', ['King', 'Fighter', 'Witch', 'Witch', 'Bandit'], 5],
    ['gap counted correctly', ['King', 'Fighter', NO_UNIT, 'Witch', 'Witch'], 4],
  ])('%s', (_name, slots, expected) => {
    expect(occupiedCount(slots)).toBe(expected);
  });
});
