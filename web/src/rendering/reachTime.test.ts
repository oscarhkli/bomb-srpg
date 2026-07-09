import { describe, it, expect } from 'vitest';
import { cardinalDistance, reachTimeMs } from './reachTime';

describe('cardinalDistance', () => {
  it('measures distance along a shared row', () => {
    expect(cardinalDistance({ x: 0, y: 0 }, { x: 3, y: 0 })).toBe(3);
  });

  it('measures distance along a shared column', () => {
    expect(cardinalDistance({ x: 2, y: 5 }, { x: 2, y: 2 })).toBe(3);
  });
});

describe('reachTimeMs', () => {
  it('scales cardinal distance by BLAST_SPEED_MS_PER_TILE (60ms/tile)', () => {
    // 4 tiles away, 60ms/tile => 240ms — a worked example independent of the implementation.
    expect(reachTimeMs({ x: 0, y: 0 }, { x: 0, y: 4 })).toBe(240);
  });
});
