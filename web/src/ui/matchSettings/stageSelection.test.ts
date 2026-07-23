import { describe, it, expect } from 'vitest';
import { formatMaxTurns, cycleMaxTurns, findStagePresetIndex } from './stageSelection';
import type { StagePreset } from '../../types/api';

function makeStagePreset(overrides: Partial<StagePreset> = {}): StagePreset {
  return {
    name: 'Plain',
    description: 'A plain field.',
    width: 10,
    height: 10,
    maxTurns: 60,
    ...overrides,
  };
}

describe('formatMaxTurns', () => {
  it.each<[number, string]>([
    [0, '💣'],
    [15, '15'],
    [60, '60'],
  ])('formats %i as %s', (value, expected) => {
    expect(formatMaxTurns(value)).toBe(expected);
  });
});

describe('cycleMaxTurns', () => {
  it.each<[number, 1 | -1, number]>([
    [15, 1, 20],
    [20, -1, 15],
    [60, 1, 0], // wraps forward past the last option back to 💣 (0)
    [0, -1, 60], // wraps backward past 💣 (0) to the last option
  ])('cycling %i by %i yields %i', (current, delta, expected) => {
    expect(cycleMaxTurns(current, delta)).toBe(expected);
  });
});

describe('findStagePresetIndex', () => {
  const presets = [
    makeStagePreset({ name: 'Plain' }),
    makeStagePreset({ name: 'Divided' }),
    makeStagePreset({ name: 'Lava' }),
  ];

  it('finds the matching preset by name', () => {
    expect(findStagePresetIndex(presets, 'Divided')).toBe(1);
  });

  it('falls back to 0 when no preset matches the name', () => {
    expect(findStagePresetIndex(presets, 'Nonexistent')).toBe(0);
  });
});
