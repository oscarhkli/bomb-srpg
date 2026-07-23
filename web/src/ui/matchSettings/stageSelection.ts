// Pure, Phaser-free stage/maxTurns selection logic for StagePage.
import type { StagePreset } from '../../types/api';

// Cyclic maxTurns options. 0 = instant sudden death (displayed as 💣).
const MAX_TURNS_OPTIONS = [0, 15, 20, 30, 45, 60] as const;

export function formatMaxTurns(value: number): string {
  return value === 0 ? '💣' : String(value);
}

// Steps `current` to the next/previous MAX_TURNS_OPTIONS entry, wrapping at both ends.
export function cycleMaxTurns(current: number, delta: 1 | -1): number {
  const index = MAX_TURNS_OPTIONS.indexOf(current as (typeof MAX_TURNS_OPTIONS)[number]);
  const len = MAX_TURNS_OPTIONS.length;
  const nextIndex = ((index === -1 ? 0 : index) + delta + len) % len;
  return MAX_TURNS_OPTIONS[nextIndex]!;
}

// Index of the preset named `name`, or 0 (first StageCard) if none matches.
export function findStagePresetIndex(stagePresets: StagePreset[], name: string): number {
  const index = stagePresets.findIndex(preset => preset.name === name);
  return index === -1 ? 0 : index;
}
