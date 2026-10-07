import { describe, it, expect } from 'vitest';
import { createMockGraphics } from './setup';
import { makeBombGraphics as bombGraphics } from './sceneHelpers';
import { makeState as state } from './fixtures';
import { occupantsMatch } from './occupantsMatch';

describe('occupantsMatch', () => {
  it('is true when live units, bombs, and softBlocks all have a matching graphics entry', () => {
    const fresh = state({
      units: [{ id: 1, hp: 1 } as never, { id: 2, hp: 1 } as never],
      bombs: [{ id: 10 } as never],
      softBlocks: [{ id: 20 } as never],
    });

    const match = occupantsMatch(
      fresh,
      new Map([
        [1, createMockGraphics() as never],
        [2, createMockGraphics() as never],
      ]),
      new Map([[10, bombGraphics()]]),
      new Map([[20, createMockGraphics() as never]])
    );

    expect(match).toBe(true);
  });

  it('ignores dead units (hp 0) when comparing against the unit graphics map', () => {
    const fresh = state({
      units: [{ id: 1, hp: 1 } as never, { id: 2, hp: 0 } as never],
    });

    expect(
      occupantsMatch(fresh, new Map([[1, createMockGraphics() as never]]), new Map(), new Map())
    ).toBe(true);
  });

  it.each([
    ['unit', 'units'],
    ['bomb', 'bombs'],
    ['softBlock', 'softBlocks'],
  ] as const)('is false when a %s has no graphics entry', (_label, collection) => {
    const fresh = state({ [collection]: [{ id: 1, hp: 1 } as never] });
    expect(occupantsMatch(fresh, new Map(), new Map(), new Map())).toBe(false);
  });

  it('is false when the graphics map holds an occupant that truth no longer has', () => {
    expect(
      occupantsMatch(state(), new Map([[1, createMockGraphics() as never]]), new Map(), new Map())
    ).toBe(false);
  });
});
